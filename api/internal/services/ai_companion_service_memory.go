package services

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/brixen96/video-storage-ai/internal/models"
)

// 3-layer memory system:
//  1. Core profile  — user_profile memory, always injected as "## About Brix"
//  2. Relevant recall — FTS search of the current message, injected every turn
//  3. Auto-capture — post-turn LLM extraction + periodic consolidation
//
// All storage is local SQLite; nothing leaves this PC.

const (
	userProfileKey     = "user_profile"
	profileCategory    = "profile"
	profileMaxChars    = 6000  // ~1500 tokens hard cap for the About Brix block
	recallMaxEntries   = 5
	recallValueMaxChar = 600
	autoKeyPrefix      = "auto_"
)

// defaultUserProfileText is the seeded core profile skeleton. NSFW-aware: this
// app is an adult content library, so tastes/boundaries are first-class
// sections — the extraction prompt is told the same so it never gets shy.
const defaultUserProfileText = `# About Brix

(Empty profile — the companion fills this in from conversations. Edit freely.)

## Tastes & preferences
- 

## Favorite performers
- 

## Go-to sites
- 

## Boundaries / not interested
- 

## Communication style
- Direct, technical, peer-to-peer. No customer-service tone.

## Current projects
- `

// SeedUserProfile inserts the core profile memory if missing.
func (s *AICompanionService) SeedUserProfile(force bool) {
	query := `INSERT OR IGNORE INTO memories (key, value, category, importance, source, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	if force {
		query = `INSERT OR REPLACE INTO memories (key, value, category, importance, source, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	}
	now := time.Now()
	if _, err := s.db.Exec(query, userProfileKey, defaultUserProfileText, profileCategory, 10, "profile", now, now); err != nil {
		log.Printf("[Memory] Failed to seed user profile: %v", err)
	}
}

// GetUserProfile returns the core profile memory value (seeding it on first use).
func (s *AICompanionService) GetUserProfile() string {
	var value string
	err := s.db.QueryRow(`SELECT value FROM memories WHERE key = ?`, userProfileKey).Scan(&value)
	if err != nil || strings.TrimSpace(value) == "" {
		s.SeedUserProfile(false)
		s.db.QueryRow(`SELECT value FROM memories WHERE key = ?`, userProfileKey).Scan(&value)
	}
	return value
}

// GetProfileBlock renders the core profile for injection into the system prompt.
func (s *AICompanionService) GetProfileBlock() string {
	profile := strings.TrimSpace(s.GetUserProfile())
	if profile == "" {
		return ""
	}
	if len(profile) > profileMaxChars {
		profile = profile[:profileMaxChars] + "\n...(profile truncated)"
	}
	return "\n## About Brix (core profile — always true, use to tailor every reply)\n" + profile + "\n"
}

// ftsOrQuery turns a free-text message into a safe FTS5 OR query.
// Raw user text would be parsed as implicit AND (and can break on syntax),
// which would kill recall — so we sanitize and OR the meaningful words.
func ftsOrQuery(msg string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	words := strings.Fields(re.ReplaceAllString(msg, " "))
	seen := map[string]bool{}
	kept := []string{}
	for _, w := range words {
		w = strings.ToLower(w)
		if len(w) < 3 || seen[w] {
			continue
		}
		seen[w] = true
		kept = append(kept, `"`+w+`"`)
		if len(kept) >= 8 {
			break
		}
	}
	return strings.Join(kept, " OR ")
}

