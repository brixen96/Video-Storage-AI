package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
)

// ── Bannerlord Campaign Service ──
//
// Scans the AIInfluence mod's save_data/ folder for campaign directories and
// builds per-campaign overviews: NPC roster (live states), diplomacy, wars.
// Each campaign folder is an isolated instance — nothing is mixed between
// campaigns. Read-only: never writes to mod files.

const defaultBannerlordModPath = `C:\SteamLibrary\steamapps\common\Mount & Blade II Bannerlord\Modules\AIInfluence`

// bannerlordModPath returns the mod folder path (env override for portability).
func bannerlordModPath() string {
	if p := os.Getenv("BANNERLORD_MOD_PATH"); p != "" {
		return p
	}
	return defaultBannerlordModPath
}

// campaignIDPattern guards against path traversal — IDs are plain folder names.
var campaignIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// skippedSaveDataDirs are non-campaign folders inside save_data/.
var skippedSaveDataDirs = map[string]bool{
	"default":       true,
	"storytools":    true,
	"_portrait_tmp": true,
}

// BannerlordCampaignInfo describes one campaign folder.
type BannerlordCampaignInfo struct {
	ID           string `json:"id"`
	Path         string `json:"path"`
	NPCCount     int    `json:"npc_count"`
	LastModified string `json:"last_modified"`
}

// ListBannerlordCampaigns scans save_data/ for campaign folders (directories
// containing NPC JSON files named "Name (string_id).json").
func ListBannerlordCampaigns() ([]BannerlordCampaignInfo, error) {
	base := filepath.Join(bannerlordModPath(), "save_data")
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil, fmt.Errorf("cannot read save_data (is Bannerlord installed at the expected path?): %w", err)
	}

	var campaigns []BannerlordCampaignInfo
	for _, e := range entries {
		if !e.IsDir() || skippedSaveDataDirs[e.Name()] {
			continue
		}
		dir := filepath.Join(base, e.Name())
		count, lastMod := scanCampaignNPCs(dir)
		if count == 0 {
			continue // no NPC files → not a real campaign
		}
		campaigns = append(campaigns, BannerlordCampaignInfo{
			ID:           e.Name(),
			Path:         dir,
			NPCCount:     count,
			LastModified: lastMod,
		})
	}
	// Most recently active first
	for i := 0; i < len(campaigns); i++ {
		for j := i + 1; j < len(campaigns); j++ {
			if campaigns[j].LastModified > campaigns[i].LastModified {
				campaigns[i], campaigns[j] = campaigns[j], campaigns[i]
			}
		}
	}
	return campaigns, nil
}

// scanCampaignNPCs counts NPC files ("Name (string_id).json") and returns the
// most recent modification time as an RFC3339 string.
func scanCampaignNPCs(dir string) (int, string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, ""
	}
	count := 0
	var latest time.Time
	for _, e := range entries {
		if e.IsDir() || !isNPCFilename(e.Name()) {
			continue
		}
		count++
		if info, err := e.Info(); err == nil && info.ModTime().After(latest) {
			latest = info.ModTime()
		}
	}
	lastMod := ""
	if !latest.IsZero() {
		lastMod = latest.Format(time.RFC3339)
	}
	return count, lastMod
}

// isNPCFilename matches the mod's NPC file pattern: "Name (string_id).json".
func isNPCFilename(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".json") && strings.Contains(name, " (") && strings.HasSuffix(name, ").json")
}

// BannerlordNPCRosterEntry is one NPC in a campaign overview.
type BannerlordNPCRosterEntry struct {
	FileName       string `json:"file_name"`
	FilePath       string `json:"file_path"`
	Name           string `json:"name"`
	StringID       string `json:"string_id"`
	Mood           string `json:"mood,omitempty"`
	RelationValue  int    `json:"relation_value"`
	RelationDesc   string `json:"relation_desc"`
	CurrentTask    string `json:"current_task,omitempty"`
	LastResponse   string `json:"last_response,omitempty"`
	ParseError     string `json:"parse_error,omitempty"`
	Watched        bool   `json:"watched"`
	WatchID        int64  `json:"watch_id,omitempty"`
	LastSnapshotAt string `json:"last_snapshot_at,omitempty"`
}

