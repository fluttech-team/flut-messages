package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flutapp/chat-service/internal/middleware"
)

type websocketAuthStub struct {
	userID string
	err    error
}

func (s websocketAuthStub) VerifyToken(string) (string, error) { return s.userID, s.err }

type ticketConsumerStub struct {
	userID string
	err    error
}

func (s ticketConsumerStub) Consume(string) (string, error) { return s.userID, s.err }

func TestWebSocketAuthAcceptsSingleUseTicket(t *testing.T) {
	handler := middleware.RequireWebSocketAuth(
		websocketAuthStub{err: errors.New("bearer should not be required")},
		ticketConsumerStub{userID: "browser-user"},
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserID(r)
		if !ok || userID != "browser-user" {
			t.Fatalf("expected browser-user in context, got %q", userID)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/ws?ticket=single-use", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, response.Code)
	}
}

func TestWebSocketAuthKeepsBearerSupportForMobile(t *testing.T) {
	handler := middleware.RequireWebSocketAuth(
		websocketAuthStub{userID: "mobile-user"},
		ticketConsumerStub{err: errors.New("ticket should not be required")},
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, _ := middleware.UserID(r)
		if userID != "mobile-user" {
			t.Fatalf("expected mobile-user in context, got %q", userID)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.Header.Set("Authorization", "Bearer mobile-jwt")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, response.Code)
	}
}

func TestWebSocketAuthRejectsMissingAndInvalidCredentials(t *testing.T) {
	handler := middleware.RequireWebSocketAuth(
		websocketAuthStub{err: errors.New("invalid bearer")},
		ticketConsumerStub{err: errors.New("invalid ticket")},
	)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("unauthenticated request reached websocket handler")
	}))

	for _, target := range []string{"/ws", "/ws?token=long-lived-jwt", "/ws?ticket=invalid"} {
		request := httptest.NewRequest(http.MethodGet, target, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("%s: expected %d, got %d", target, http.StatusUnauthorized, response.Code)
		}
	}
}
