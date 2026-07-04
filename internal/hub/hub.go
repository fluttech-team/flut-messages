package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

// Client represents a connected WebSocket client
type Client struct {
	ID    string
	Conn  interface{}
	Send  chan []byte
	Rooms map[string]bool
	mu    sync.RWMutex
}

// broadcastMessage represents a message to be broadcasted to specific rooms
type broadcastMessage struct {
	rooms   []string
	userID  string
	message []byte
}

// Hub manages all connected clients and rooms
type Hub struct {
	clients      map[*Client]bool
	rooms        map[string]map[*Client]bool
	register     chan *Client
	unregister   chan *Client
	broadcast    chan broadcastMessage
	mu           sync.RWMutex
}

// NewHub creates a new Hub instance
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
		broadcast:  make(chan broadcastMessage, 256),
	}
}

// Run starts the main event loop for the hub
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// Clean up all clients
			h.mu.Lock()
			for client := range h.clients {
				close(client.Send)
			}
			h.clients = make(map[*Client]bool)
			h.rooms = make(map[string]map[*Client]bool)
			h.mu.Unlock()
			log.Println("Hub stopped")
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Client registered: %s", client.ID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)

				// Remove client from all rooms
				for roomID := range h.rooms {
					if _, exists := h.rooms[roomID][client]; exists {
						delete(h.rooms[roomID], client)
						if len(h.rooms[roomID]) == 0 {
							delete(h.rooms, roomID)
						}
					}
				}

				h.mu.Unlock()
				log.Printf("Client unregistered: %s", client.ID)
			} else {
				h.mu.Unlock()
			}

		case msg := <-h.broadcast:
			h.mu.RLock()
			for _, roomID := range msg.rooms {
				if room, exists := h.rooms[roomID]; exists {
					for client := range room {
						select {
						case client.Send <- msg.message:
						default:
							// Channel full, skip sending to this client
						}
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// JoinRoom adds a client to a room
func (h *Hub) JoinRoom(client *Client, roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Add room if it doesn't exist
	if _, exists := h.rooms[roomID]; !exists {
		h.rooms[roomID] = make(map[*Client]bool)
	}

	// Add client to room
	h.rooms[roomID][client] = true

	// Add room to client's rooms
	client.mu.Lock()
	client.Rooms[roomID] = true
	client.mu.Unlock()

	log.Printf("Client %s joined room %s", client.ID, roomID)
}

// LeaveRoom removes a client from a room
func (h *Hub) LeaveRoom(client *Client, roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if room, exists := h.rooms[roomID]; exists {
		if _, clientExists := room[client]; clientExists {
			delete(room, client)
			if len(room) == 0 {
				delete(h.rooms, roomID)
			}
		}
	}

	client.mu.Lock()
	delete(client.Rooms, roomID)
	client.mu.Unlock()

	log.Printf("Client %s left room %s", client.ID, roomID)
}

// Broadcast sends a message to all clients in specified rooms
func (h *Hub) Broadcast(rooms []string, payload interface{}) error {
	message, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := broadcastMessage{
		rooms:   rooms,
		userID:  "",
		message: message,
	}

	h.broadcast <- msg
	return nil
}

// SendToUser sends a message to a specific user by ID
func (h *Hub) SendToUser(userID string, payload interface{}) error {
	message, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	h.mu.RLock()
	var targetClient *Client
	for client := range h.clients {
		if client.ID == userID {
			targetClient = client
			break
		}
	}
	h.mu.RUnlock()

	if targetClient == nil {
		return fmt.Errorf("user %s not found", userID)
	}

	select {
	case targetClient.Send <- message:
	default:
		return fmt.Errorf("failed to send message to user %s: channel full", userID)
	}

	return nil
}

// GetConnectedUsers returns a slice of unique connected user IDs
func (h *Hub) GetConnectedUsers() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	users := make([]string, 0, len(h.clients))
	userMap := make(map[string]bool)

	for client := range h.clients {
		if !userMap[client.ID] {
			users = append(users, client.ID)
			userMap[client.ID] = true
		}
	}

	return users
}