// GetBannerlordCampaignOverview builds the full overview for one campaign:
// NPC roster (with watch status), diplomacy events/statements, kingdom wars.
func GetBannerlordCampaignOverview(campaignID string) (map[string]interface{}, error) {
	if !campaignIDPattern.MatchString(campaignID) {
		return nil, fmt.Errorf("invalid campaign id")
	}
	dir := filepath.Join(bannerlordModPath(), "save_data", campaignID)
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return nil, fmt.Errorf("campaign not found: %s", campaignID)
	}

	// Watch lookup by file path (watches are absolute paths → per-campaign isolation is automatic)
	watchByPath := map[string]struct {
		ID       int64
		LastSnap string
	}{}
	if db := database.GetDB(); db != nil {
		rows, err := db.Query(`SELECT w.id, w.file_path,
			COALESCE((SELECT MAX(created_at) FROM roleplay_snapshots WHERE watch_id = w.id), '')
			FROM roleplay_watches w`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var id int64
				var path, lastSnap string
				if rows.Scan(&id, &path, &lastSnap) == nil {
					watchByPath[path] = struct {
						ID       int64
						LastSnap string
					}{id, lastSnap}
				}
			}
		}
	}

	// ── NPC roster ──
	entries, _ := os.ReadDir(dir)
	roster := []BannerlordNPCRosterEntry{}
	for _, e := range entries {
		if e.IsDir() || !isNPCFilename(e.Name()) {
			continue
		}
		path := filepath.Join(dir, e.Name())
		item := BannerlordNPCRosterEntry{FileName: e.Name(), FilePath: path}
		if w, ok := watchByPath[path]; ok {
			item.Watched = true
			item.WatchID = w.ID
			item.LastSnapshotAt = w.LastSnap
		}
		if err := parseNPCRosterEntry(path, &item); err != nil {
			item.ParseError = err.Error()
			// Still derive name from filename: "Empress_Rhagaea (lord_1_14).json"
			item.Name = strings.TrimSuffix(strings.Split(e.Name(), " (")[0], ".json")
		}
		roster = append(roster, item)
	}

	// ── Diplomacy bundle ──
	dynEvents := []map[string]interface{}{}
	kingdomStatements := []map[string]interface{}{}
	var campaignDays float64
	dipRaw, err := os.ReadFile(filepath.Join(dir, "aiinfluence_campaign_diplomacy.json"))
	if err == nil {
		var bundle struct {
			DynamicEvents []struct {
				ID               string   `json:"id"`
				Type             string   `json:"type"`
				Title            string   `json:"title"`
				Description      string   `json:"description"`
				Importance       int      `json:"importance"`
				KingdomsInvolved []string `json:"kingdoms_involved"`
				PlayerInvolved   bool     `json:"player_involved"`
			} `json:"dynamic_events"`
			KingdomStatements []struct {
				KingdomID     string   `json:"kingdom_id"`
				StatementText string   `json:"statement_text"`
				Action        string   `json:"action"`
				Actions       []string `json:"actions"`
				TargetKingdom string   `json:"target_kingdom_id"`
				CampaignDays  float64  `json:"campaign_days"`
			} `json:"kingdom_statements"`
			SavedCampaignDays float64 `json:"saved_campaign_days"`
		}
		if json.Unmarshal(dipRaw, &bundle) == nil {
			campaignDays = bundle.SavedCampaignDays
			for _, ev := range bundle.DynamicEvents {
				dynEvents = append(dynEvents, map[string]interface{}{
					"id": ev.ID, "type": ev.Type, "title": ev.Title,
					"description":      truncate(ev.Description, 300),
					"importance":       ev.Importance,
					"kingdoms_involved": ev.KingdomsInvolved,
					"player_involved":  ev.PlayerInvolved,
				})
			}
			for _, st := range bundle.KingdomStatements {
				kingdomStatements = append(kingdomStatements, map[string]interface{}{
					"kingdom_id":      st.KingdomID,
					"statement_text":  truncate(st.StatementText, 300),
					"action":          st.Action,
					"actions":         st.Actions,
					"target_kingdom":  st.TargetKingdom,
				})
			}
		}
	}

	// ── War statistics ──
	kingdomWars := []map[string]interface{}{}
	warRaw, err := os.ReadFile(filepath.Join(dir, "war_statistics.json"))
	if err == nil {
		var war struct {
			KingdomStats map[string]struct {
				CurrentTroops      int     `json:"current_troops"`
				TotalCasualties    int     `json:"total_casualties"`
				DaysAtWar          int     `json:"days_at_war"`
				CurrentSettlements int     `json:"current_settlements"`
				WarFatigue         float64 `json:"war_fatigue"`
				TotalLordsCaptured int     `json:"total_lords_captured"`
				TotalLordsKilled   int     `json:"total_lords_killed"`
			} `json:"kingdom_stats"`
		}
		if json.Unmarshal(warRaw, &war) == nil {
			for id, ks := range war.KingdomStats {
				kingdomWars = append(kingdomWars, map[string]interface{}{
					"kingdom":           id,
					"current_troops":    ks.CurrentTroops,
					"total_casualties":  ks.TotalCasualties,
					"days_at_war":       ks.DaysAtWar,
					"settlements":       ks.CurrentSettlements,
					"war_fatigue":       ks.WarFatigue,
					"lords_captured":    ks.TotalLordsCaptured,
					"lords_killed":      ks.TotalLordsKilled,
				})
			}
		}
	}

	// ── Alliances & tributes (small files) ──
	allianceCount := 0
	if raw, err := os.ReadFile(filepath.Join(dir, "alliances.json")); err == nil {
		var a struct {
			Alliances map[string]interface{} `json:"alliances"`
		}
		if json.Unmarshal(raw, &a) == nil {
			allianceCount = len(a.Alliances)
		}
	}
	tributeCount := 0
	if raw, err := os.ReadFile(filepath.Join(dir, "tributes.json")); err == nil {
		var t struct {
			Tributes map[string]interface{} `json:"tributes"`
		}
		if json.Unmarshal(raw, &t) == nil {
			tributeCount = len(t.Tributes)
		}
	}

	watchedCount := 0
	for _, r := range roster {
		if r.Watched {
			watchedCount++
		}
	}

	log.Printf("[Bannerlord] Overview built for %s: %d NPCs (%d watched), %d events, %d statements",
		campaignID, len(roster), watchedCount, len(dynEvents), len(kingdomStatements))

	return map[string]interface{}{
		"campaign_id":       campaignID,
		"campaign_path":     dir,
		"campaign_days":     campaignDays,
		"npcs":              roster,
		"npc_count":         len(roster),
		"watched_count":     watchedCount,
		"dynamic_events":     dynEvents,
		"kingdom_statements": kingdomStatements,
		"kingdom_wars":       kingdomWars,
		"alliance_count":     allianceCount,
		"tribute_count":      tributeCount,
		"generated_at":       time.Now().Format(time.RFC3339),
	}, nil
}

