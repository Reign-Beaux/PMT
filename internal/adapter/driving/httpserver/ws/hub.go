package ws

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"project-management-tools/internal/application/notification"
	"project-management-tools/internal/domain/shared"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

// client represents a single WebSocket connection.
type client struct {
	ownerID shared.ID
	conn    *websocket.Conn
	send    chan []byte
	hub     *Hub
}

// writePump pumps messages from the send channel to the WebSocket connection.
// Runs in its own goroutine per client.
func (c *client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel.
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump reads incoming messages (discards them) and handles the connection lifecycle.
// Runs in its own goroutine per client.
func (c *client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		// Clients are consumers-only; discard any incoming message.
		if _, _, err := c.conn.ReadMessage(); err != nil {
			break
		}
	}
}

// Hub maintains the set of active WebSocket connections and broadcasts events.
// It implements notification.Notifier.
type Hub struct {
	mu         sync.RWMutex
	clients    map[shared.ID]map[*client]struct{} // ownerID → connections
	register   chan *client
	unregister chan *client
	broadcast  chan broadcastMsg
}

type broadcastMsg struct {
	ownerID shared.ID
	data    []byte
}

// NewHub creates and returns a new Hub. Call Run in a goroutine before using it.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[shared.ID]map[*client]struct{}),
		register:   make(chan *client),
		unregister: make(chan *client),
		broadcast:  make(chan broadcastMsg, 256),
	}
}

// Run processes register/unregister/broadcast events.
// Must be called in a dedicated goroutine before accepting connections.
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			if h.clients[c.ownerID] == nil {
				h.clients[c.ownerID] = make(map[*client]struct{})
			}
			h.clients[c.ownerID][c] = struct{}{}
			h.mu.Unlock()

		case c := <-h.unregister:
			h.mu.Lock()
			if set, ok := h.clients[c.ownerID]; ok {
				if _, ok := set[c]; ok {
					delete(set, c)
					close(c.send)
					if len(set) == 0 {
						delete(h.clients, c.ownerID)
					}
				}
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			set := h.clients[msg.ownerID]
			targets := make([]*client, 0, len(set))
			for c := range set {
				targets = append(targets, c)
			}
			h.mu.RUnlock()

			for _, c := range targets {
				select {
				case c.send <- msg.data:
				default:
					c.conn.Close()
				}
			}
		}
	}
}

// Notify implements notification.Notifier.
// It serialises the event to JSON and enqueues it for all clients owned by ownerID.
// Non-blocking: if the broadcast channel is full the event is dropped rather than
// stalling the calling use case.
func (h *Hub) Notify(ownerID shared.ID, event notification.Event) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	select {
	case h.broadcast <- broadcastMsg{ownerID: ownerID, data: data}:
	default:
		// Hub backpressure: drop the event to avoid blocking the caller.
	}
}
roadcast <- broadcastMsg{ownerID: ownerID, data: data}:
	default:
		// Hub backpressure: drop the event to avoid blocking the caller.
	}
}
