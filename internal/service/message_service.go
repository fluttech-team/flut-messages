package service

import (
	"context"
	"strings"

	"github.com/flutapp/chat-service/internal/domain"
	"github.com/flutapp/chat-service/internal/repository"
	"github.com/flutapp/chat-service/internal/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MessageService interface {
	SendMessage(ctx context.Context, convID string, senderID, receiverID, text string, attachments []domain.Attachment) (*domain.Message, error)
	GetMessages(ctx context.Context, convID string, userID string, limit, offset int) ([]*domain.Message, error)
	MarkAsRead(ctx context.Context, messageID string, userID string) error
	DeleteMessage(ctx context.Context, messageID string, userID string) error
	EditMessage(ctx context.Context, messageID string, userID string, newText string) error
	SearchMessages(ctx context.Context, convID string, userID string, query string) ([]*domain.Message, error)
	GetUnreadMessages(ctx context.Context, userID string) ([]*domain.Message, error)
}

type messageService struct {
	msgRepo    repository.MessageRepository
	convRepo   repository.ConversationRepository
	blockRepo  repository.BlockRepository
}

func NewMessageService(msgRepo repository.MessageRepository, convRepo repository.ConversationRepository, blockRepo repository.BlockRepository) MessageService {
	return &messageService{
		msgRepo:   msgRepo,
		convRepo:  convRepo,
		blockRepo: blockRepo,
	}
}

func (s *messageService) SendMessage(ctx context.Context, convID string, senderID, receiverID, text string, attachments []domain.Attachment) (*domain.Message, error) {
	// Validate text and attachments
	if strings.TrimSpace(text) == "" && (attachments == nil || len(attachments) == 0) {
		return nil, utils.ErrInvalidPayload
	}

	// Parse conversation ID
	convObjID, err := primitive.ObjectIDFromHex(convID)
	if err != nil {
		return nil, utils.ErrInvalidPayload
	}

	// Get conversation and verify participant
	conv, err := s.convRepo.FindByID(ctx, convObjID)
	if err != nil {
		return nil, err
	}
	if conv == nil {
		return nil, utils.ErrConversationNotFound
	}

	// Verify sender is a participant
	isParticipant := false
	for _, p := range conv.ParticipantIDs {
		if p == senderID {
			isParticipant = true
			break
		}
	}
	if !isParticipant {
		return nil, utils.ErrUserNotParticipant
	}

	// Check if sender has blocked the receiver or vice versa
	blockedBySender, err := s.blockRepo.IsBlocked(ctx, senderID, receiverID)
	if err != nil {
		return nil, err
	}
	if blockedBySender {
		return nil, utils.ErrUserBlocked
	}

	blockedByReceiver, err := s.blockRepo.IsBlocked(ctx, receiverID, senderID)
	if err != nil {
		return nil, err
	}
	if blockedByReceiver {
		return nil, utils.ErrUserBlocked
	}

	// Create and insert message
	msg := &domain.Message{
		ConversationID: convObjID,
		SenderID:       senderID,
		ReceiverID:     receiverID,
		Text:           text,
		Attachments:    attachments,
	}

	savedMsg, err := s.msgRepo.Insert(ctx, msg)
	if err != nil {
		return nil, err
	}

	// Update conversation with last message and unread count
	preview := &domain.MessagePreview{
		ID:        savedMsg.ID,
		Text:      savedMsg.Text,
		SenderID:  savedMsg.SenderID,
		CreatedAt: savedMsg.CreatedAt,
	}

	err = s.convRepo.UpdateLastMessage(ctx, convObjID, preview)
	if err != nil {
		return nil, err
	}

	// Increment unread count for receiver
	err = s.convRepo.UpdateUnreadCount(ctx, convObjID, receiverID, 1)
	if err != nil {
		return nil, err
	}

	return savedMsg, nil
}

