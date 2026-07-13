package dto

import (
	"time"

	"github.com/flutapp/chat-service/internal/domain"
)

type MessageResponse struct {
	ID           string              `json:"id"`
	Text         string              `json:"text"`
	SenderID     string              `json:"sender_id"`
	ReceiverID   string              `json:"receiver_id"`
	Status       string              `json:"status"`
	CreatedAt    time.Time           `json:"created_at"`
	DeliveredAt  *time.Time          `json:"delivered_at"`
	ReadAt       *time.Time          `json:"read_at"`
	EditedAt     *time.Time          `json:"edited_at"`
	Attachments  []domain.Attachment `json:"attachments"`
	TemplateType *string             `json:"template_type,omitempty"`
}

func MessageToDTO(msg *domain.Message) MessageResponse {
	return MessageResponse{
		ID:           msg.ID.Hex(),
		Text:         msg.Text,
		SenderID:     msg.SenderID,
		ReceiverID:   msg.ReceiverID,
		Status:       msg.Status,
		CreatedAt:    msg.CreatedAt,
		DeliveredAt:  msg.DeliveredAt,
		ReadAt:       msg.ReadAt,
		EditedAt:     msg.EditedAt,
		Attachments:  msg.Attachments,
		TemplateType: msg.TemplateType,
	}
}
