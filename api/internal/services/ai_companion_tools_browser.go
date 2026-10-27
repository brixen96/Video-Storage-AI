package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// browser timeouts per command class (extension may need a moment to wake up).
const (
	browserReadTimeout    = 12 * time.Second
	browserActionTimeout  = 10 * time.Second
	browserQueryTimeout   = 8 * time.Second
	browserSnapshotMaxLen = 60000
)

func bArgStr(args map[string]interface{}, key, def string) string {
	if v, ok := args[key].(string); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return def
}

func bArgInt(args map[string]interface{}, key string, def, max int) int {
	if v, ok := args[key].(float64); ok && v > 0 {
		n := int(v)
		if n > max {
			return max
		}
		return n
	}
	return def
}

func bArgBool(args map[string]interface{}, key string) bool {
	v, _ := args[key].(bool)
	return v
}

// bResult converts an extension result payload into a tool return value.
func bResult(raw json.RawMessage) (interface{}, error) {
	var out interface{}
	if len(raw) == 0 {
		return map[string]interface{}{"ok": true}, nil
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return string(raw), nil
	}
	return out, nil
}

// bErr adds the failing action name to bridge errors so the LLM can recover or advise.
func bErr(action string, err error) error {
	return fmt.Errorf("browser_%s failed: %w", action, err)
}

