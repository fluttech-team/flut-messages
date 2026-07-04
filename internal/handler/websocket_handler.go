package handler

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/flutapp/chat-service/internal/domain"
	"github.com/flutapp/chat-service/internal/dto"
	"github.com/flutapp/chat-service/internal/hub"
	"github.com/flutapp/chat-service/internal/service"
	"github.com/flutapp/chat-service/internal/utils"
)

// WebSocketEvent represents an incoming WebSocket event
type WebSocketEvent struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// AckResponse represents the acknowledgment response for a WebSocket event
type AckResponse struct {
	Status  string      `json:"status"` // "ok" or "error"
	Code    string      `json:"code"`   // Error code (e.g., "FORBIDDEN", "BLOCKED")
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// BroadcastEvent represents an event to be broadcasted
type BroadcastEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// WebSocketHandler handles WebSocket events
type WebSocketHandler struct {
	msgService  service.MessageService
	convService service.ConversationService
	hub         *hub.Hub
	blockSvc    service.BlockService
}

// NewWebSocketHandler creates a new WebSocketHandler
func NewWebSocketHandler(
	msgService service.MessageService,
	convService service.ConversationService,
	hub *hub.Hub,
	blockSvc service.BlockService,
) *WebSocketHandler {
	return &WebSocketHandler{
		msgService:  msgService,
		convService: convService,
		hub:         hub,
		blockSvc:    blockSvc,
	}
}

// HandleEvent dispatches incoming WebSocket events to appropriate handlers
func (h *WebSocketHandler) HandleEvent(ctx context.Context, client *hub.Client, event WebSocketEvent) AckResponse {
	switch event.Type {
	case "join_conversation":
		return h.handleJoinConversation(ctx, client, event.Payload)
	case "send_message":
		return h.handleSendMessage(ctx, client, event.Payload)
	case "mark_as_read":
		return h.handleMarkAsRead(ctx, client, event.Payload)
	case "typing":
		return h.handleTyping(ctx, client, event.Payload)
	case "delete_message":
		return h.handleDeleteMessage(ctx, client, event.Payload)
	case "edit_message":
		return h.handleEditMessage(ctx, client, event.Payload)
	case "leave_conversation":
		return h.handleLeaveConversation(ctx, client, event.Payload)
	default:
		return AckResponse{
			Status:  "error",
			Code:    "UNKNOWN_EVENT",
			Message: "Unknown event type",
		}
	}
}

// handleJoinConversation - join_conversation event
func (h *WebSocketHandler) handleJoinConversation(ctx context.Context, client *hub.Client, payload json.RawMessage) AckResponse {
	var req struct {
		ConversationID string `json:"conversation_id"`
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		return AckResponse{
			Status:  "error",
			Code:    "INVALID_PAYLOAD",
			Message: "Invalid event payload",
		}
	}

	// Verify user is a participant in the conversation
	conv, err := h.convService.GetConversation(ctx, req.ConversationID, client.ID)
	if err != nil {
		return h.mapError(err)
	}

	// Join room with conversation ID
	h.hub.JoinRoom(client, req.ConversationID)

	// Broadcast online status
	h.hub.Broadcast([]string{req.ConversationID}, BroadcastEvent{
		Type: "user_online",
		Payload: map[string]interface{}{
			"user_id":           client.ID,
			"conversation_id":   req.ConversationID,
			"timestamp":         time.Now(),
		},
	})

	log.Printf("Client %s joined conversation %s", client.ID, req.ConversationID)

	return AckResponse{
		Status:  "ok",
		Code:    "SUCCESS",
		Message: "Joined conversation",
		Data: map[string]interface{}{
			"conversation_id": conv.ID.Hex(),
		},
	}
}

// handleSendMessage - send_message event
func (h *WebSocketHandler) handleSendMessage(ctx context.Context, client *hub.Client, payload json.RawMessage) AckResponse {
	var req struct {
		ConversationID string               `json:"conversation_id"`
		ReceiverID     string               `json:"receiver_id"`
		Text           string               `json:"text"`
		Attachments    []domain.Attachment `json:"attachments"`
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		return AckResponse{
			Status:  "error",
			Code:    "INVALID_PAYLOAD",
			Message: "Invalid event payload",
		}
	}

	// Send message using service
	msg, err := h.msgService.SendMessage(ctx, req.ConversationID, client.ID, req.ReceiverID, req.Text, req.Attachments)
	if err != nil {
		return h.mapError(err)
	}

	// Convert message to DTO
	msgDTO := dto.MessageToDTO(msg)

	// Update delivery status to "delivered"
	// (In production, this would be async, but for simplicity we do it here)
	deliveredTime := time.Now()
	msgDTO.DeliveredAt = &deliveredTime
	msgDTO.Status = "delivered"

	// Broadcast new message event
	h.hub.Broadcast([]string{req.ConversationID}, BroadcastEvent{
		Type:    "new_message",
		Payload: msgDTO,
	})

	return AckResponse{
		Status:  "ok",
		Code:    "SUCCESS",
		Message: "Message sent",
		Data: map[string]interface{}{
			"message_id": msg.ID.Hex(),
			"created_at": msg.CreatedAt,
		},
	}
}

// handleMarkAsRead - mark_as_read event
func (h *WebSocketHandler) handleMarkAsRead(ctx context.Context, client *hub.Client, payload json.RawMessage) AckResponse {
	var req struct {
		MessageID      string `json:"message_id"`
		ConversationID string `json:"conversation_id"`
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		return AckResponse{
			Status:  "error",
			Code:    "INVALID_PAYLOAD",
			Message: "Invalid event payload",
		}
	}

	// Mark message as read using service
	if err := h.msgService.MarkAsRead(ctx, req.MessageID, client.ID); err != nil {
		return h.mapError(err)
	}

	// Broadcast message_read event
	h.hub.Broadcast([]string{req.ConversationID}, BroadcastEvent{
		Type: "message_read",
		Payload: map[string]interface{}{
			"message_id":       req.MessageID,
			"user_id":          client.ID,
			"conversation_id":  req.ConversationID,
			"read_at":          time.Now(),
		},
	})

	return AckResponse{
		Status:  "ok",
		Code:    "SUCCESS",
		Message: "Message marked as read",
	}
}

// handleTyping - typing event
// Note: typing events don't require acknowledgment, so we return ok
func (h *WebSocketHandler) handleTyping(ctx context.Context, client *hub.Client, payload json.RawMessage) AckResponse {
	var req struct {
		ConversationID string `json:"conversation_id"`
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		return AckResponse{
			Status:  "error",
			Code:    "INVALID_PAYLOAD",
			Message: "Invalid event payload",
		}
	}

	// Broadcast typing event
	h.hub.Broadcast([]string{req.ConversationID}, BroadcastEvent{
		Type: "user_typing",
		Payload: map[string]interface{}{
			"user_id":          client.ID,
			"conversation_id":  req.ConversationID,
			"timestamp":        time.Now(),
		},
	})

	return AckResponse{
		Status:  "ok",
		Code:    "SUCCESS",
		Message: "Typing broadcasted",
	}
}

// handleDeleteMessage - delete_message event
func (h *WebSocketHandler) handleDeleteMessage(ctx context.Context, client *hub.Client, payload json.RawMessage) AckResponse {
	var req struct {
		MessageID      string `json:"message_id"`
		ConversationID string `json:"conversation_id"`
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		return AckResponse{
			Status:  "error",
			Code:    "INVALID_PAYLOAD",
			Message: "Invalid event payload",
		}
	}

	// Delete message using service
	if err := h.msgService.DeleteMessage(ctx, req.MessageID, client.ID); err != nil {
		return h.mapError(err)
	}

	// Broadcast message_deleted event
	h.hub.Broadcast([]string{req.ConversationID}, BroadcastEvent{
		Type: "message_deleted",
		Payload: map[string]interface{}{
			"message_id":       req.MessageID,
			"user_id":          client.ID,
			"conversation_id":  req.ConversationID,
			"deleted_at":       time.Now(),
		},
	})

	return AckResponse{
		Status:  "ok",
		Code:    "SUCCESS",
		Message: "Message deleted",
	}
}

// handleEditMessage - edit_message event
func (h *WebSocketHandler) handleEditMessage(ctx context.Context, client *hub.Client, payload json.RawMessage) AckResponse {
	var req struct {
		MessageID      string `json:"message_id"`
		ConversationID string `json:"conversation_id"`
		NewText        string `json:"new_text"`
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		return AckResponse{
			Status:  "error",
			Code:    "INVALID_PAYLOAD",
			Message: "Invalid event payload",
		}
	}

	// Edit message using service
	if err := h.msgService.EditMessage(ctx, req.MessageID, client.ID, req.NewText); err != nil {
		return h.mapError(err)
	}

	// Broadcast message_edited event
	h.hub.Broadcast([]string{req.ConversationID}, BroadcastEvent{
		Type: "message_edited",
		Payload: map[string]interface{}{
			"message_id":       req.MessageID,
			"user_id":          client.ID,
			"conversation_id":  req.ConversationID,
			"new_text":         req.NewText,
			"edited_at":        time.Now(),
		},
	})

	return AckResponse{
		Status:  "ok",
		Code:    "SUCCESS",
		Message: "Message edited",
	}
}

// handleLeaveConversation - leave_conversation event
func (h *WebSocketHandler) handleLeaveConversation(ctx context.Context, client *hub.Client, payload json.RawMessage) AckResponse {
	var req struct {
		ConversationID string `json:"conversation_id"`
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		return AckResponse{
			Status:  "error",
			Code:    "INVALID_PAYLOAD",
			Message: "Invalid event payload",
		}
	}

	// Leave room
	h.hub.LeaveRoom(client, req.ConversationID)

	// Broadcast offline status
	h.hub.Broadcast([]string{req.ConversationID}, BroadcastEvent{
		Type: "user_offline",
		Payload: map[string]interface{}{
			"user_id":          client.ID,
			"conversation_id":  req.ConversationID,
			"timestamp":        time.Now(),
		},
	})

	log.Printf("Client %s left conversation %s", client.ID, req.ConversationID)

	return AckResponse{
		Status:  "ok",
		Code:    "SUCCESS",
		Message: "Left conversation",
	}
}

// mapError maps service errors to appropriate AckResponse
func (h *WebSocketHandler) mapError(err error) AckResponse {
	switch err {
	case utils.ErrUnauthorized:
		return AckResponse{
			Status:  "error",
			Code:    "UNAUTHORIZED",
			Message: "Unauthorized",
		}
	case utils.ErrForbidden, utils.ErrUserNotParticipant:
		return AckResponse{
			Status:  "error",
			Code:    "FORBIDDEN",
			Message: "Forbidden - not a participant",
		}
	case utils.ErrUserBlocked:
		return AckResponse{
			Status:  "error",
			Code:    "BLOCKED",
			Message: "User blocked communication",
		}
	case utils.ErrConversationNotFound:
		return AckResponse{
			Status:  "error",
			Code:    "CONVERSATION_NOT_FOUND",
			Message: "Conversation not found",
		}
	case utils.ErrMessageNotFound:
		return AckResponse{
			Status:  "error",
			Code:    "MESSAGE_NOT_FOUND",
			Message: "Message not found",
		}
	case utils.ErrInvalidPayload:
		return AckResponse{
			Status:  "error",
			Code:    "INVALID_PAYLOAD",
			Message: "Invalid payload",
		}
	case utils.ErrDatabaseError:
		return AckResponse{
			Status:  "error",
			Code:    "DATABASE_ERROR",
			Message: "Database error",
		}
	default:
		return AckResponse{
			Status:  "error",
			Code:    "INTERNAL_ERROR",
			Message: "Internal server error",
		}
	}
}