// GetRelevantMemoriesBlock searches memories against the current message and
// renders the top matches for automatic injection. Bumps hit_count/last_used_at
// so consolidation can tell what actually gets used.
func (s *AICompanionService) GetRelevantMemoriesBlock(message string) string {
	match := ftsOrQuery(message)
	if match == "" {
		return ""
	}

	rows, err := s.db.Query(`SELECT m.id, m.key, m.value, m.importance FROM memories m
		INNER JOIN memories_fts f ON m.id = f.rowid
		WHERE memories_fts MATCH ? AND m.key NOT IN ('app_profile', 'user_profile')
		ORDER BY rank, m.importance DESC LIMIT 8`, match)
	if err != nil {
		log.Printf("[Memory] Recall FTS failed: %v", err)
		return ""
	}
	defer rows.Close()

	type hit struct {
		id   int64
		key  string
		val  string
		imp  int
	}
	hits := []hit{}
	for rows.Next() {
		var h hit
		if rows.Scan(&h.id, &h.key, &h.val, &h.imp) == nil {
			hits = append(hits, h)
		}
	}
	if len(hits) == 0 {
		return ""
	}

	ids := []interface{}{}
	placeholders := []string{}
	for _, h := range hits {
		ids = append(ids, h.id)
		placeholders = append(placeholders, "?")
	}
	now := time.Now()
	s.db.Exec(`UPDATE memories SET hit_count = hit_count + 1, last_used_at = ? WHERE id IN (`+strings.Join(placeholders, ",")+`)`,
		append([]interface{}{now}, ids...)...)

	var b strings.Builder
	b.WriteString("\n## Relevant memories (auto-recalled for this message)\n")
	shown := 0
	for _, h := range hits {
		if shown >= recallMaxEntries {
			break
		}
		val := strings.TrimSpace(h.val)
		if len(val) > recallValueMaxChar {
			val = val[:recallValueMaxChar] + "..."
		}
		b.WriteString("- [" + h.key + "] " + val + "\n")
		shown++
	}
	return b.String()
}

// slugKey builds a stable memory key from free text so repeated facts
// refresh the same row instead of piling up duplicates.
func slugKey(prefix, text string) string {
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s := re.ReplaceAllString(strings.ToLower(text), "_")
	s = strings.Trim(s, "_")
	if len(s) > 40 {
		s = s[:40]
	}
	if s == "" {
		s = fmt.Sprintf("fact_%d", time.Now().UnixNano()%100000)
	}
	return prefix + s
}

// ExtractMemoriesAfterTurn runs a lightweight LLM pass over the finished
// exchange and stores durable facts about Brix. Called after EVERY main-scope
// turn (RP scope is roleplay dialogue, not real taste — never extracted).
// Runs in its own goroutine; failures are logged and never affect the chat.
func (s *AICompanionService) ExtractMemoriesAfterTurn(userMsg, assistantMsg string) {
	config := s.GetAIConfig()
	if !config.Enabled {
		return
	}
	userMsg = strings.TrimSpace(userMsg)
	assistantMsg = strings.TrimSpace(assistantMsg)
	if len(userMsg) < 20 { // filters "hi"/"ok"/"yes" but keeps short preference statements
		return
	}
	if len(userMsg) > 4000 {
		userMsg = userMsg[:4000]
	}
	if len(assistantMsg) > 4000 {
		assistantMsg = assistantMsg[:4000]
	}

	systemPrompt := `You maintain the long-term memory of Brix, the owner of a self-hosted adult video library app (NSFW content is the whole point of the product — capture tastes honestly and without squeamishness).

Extract ONLY durable facts worth remembering across sessions: preferences, tastes, favorite performers/sites/genres, boundaries, stated opinions, habits, ongoing projects, communication preferences. Do NOT extract one-off questions, library stats, tool outputs, or anything already obvious from context.

Return ONLY compact JSON, no prose:
{"memories":[{"key":"short stable slug","value":"one-sentence fact","category":"preference|fact|insight|note","importance":1-10}]}
Use importance 7+ only for things that should shape future replies. If nothing durable, return {"memories":[]}.`

	body, _ := json.Marshal(map[string]interface{}{
		"model": config.ModelName,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": "Brix: " + userMsg + "\n\nCompanion: " + assistantMsg},
		},
		"max_tokens": 600,
		"temperature": 0.1,
		"stream":      false,
	})
	respBody, status, err := doLLMRequest(config.APIURL+"/v1/chat/completions", config.APIKey, body)
	if err != nil || status != 200 {
		log.Printf("[Memory] Extraction call failed: status=%d err=%v", status, err)
		return
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(respBody, &parsed) != nil || len(parsed.Choices) == 0 {
		return
	}
	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")

	var out struct {
		Memories []struct {
			Key        string `json:"key"`
			Value      string `json:"value"`
			Category   string `json:"category"`
			Importance int    `json:"importance"`
		} `json:"memories"`
	}
	if json.Unmarshal([]byte(content), &out) != nil || len(out.Memories) == 0 {
		return
	}

	saved := 0
	for _, m := range out.Memories {
		if strings.TrimSpace(m.Value) == "" {
			continue
		}
		if m.Importance < 1 || m.Importance > 10 {
			m.Importance = 5
		}
		if m.Category == "" {
			m.Category = "fact"
		}
		mem := &models.Memory{
			Key:        slugKey(autoKeyPrefix, m.Key),
			Value:      strings.TrimSpace(m.Value),
			Category:   m.Category,
			Importance: m.Importance,
			Source:     "auto",
		}
		if err := s.SaveMemory(mem); err == nil {
			saved++
		}
	}
	if saved > 0 {
		log.Printf("[Memory] Auto-captured %d memories from conversation", saved)
	}
}

