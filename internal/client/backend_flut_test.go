package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveChatContextUsesActorBearerAndSelectedCompany(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/company/chat-context" || r.Header.Get("Authorization") != "Bearer actor-token" || r.Header.Get("X-Company-ID") != "owner" {
			t.Errorf("wrong backend context request")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"actor_id":"member","company_id":"owner","owner_id":"owner"}}`))
	}))
	defer server.Close()
	result, err := NewBackendFlutClient(server.URL).ResolveChatContext(context.Background(), "Bearer actor-token", "owner")
	if err != nil || result.ActorID != "member" || result.OwnerID != "owner" {
		t.Fatalf("context: %+v %v", result, err)
	}
}
