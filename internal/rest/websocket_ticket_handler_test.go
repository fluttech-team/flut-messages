package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/flutapp/chat-service/internal/middleware"
	"github.com/flutapp/chat-service/internal/service"
)

type ticketHandlerAuthStub struct{}

func (ticketHandlerAuthStub) VerifyToken(string) (string, error) { return "user-1", nil }

func TestIssueWebSocketTicketReturnsShortLivedCredential(t *testing.T) {
	now := time.Date(2026, time.September, 14, 12, 0, 0, 0, time.UTC)
	ticketService := service.NewWebSocketTicketService(30*time.Second, func() time.Time { return now })
	handler := NewRESTHandler(nil, nil, nil, ticketService)
	protected := middleware.RequireAuth(ticketHandlerAuthStub{})(http.HandlerFunc(handler.IssueWebSocketTicket))
	request := httptest.NewRequest(http.MethodPost, "/ws-tickets", nil)
	request.Header.Set("Authorization", "Bearer valid-jwt")
	response := httptest.NewRecorder()

	protected.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, response.Code)
	}
	var body struct {
		Ticket    string    `json:"ticket"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Ticket == "" {
		t.Fatal("expected a ticket")
	}
	if want := now.Add(30 * time.Second); !body.ExpiresAt.Equal(want) {
		t.Fatalf("expected expiry %s, got %s", want, body.ExpiresAt)
	}
}
