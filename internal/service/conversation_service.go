package service

import (
	"context"

	"github.com/flutapp/chat-service/internal/domain"
	"github.com/flutapp/chat-service/internal/repository"
	"github.com/flutapp/chat-service/internal/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ConversationService interface {
	CreateOrGetConversation(ctx context.Context, userAID, userBID string) (*domain.Conversation, error)
	GetConversations(ctx context.Context, userID string, limit, offset int) ([]*domain.Conversation, error)
	GetConversation(ctx context.Context, convID string, userID string) (*domain.Conversation, error)
}

type conversationService struct {
	repo repository.ConversationRepository
}

func NewConversationService(repo repository.ConversationRepository) ConversationService {
	return &conversationService{repo}
}

func (s *conversationService) CreateOrGetConversation(ctx context.Context, userAID, userBID string) (*domain.Conversation, error) {
	// Try to find existing conversation
	conv, err := s.repo.FindByParticipants(ctx, []string{userAID, userBID})
	if err != nil {
		return nil, err
	}
	if conv != nil {
		return conv, nil
	}

	// Create new conversation if not found
	newConv := &domain.Conversation{
		ParticipantIDs: []string{userAID, userBID},
	}
	return s.repo.Create(ctx, newConv)
}

func (s *conversationService) GetConversations(ctx context.Context, userID string, limit, offset int) ([]*domain.Conversation, error) {
	return s.repo.FindByUserID(ctx, userID, int64(limit), int64(offset))
}

func (s *conversationService) GetConversation(ctx context.Context, convID string, userID string) (*domain.Conversation, error) {
	objID, err := primitive.ObjectIDFromHex(convID)
	if err != nil {
		return nil, utils.ErrInvalidPayload
	}

	conv, err := s.repo.FindByID(ctx, objID)
	if err != nil {
		return nil, err
	}
	if conv == nil {
		return nil, utils.ErrConversationNotFound
	}

	// Check if user is a participant
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

	return conv, nil
}
