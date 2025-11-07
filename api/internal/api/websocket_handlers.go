package api

import (
	"log"
	"net/http"

	"github.com/brixen96/video-storage-ai/internal/websocket"
	"github.com/gin-gonic/gin"
	ws "github.com/gorilla/websocket"
)

var upgrader = ws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

var wsHub *websocket.Hub

// InitWebSocket initializes the WebSocket hub
func InitWebSocket() *websocket.Hub {
	wsHub = websocket.NewHub()
	go wsHub.Run()
	return wsHub
}

// GetWebSocketHub returns the WebSocket hub instance
func GetWebSocketHub() *websocket.Hub {
	return wsHub
}

// handleWebSocket handles WebSocket connections
func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Create new client using the constructor
	client := websocket.NewClient(wsHub, conn)

	// Register client with hub
	wsHub.Register(client)

	// Start client pumps
	client.Start()
}
