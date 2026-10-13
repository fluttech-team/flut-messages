package service

import (
	"context"
	"testing"

	"github.com/flutapp/chat-service/internal/client"
	"github.com/flutapp/chat-service/internal/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type duplicateConversationRepo struct {
	finds int
	ready *domain.Conversation
}

func (r *duplicateConversationRepo) Create(context.Context, *domain.Conversation) (*domain.Conversation, error) {
	return nil, mongo.WriteException{WriteErrors: []mongo.WriteError{{Code: 11000}}}
}
func (r *duplicateConversationRepo) FindByApplicationID(context.Context, string) (*domain.Conversation, error) {
	r.finds++
	if r.finds == 1 {
		return nil, nil
	}
	return r.ready, nil
}
func (*duplicateConversationRepo) FindByID(context.Context, primitive.ObjectID) (*domain.Conversation, error) {
	return nil, nil
}
func (*duplicateConversationRepo) FindByUserID(context.Context, string, int64, int64) ([]*domain.Conversation, error) {
	return nil, nil
}
func (*duplicateConversationRepo) UpdateLastMessage(context.Context, primitive.ObjectID, *domain.MessagePreview) error {
	return nil
}
func (*duplicateConversationRepo) UpdateUnreadCount(context.Context, primitive.ObjectID, string, int) error {
	return nil
}
func (*duplicateConversationRepo) ResetUnreadCount(context.Context, primitive.ObjectID, string) error {
	return nil
}

type participantClient struct{}

func (participantClient) GetApplicationParticipants(context.Context, string, string) (*client.ApplicationParticipants, error) {
	return &client.ApplicationParticipants{ApplicantID: "candidate", CompanyID: "company"}, nil
}

func TestCreateOrGetConversationRecoversConcurrentCreate(t *testing.T) {
	existing := &domain.Conversation{ApplicationID: "application", ParticipantIDs: []string{"candidate", "company"}}
	repo := &duplicateConversationRepo{ready: existing}
	svc := NewConversationService(repo, participantClient{})

	conversation, err := svc.CreateOrGetConversation(context.Background(), "company", "Bearer token", "application")
	if err != nil {
		t.Fatalf("expected duplicate create to return the existing conversation: %v", err)
	}
	if conversation != existing {
		t.Fatalf("expected existing conversation, got %#v", conversation)
	}
}
