package websocket

// SystemEvent represents a system-level message, like an idle notification.
type SystemEvent struct {
	Type  string `json:"type"`
	Event string `json:"event"`
}