// GetBannerlordNPCDialogue returns the full conversation history and identity
// for one NPC file. Read-only. Path must match the NPC file pattern.
func GetBannerlordNPCDialogue(filePath string) (map[string]interface{}, error) {
	if !strings.Contains(filePath, " (") || !strings.HasSuffix(filePath, ").json") {
		return nil, fmt.Errorf("not an NPC file path")
	}
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read NPC file: %w", err)
	}
	var npc struct {
		Name                 string   `json:"Name"`
		StringID             string   `json:"StringId"`
		Gender               string   `json:"Gender"`
		CharacterDescription string   `json:"CharacterDescription"`
		AIGeneratedPersonality string `json:"AIGeneratedPersonality"`
		AIGeneratedBackstory string   `json:"AIGeneratedBackstory"`
		CurrentTask          string   `json:"CurrentTask"`
		LocationType         string   `json:"LocationType"`
		ConversationHistory  []string `json:"ConversationHistory"`
		EmotionalState       struct {
			Mood   string `json:"Mood"`
			Reason string `json:"Reason"`
		} `json:"EmotionalState"`
		PlayerRelation struct {
			Value       int    `json:"Value"`
			Description string `json:"Description"`
		} `json:"PlayerRelation"`
	}
	if err := json.Unmarshal(raw, &npc); err != nil {
		return nil, fmt.Errorf("invalid NPC JSON: %w", err)
	}
	if npc.Name == "" && npc.StringID == "" {
		return nil, fmt.Errorf("not an NPC file (no Name/StringId)")
	}
	return map[string]interface{}{
		"name":                  npc.Name,
		"string_id":             npc.StringID,
		"gender":                npc.Gender,
		"character_description": truncate(npc.CharacterDescription, 500),
		"personality":           truncate(npc.AIGeneratedPersonality, 500),
		"backstory":             truncate(npc.AIGeneratedBackstory, 500),
		"current_task":          npc.CurrentTask,
		"location":              npc.LocationType,
		"mood":                  npc.EmotionalState.Mood,
		"mood_reason":           npc.EmotionalState.Reason,
		"relation_value":        npc.PlayerRelation.Value,
		"relation_desc":         npc.PlayerRelation.Description,
		"conversation_history":  npc.ConversationHistory,
		"entry_count":           len(npc.ConversationHistory),
	}, nil
}

