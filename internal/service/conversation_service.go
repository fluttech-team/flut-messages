package service

import (
	"context"
	"slices"

	"github.com/flutapp/chat-service/internal/client"
	"github.com/flutapp/chat-service/internal/domain"
	"github.com/flutapp/chat-service/internal/repository"
	"github.com/flutapp/chat-service/internal/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ConversationService interface {
	// CreateOrGetConversation scopes a conversation to a job application: it
	// resolves applicant/company via backend-flut (forwarding authHeader) and
	// verifies requesterID is one of them.
	CreateOrGetConversation(ctx context.Context, requesterID, authHeader, applicationID string) (*domain.Conversation, error)
	GetConversations(ctx context.Context, userID string, limit, offset int) ([]*domain.Conversation, error)
	GetConversation(ctx context.Context, convID string, userID string) (*domain.Conversation, error)
}

type conversationService struct {
	repo        repository.ConversationRepository
	backendFlut client.BackendFlutClient
}

func NewConversationService(repo repository.ConversationRepository, backendFlut client.BackendFlutClient) ConversationService {
	return &conversationService{repo: repo, backendFlut: backendFlut}
}

func (s *conversationService) CreateOrGetConversation(ctx context.Context, requesterID, authHeader, applicationID string) (*domain.Conversation, error) {
	// Already created for this application? Just verify the caller belongs to it.
	conv, err := s.repo.FindByApplicationID(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	if conv != nil {
		if !slices.Contains(conv.ParticipantIDs, requesterID) {
			return nil, utils.ErrForbidden
		}
		return conv, nil
	}

	// First time: ask backend-flut who the two parties are. It re-validates
	// the JWT and confirms requesterID is actually a party to this application.
	participants, err := s.backendFlut.GetApplicationParticipants(ctx, authHeader, applicationID)
	if err != nil {
		return nil, err
	}
	if requesterID != participants.ApplicantID && requesterID != participants.CompanyID {
		return nil, utils.ErrForbidden
	}

	newConv := &domain.Conversation{
		ApplicationID:  applicationID,
		ParticipantIDs: []string{participants.ApplicantID, participants.CompanyID},
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

	if !slices.Contains(conv.ParticipantIDs, userID) {
		return nil, utils.ErrUserNotParticipant
	}

	return conv, nil
}
