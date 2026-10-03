package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/flutapp/chat-service/internal/domain"
	"github.com/flutapp/chat-service/internal/utils"
)

// ApplicationParticipants mirrors backend-flut's
// GET /applications/:id/participants response.
type ApplicationParticipants struct {
	ApplicationID string `json:"application_id"`
	JobID         string `json:"job_id"`
	ApplicantID   string `json:"applicant_id"`
	CompanyID     string `json:"company_id"`
}

type BackendFlutClient interface {
	// GetApplicationParticipants forwards the caller's own Authorization header
	// to backend-flut, which re-validates the JWT (including session revocation)
	// and enforces that the caller is a party to the application.
	GetApplicationParticipants(ctx context.Context, authHeader, applicationID string) (*ApplicationParticipants, error)
}

type backendFlutClient struct {
	baseURL string
	http    *http.Client
}

func NewBackendFlutClient(baseURL string) *backendFlutClient {
	return &backendFlutClient{baseURL: baseURL, http: &http.Client{Timeout: 5 * time.Second}}
}

type envelope struct {
	Data ApplicationParticipants `json:"data"`
}

func (c *backendFlutClient) GetApplicationParticipants(ctx context.Context, authHeader, applicationID string) (*ApplicationParticipants, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/applications/"+applicationID+"/participants", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("X-Company-ID", domain.ChatCompany(ctx))

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var body envelope
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return nil, err
		}
		return &body.Data, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, utils.ErrForbidden
	case http.StatusNotFound:
		return nil, utils.ErrConversationNotFound
	default:
		return nil, fmt.Errorf("backend-flut: unexpected status %d", resp.StatusCode)
	}
}

// ChatContext is resolved by backend-flut using the actor's live session and membership.
type ChatContext struct {
	ActorID   string `json:"actor_id"`
	CompanyID string `json:"company_id"`
	OwnerID   string `json:"owner_id"`
}

func (c *backendFlutClient) ResolveChatContext(ctx context.Context, authHeader, companyID string) (*ChatContext, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/company/chat-context", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("X-Company-ID", companyID)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, utils.ErrForbidden
	}
	var result struct {
		Data ChatContext `json:"data"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}