// GetBannerlordRPContext builds the full roleplay writing context for one NPC:
// the campaign's editable prompts (world lore, RP rules, player description)
// plus the NPC's identity and recent dialogue. Read-only.
func GetBannerlordRPContext(npcPath string) (map[string]interface{}, error) {
	if !strings.Contains(npcPath, " (") || !strings.HasSuffix(npcPath, ").json") {
		return nil, fmt.Errorf("not an NPC file path")
	}
	if _, err := os.Stat(npcPath); err != nil {
		return nil, fmt.Errorf("NPC file not found")
	}
	campaignDir := filepath.Dir(npcPath)

	readCampaignFile := func(rel string, max int) string {
		b, err := os.ReadFile(filepath.Join(campaignDir, rel))
		if err != nil {
			return ""
		}
		return truncate(string(b), max)
	}

	playerDesc := readCampaignFile(filepath.Join("prompts", "playerdescription.txt"), 2000)
	worldLore := readCampaignFile(filepath.Join("prompts", "world_data", "world.txt"), 6000)
	rpRules := readCampaignFile(filepath.Join("prompts", "rules", "actionrules.txt"), 8000)

	// Parse NPC identity + recent dialogue
	raw, err := os.ReadFile(npcPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read NPC file: %w", err)
	}
	var npc struct {
		Name                   string   `json:"Name"`
		StringID               string   `json:"StringId"`
		CharacterDescription   string   `json:"CharacterDescription"`
		AIGeneratedPersonality string   `json:"AIGeneratedPersonality"`
		AIGeneratedBackstory   string   `json:"AIGeneratedBackstory"`
		CurrentTask            string   `json:"CurrentTask"`
		ConversationHistory    []string `json:"ConversationHistory"`
		EmotionalState         struct {
			Mood   string `json:"Mood"`
			Reason string `json:"Reason"`
		} `json:"EmotionalState"`
		PlayerRelation struct {
			Value       int    `json:"Value"`
			Description string `json:"Description"`
		} `json:"PlayerRelation"`
	}
	if err := json.Unmarshal(raw, &npc); err != nil {
		return nil, fmt.Errorf("invalid NPC JSON: %w", err)
	}
	if npc.Name == "" && npc.StringID == "" {
		return nil, fmt.Errorf("not an NPC file (no Name/StringId)")
	}

	// Last 12 dialogue entries, each capped at 800 chars (oldest first).
	// The dialogue is the Companion's ONLY knowledge source for RP writing.
	recent := []string{}
	if len(npc.ConversationHistory) > 0 {
		start := len(npc.ConversationHistory) - 12
		if start < 0 {
			start = 0
		}
		for _, e := range npc.ConversationHistory[start:] {
			recent = append(recent, truncate(e, 800))
		}
	}

	return map[string]interface{}{
		"campaign_id":         filepath.Base(campaignDir),
		"player_description":  playerDesc,
		"world_lore":          worldLore,
		"rp_rules":            rpRules,
		"npc": map[string]interface{}{
			"name":                  npc.Name,
			"string_id":             npc.StringID,
			"character_description": truncate(npc.CharacterDescription, 600),
			"personality":           truncate(npc.AIGeneratedPersonality, 600),
			"backstory":             truncate(npc.AIGeneratedBackstory, 600),
			"current_task":          truncate(npc.CurrentTask, 200),
			"mood":                  npc.EmotionalState.Mood,
			"mood_reason":           npc.EmotionalState.Reason,
			"relation_value":        npc.PlayerRelation.Value,
			"relation_desc":         npc.PlayerRelation.Description,
		},
		"recent_dialogue": recent,
	}, nil
}

// parseNPCRosterEntry extracts the compact roster fields from an NPC file.
func parseNPCRosterEntry(path string, item *BannerlordNPCRosterEntry) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var npc struct {
		Name                string   `json:"Name"`
		StringID            string   `json:"StringId"`
		CurrentTask         string   `json:"CurrentTask"`
		LastDynamicResponse string   `json:"LastDynamicResponse"`
		EmotionalState      struct {
			Mood string `json:"Mood"`
		} `json:"EmotionalState"`
		PlayerRelation struct {
			Value       int    `json:"Value"`
			Description string `json:"Description"`
		} `json:"PlayerRelation"`
	}
	if err := json.Unmarshal(raw, &npc); err != nil {
		return fmt.Errorf("invalid NPC JSON")
	}
	item.Name = npc.Name
	item.StringID = npc.StringID
	item.Mood = npc.EmotionalState.Mood
	item.RelationValue = npc.PlayerRelation.Value
	item.RelationDesc = npc.PlayerRelation.Description
	item.CurrentTask = truncate(npc.CurrentTask, 120)
	item.LastResponse = truncate(npc.LastDynamicResponse, 200)
	return nil
}