func (s *messageService) GetMessages(ctx context.Context, convID string, userID string, limit, offset int) ([]*domain.Message, error) {
	// Parse conversation ID
	convObjID, err := primitive.ObjectIDFromHex(convID)
	if err != nil {
		return nil, utils.ErrInvalidPayload
	}

	// Get conversation and verify participant
	conv, err := s.convRepo.FindByID(ctx, convObjID)
	if err != nil {
		return nil, err
	}
	if conv == nil {
		return nil, utils.ErrConversationNotFound
	}

	// Verify user is a participant
	isParticipant := false
	for _, p := range conv.ParticipantIDs {
		if p == userID {
			isParticipant = true
			break
		}
	}
	if !isParticipant {
		return nil, utils.ErrUserNotParticipant
	}

	// Get messages
	return s.msgRepo.FindByConversationID(ctx, convObjID, int64(limit), int64(offset))
}

func (s *messageService) MarkAsRead(ctx context.Context, messageID string, userID string) error {
	// Parse message ID
	msgObjID, err := primitive.ObjectIDFromHex(messageID)
	if err != nil {
		return utils.ErrInvalidPayload
	}

	// Get message
	msg, err := s.msgRepo.FindByID(ctx, msgObjID)
	if err != nil {
		return err
	}
	if msg == nil {
		return utils.ErrMessageNotFound
	}

	// Verify user is the receiver only
	if msg.ReceiverID != userID {
		return utils.ErrForbidden
	}

	// Update status to read
	return s.msgRepo.UpdateStatus(ctx, msgObjID, "read")
}

func (s *messageService) DeleteMessage(ctx context.Context, messageID string, userID string) error {
	// Parse message ID
	msgObjID, err := primitive.ObjectIDFromHex(messageID)
	if err != nil {
		return utils.ErrInvalidPayload
	}

	// Get message
	msg, err := s.msgRepo.FindByID(ctx, msgObjID)
	if err != nil {
		return err
	}
	if msg == nil {
		return utils.ErrMessageNotFound
	}

	// Verify user is the sender only
	if msg.SenderID != userID {
		return utils.ErrForbidden
	}

	// Mark as deleted
	return s.msgRepo.MarkDeleted(ctx, msgObjID)
}

func (s *messageService) EditMessage(ctx context.Context, messageID string, userID string, newText string) error {
	// Validate new text
	if strings.TrimSpace(newText) == "" {
		return utils.ErrInvalidPayload
	}

	// Parse message ID
	msgObjID, err := primitive.ObjectIDFromHex(messageID)
	if err != nil {
		return utils.ErrInvalidPayload
	}

	// Get message
	msg, err := s.msgRepo.FindByID(ctx, msgObjID)
	if err != nil {
		return err
	}
	if msg == nil {
		return utils.ErrMessageNotFound
	}

	// Verify user is the sender only
	if msg.SenderID != userID {
		return utils.ErrForbidden
	}

	// Update message text
	return s.msgRepo.Update(ctx, msgObjID, newText)
}

func (s *messageService) SearchMessages(ctx context.Context, convID string, userID string, query string) ([]*domain.Message, error) {
	// Parse conversation ID
	convObjID, err := primitive.ObjectIDFromHex(convID)
	if err != nil {
		return nil, utils.ErrInvalidPayload
	}

	// Get conversation and verify participant
	conv, err := s.convRepo.FindByID(ctx, convObjID)
	if err != nil {
		return nil, err
	}
	if conv == nil {
		return nil, utils.ErrConversationNotFound
	}

	// Verify user is a participant
	isParticipant := false
	for _, p := range conv.ParticipantIDs {
		if p == userID {
			isParticipant = true
			break
		}
	}
	if !isParticipant {
		return nil, utils.ErrUserNotParticipant
	}

	// Search messages
	return s.msgRepo.SearchByText(ctx, convObjID, query)
}

func (s *messageService) GetUnreadMessages(ctx context.Context, userID string) ([]*domain.Message, error) {
	return s.msgRepo.FindUnreadByUserID(ctx, userID)
}
