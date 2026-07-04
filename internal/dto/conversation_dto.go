package dto

import (
	"time"

	"github.com/flutapp/chat-service/internal/domain"
)

type ConversationResponse struct {
	ID             string             `json:"id"`
	ParticipantIDs []string           `json:"participant_ids"`
	LastMessage    *MessagePreviewDTO `json:"last_message"`
	UnreadCount    int                `json:"unread_count"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

type MessagePreviewDTO struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	SenderID  string    `json:"sender_id"`
	CreatedAt time.Time `json:"created_at"`
}

func ConversationToDTO(conv *domain.Conversation, userID string) ConversationResponse {
	var lastMsgDTO *MessagePreviewDTO
	if conv.LastMessage != nil {
		lastMsgDTO = &MessagePreviewDTO{
			ID:        conv.LastMessage.ID.Hex(),
			Text:      conv.LastMessage.Text,
			SenderID:  conv.LastMessage.SenderID,
			CreatedAt: conv.LastMessage.CreatedAt,
		}
	}

	return ConversationResponse{
		ID:             conv.ID.Hex(),
		ParticipantIDs: conv.ParticipantIDs,
		LastMessage:    lastMsgDTO,
		UnreadCount:    conv.UnreadCount[userID],
		CreatedAt:      conv.CreatedAt,
		UpdatedAt:      conv.UpdatedAt,
	}
}
