package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

// handleBrowserWS upgrades the Firefox extension's bridge connection.
// The extension is the sole client of this endpoint; the AI Companion and
// frontend talk to the bridge through the tools/status/screenshot routes.
func handleBrowserWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Browser bridge: upgrade failed: %v", err)
		return
	}
	services.GetBrowserBridge().HandleConnection(conn)
}

// getBrowserStatus returns connection state, active tab, open tabs and cache info.
func getBrowserStatus(c *gin.Context) {
	respondSuccess(c, services.GetBrowserBridge().Status(), "Browser bridge status")
}

// getBrowserScreenshot returns the most recent visible-tab screenshot as a data URL.
func getBrowserScreenshot(c *gin.Context) {
	frame, capturedAt, ok := services.GetBrowserBridge().GetFrame()
	if !ok {
		respondError(c, http.StatusNotFound, "No screenshot captured yet", nil)
		return
	}
	respondSuccess(c, gin.H{"image": frame, "captured_at": capturedAt}, "Latest screenshot")
}

// captureBrowserScreenshot asks the extension for a FRESH screenshot of the active tab, caches it and returns it. Powers the live Browser tab view.
func captureBrowserScreenshot(c *gin.Context) {
	raw, err := services.GetBrowserBridge().Request("screenshot", nil, 8*time.Second)
	if err != nil {
		respondBadRequest(c, "Screenshot failed", err)
		return
	}
	var shot struct {
		Image string `json:"image"`
		URL   string `json:"url"`
		Title string `json:"title"`
	}
	if json.Unmarshal(raw, &shot) != nil || shot.Image == "" {
		respondInternalError(c, "Could not parse screenshot result", nil)
		return
	}
	services.GetBrowserBridge().SetFrame(shot.Image)
	respondSuccess(c, gin.H{
		"image":        shot.Image,
		"url":          shot.URL,
		"title":        shot.Title,
		"captured_at":  time.Now().UTC().Format(time.RFC3339),
	}, "Screenshot captured")
}
