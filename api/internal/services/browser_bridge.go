package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// BrowserTabInfo describes one open tab as reported by the Firefox extension.
type BrowserTabInfo struct {
	TabID   int    `json:"tab_id"`
	URL     string `json:"url"`
	Title   string `json:"title"`
	FavIcon string `json:"fav_icon"`
	Active  bool   `json:"active"`
}

// BrowserState is the live browsing state broadcast to the frontend on every change.
type BrowserState struct {
	Connected bool             `json:"connected"`
	Active    *BrowserTabInfo  `json:"active,omitempty"`
	Tabs      []BrowserTabInfo `json:"tabs"`
	UpdatedAt string           `json:"updated_at"`
}

// BrowserStatus is the full bridge status returned by the status endpoint.
type BrowserStatus struct {
	Connected      bool         `json:"connected"`
	State          BrowserState `json:"state"`
	HasFrame       bool         `json:"has_frame"`
	FrameUpdatedAt string       `json:"frame_updated_at,omitempty"`
	BookmarkCount  int          `json:"bookmark_count"`
	LastError      string       `json:"last_error,omitempty"`
}

// BrowserBookmark is a flattened Firefox bookmark entry (title+url only).
type BrowserBookmark struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Added int64  `json:"added"`
}

// BrowserStateBroadcaster is implemented by the websocket hub (browser_state events).
type BrowserStateBroadcaster interface {
	BroadcastBrowserState(state interface{})
}

// bridgePending is an in-flight command waiting for the extension's reply.
type bridgePending struct {
	action  string
	args    map[string]interface{}
	created time.Time
	ch      chan bridgeResult
}

// bridgeResult is the extension's reply to a single command.
type bridgeResult struct {
	ok   bool
	data json.RawMessage
	err  string
}

// BrowserBridge connects the Firefox extension to the AI Companion tools.
// Single-extension design: the most recently connected Firefox instance wins.
type BrowserBridge struct {
	mu        sync.Mutex
	writeMu   sync.Mutex
	conn      *websocket.Conn
	seq       int
	pending   map[string]*bridgePending
	state     BrowserState
	frame     string
	frameAt   time.Time
	bookmarks []BrowserBookmark
	lastError string
	broadcast BrowserStateBroadcaster
}

var (
	browserBridgeOnce sync.Once
	browserBridge     *BrowserBridge
)

// GetBrowserBridge returns the lazily-initialized singleton bridge.
func GetBrowserBridge() *BrowserBridge {
	browserBridgeOnce.Do(func() {
		browserBridge = &BrowserBridge{
			pending: make(map[string]*bridgePending),
			state:   BrowserState{Tabs: []BrowserTabInfo{}},
		}
	})
	return browserBridge
}

// SetBrowserHub wires the websocket hub used for browser_state broadcasts.
func SetBrowserHub(hub BrowserStateBroadcaster) {
	br := GetBrowserBridge()
	br.mu.Lock()
	br.broadcast = hub
	br.mu.Unlock()
}

// HandleConnection runs the extension read loop until the connection drops.
// Pending commands younger than 20s are replayed on each fresh connection so
// a background-suspension blip does not eat an in-flight AI action.
func (b *BrowserBridge) HandleConnection(conn *websocket.Conn) {
	b.mu.Lock()
	b.conn = conn
	b.state.Connected = true
	b.state.UpdatedAt = nowISO()
	b.mu.Unlock()
	b.resendPending(conn)
	b.broadcastState()

	defer func() {
		b.mu.Lock()
		if b.conn == conn {
			b.conn = nil
			b.state.Connected = false
			b.state.UpdatedAt = nowISO()
		}
		b.mu.Unlock()
		conn.Close()
		b.broadcastState()
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		b.handleMessage(msg)
	}
}

// handleMessage processes one envelope sent by the extension.
func (b *BrowserBridge) handleMessage(msg []byte) {
	var env struct {
		ID    string          `json:"id"`
		Type  string          `json:"type"`
		OK    *bool           `json:"ok"`
		Data  json.RawMessage `json:"data"`
		Error string          `json:"error"`
	}
	if err := json.Unmarshal(msg, &env); err != nil {
		return
	}
	switch env.Type {
	case "hello":
		b.broadcastState()
	case "tab_state":
		var st struct {
			Tabs   []BrowserTabInfo `json:"tabs"`
			Active *BrowserTabInfo  `json:"active"`
		}
		if json.Unmarshal(env.Data, &st) == nil {
			b.mu.Lock()
			if st.Tabs != nil {
				b.state.Tabs = st.Tabs
			}
			b.state.Active = st.Active
			b.state.Connected = true
			b.state.UpdatedAt = nowISO()
			b.mu.Unlock()
			b.broadcastState()
		}
	case "bookmarks":
		var list []BrowserBookmark
		if json.Unmarshal(env.Data, &list) == nil {
			b.mu.Lock()
			b.bookmarks = list
			b.mu.Unlock()
		}
	case "result":
		b.mu.Lock()
		p := b.pending[env.ID]
		delete(b.pending, env.ID)
		b.mu.Unlock()
		if p == nil {
			return
		}
		ok := env.OK != nil && *env.OK
		if !ok {
			msgText := env.Error
			if msgText == "" {
				msgText = "extension reported failure"
			}
			b.setLastError(msgText)
		}
		select {
		case p.ch <- bridgeResult{ok: ok, data: env.Data, err: env.Error}:
		default:
		}
	}
}

