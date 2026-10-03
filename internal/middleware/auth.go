package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/flutapp/chat-service/internal/client"
	"github.com/flutapp/chat-service/internal/domain"
	"log"
	"net/http"
	"strings"

	"github.com/flutapp/chat-service/internal/service"
)

type contextKey string

const userIDContextKey contextKey = "userID"

type WebSocketTicketConsumer interface {
	Consume(ticket string) (userID string, err error)
}

// RequireAuth verifies the "Authorization: Bearer <jwt>" header and stores the
// resolved userID in the request context for handlers to read via UserID.
func RequireAuth(authService service.AuthService, resolvers ...CompanyResolver) func(http.Handler) http.Handler {
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

			serveAuthenticated(w, r, userID, resolvers, next)
		})
	}
}

// RequireWebSocketAuth accepts the existing Bearer flow used by native mobile
// clients or a short-lived, single-use ticket used by browser clients.
func RequireWebSocketAuth(authService service.AuthService, tickets WebSocketTicketConsumer, resolvers ...CompanyResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				userID, err := authService.VerifyToken(strings.TrimPrefix(authHeader, "Bearer "))
				if err != nil {
					log.Printf("[ws-auth] bearer verification failed: %v", err)
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				serveAuthenticated(w, r, userID, resolvers, next)
				return
			}

			userID, err := tickets.Consume(r.URL.Query().Get("ticket"))
			if err != nil {
				log.Printf("[ws-auth] ticket verification failed")
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			serveAuthenticated(w, r, userID, resolvers, next)
		})
	}
}

func withUserID(r *http.Request, userID string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), userIDContextKey, userID))
}

// UserID reads the userID stored by RequireAuth.
func UserID(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(userIDContextKey).(string)
	return userID, ok
}

type CompanyResolver interface {
	ResolveChatContext(context.Context, string, string) (*client.ChatContext, error)
}
type companyIdentity struct {
	Actor         string
	Authorization string
	Company       string
}

const companyIdentityKey contextKey = "companyIdentity"

func ActorID(r *http.Request) string {
	if identity, ok := r.Context().Value(companyIdentityKey).(companyIdentity); ok {
		return identity.Actor
	}
	id, _ := UserID(r)
	return id
}
func TicketIdentity(r *http.Request) string {
	if identity, ok := r.Context().Value(companyIdentityKey).(companyIdentity); ok {
		encoded, _ := json.Marshal(identity)
		return "company:" + string(encoded)
	}
	id, _ := UserID(r)
	return id
}
func serveAuthenticated(w http.ResponseWriter, r *http.Request, userID string, resolvers []CompanyResolver, next http.Handler) {
	identity := companyIdentity{Actor: userID, Authorization: r.Header.Get("Authorization"), Company: r.Header.Get("X-Company-ID")}
	if strings.HasPrefix(userID, "company:") {
		if json.Unmarshal([]byte(strings.TrimPrefix(userID, "company:")), &identity) != nil {
			http.Error(w, "unauthorized", 401)
			return
		}
	}
	if identity.Company == "" {
		next.ServeHTTP(w, withUserID(r, userID))
		return
	}
	if len(resolvers) == 0 {
		http.Error(w, "forbidden", 403)
		return
	}
	resolve := func(ctx context.Context) error {
		resolved, err := resolvers[0].ResolveChatContext(ctx, identity.Authorization, identity.Company)
		if err != nil || resolved == nil || resolved.ActorID != identity.Actor || resolved.CompanyID != identity.Company || resolved.OwnerID != identity.Company {
			return fmt.Errorf("company access denied")
		}
		return nil
	}
	if resolve(r.Context()) != nil {
		http.Error(w, "forbidden", 403)
		return
	}
	ctx := context.WithValue(r.Context(), companyIdentityKey, identity)
	ctx = domain.WithChatActor(ctx, identity.Actor)
	ctx = domain.WithChatCompany(ctx, identity.Company)
	ctx = context.WithValue(ctx, validatorKey, resolve)
	next.ServeHTTP(w, withUserID(r.WithContext(ctx), identity.Company))
}

const validatorKey contextKey = "companyValidator"

func CompanyValidator(r *http.Request) func(context.Context) error {
	fn, _ := r.Context().Value(validatorKey).(func(context.Context) error)
	return fn
}
