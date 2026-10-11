package services

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/fsnotify/fsnotify"
)

// stripBOM removes a UTF-8 byte order mark if present. Editors like VS Code
// can save JSON files with a BOM, which makes encoding/json fail with
// "invalid character '\u00ef' looking for beginning of value".
func stripBOM(b []byte) []byte {
	return bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})
}

// ── Roleplay Watcher ──
//
// Watches AIInfluence NPC JSON files on disk. When the mod writes a new NPC
// response (game save / dialogue update), the watcher extracts a compact state
// snapshot (latest responses, mood, relation, task) and stores it. Snapshots
// are injected into the AI Companion's system prompt context, so the
// Companion automatically knows what the NPC just said — no manual "the
// conversation was updated" messages needed.
//
// Scope (v1): read-only observation of user-chosen NPC files.

// roleplayWatch mirrors a row in roleplay_watches.
type roleplayWatch struct {
	ID        int64     `json:"id"`
	FilePath  string    `json:"file_path"`
	NPCName   string    `json:"npc_name"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

// roleplayState is the compact NPC state extracted from a mod JSON file.
type roleplayState struct {
	NPCName          string   `json:"npc_name"`
	StringID         string   `json:"string_id"`
	Mood             string   `json:"mood"`
	MoodReason       string   `json:"mood_reason,omitempty"`
	RelationValue    int      `json:"relation_value"`
	RelationDesc     string   `json:"relation_desc"`
	CurrentTask      string   `json:"current_task,omitempty"`
	LocationType     string   `json:"location_type,omitempty"`
	LastResponse     string   `json:"last_response,omitempty"`
	RecentHistory    []string `json:"recent_history,omitempty"`
	IsSick           bool     `json:"is_sick,omitempty"`
	RomancePartners  []string `json:"romance_partners,omitempty"`
	DynamicEvents    []string `json:"dynamic_events,omitempty"`
	ExtractedAt      string   `json:"extracted_at"`
	FileModifiedTime string   `json:"file_modified_time"`
}

// roleplayTarget identifies one watched NPC file inside its directory.
type roleplayTarget struct {
	Dir      string
	FileName string
}

// roleplayWatcherService manages fsnotify watchers for NPC JSON files.
// Windows note: we watch the PARENT DIRECTORY (not the file) so writes via
// temp-file+rename (common on Windows) still produce events, plus a periodic
// ModTime sweep as a guaranteed safety net.
type roleplayWatcherService struct {
	db        *sql.DB
	companion *AICompanionService
	mu        sync.Mutex
	watcher   *fsnotify.Watcher
	active    map[int64]roleplayTarget // watch_id -> dir + file name
	started   bool
}

// roleplayWatcher is the package-level singleton (lazy init, no package vars).
var (
	roleplayWatcherOnce sync.Once
	roleplayWatcherInst *roleplayWatcherService
)

func getRoleplayWatcher() *roleplayWatcherService {
	roleplayWatcherOnce.Do(func() {
		w, err := fsnotify.NewWatcher()
		if err != nil {
			log.Printf("[Roleplay] Failed to create file watcher: %v", err)
			return
		}
		roleplayWatcherInst = &roleplayWatcherService{
			db:      database.GetDB(),
			watcher: w,
			active:  make(map[int64]roleplayTarget),
		}
	})
	return roleplayWatcherInst
}

// StartRoleplayWatcher initializes the watcher and loads persisted watches.
// Called once at service startup. Safe to call multiple times.
func StartRoleplayWatcher(svc *AICompanionService) {
	rw := getRoleplayWatcher()
	if rw == nil || rw.watcher == nil {
		return
	}
	rw.companion = svc
	rw.mu.Lock()
	if rw.started {
		rw.mu.Unlock()
		return
	}
	rw.started = true
	rw.mu.Unlock()

	// Load persisted watches and activate them
	var watches []roleplayWatch
	rows, err := rw.db.Query(`SELECT id, file_path, npc_name, enabled FROM roleplay_watches WHERE enabled = 1`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var w roleplayWatch
			var enabled int
			if rows.Scan(&w.ID, &w.FilePath, &w.NPCName, &enabled) == nil {
				w.Enabled = enabled == 1
				watches = append(watches, w)
			}
		}
	}

	activated := 0
	for _, w := range watches {
		if rw.activateWatch(w.ID, w.FilePath) {
			activated++
		} else {
			log.Printf("[Roleplay] Cannot watch (missing?): %s", w.FilePath)
		}
	}
	log.Printf("[Roleplay] Watcher started — %d NPC file(s) watched (dir-level + 30s sweep)", activated)

	go rw.eventLoop()
}

// activateWatch registers the watch: fsnotify on the parent directory plus
// in-memory target entry. Returns false if the file is missing.
// Caller must NOT hold rw.mu.
func (rw *roleplayWatcherService) activateWatch(watchID int64, filePath string) bool {
	if _, err := os.Stat(filePath); err != nil {
		return false
	}
	dir := filepath.Dir(filePath)
	fileName := filepath.Base(filePath)

	rw.mu.Lock()
	defer rw.mu.Unlock()

	// Watch the directory once even if multiple NPC files live in it
	alreadyWatchingDir := false
	for _, t := range rw.active {
		if strings.EqualFold(t.Dir, dir) {
			alreadyWatchingDir = true
			break
		}
	}
	if !alreadyWatchingDir {
		if err := rw.watcher.Add(dir); err != nil {
			log.Printf("[Roleplay] Cannot watch dir %s: %v", dir, err)
			return false
		}
	}
	rw.active[watchID] = roleplayTarget{Dir: dir, FileName: fileName}
	return true
}

// eventLoop processes fsnotify events (debounced) and runs a periodic ModTime
// sweep so changes are NEVER missed even if fsnotify drops an event.
func (rw *roleplayWatcherService) eventLoop() {
	debounce := make(map[string]*time.Timer)
	var debounceMu sync.Mutex
	sweep := time.NewTicker(30 * time.Second)
	defer sweep.Stop()

	for {
		select {
		case event, ok := <-rw.watcher.Events:
			if !ok {
				return
			}
			// Only care about write/create/rename
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}
			path := rw.matchWatchedFile(event.Name)
			if path == "" {
				continue
			}
			debounceMu.Lock()
			if t, exists := debounce[path]; exists {
				t.Stop()
			}
			// Wait 700ms after the last write event (mod may write multiple times)
			debounce[path] = time.AfterFunc(700*time.Millisecond, func() {
				rw.handleFileChange(path)
			})
			debounceMu.Unlock()

		case <-sweep.C:
			rw.sweepChanges()

		case err, ok := <-rw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[Roleplay] Watcher error: %v", err)
		}
	}
}

// matchWatchedFile returns the full path when the fsnotify event path equals
// one of the watched NPC files (case-insensitive — Windows paths).
func (rw *roleplayWatcherService) matchWatchedFile(eventPath string) string {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	for _, t := range rw.active {
		full := filepath.Join(t.Dir, t.FileName)
		if strings.EqualFold(eventPath, full) {
			return full
		}
	}
	return ""
}

// sweepChanges stats every watched file and triggers a snapshot when the file's
// ModTime is newer than the latest stored snapshot. Guaranteed fallback for
// missed fsnotify events (Windows temp+rename patterns, service restarts).
func (rw *roleplayWatcherService) sweepChanges() {
	rw.mu.Lock()
	targets := make(map[int64]roleplayTarget, len(rw.active))
	for id, t := range rw.active {
		targets[id] = t
	}
	rw.mu.Unlock()

	for id, t := range targets {
		full := filepath.Join(t.Dir, t.FileName)
		fi, err := os.Stat(full)
		if err != nil {
			continue
		}
		var lastSnap string
		rw.db.QueryRow(`SELECT COALESCE(MAX(created_at), '') FROM roleplay_snapshots WHERE watch_id = ?`, id).Scan(&lastSnap)
		snapTime := time.Time{}
		if lastSnap != "" {
			if parsed, pErr := time.Parse("2006-01-02 15:04:05.999999999-07:00", lastSnap); pErr == nil {
				snapTime = parsed
			} else if parsed, pErr := time.Parse(time.RFC3339, lastSnap); pErr == nil {
				snapTime = parsed
			}
		}
		if fi.ModTime().After(snapTime.Add(time.Second)) {
			log.Printf("[Roleplay] Sweep detected change: %s (mod %s)", t.FileName, fi.ModTime().Format("15:04:05"))
			rw.handleFileChange(full)
		}
	}
}

// handleFileChange extracts a state snapshot from a changed NPC file.
func (rw *roleplayWatcherService) handleFileChange(path string) {
	state, err := extractRoleplayState(path)
	if err != nil {
		// File may be mid-write; skip quietly
		return
	}

	// Find the watch ID
	var watchID int64
	rw.db.QueryRow(`SELECT id FROM roleplay_watches WHERE file_path = ?`, path).Scan(&watchID)
	if watchID == 0 {
		return
	}

	stateJSON, _ := json.Marshal(state)
	_, err = rw.db.Exec(
		`INSERT INTO roleplay_snapshots (watch_id, npc_name, file_path, state_json, created_at) VALUES (?, ?, ?, ?, ?)`,
		watchID, state.NPCName, path, string(stateJSON), time.Now(),
	)
	if err != nil {
		log.Printf("[Roleplay] Failed to store snapshot: %v", err)
		return
	}
	// Prune: keep 50 snapshots per watch
	rw.db.Exec(`DELETE FROM roleplay_snapshots WHERE watch_id = ? AND id NOT IN (
		SELECT id FROM roleplay_snapshots WHERE watch_id = ? ORDER BY created_at DESC LIMIT 50
	)`, watchID, watchID)

	log.Printf("[Roleplay] NPC update detected: %s (mood=%s, relation=%s)",
		state.NPCName, state.Mood, state.RelationDesc)

	// Update NPC name on the watch row if it was empty
	if state.NPCName != "" {
		rw.db.Exec(`UPDATE roleplay_watches SET npc_name = ? WHERE id = ? AND (npc_name = '' OR npc_name IS NULL)`, state.NPCName, watchID)
	}

	// Broadcast so the frontend can toast the update
	if rw.companion != nil {
		rw.companion.emitEvent(CompanionEvent{
			Type:      "roleplay_update",
			Source:    "roleplay_watcher",
			Message:   fmt.Sprintf("%s: NPC state updated", state.NPCName),
			Severity:  "info",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"npc":       state.NPCName,
				"mood":      state.Mood,
				"relation":  state.RelationDesc,
				"file_path": path,
			},
		})
	}
}

// AddRoleplayWatch registers an NPC file for watching and activates it.
func AddRoleplayWatch(filePath string) (map[string]interface{}, error) {
	rw := getRoleplayWatcher()
	if rw == nil {
		return nil, fmt.Errorf("watcher not available")
	}

	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return nil, fmt.Errorf("file path is required")
	}
	if !strings.HasSuffix(strings.ToLower(filePath), ".json") {
		return nil, fmt.Errorf("only .json files can be watched")
	}
	if _, err := os.Stat(filePath); err != nil {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}

	// Pre-extract to validate it's an NPC file + capture the name
	state, err := extractRoleplayState(filePath)
	if err != nil {
		return nil, fmt.Errorf("file is not a valid NPC JSON: %v", err)
	}

	_, err = rw.db.Exec(
		`INSERT OR IGNORE INTO roleplay_watches (file_path, npc_name, enabled, created_at) VALUES (?, ?, 1, ?)`,
		filePath, state.NPCName, time.Now(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save watch: %v", err)
	}

	var watchID int64
	rw.db.QueryRow(`SELECT id FROM roleplay_watches WHERE file_path = ?`, filePath).Scan(&watchID)
	if watchID == 0 {
		return nil, fmt.Errorf("failed to register watch")
	}

	if !rw.activateWatch(watchID, filePath) {
		return nil, fmt.Errorf("failed to watch file (missing?)")
	}

	// Baseline snapshot: extract current state immediately so the Companion
	// has context without waiting for the next game save.
	rw.handleFileChange(filePath)

	log.Printf("[Roleplay] Now watching: %s (%s)", state.NPCName, filepath.Base(filePath))
	return map[string]interface{}{
		"watch_id": watchID, "npc_name": state.NPCName, "file_path": filePath,
		"mood": state.Mood, "relation": state.RelationDesc,
	}, nil
}

// hasPath checks if a file path is already watched (caller holds lock).
func (rw *roleplayWatcherService) hasPath(path string) bool {
	for _, t := range rw.active {
		if strings.EqualFold(filepath.Join(t.Dir, t.FileName), path) {
			return true
		}
	}
	return false
}

// RemoveRoleplayWatch stops watching a file and deletes the watch row.
// Snapshots are kept.
func RemoveRoleplayWatch(watchID int64) error {
	rw := getRoleplayWatcher()
	if rw == nil {
		return fmt.Errorf("watcher not available")
	}

	rw.mu.Lock()
	if t, ok := rw.active[watchID]; ok {
		delete(rw.active, watchID)
		// Remove the dir watch only if no other watch lives in the same dir
		stillNeeded := false
		for _, other := range rw.active {
			if strings.EqualFold(other.Dir, t.Dir) {
				stillNeeded = true
				break
			}
		}
		if !stillNeeded {
			rw.watcher.Remove(t.Dir)
		}
	}
	rw.mu.Unlock()

	_, err := rw.db.Exec(`DELETE FROM roleplay_watches WHERE id = ?`, watchID)
	if err != nil {
		return fmt.Errorf("failed to remove watch: %v", err)
	}
	log.Printf("[Roleplay] Removed watch #%d", watchID)
	return nil
}

// GetRoleplayWatches returns all watches with their latest snapshot time.
func GetRoleplayWatches() ([]map[string]interface{}, error) {
	rw := getRoleplayWatcher()
	if rw == nil {
		return nil, fmt.Errorf("watcher not available")
	}

	rows, err := rw.db.Query(`
		SELECT w.id, w.file_path, w.npc_name, w.enabled, w.created_at,
		       (SELECT MAX(created_at) FROM roleplay_snapshots WHERE watch_id = w.id)
		FROM roleplay_watches w ORDER BY w.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		var id int64
		var filePath, npcName, createdAt string
		var enabled int
		var lastSnapshot sql.NullString
		if rows.Scan(&id, &filePath, &npcName, &enabled, &createdAt, &lastSnapshot) == nil {
			result = append(result, map[string]interface{}{
				"id": id, "file_path": filePath, "npc_name": npcName,
				"enabled": enabled == 1, "created_at": createdAt,
				"last_snapshot_at": lastSnapshot.String,
			})
		}
	}
	return result, nil
}

