package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrInvalidWebSocketTicket = errors.New("invalid or expired websocket ticket")

type WebSocketTicketService interface {
	Issue(userID string) (ticket string, expiresAt time.Time, err error)
	Consume(ticket string) (userID string, err error)
}

type websocketTicket struct {
	userID    string
	expiresAt time.Time
}

type inMemoryWebSocketTicketService struct {
	mu      sync.Mutex
	tickets map[string]websocketTicket
	ttl     time.Duration
	now     func() time.Time
}

func NewWebSocketTicketService(ttl time.Duration, now func() time.Time) WebSocketTicketService {
	return &inMemoryWebSocketTicketService{
		tickets: make(map[string]websocketTicket),
		ttl:     ttl,
		now:     now,
	}
}

func (s *inMemoryWebSocketTicketService) Issue(userID string) (string, time.Time, error) {
	if userID == "" {
		return "", time.Time{}, fmt.Errorf("user id is required")
	}
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", time.Time{}, fmt.Errorf("generate websocket ticket: %w", err)
	}
	ticket := base64.RawURLEncoding.EncodeToString(random)
	expiresAt := s.now().Add(s.ttl)

	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	for key, record := range s.tickets {
		if !now.Before(record.expiresAt) {
			delete(s.tickets, key)
		}
	}
	s.tickets[ticket] = websocketTicket{userID: userID, expiresAt: expiresAt}
	return ticket, expiresAt, nil
}

func (s *inMemoryWebSocketTicketService) Consume(ticket string) (string, error) {
	if ticket == "" {
		return "", ErrInvalidWebSocketTicket
	}

	s.mu.Lock()
	record, ok := s.tickets[ticket]
	if ok {
		delete(s.tickets, ticket)
	}
	s.mu.Unlock()

	if !ok || !s.now().Before(record.expiresAt) {
		return "", ErrInvalidWebSocketTicket
	}
	return record.userID, nil
}
