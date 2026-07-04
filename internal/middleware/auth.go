package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/flutapp/chat-service/internal/service"
	"github.com/gorilla/websocket"
)

func AuthWebSocket(authService service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := r.URL.Query().Get("token")
			if tokenStr == "" {
				http.Error(w, "missing token", http.StatusUnauthorized)
				return
			}

			// Remove "Bearer " prefix if present
			if strings.HasPrefix(tokenStr, "Bearer ") {
				tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
			}

			userID, err := authService.VerifyToken(tokenStr)
			if err != nil {
				log.Printf("[auth] token verification failed: %v", err)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Store userID in request header for handler to use
			r.Header.Set("X-User-ID", userID)
			next.ServeHTTP(w, r)
		})
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// For development, allow all origins. In production, restrict to known origins.
		return true
	},
}

func GetUpgrader() websocket.Upgrader {
	return upgrader
}
