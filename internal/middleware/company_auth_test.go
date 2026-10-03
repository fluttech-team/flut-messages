package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flutapp/chat-service/internal/client"
)

type companyAuthStub struct{}

func (companyAuthStub) VerifyToken(string) (string, error) { return "member", nil }

type companyResolverStub struct{ denied bool }

func (s *companyResolverStub) ResolveChatContext(context.Context, string, string) (*client.ChatContext, error) {
	if s.denied {
		return nil, errors.New("revoked")
	}
	return &client.ChatContext{ActorID: "member", CompanyID: "owner", OwnerID: "owner"}, nil
}
func TestCompanyAuthPreservesActorAndRechecksMembership(t *testing.T) {
	resolver := &companyResolverStub{}
	handler := RequireAuth(companyAuthStub{}, resolver)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := UserID(r)
		if id != "owner" || ActorID(r) != "member" {
			t.Fatalf("wrong identity %s %s", id, ActorID(r))
		}
		w.WriteHeader(204)
	}))
	request := httptest.NewRequest("GET", "/conversations", nil)
	request.Header.Set("Authorization", "Bearer jwt")
	request.Header.Set("X-Company-ID", "owner")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 204 {
		t.Fatal(response.Code)
	}
	resolver.denied = true
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 403 {
		t.Fatal(response.Code)
	}
}

func TestCompanyAuthRejectsMismatchedContextAndMissingResolver(t *testing.T) {
	for _, company := range []string{"other-owner", "owner"} {
		var resolvers []CompanyResolver
		if company == "other-owner" {
			resolvers = []CompanyResolver{&companyResolverStub{}}
		}
		handler := RequireAuth(companyAuthStub{}, resolvers...)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("unauthorized context passed") }))
		request := httptest.NewRequest("GET", "/conversations", nil)
		request.Header.Set("Authorization", "Bearer jwt")
		request.Header.Set("X-Company-ID", company)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != 403 {
			t.Fatal(response.Code)
		}
	}
}

type identityTicketStub struct{ identity string }

func (s identityTicketStub) Consume(string) (string, error) { return s.identity, nil }
func TestCompanyTicketRevalidatesAndRetainsActor(t *testing.T) {
	resolver := &companyResolverStub{}
	var identity string
	var validate func(context.Context) error
	issuance := RequireAuth(companyAuthStub{}, resolver)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity = TicketIdentity(r)
		validate = CompanyValidator(r)
	}))
	request := httptest.NewRequest("POST", "/ws-tickets", nil)
	request.Header.Set("Authorization", "Bearer jwt")
	request.Header.Set("X-Company-ID", "owner")
	issuance.ServeHTTP(httptest.NewRecorder(), request)
	if identity == "" || validate == nil {
		t.Fatal("missing company ticket identity")
	}
	ws := RequireWebSocketAuth(companyAuthStub{}, identityTicketStub{identity}, resolver)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ActorID(r) != "member" {
			t.Fatal("lost actor")
		}
		w.WriteHeader(204)
	}))
	response := httptest.NewRecorder()
	ws.ServeHTTP(response, httptest.NewRequest("GET", "/ws?ticket=opaque", nil))
	if response.Code != 204 {
		t.Fatal(response.Code)
	}
	resolver.denied = true
	if validate(context.Background()) == nil {
		t.Fatal("live socket did not detect revocation")
	}
	response = httptest.NewRecorder()
	ws.ServeHTTP(response, httptest.NewRequest("GET", "/ws?ticket=opaque", nil))
	if response.Code != 403 {
		t.Fatal(response.Code)
	}
}
