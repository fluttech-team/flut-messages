package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flutapp/chat-service/internal/middleware"
)

type stubAuthService struct {
	userID string
	err    error
}

func (s stubAuthService) VerifyToken(string) (string, error) {
	return s.userID, s.err
}

func TestWebSocketRejectsTokenInQueryWithoutBearerHeader(t *testing.T) {
	auth := stubAuthService{userID: "user-1"}
	handler := middleware.RequireAuth(auth)(http.HandlerFunc(handleWebSocket(nil, nil)))
	request := httptest.NewRequest(http.MethodGet, "/ws?token=jwt-in-url", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func TestWebSocketAcceptsBearerAuthenticationBeforeUpgrade(t *testing.T) {
	auth := stubAuthService{userID: "user-1"}
	handler := middleware.RequireAuth(auth)(http.HandlerFunc(handleWebSocket(nil, nil)))
	request := httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.Header.Set("Authorization", "Bearer valid-jwt")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected WebSocket upgrade status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestWebSocketRejectsInvalidBearerToken(t *testing.T) {
	auth := stubAuthService{err: errors.New("invalid token")}
	handler := middleware.RequireAuth(auth)(http.HandlerFunc(handleWebSocket(nil, nil)))
	request := httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.Header.Set("Authorization", "Bearer invalid-jwt")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}