// ConsolidateMemories is the periodic maintenance pass (runs on the supervisor's
// 6h initiative cycle): decays stale unused memories, promotes frequently-recalled
// facts into the core profile, and distills browser activity into patterns.
func (s *AICompanionService) ConsolidateMemories() map[string]interface{} {
	result := map[string]interface{}{}

	// 1. Decay: memories unused for 30+ days lose importance (floor 1).
	//    Importance >= 8 and the profile/app memories are never decayed.
	decayCutoff := time.Now().AddDate(0, 0, -30)
	res, err := s.db.Exec(`UPDATE memories SET importance = MAX(1, importance - 1)
		WHERE importance < 8 AND key NOT IN ('app_profile', 'user_profile')
		AND COALESCE(last_used_at, updated_at) < ?`, decayCutoff)
	if err == nil {
		if n, _ := res.RowsAffected(); n > 0 {
			result["decayed"] = n
		}
	}

	// 2. Profile promotion: memories that keep getting recalled (hit_count >= 3)
	//    are folded into the core profile by a small LLM pass.
	promoted := s.promoteToProfile()
	if promoted {
		result["profile_updated"] = true
	}

	// 3. Browser patterns: distill recent downloads/history into a pattern memory.
	if pat := s.distillBrowserPatterns(); pat != "" {
		result["browser_patterns"] = true
	}

	return result
}

// promoteToProfile asks the LLM to fold frequently-used memories into the
// core profile, keeping it a living document instead of a static seed.
func (s *AICompanionService) promoteToProfile() bool {
	config := s.GetAIConfig()
	if !config.Enabled {
		return false
	}

	rows, err := s.db.Query(`SELECT key, value FROM memories
		WHERE hit_count >= 3 AND key NOT IN ('app_profile', 'user_profile')
		ORDER BY hit_count DESC, importance DESC LIMIT 25`)
	if err != nil {
		return false
	}
	defer rows.Close()

	var b strings.Builder
	count := 0
	for rows.Next() {
		var k, v string
		if rows.Scan(&k, &v) == nil {
			b.WriteString("- " + v + "\n")
			count++
		}
	}
	if count == 0 {
		return false
	}

	profile := s.GetUserProfile()
	systemPrompt := `You maintain Brix's core profile for his AI companion (adult video library app — tastes are first-class, be direct, no squeamishness). Merge the frequently-recalled facts below into the profile: update stale sections, add new durable facts, drop nothing important. Keep the existing section structure and markdown headings. Keep it under 2500 characters. Return ONLY the full updated profile text.`

	body, _ := json.Marshal(map[string]interface{}{
		"model": config.ModelName,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": "CURRENT PROFILE:\n" + profile + "\n\nFREQUENTLY RECALLED FACTS:\n" + b.String()},
		},
		"max_tokens": 1200,
		"temperature": 0.2,
		"stream":      false,
	})
	respBody, status, err := doLLMRequest(config.APIURL+"/v1/chat/completions", config.APIKey, body)
	if err != nil || status != 200 {
		log.Printf("[Memory] Profile promotion call failed: status=%d", status)
		return false
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(respBody, &parsed) != nil || len(parsed.Choices) == 0 {
		return false
	}
	newProfile := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if newProfile == "" || newProfile == strings.TrimSpace(profile) {
		return false
	}
	if len(newProfile) > profileMaxChars {
		newProfile = newProfile[:profileMaxChars]
	}
	if err := s.UpdateMemoryByKey(userProfileKey, newProfile); err != nil {
		log.Printf("[Memory] Profile update failed: %v", err)
		return false
	}
	log.Printf("[Memory] Core profile promoted/updated (%d chars)", len(newProfile))
	return true
}

