package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/flutapp/chat-service/internal/service"
)

type contextKey string

const userIDContextKey contextKey = "userID"

// RequireAuth verifies the "Authorization: Bearer <jwt>" header and stores the
// resolved userID in the request context for handlers to read via UserID.
func RequireAuth(authService service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}

			userID, err := authService.VerifyToken(strings.TrimPrefix(authHeader, "Bearer "))
			if err != nil {
				log.Printf("[auth] token verification failed: %v", err)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDContextKey, userID)))
		})
	}
}

// UserID reads the userID stored by RequireAuth.
func UserID(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(userIDContextKey).(string)
	return userID, ok
}