// registerBrowserTools registers the Firefox bridge tool suite. Read-only tools
// are observation; action tools are user-driven only (never supervisor-proposed,
// see supervisorProposalTools). Registration is additive via buildToolRegistry.
func (s *AICompanionService) registerBrowserTools() {
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "browser_status",
			Description: "Check the live state of Brix's real Firefox browser: whether the VideoStorageAI extension is connected, which page/tab is active, and which tabs are open. Use before any browser task to confirm you have a connection, and when asked 'what am I looking at / what am I browsing'.",
			Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "required": []string{}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		st := GetBrowserBridge().Status()
		tabs := st.State.Tabs
		if len(tabs) > 20 {
			tabs = tabs[:20]
		}
		return map[string]interface{}{
			"connected":         st.Connected,
			"active_tab":        st.State.Active,
			"open_tabs":         tabs,
			"tab_count":         len(st.State.Tabs),
			"state_updated_at":  st.State.UpdatedAt,
			"screenshot_ready":  st.HasFrame,
			"bookmark_count":    st.BookmarkCount,
			"last_bridge_error": st.LastError,
			"hint":              "If connected is false, tell Brix to enable the VideoStorageAI extension in about:addons.",
		}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "browser_read",
			Description: "Read the page Brix is currently viewing in his real Firefox browser. Returns the page URL/title plus a structured snapshot of its elements (headings, links, buttons, inputs, labels) with ready-to-use CSS selectors, and captures a screenshot for his Browser panel. ALWAYS call this before browser_click/browser_type/browser_scroll so your selectors are current - selectors go stale after any navigation.",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		br := GetBrowserBridge()
		snapRaw, err := br.Request("snapshot", map[string]interface{}{}, browserReadTimeout)
		if err != nil {
			return nil, bErr("read", err)
		}
		frameRaw, frameErr := br.Request("screenshot", map[string]interface{}{}, browserReadTimeout)
		if frameErr == nil && len(frameRaw) > 0 {
			var frame struct {
				Image string `json:"image"`
			}
			if json.Unmarshal(frameRaw, &frame) == nil && frame.Image != "" {
				br.SetFrame(frame.Image)
			}
		}
		var pageOut interface{}
		snapStr := string(snapRaw)
		truncated := false
		if len(snapStr) > browserSnapshotMaxLen {
			pageOut = snapStr[:browserSnapshotMaxLen] + " ...(SNAPSHOT TRUNCATED - top elements shown above)"
			truncated = true
		} else if err := json.Unmarshal(snapRaw, &pageOut); err != nil {
			return nil, fmt.Errorf("could not parse page snapshot: %w", err)
		}
		return map[string]interface{}{
			"page":              pageOut,
			"truncated":         truncated,
			"screenshot_taken":  frameErr == nil,
			"note":              "Brix sees this page live in his Browser panel. Use the 'selector' values from this snapshot with browser_click / browser_type / browser_scroll.",
			"screenshot_failed": frameErr != nil,
		}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "browser_downloads",
			Description: "List recent downloads in Brix's Firefox browser (filename, source URL, progress, state: queued/complete/interrupted, error). Use when asked about downloads, download progress, or where a downloaded file came from.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"limit": map[string]interface{}{"type": "integer", "description": "Max downloads to return (default 10, max 50)"},
				},
				"required": []string{},
			},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		limit := bArgInt(args, "limit", 10, 50)
		raw, err := GetBrowserBridge().Request("downloads", map[string]interface{}{"limit": limit}, browserQueryTimeout)
		if err != nil {
			return nil, bErr("downloads", err)
		}
		return bResult(raw)
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "browser_history",
			Description: "Search Brix's Firefox browsing history. Returns matching pages with title, URL and visit time. Use when asked what he visited, when he was on a site, or to find a page he can't locate.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text":  map[string]interface{}{"type": "string", "description": "Search words (empty = most recent history)"},
					"days":  map[string]interface{}{"type": "integer", "description": "Look back this many days (default 7)"},
					"limit": map[string]interface{}{"type": "integer", "description": "Max entries (default 20, max 50)"},
				},
				"required": []string{},
			},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		q := map[string]interface{}{
			"text":  bArgStr(args, "text", ""),
			"days":  bArgInt(args, "days", 7, 365),
			"limit": bArgInt(args, "limit", 20, 50),
		}
		raw, err := GetBrowserBridge().Request("history", q, browserQueryTimeout)
		if err != nil {
			return nil, bErr("history", err)
		}
		return bResult(raw)
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "browser_bookmarks",
			Description: "Search Brix's Firefox bookmarks (cached locally from his real bookmark library). Returns matching title + URL. Use when asked about his bookmarks or to open something bookmarked (combine with browser_navigate to open it).",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string", "description": "Words to match in bookmark title or URL (empty = list most recent)"},
					"limit": map[string]interface{}{"type": "integer", "description": "Max results (default 30, max 100)"},
				},
				"required": []string{},
			},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		br := GetBrowserBridge()
		limit := bArgInt(args, "limit", 30, 100)
		query := bArgStr(args, "query", "")
		bookmarks := br.GetBookmarks()
		if len(bookmarks) == 0 {
			raw, err := br.Request("bookmarks_get", map[string]interface{}{}, browserQueryTimeout)
			if err != nil {
				return nil, bErr("bookmarks", err)
			}
			var list []BrowserBookmark
			if json.Unmarshal(raw, &list) == nil {
				br.SetBookmarks(list)
				bookmarks = list
			}
		}
		matches := make([]BrowserBookmark, 0, limit)
		q := strings.ToLower(strings.TrimSpace(query))
		for _, bm := range bookmarks {
			if q == "" || strings.Contains(strings.ToLower(bm.Title), q) || strings.Contains(strings.ToLower(bm.URL), q) {
				matches = append(matches, bm)
				if len(matches) >= limit {
					break
				}
			}
		}
		return map[string]interface{}{
			"query":           query,
			"total_cached":    len(bookmarks),
			"matches":         matches,
			"bookmark_count":  len(bookmarks),
		}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "browser_navigate",
			Description: "Navigate Brix's active Firefox tab to a URL (or a search phrase - falls back to DuckDuckGo). The user's real browser with his real logins is used. After navigating, call browser_read to see the new page before acting on it.",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{"type": "string", "description": "Full URL like https://example.com, or a search phrase"},
				},
				"required": []string{"url"},
			},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		target := strings.TrimSpace(bArgStr(args, "url", ""))
		if target == "" {
			return nil, fmt.Errorf("browser_navigate: url is required")
		}
		if !strings.Contains(target, " ") && !strings.Contains(target, "://") && !strings.HasPrefix(target, "about:") {
			target = "https://" + target
		} else if strings.Contains(target, " ") {
			target = "https://duckduckgo.com/?q=" + strings.ReplaceAll(target, " ", "+")
		}
		raw, err := GetBrowserBridge().Request("navigate", map[string]interface{}{"url": target}, browserActionTimeout)
		if err != nil {
			return nil, bErr("navigate", err)
		}
		return bResult(raw)
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "browser_history_nav",
			Description: "Move Brix's active Firefox tab back, forward, or reload it. Use after a click led somewhere unintended or when he asks to go back/retry.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"direction": map[string]interface{}{"type": "string", "enum": []string{"back", "forward", "reload"}, "description": "Navigation direction"},
				},
				"required": []string{"direction"},
			},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		dir := bArgStr(args, "direction", "")
		if dir != "back" && dir != "forward" && dir != "reload" {
			return nil, fmt.Errorf("browser_history_nav: direction must be back, forward or reload")
		}
		raw, err := GetBrowserBridge().Request("history_nav", map[string]interface{}{"direction": dir}, browserActionTimeout)
		if err != nil {
			return nil, bErr("history_nav", err)
		}
		return bResult(raw)
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "browser_click",
			Description: "Click an element on Brix's current Firefox page. ONLY pass a 'selector' that came from a fresh browser_read snapshot (selectors go stale after navigation). Clicks act inside his real logged-in browser, so only click what he asked for - never confirm purchases, deletions or account changes unless he explicitly told you to.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector": map[string]interface{}{"type": "string", "description": "CSS selector exactly as returned by browser_read"},
				},
				"required": []string{"selector"},
			},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		sel := strings.TrimSpace(bArgStr(args, "selector", ""))
		if sel == "" {
			return nil, fmt.Errorf("browser_click: selector is required")
		}
		raw, err := GetBrowserBridge().Request("click", map[string]interface{}{"selector": sel}, browserActionTimeout)
		if err != nil {
			return nil, bErr("click", err)
		}
		return bResult(raw)
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "browser_type",
			Description: "Type text into an input/textarea on Brix's current Firefox page (replaces existing value unless keep_value is true; works with React/Vue sites). Only use submit=true when Brix explicitly asked you to submit/send the form - his real logins and accounts are in play.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"selector":   map[string]interface{}{"type": "string", "description": "CSS selector from a fresh browser_read snapshot"},
					"text":       map[string]interface{}{"type": "string", "description": "Text to type"},
					"submit":     map[string]interface{}{"type": "boolean", "description": "Submit the surrounding form after typing (default false) - only when explicitly asked"},
					"keep_value": map[string]interface{}{"type": "boolean", "description": "Append instead of replacing the current value (default false)"},
				},
				"required": []string{"selector", "text"},
			},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		sel := strings.TrimSpace(bArgStr(args, "selector", ""))
		text := bArgStr(args, "text", "")
		if sel == "" {
			return nil, fmt.Errorf("browser_type: selector is required")
		}
		q := map[string]interface{}{
			"selector":   sel,
			"text":       text,
			"submit":     bArgBool(args, "submit"),
			"keep_value": bArgBool(args, "keep_value"),
		}
		raw, err := GetBrowserBridge().Request("type", q, browserActionTimeout)
		if err != nil {
			return nil, bErr("type", err)
		}
		return bResult(raw)
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "browser_scroll",
			Description: "Scroll Brix's current Firefox page (page-level). Use to read content below the fold; combine with browser_read to see new elements.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"direction": map[string]interface{}{"type": "string", "enum": []string{"down", "up", "top", "bottom"}, "description": "Scroll direction"},
					"amount":    map[string]interface{}{"type": "integer", "description": "Pixels to scroll (default 700, ignored for top/bottom)"},
				},
				"required": []string{"direction"},
			},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		dir := bArgStr(args, "direction", "")
		if dir != "down" && dir != "up" && dir != "top" && dir != "bottom" {
			return nil, fmt.Errorf("browser_scroll: direction must be down, up, top or bottom")
		}
		q := map[string]interface{}{
			"direction": dir,
			"amount":    bArgInt(args, "amount", 700, 20000),
		}
		raw, err := GetBrowserBridge().Request("scroll", q, browserActionTimeout)
		if err != nil {
			return nil, bErr("scroll", err)
		}
		return bResult(raw)
	})
}