// GetRoleplaySnapshots returns the latest snapshot per watch (the Companion's
// current view of each watched NPC).
func GetRoleplaySnapshots() ([]map[string]interface{}, error) {
	rw := getRoleplayWatcher()
	if rw == nil {
		return nil, fmt.Errorf("watcher not available")
	}

	rows, err := rw.db.Query(`
		SELECT s.npc_name, s.file_path, s.state_json, s.created_at
		FROM roleplay_snapshots s
		INNER JOIN (
			SELECT watch_id, MAX(created_at) AS max_created
			FROM roleplay_snapshots GROUP BY watch_id
		) latest ON s.watch_id = latest.watch_id AND s.created_at = latest.max_created
		ORDER BY s.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		var npcName, filePath, stateJSON, createdAt string
		if rows.Scan(&npcName, &filePath, &stateJSON, &createdAt) == nil {
			item := map[string]interface{}{
				"npc_name": npcName, "file_path": filePath, "created_at": createdAt,
			}
			var state roleplayState
			if json.Unmarshal([]byte(stateJSON), &state) == nil {
				item["state"] = state
			}
			result = append(result, item)
		}
	}
	return result, nil
}

// extractRoleplayState parses an AIInfluence NPC JSON file into a compact
// state snapshot. Tolerant: missing fields are left empty.
func extractRoleplayState(path string) (*roleplayState, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	raw = stripBOM(raw)

	var npc struct {
		Name               string   `json:"Name"`
		StringID           string   `json:"StringId"`
		CurrentTask        string   `json:"CurrentTask"`
		LocationType       string   `json:"LocationType"`
		LastDynamicResponse string  `json:"LastDynamicResponse"`
		IsSick             bool     `json:"IsSick"`
		ConversationHistory []string `json:"ConversationHistory"`
		EmotionalState     struct {
			Mood   string `json:"Mood"`
			Reason string `json:"Reason"`
		} `json:"EmotionalState"`
		PlayerRelation struct {
			Value       int    `json:"Value"`
			Description string `json:"Description"`
		} `json:"PlayerRelation"`
		RomancePartners map[string]interface{} `json:"RomancePartners"`
		DynamicEvents   []string               `json:"DynamicEvents"`
	}
	if err := json.Unmarshal(raw, &npc); err != nil {
		return nil, fmt.Errorf("invalid NPC JSON: %w", err)
	}
	if npc.Name == "" && npc.StringID == "" {
		return nil, fmt.Errorf("not an NPC file (no Name/StringId)")
	}

	state := &roleplayState{
		NPCName:       npc.Name,
		StringID:      npc.StringID,
		Mood:          npc.EmotionalState.Mood,
		MoodReason:    npc.EmotionalState.Reason,
		RelationValue: npc.PlayerRelation.Value,
		RelationDesc:  npc.PlayerRelation.Description,
		CurrentTask:   npc.CurrentTask,
		LocationType:  npc.LocationType,
		LastResponse:  truncate(npc.LastDynamicResponse, 600),
		IsSick:        npc.IsSick,
		DynamicEvents: npc.DynamicEvents,
		ExtractedAt:   time.Now().Format(time.RFC3339),
	}
	// Last 5 conversation entries (most recent last in the array? The mod
	// appends — take the tail)
	if len(npc.ConversationHistory) > 0 {
		start := len(npc.ConversationHistory) - 5
		if start < 0 {
			start = 0
		}
		for _, entry := range npc.ConversationHistory[start:] {
			state.RecentHistory = append(state.RecentHistory, truncate(entry, 400))
		}
	}
	for partner := range npc.RomancePartners {
		state.RomancePartners = append(state.RomancePartners, partner)
	}

	if fi, err := os.Stat(path); err == nil {
		state.FileModifiedTime = fi.ModTime().Format(time.RFC3339)
	}
	return state, nil
}

// formatRoleplayContext builds the compact roleplay section for the system
// prompt. Returns empty string when no watches/snapshots exist.
func (s *AICompanionService) formatRoleplayContext() string {
	rows, err := s.db.Query(`
		SELECT s.npc_name, s.state_json, s.created_at
		FROM roleplay_snapshots s
		INNER JOIN (
			SELECT watch_id, MAX(created_at) AS max_created
			FROM roleplay_snapshots GROUP BY watch_id
		) latest ON s.watch_id = latest.watch_id AND s.created_at = latest.max_created
		ORDER BY s.created_at DESC LIMIT 5`)
	if err != nil {
		return ""
	}
	defer rows.Close()

	var sb strings.Builder
	count := 0
	for rows.Next() {
		var npcName, stateJSON, createdAt string
		if rows.Scan(&npcName, &stateJSON, &createdAt) != nil {
			continue
		}
		var st roleplayState
		if json.Unmarshal([]byte(stateJSON), &st) != nil {
			continue
		}
		if count == 0 {
			sb.WriteString("\n### Roleplay Context — Bannerlord AIInfluence campaign\n")
			sb.WriteString("Brix plays Emil (19yo Battanian lord). NPC files on disk are watched — the states below are the CURRENT live state of each NPC, updated when the game saves.\n")
		}
		count++
		fmt.Fprintf(&sb, "\n**%s** (%s):", st.NPCName, st.StringID)
		if st.Mood != "" {
			fmt.Fprintf(&sb, " mood=%s (%s)", st.Mood, st.MoodReason)
		}
		if st.RelationDesc != "" {
			fmt.Fprintf(&sb, " player_relation=%s (%d)", st.RelationDesc, st.RelationValue)
		}
		if st.CurrentTask != "" {
			fmt.Fprintf(&sb, " task=%s", truncate(st.CurrentTask, 120))
		}
		if st.LastResponse != "" {
			fmt.Fprintf(&sb, "\n  Latest NPC response: \"%s\"", truncate(st.LastResponse, 400))
		}
		if len(st.RecentHistory) > 1 {
			fmt.Fprintf(&sb, "\n  Recent dialogue (%d entries):", len(st.RecentHistory))
			for i, h := range st.RecentHistory {
				fmt.Fprintf(&sb, "\n    [%d] %s", i+1, truncate(h, 200))
			}
		}
		sb.WriteString("\n")
	}
	if count == 0 {
		return ""
	}
	return sb.String()
}
