package service

import (
	"context"
	"testing"

	"github.com/flutapp/chat-service/internal/domain"
	"github.com/flutapp/chat-service/internal/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type messageRepoStub struct{ message *domain.Message }

func (messageRepoStub) Insert(context.Context, *domain.Message) (*domain.Message, error) {
	panic("unexpected insert")
}
func (messageRepoStub) FindByConversationID(context.Context, primitive.ObjectID, int64, int64) ([]*domain.Message, error) {
	return nil, nil
}
func (messageRepoStub) FindByID(context.Context, primitive.ObjectID) (*domain.Message, error) {
	return nil, nil
}
func (messageRepoStub) UpdateStatus(context.Context, primitive.ObjectID, string) error { return nil }
func (messageRepoStub) MarkDeleted(context.Context, primitive.ObjectID) error          { return nil }
func (messageRepoStub) Update(context.Context, primitive.ObjectID, string) error       { return nil }
func (messageRepoStub) SearchByText(context.Context, primitive.ObjectID, string) ([]*domain.Message, error) {
	return nil, nil
}
func (messageRepoStub) FindUnreadByUserID(context.Context, string) ([]*domain.Message, error) {
	return nil, nil
}

type conversationRepoStub struct{ conversation *domain.Conversation }

func (conversationRepoStub) Create(context.Context, *domain.Conversation) (*domain.Conversation, error) {
	return nil, nil
}
func (conversationRepoStub) FindByApplicationID(context.Context, string) (*domain.Conversation, error) {
	return nil, nil
}
func (s conversationRepoStub) FindByID(context.Context, primitive.ObjectID) (*domain.Conversation, error) {
	return s.conversation, nil
}
func (conversationRepoStub) FindByUserID(context.Context, string, int64, int64) ([]*domain.Conversation, error) {
	return nil, nil
}
func (conversationRepoStub) UpdateLastMessage(context.Context, primitive.ObjectID, *domain.MessagePreview) error {
	return nil
}
func (conversationRepoStub) UpdateUnreadCount(context.Context, primitive.ObjectID, string, int) error {
	return nil
}
func (conversationRepoStub) ResetUnreadCount(context.Context, primitive.ObjectID, string) error {
	return nil
}

type blockRepoStub struct{}

func (blockRepoStub) Create(context.Context, string, string) (*domain.Block, error) { return nil, nil }
func (blockRepoStub) Delete(context.Context, string, string) error                  { return nil }
func (blockRepoStub) IsBlocked(context.Context, string, string) (bool, error)       { return false, nil }
func (blockRepoStub) GetBlockedList(context.Context, string) ([]string, error)      { return nil, nil }

func TestSendMessageRejectsReceiverOutsideConversation(t *testing.T) {
	conversationID := primitive.NewObjectID()
	svc := NewMessageService(
		messageRepoStub{},
		conversationRepoStub{conversation: &domain.Conversation{
			ID:             conversationID,
			ParticipantIDs: []string{"candidate-1", "company-1"},
		}},
		blockRepoStub{},
	)

	_, err := svc.SendMessage(
		context.Background(),
		conversationID.Hex(),
		"company-1",
		"attacker-controlled-user",
		"hello",
		nil,
	)
	if err != utils.ErrUserNotParticipant {
		t.Fatalf("expected ErrUserNotParticipant, got %v", err)
	}
}

type markReadMessageRepoStub struct {
	message *domain.Message
}

func (s markReadMessageRepoStub) Insert(context.Context, *domain.Message) (*domain.Message, error) {
	return nil, nil
}
func (s markReadMessageRepoStub) FindByConversationID(context.Context, primitive.ObjectID, int64, int64) ([]*domain.Message, error) {
	return nil, nil
}
func (s markReadMessageRepoStub) FindByID(context.Context, primitive.ObjectID) (*domain.Message, error) {
	return s.message, nil
}
func (s markReadMessageRepoStub) UpdateStatus(context.Context, primitive.ObjectID, string) error {
	return nil
}
func (s markReadMessageRepoStub) MarkDeleted(context.Context, primitive.ObjectID) error { return nil }
func (s markReadMessageRepoStub) Update(context.Context, primitive.ObjectID, string) error {
	return nil
}
func (s markReadMessageRepoStub) SearchByText(context.Context, primitive.ObjectID, string) ([]*domain.Message, error) {
	return nil, nil
}
func (s markReadMessageRepoStub) FindUnreadByUserID(context.Context, string) ([]*domain.Message, error) {
	return nil, nil
}

type markReadConversationRepoStub struct {
	conversationRepoStub
	resetConversationID primitive.ObjectID
	resetUserID         string
}

func (s *markReadConversationRepoStub) ResetUnreadCount(_ context.Context, conversationID primitive.ObjectID, userID string) error {
	s.resetConversationID = conversationID
	s.resetUserID = userID
	return nil
}

func TestMarkAsReadResetsConversationUnreadCount(t *testing.T) {
	conversationID := primitive.NewObjectID()
	messageID := primitive.NewObjectID()
	convRepo := &markReadConversationRepoStub{}
	svc := NewMessageService(
		markReadMessageRepoStub{message: &domain.Message{
			ID:             messageID,
			ConversationID: conversationID,
			ReceiverID:     "candidate-1",
		}},
		convRepo,
		blockRepoStub{},
	)

	if err := svc.MarkAsRead(context.Background(), messageID.Hex(), "candidate-1"); err != nil {
		t.Fatalf("MarkAsRead returned error: %v", err)
	}
	if convRepo.resetConversationID != conversationID || convRepo.resetUserID != "candidate-1" {
		t.Fatalf("expected unread count reset for conversation and receiver")
	}
}
