package handler

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/flutapp/chat-service/internal/domain"
	"github.com/flutapp/chat-service/internal/hub"
	"github.com/flutapp/chat-service/internal/utils"
)

type conversationServiceStub struct{ getErr error }

func (conversationServiceStub) CreateOrGetConversation(context.Context, string, string, string) (*domain.Conversation, error) {
	return nil, nil
}
func (conversationServiceStub) GetConversations(context.Context, string, int, int) ([]*domain.Conversation, error) {
	return nil, nil
}
func (s conversationServiceStub) GetConversation(context.Context, string, string) (*domain.Conversation, error) {
	return nil, s.getErr
}

type messageServiceStub struct{}

func (messageServiceStub) SendMessage(context.Context, string, string, string, string, []domain.Attachment) (*domain.Message, error) {
	return nil, nil
}
func (messageServiceStub) GetMessages(context.Context, string, string, int, int) ([]*domain.Message, error) {
	return nil, nil
}
func (messageServiceStub) MarkAsRead(context.Context, string, string) error          { return nil }
func (messageServiceStub) DeleteMessage(context.Context, string, string) error       { return nil }
func (messageServiceStub) EditMessage(context.Context, string, string, string) error { return nil }
func (messageServiceStub) SearchMessages(context.Context, string, string, string) ([]*domain.Message, error) {
	return nil, nil
}
func (messageServiceStub) GetUnreadMessages(context.Context, string) ([]*domain.Message, error) {
	return nil, nil
}

type blockServiceStub struct{}

func (blockServiceStub) BlockUser(context.Context, string, string) error          { return nil }
func (blockServiceStub) UnblockUser(context.Context, string, string) error        { return nil }
func (blockServiceStub) IsBlocked(context.Context, string, string) (bool, error)  { return false, nil }
func (blockServiceStub) GetBlockedList(context.Context, string) ([]string, error) { return nil, nil }

func TestJoinConversationRejectsNonParticipant(t *testing.T) {
	h := NewWebSocketHandler(
		messageServiceStub{},
		conversationServiceStub{getErr: utils.ErrUserNotParticipant},
		hub.NewHub(),
		blockServiceStub{},
	)
	payload, _ := json.Marshal(map[string]string{"conversation_id": "conversation-1"})

	response := h.HandleEvent(context.Background(), &hub.Client{ID: "outsider", Rooms: map[string]bool{}}, WebSocketEvent{
		Type:    "join_conversation",
		Payload: payload,
	})

	if response.Status != "error" || response.Code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN response, got %+v", response)
	}
}