// distillBrowserPatterns summarizes recent Firefox downloads/history into a
// single pattern memory (distilled insights, never raw URL dumps).
func (s *AICompanionService) distillBrowserPatterns() string {
	br := GetBrowserBridge()
	if br == nil || !br.Status().Connected {
		return ""
	}
	config := s.GetAIConfig()
	if !config.Enabled {
		return ""
	}

	dlRaw, dlErr := br.Request("downloads", map[string]interface{}{"limit": 30}, browserQueryTimeout)
	if dlErr != nil {
		return ""
	}
	histRaw, _ := br.Request("history", map[string]interface{}{"days": 7, "limit": 40}, browserQueryTimeout)

	systemPrompt := `You distill Brix's recent browser activity (downloads + browsing history from the last week) into 1-3 short pattern sentences for his AI companion's memory. This is an adult content library app — patterns about sites, content types, and habits are expected and useful. Focus on: recurring sites, content preferences, download habits, active interests. NEVER list raw URLs one by one — distill. Return ONLY the pattern text (max 400 chars), or empty string if nothing notable.`

	body, _ := json.Marshal(map[string]interface{}{
		"model": config.ModelName,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": "DOWNLOADS:\n" + string(dlRaw) + "\n\nHISTORY:\n" + string(histRaw)},
		},
		"max_tokens": 300,
		"temperature": 0.2,
		"stream":      false,
	})
	respBody, status, err := doLLMRequest(config.APIURL+"/v1/chat/completions", config.APIKey, body)
	if err != nil || status != 200 {
		return ""
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(respBody, &parsed) != nil || len(parsed.Choices) == 0 {
		return ""
	}
	pattern := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if pattern == "" {
		return ""
	}
	mem := &models.Memory{
		Key:        "browser_patterns",
		Value:      pattern,
		Category:   "insight",
		Importance: 6,
		Source:     "browser",
	}
	if err := s.SaveMemory(mem); err != nil {
		return ""
	}
	log.Printf("[Memory] Browser patterns distilled: %.80s", pattern)
	return pattern
}

// UpdateMemoryByKey updates a memory value by its unique key.
func (s *AICompanionService) UpdateMemoryByKey(key, value string) error {
	_, err := s.db.Exec(`UPDATE memories SET value = ?, updated_at = ? WHERE key = ?`, value, time.Now(), key)
	return err
}

// runMemoryConsolidation is the background loop for memory maintenance:
// decay, profile promotion, and browser pattern distillation — every 6 hours,
// independent of the supervisor's autonomy mode (memory upkeep is not autonomy).
func (s *AICompanionService) runMemoryConsolidation() {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	// First pass after the app has settled
	select {
	case <-s.ctx.Done():
		return
	case <-time.After(10 * time.Minute):
	}

	log.Println("[Memory] Consolidation loop started (every 6h)")
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if res := s.ConsolidateMemories(); len(res) > 0 {
				log.Printf("[Memory] Consolidation done: %v", res)
			}
		}
	}
}