// resendPending re-sends recent unresolved commands to a fresh connection.
func (b *BrowserBridge) resendPending(conn *websocket.Conn) {
	b.mu.Lock()
	cmds := make([]map[string]interface{}, 0, len(b.pending))
	for id, p := range b.pending {
		if time.Since(p.created) < 20*time.Second {
			cmds = append(cmds, map[string]interface{}{
				"type": "command", "id": id, "action": p.action, "args": p.args,
			})
		}
	}
	b.mu.Unlock()
	if len(cmds) == 0 {
		return
	}
	b.writeMu.Lock()
	defer b.writeMu.Unlock()
	for _, cmd := range cmds {
		_ = conn.WriteJSON(cmd)
	}
}

// Request sends a command to the extension and waits for its result.
func (b *BrowserBridge) Request(action string, args map[string]interface{}, timeout time.Duration) (json.RawMessage, error) {
	b.mu.Lock()
	if b.conn == nil {
		b.mu.Unlock()
		return nil, fmt.Errorf("Firefox extension not connected - the VideoStorageAI extension must be enabled in Firefox")
	}
	b.seq++
	id := fmt.Sprintf("br%d", b.seq)
	p := &bridgePending{action: action, args: args, created: time.Now(), ch: make(chan bridgeResult, 1)}
	b.pending[id] = p
	b.mu.Unlock()

	if err := b.write(map[string]interface{}{"type": "command", "id": id, "action": action, "args": args}); err != nil {
		b.mu.Lock()
		delete(b.pending, id)
		b.mu.Unlock()
		return nil, fmt.Errorf("browser bridge write failed: %w", err)
	}

	select {
	case res := <-p.ch:
		if !res.ok {
			msgText := res.err
			if msgText == "" {
				msgText = "extension reported failure"
			}
			return nil, fmt.Errorf("browser action %q failed: %s", action, msgText)
		}
		return res.data, nil
	case <-time.After(timeout):
		b.mu.Lock()
		delete(b.pending, id)
		b.mu.Unlock()
		return nil, fmt.Errorf("browser action %q timed out after %s (extension background may be asleep - a retry usually works)", action, timeout)
	}
}

// write serializes one message to the current extension connection.
func (b *BrowserBridge) write(v interface{}) error {
	b.mu.Lock()
	conn := b.conn
	b.mu.Unlock()
	if conn == nil {
		return fmt.Errorf("extension not connected")
	}
	b.writeMu.Lock()
	defer b.writeMu.Unlock()
	return conn.WriteJSON(v)
}

// Status returns the full bridge status for the status endpoint and tools.
func (b *BrowserBridge) Status() BrowserStatus {
	b.mu.Lock()
	defer b.mu.Unlock()
	st := BrowserStatus{
		Connected:  b.conn != nil,
		State:      b.state,
		HasFrame:   b.frame != "",
		BookmarkCount: len(b.bookmarks),
		LastError:  b.lastError,
	}
	st.State.Connected = b.conn != nil
	if !b.frameAt.IsZero() {
		st.FrameUpdatedAt = b.frameAt.UTC().Format(time.RFC3339)
	}
	return st
}

// SetFrame stores the latest screenshot dataURL for the frontend Browser tab.
func (b *BrowserBridge) SetFrame(dataURL string) {
	b.mu.Lock()
	b.frame = dataURL
	b.frameAt = time.Now()
	b.mu.Unlock()
}

// GetFrame returns the latest screenshot dataURL and its capture time.
func (b *BrowserBridge) GetFrame() (string, string, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.frame == "" {
		return "", "", false
	}
	return b.frame, b.frameAt.UTC().Format(time.RFC3339), true
}

// GetBookmarks returns a copy of the cached bookmark list.
func (b *BrowserBridge) GetBookmarks() []BrowserBookmark {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]BrowserBookmark, len(b.bookmarks))
	copy(out, b.bookmarks)
	return out
}

// SetBookmarks replaces the cached bookmark list (pull fetch or push update).
func (b *BrowserBridge) SetBookmarks(list []BrowserBookmark) {
	b.mu.Lock()
	b.bookmarks = list
	b.mu.Unlock()
}

// SearchBookmarks filters the cached bookmarks by title/url substring.
func (b *BrowserBridge) SearchBookmarks(query string, limit int) []BrowserBookmark {
	b.mu.Lock()
	defer b.mu.Unlock()
	q := strings.ToLower(strings.TrimSpace(query))
	out := []BrowserBookmark{}
	for _, bm := range b.bookmarks {
		if q == "" || strings.Contains(strings.ToLower(bm.Title), q) || strings.Contains(strings.ToLower(bm.URL), q) {
			out = append(out, bm)
			if len(out) >= limit {
				break
			}
		}
	}
	return out
}

func (b *BrowserBridge) setLastError(msgText string) {
	b.mu.Lock()
	b.lastError = msgText
	b.mu.Unlock()
}

func (b *BrowserBridge) broadcastState() {
	b.mu.Lock()
	hub := b.broadcast
	snap := b.state
	snap.Connected = b.conn != nil
	b.mu.Unlock()
	if hub != nil {
		hub.BroadcastBrowserState(snap)
	}
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}
