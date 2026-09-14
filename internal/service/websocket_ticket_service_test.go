package service

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWebSocketTicketCanBeConsumedOnlyOnce(t *testing.T) {
	now := time.Date(2026, time.September, 14, 12, 0, 0, 0, time.UTC)
	tickets := NewWebSocketTicketService(30*time.Second, func() time.Time { return now })

	ticket, expiresAt, err := tickets.Issue("user-1")
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}
	if ticket == "" {
		t.Fatal("expected a non-empty ticket")
	}
	if want := now.Add(30 * time.Second); !expiresAt.Equal(want) {
		t.Fatalf("expected expiry %s, got %s", want, expiresAt)
	}

	userID, err := tickets.Consume(ticket)
	if err != nil {
		t.Fatalf("consume ticket: %v", err)
	}
	if userID != "user-1" {
		t.Fatalf("expected user-1, got %s", userID)
	}
	if _, err := tickets.Consume(ticket); err != ErrInvalidWebSocketTicket {
		t.Fatalf("expected invalid ticket after first use, got %v", err)
	}
}

func TestWebSocketTicketExpires(t *testing.T) {
	now := time.Date(2026, time.September, 14, 12, 0, 0, 0, time.UTC)
	tickets := NewWebSocketTicketService(30*time.Second, func() time.Time { return now })
	ticket, _, err := tickets.Issue("user-1")
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}

	now = now.Add(31 * time.Second)
	if _, err := tickets.Consume(ticket); err != ErrInvalidWebSocketTicket {
		t.Fatalf("expected expired ticket to be invalid, got %v", err)
	}
}

func TestWebSocketTicketConcurrentConsumptionHasOneWinner(t *testing.T) {
	tickets := NewWebSocketTicketService(time.Minute, time.Now)
	ticket, _, err := tickets.Issue("user-1")
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}

	var winners atomic.Int32
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := tickets.Consume(ticket); err == nil {
				winners.Add(1)
			}
		}()
	}
	wg.Wait()

	if got := winners.Load(); got != 1 {
		t.Fatalf("expected exactly one successful consumer, got %d", got)
	}
}
