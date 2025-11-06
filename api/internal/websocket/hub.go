
package websocket

import (
    "encoding/json"
    "log"

    "github.com/brixen96/video-storage-ai/internal/models"
)

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
    clients    map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
    return &Hub{
        clients:    make(map[*Client]bool),
        broadcast:  make(chan []byte, 256),
        register:   make(chan *Client),
        unregister: make(chan *Client),
    }
}

// Run starts the hub's main loop
func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.clients[client] = true
            log.Printf("Client connected. Total clients: %d", len(h.clients))

        case client := <-h.unregister:
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
                log.Printf("Client disconnected. Total clients: %d", len(h.clients))
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

// Register registers a new client with the hub
func (h *Hub) Register(client *Client) {
    h.register <- client
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(client *Client) {
    h.unregister <- client
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(message []byte) {
    h.broadcast <- message
}

// BroadcastActivityUpdate broadcasts an activity update to all clients
func (h *Hub) BroadcastActivityUpdate(activity *models.Activity) {
    message := map[string]interface{}{
        "type": "activity_update",
        "data": activity,
    }

    data, err := json.Marshal(message)
    if err != nil {
        log.Printf("Failed to marshal activity update: %v", err)
        return
    }

    h.Broadcast(data)
}

// BroadcastStatusUpdate broadcasts a status update to all clients
func (h *Hub) BroadcastStatusUpdate(status *models.ActivityStatus) {
    message := map[string]interface{}{
        "type": "status_update",
        "data": status,
    }

    data, err := json.Marshal(message)
    if err != nil {
        log.Printf("Failed to marshal status update: %v", err)
        return
    }

    h.Broadcast(data)
}

// BroadcastMessage broadcasts a generic message to all clients
func (h *Hub) BroadcastMessage(messageType string, data interface{}) {
    message := map[string]interface{}{
        "type": messageType,
        "data": data,
    }

    jsonData, err := json.Marshal(message)
    if err != nil {
        log.Printf("Failed to marshal message: %v", err)
        return
    }

    h.Broadcast(jsonData)
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
    return len(h.clients)
}
