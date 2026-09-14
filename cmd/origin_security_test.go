package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebSocketOriginChecker(t *testing.T) {
	checkOrigin := originChecker([]string{"https://dashboard.flut.id", "http://localhost:5173"})

	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		{name: "native client without origin", want: true},
		{name: "configured dashboard", origin: "https://dashboard.flut.id", want: true},
		{name: "configured local dashboard", origin: "http://localhost:5173", want: true},
		{name: "untrusted browser", origin: "https://evil.example", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/ws", nil)
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			}

			if got := checkOrigin(request); got != test.want {
				t.Fatalf("origin check = %v, want %v", got, test.want)
			}
		})
	}
}

func TestCORSMiddlewareOnlyAllowsConfiguredBrowserOrigins(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := corsMiddleware([]string{"https://dashboard.flut.id"}, next)

	allowedRequest := httptest.NewRequest(http.MethodGet, "/conversations", nil)
	allowedRequest.Header.Set("Origin", "https://dashboard.flut.id")
	allowedResponse := httptest.NewRecorder()
	handler.ServeHTTP(allowedResponse, allowedRequest)

	if got := allowedResponse.Header().Get("Access-Control-Allow-Origin"); got != "https://dashboard.flut.id" {
		t.Fatalf("allowed origin header = %q", got)
	}

	blockedRequest := httptest.NewRequest(http.MethodOptions, "/conversations", nil)
	blockedRequest.Header.Set("Origin", "https://evil.example")
	blockedResponse := httptest.NewRecorder()
	handler.ServeHTTP(blockedResponse, blockedRequest)

	if blockedResponse.Code != http.StatusForbidden {
		t.Fatalf("blocked preflight status = %d, want %d", blockedResponse.Code, http.StatusForbidden)
	}
	if got := blockedResponse.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("blocked origin must not receive CORS permission, got %q", got)
	}
}
