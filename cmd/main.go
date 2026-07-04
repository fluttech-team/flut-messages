package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/flutapp/chat-service/internal/config"
	"github.com/flutapp/chat-service/internal/handler"
	"github.com/flutapp/chat-service/internal/hub"
	"github.com/flutapp/chat-service/internal/repository"
	"github.com/flutapp/chat-service/internal/rest"
	"github.com/flutapp/chat-service/internal/service"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingInterval = 54 * time.Second

	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Maximum message size allowed from peer.
	maxMessageSize = 512 * 1024 // 512KB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

func main() {
	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Config validation failed: %v", err)
	}

	log.Printf("Starting server on port %s (env: %s)", cfg.Port, cfg.Env)

	// Connect to MongoDB
	mongoClient, err := repository.NewMongoClient(cfg.MongoDBURI)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if err := mongoClient.Disconnect(context.Background()); err != nil {
			log.Printf("Error disconnecting MongoDB: %v", err)
		}
	}()

	// Get database
	db := repository.GetDatabase(mongoClient, "chat_db")

	// Initialize repositories
	conversationRepo, err := repository.NewConversationRepository(db)
	if err != nil {
		log.Fatalf("Failed to initialize conversation repository: %v", err)
	}

	messageRepo, err := repository.NewMessageRepository(db)
	if err != nil {
		log.Fatalf("Failed to initialize message repository: %v", err)
	}

	blockRepo, err := repository.NewBlockRepository(db)
	if err != nil {
		log.Fatalf("Failed to initialize block repository: %v", err)
	}

	// Initialize services
	authService := service.NewAuthService(cfg.JWTSecret)
	conversationService := service.NewConversationService(conversationRepo)
	messageService := service.NewMessageService(messageRepo, conversationRepo, blockRepo)
	blockService := service.NewBlockService(blockRepo)

	// Initialize Hub and start it
	h := hub.NewHub()
	go h.Run(context.Background())
	log.Println("WebSocket Hub started")

	// Initialize handlers
	wsHandler := handler.NewWebSocketHandler(messageService, conversationService, h, blockService)
	restHandler := rest.NewRESTHandler(conversationService, messageService, blockService)

	// Setup HTTP routes
	mux := http.NewServeMux()

	// REST endpoints
	mux.HandleFunc("GET /conversations", restHandler.GetConversations)
	mux.HandleFunc("GET /conversations/{id}/messages", restHandler.GetMessages)
	mux.HandleFunc("GET /conversations/{id}/search", restHandler.SearchMessages)
	mux.HandleFunc("POST /users/{id}/block", restHandler.BlockUser)
	mux.HandleFunc("DELETE /users/{id}/block", restHandler.UnblockUser)
	mux.HandleFunc("GET /users/blocked-list", restHandler.GetBlockedList)

	// WebSocket endpoint
	mux.HandleFunc("/ws", handleWebSocket(h, authService, wsHandler))

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

// handleWebSocket returns a handler for WebSocket connections
func handleWebSocket(h *hub.Hub, authService service.AuthService, wsHandler *handler.WebSocketHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract token from query parameter
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}

		// Verify token with auth service
		userID, err := authService.VerifyToken(token)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Upgrade connection
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		// Create client
		client := &hub.Client{
			ID:    userID,
			Conn:  conn,
			Send:  make(chan []byte, 256),
			Rooms: make(map[string]bool),
		}

		// Register client with hub
		h.Register(client)

		// Start read and write pumps
		go readPump(h, wsHandler, client)
		go writePump(client)
	}
}

// readPump reads messages from the WebSocket connection
func readPump(h *hub.Hub, wsHandler *handler.WebSocketHandler, client *hub.Client) {
	defer func() {
		h.Unregister(client)
		conn := client.Conn.(*websocket.Conn)
		conn.Close()
	}()

	conn := client.Conn.(*websocket.Conn)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	conn.SetReadLimit(maxMessageSize)

	for {
		// Read WebSocket event
		var event handler.WebSocketEvent
		err := conn.ReadJSON(&event)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle event
		ctx := context.Background()
		response := wsHandler.HandleEvent(ctx, client, event)

		// Send ack response back via client.Send channel
		responseBytes, err := json.Marshal(response)
		if err != nil {
			log.Printf("Failed to marshal response: %v", err)
			continue
		}

		select {
		case client.Send <- responseBytes:
		default:
			log.Printf("Client %s send channel full, closing connection", client.ID)
			break
		}
	}
}

// writePump writes messages to the WebSocket connection
func writePump(client *hub.Client) {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		conn := client.Conn.(*websocket.Conn)
		conn.Close()
	}()

	conn := client.Conn.(*websocket.Conn)

	for {
		select {
		case message, ok := <-client.Send:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message.
			n := len(client.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte("\n"))
				w.Write(<-client.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
