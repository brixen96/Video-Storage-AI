package websocket

import (
	"encoding/json"

	"github.com/brixen96/video-storage-ai/internal/models"
)

// Hub maintains the set of active clients and broadcasts messages to the
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

// Run starts the hub's event loop.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

// Broadcast broadcasts a message to all clients.
func (h *Hub) Broadcast(message []byte) {
	h.broadcast <- message
}

// BroadcastActivityUpdate broadcasts an activity update to all clients.
func (h *Hub) BroadcastActivityUpdate(activity *models.Activity) {
	// Wrap message with type
	wrapper := map[string]interface{}{
		"type": "activity_update",
		"data": activity,
	}
	message, err := json.Marshal(wrapper)
	if err == nil {
		h.Broadcast(message)
	}
}

// BroadcastStatusUpdate broadcasts a status update to all clients.
func (h *Hub) BroadcastStatusUpdate(status *models.ActivityStatus) {
	// Wrap message with type
	wrapper := map[string]interface{}{
		"type": "status_update",
		"data": status,
	}
	message, err := json.Marshal(wrapper)
	if err == nil {
		h.Broadcast(message)
	}
}

// BroadcastSystemEvent broadcasts a system event to all clients.
func (h *Hub) BroadcastSystemEvent(event string) {
	message, err := json.Marshal(SystemEvent{
		Type:  "system",
		Event: event,
	})
	if err == nil {
		h.Broadcast(message)
	}
}

// BroadcastConsoleLog broadcasts a console log entry to all clients.
func (h *Hub) BroadcastConsoleLog(log *models.ConsoleLog) {
	// Wrap message with type
	wrapper := map[string]interface{}{
		"type": "console_log",
		"data": log,
	}
	message, err := json.Marshal(wrapper)
	if err == nil {
		h.Broadcast(message)
	}
}

// BroadcastNotification broadcasts a notification to all clients.
func (h *Hub) BroadcastNotification(notification *models.Notification) {
	// Wrap message with type
	wrapper := map[string]interface{}{
		"type": "notification",
		"data": notification,
	}
	message, err := json.Marshal(wrapper)
	if err == nil {
		h.Broadcast(message)
	}
}

// Register registers a new client with the hub.
func (h *Hub) Register(client *Client) {
	h.register <- client
}