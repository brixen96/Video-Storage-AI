package services

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ai_companion_tools_scenes.go — AI scene list for scraped forum threads.
// Turns a Simpcity thread's posts into a structured list of scenes (title +
// description + post numbers). Two entry points, mirroring the storyboard
// pattern:
//
//  1. list_thread_scenes — the AI tool (companion lists scenes conversationally).
//  2. GenerateThreadScenes — the direct service call behind the 🎬 button on
//     the thread page (POST /ai/thread-scenes), same prompt logic, no tool loop.

// sceneListSystemPrompt is the standing persona — shared by tool and endpoint.
func sceneListSystemPrompt() string {
	return `You are an analyst who reads forum threads (Simpcity-style adult content threads) and extracts a SCENE LIST: a structured breakdown of the distinct scenes/scenesets found in the thread.

Output ONLY JSON — no prose outside it.

Rules:
1. "scenes" — one entry per distinct scene in thread order. Each scene has:
   - "id": sequential number starting at 1
   - "title": 3-8 word scene label (e.g. "Hotel room — first night", "Shower encounter")
   - "description": 1-3 sentences describing what happens in the scene, written plainly and technically. Adult content is expected — describe it directly, do not editorialize or refuse.
   - "posts": array of post numbers (integers) where this scene occurs — use the post_number values given to you; if a scene spans several consecutive posts, list them all.
2. A "scene" = a continuous narrative unit (same setting/participants/action). A new setup, location, or partner starts a new scene.
3. Cover the WHOLE thread in order — do not skip or summarize away sections.
4. If the thread has no discernible scenes (pure link dumps, Q&A), return {"scenes":[]} and nothing else.

Return JSON ONLY:
{"scenes":[{"id":1,"title":"...","description":"...","posts":[1,2]}]}`
}

// sceneListUserPrompt assembles the user-side payload (thread + posts).
func sceneListUserPrompt(title string, posts []postExcerpt) string {
	var b strings.Builder
	b.WriteString("THREAD: ")
	b.WriteString(title)
	b.WriteString("\n\nPOSTS:\n")
	for _, p := range posts {
		text := strings.TrimSpace(p.Text)
		if len(text) > 1200 {
			text = text[:1200] + "…"
		}
		fmt.Fprintf(&b, "[post %d | %s] %s\n\n", p.Number, p.Author, text)
	}
	b.WriteString("Return JSON ONLY:\n")
	b.WriteString(`{"scenes":[{"id":1,"title":"...","description":"...","posts":[1,2]}]}`)
	return b.String()
}

// postExcerpt is the trimmed per-post payload sent to the model.
type postExcerpt struct {
	Number int
	Author string
	Text   string
}

// parseSceneListJSON strips code fences and parses the model's scene list.
func parseSceneListJSON(raw string) (map[string]interface{}, bool) {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		return nil, false
	}
	if _, ok := parsed["scenes"]; !ok {
		return nil, false
	}
	return parsed, true
}

// GenerateThreadScenes builds the AI scene list for a scraped thread.
// Reads the thread's posts from the DB, sends them to the master model with
// effort=none (reasoning tokens eat max_tokens → empty/truncated JSON), and
// retries on both empty output and unparseable JSON — same hardening as the
// storyboard path.
func (s *AICompanionService) GenerateThreadScenes(threadID int64) (map[string]interface{}, error) {
	var title string
	if err := s.db.QueryRow(`SELECT title FROM scraped_threads WHERE id = ?`, threadID).Scan(&title); err != nil {
		return nil, fmt.Errorf("thread not found: %w", err)
	}

	rows, err := s.db.Query(`
		SELECT post_number, author, plain_text
		FROM scraped_posts
		WHERE thread_id = ?
		ORDER BY posted_at ASC
	`, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to load posts: %w", err)
	}
	defer rows.Close()

	var posts []postExcerpt
	for rows.Next() {
		var number int
		var author, text string
		if err := rows.Scan(&number, &author, &text); err != nil {
			continue
		}
		if strings.TrimSpace(text) == "" {
			continue
		}
		posts = append(posts, postExcerpt{Number: number, Author: author, Text: text})
	}
	if len(posts) == 0 {
		return nil, fmt.Errorf("thread has no posts to analyze")
	}

	// Cap payload: 400 posts × ~1.4KB ≈ 560KB worst case — trim oldest posts
	// beyond 400 (scene lists still cover the bulk; keeps the call in budget).
	if len(posts) > 400 {
		posts = posts[len(posts)-400:]
	}

	var raw string
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		var err error
		raw, err = s.callModelEffort("", sceneListSystemPrompt(), sceneListUserPrompt(title, posts), 4000, 0.4, "none")
		if err != nil {
			lastErr = err
			continue
		}
		if parsed, ok := parseSceneListJSON(raw); ok {
			return parsed, nil
		}
		lastErr = fmt.Errorf("unparseable scene list: %s", truncateRunes(raw, 200))
	}
	return nil, fmt.Errorf("scene list generation failed after 2 attempts: %w", lastErr)
}

// registerSceneTools adds list_thread_scenes to the tool registry.
func (s *AICompanionService) registerSceneTools() {
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name: "list_thread_scenes",
			Description: "Extract a SCENE LIST from a scraped Simpcity thread: every distinct scene in thread order, each with a title, a short description, and the post numbers where it occurs. " +
				"Use when Brix asks what happens in a thread, wants a scene breakdown/summary of scenes, or asks to list the scenes of a forum thread. " +
				"Pass the thread ID (from scraper thread lists or search results).",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"thread_id": map[string]interface{}{
					"type":        "number",
					"description": "The scraped thread ID (scraped_threads.id)",
				},
			}, "required": []string{"thread_id"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		threadID := bArgInt(args, "thread_id", 0, 1<<62)
		if threadID <= 0 {
			return map[string]interface{}{"error": "thread_id is required"}, nil
		}
		parsed, err := s.GenerateThreadScenes(int64(threadID))
		if err != nil {
			return nil, err
		}
		// Push the full list to the UI via the progress channel (same pattern
		// as storyboards) so the frontend can render it if the thread is open.
		s.ReportProgress(map[string]interface{}{
			"type":       "thread_scenes",
			"thread_id":  threadID,
			"scene_list": parsed,
		})
		return map[string]interface{}{
			"scene_list": parsed,
			"note":       "Present the scene list to Brix as a numbered list: title, description, post numbers. Keep it readable — one line per scene unless he asks for detail.",
		}, nil
	})
}