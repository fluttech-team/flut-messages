package rest

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/flutapp/chat-service/internal/dto"
	"github.com/flutapp/chat-service/internal/service"
	"github.com/flutapp/chat-service/internal/utils"
)

// RESTHandler handles REST API endpoints
type RESTHandler struct {
	convService  service.ConversationService
	msgService   service.MessageService
	blockService service.BlockService
}

// NewRESTHandler creates a new RESTHandler
func NewRESTHandler(
	convService service.ConversationService,
	msgService service.MessageService,
	blockService service.BlockService,
) *RESTHandler {
	return &RESTHandler{
		convService:  convService,
		msgService:   msgService,
		blockService: blockService,
	}
}

// getUserID extracts and validates the X-User-ID header
func getUserID(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing X-User-ID header"})
		return "", false
	}
	return userID, true
}

// parseQueryParams parses limit and offset query parameters
func parseQueryParams(r *http.Request, defaultLimit, maxLimit int) (int, int) {
	limit := defaultLimit
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			if parsed > maxLimit {
				parsed = maxLimit
			}
			limit = parsed
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	return limit, offset
}

// GetConversations lists user's conversations
func (h *RESTHandler) GetConversations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get X-User-ID header
	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	// Parse limit and offset
	limit, offset := parseQueryParams(r, 20, 100)

	// Get conversations from service
	conversations, err := h.convService.GetConversations(r.Context(), userID, limit, offset)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to fetch conversations"})
		return
	}

	// Convert to DTOs
	var results []dto.ConversationResponse
	for _, conv := range conversations {
		results = append(results, dto.ConversationToDTO(conv, userID))
	}

	// Return JSON array
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(results)
}

// GetMessages fetches message history with pagination
func (h *RESTHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get X-User-ID header
	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	// Extract convID from URL path
	convID := strings.TrimPrefix(r.URL.Path, "/conversations/")
	convID = strings.TrimSuffix(convID, "/messages")
	if convID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid conversation ID"})
		return
	}

	// Parse limit and offset
	limit, offset := parseQueryParams(r, 50, 100)

	// Get messages from service
	messages, err := h.msgService.GetMessages(r.Context(), convID, userID, limit, offset)
	if err != nil {
		if err == utils.ErrConversationNotFound || err == utils.ErrUserNotParticipant {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to fetch messages"})
		}
		return
	}

	// Convert to DTOs
	var results []dto.MessageResponse
	for _, msg := range messages {
		results = append(results, dto.MessageToDTO(msg))
	}

	// Return JSON array
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(results)
}

// SearchMessages searches messages in a conversation
func (h *RESTHandler) SearchMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get X-User-ID header
	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	// Extract convID from URL path
	convID := strings.TrimPrefix(r.URL.Path, "/conversations/")
	convID = strings.TrimSuffix(convID, "/search")
	if convID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid conversation ID"})
		return
	}

	// Get query string param
	query := r.URL.Query().Get("q")
	if query == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing 'q' query parameter"})
		return
	}

	// Search messages
	messages, err := h.msgService.SearchMessages(r.Context(), convID, userID, query)
	if err != nil {
		if err == utils.ErrConversationNotFound || err == utils.ErrUserNotParticipant {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to search messages"})
		}
		return
	}

	// Convert to DTOs
	var results []dto.MessageResponse
	for _, msg := range messages {
		results = append(results, dto.MessageToDTO(msg))
	}

	// Return JSON array
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(results)
}

// BlockUser blocks a user (one-directional)
func (h *RESTHandler) BlockUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get X-User-ID header
	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	// Extract targetID from URL path
	targetID := strings.TrimPrefix(r.URL.Path, "/users/")
	targetID = strings.TrimSuffix(targetID, "/block")
	if targetID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid target ID"})
		return
	}

	// Block user
	err := h.blockService.BlockUser(r.Context(), userID, targetID)
	if err != nil {
		if err == utils.ErrInvalidPayload {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to block user"})
		}
		return
	}

	// Return 204 NoContent on success
	w.WriteHeader(http.StatusNoContent)
}

// UnblockUser unblocks a user
func (h *RESTHandler) UnblockUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get X-User-ID header
	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	// Extract targetID from URL path
	targetID := strings.TrimPrefix(r.URL.Path, "/users/")
	targetID = strings.TrimSuffix(targetID, "/unblock")
	if targetID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid target ID"})
		return
	}

	// Unblock user
	err := h.blockService.UnblockUser(r.Context(), userID, targetID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to unblock user"})
		return
	}

	// Return 204 NoContent on success
	w.WriteHeader(http.StatusNoContent)
}

// GetBlockedList gets list of users blocked by requester
func (h *RESTHandler) GetBlockedList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get X-User-ID header
	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	// Get blocked list
	blockedList, err := h.blockService.GetBlockedList(r.Context(), userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to fetch blocked list"})
		return
	}

	// Return JSON: {"blocked_users": [...]}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string][]string{"blocked_users": blockedList})
}
