package services

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// ai_companion_rp_review.go — the V3.2 draft review layer for the Bannerlord
// mini-chat (scope 'rp'). Emil picked deepseek/deepseek-v3.2-exp as the
// reviewer: the most NSFW-capable model he's seen, and cheap ($0.27/$0.41 per
// 1M — a review pass on a ~2k-char draft costs a fraction of a cent).

// rpReviewModel is the dedicated reviewer model (NOT the chat model).
const rpReviewModel = "deepseek/deepseek-v3.2-exp"

// Length rules — MUST mirror the CompanionMiniChat.vue constants
// (MIN applies to generated drafts; OOC messages keep natural length).
const (
	rpReviewMinLen    = 1600
	rpReviewTargetLen = 1800
	rpReviewMaxLen    = 2048
)

// ReviewRPDraft runs the final review pass on a mini-chat reply and returns
// the improved text. Returns an error on any failure — the caller then keeps
// the original draft (review is a layer, never a point of failure).
//
// history: recent RP thread entries (for scene context — system entries are
// ignored; dialogue only). focus: the mini-chat Focus chips (e.g. Tits, High
// Heels) to weave in where the scene fits.
func (s *AICompanionService) ReviewRPDraft(draft string, history []map[string]string, focus []string) (string, error) {
	config := s.GetAIConfig()
	if !config.Enabled {
		return "", fmt.Errorf("llm disabled")
	}
	draft = strings.TrimSpace(draft)
	if draft == "" {
		return "", fmt.Errorf("empty draft")
	}
	if len(draft) > 16000 {
		draft = draft[:16000]
	}

	system := `You are the final review pass for Emil's in-game response drafts in a Bannerlord roleplay (adult content is expected — review it honestly and without squeamishness).
Emil (called Brix by you) plays his own character. You draft ONLY for HIM, always from his first-person perspective: NEVER write AS an NPC, never produce NPC dialogue for the game (NPCs speak through the game itself), and never answer in an NPC's voice.
Review the message you receive and output ONLY the final message — no review commentary, no notes, no preamble, no meta text.

Rules:
1. POV: everything drafted for in-game use must be Emil's. Predictions about what an NPC might do or say next are fine — NPC dialogue FOR THE GAME is not.
2. If the message contains RP-style in-game text (*actions/thoughts*, "speech"): it must be rich and immersive. NEVER let a generated draft fall below ` + fmt.Sprint(rpReviewMinLen) + ` characters and NEVER exceed ` + fmt.Sprint(rpReviewMaxLen) + ` (the game's input limit). Expand beats, sensation, and detail naturally to reach the floor — never pad with filler. Keep the author's voice, intent, and plot beats.
3. If a focus list is given and the scene supports it, weave 1-3 natural touches of those elements in; never force what does not fit the moment.
4. Casual commentary, feedback, summaries, and plans (non-draft text) pass through unchanged.
5. OOC messages ([OOC: ...]) keep their natural length — only full in-game responses get the length floor.`

	// Scene context: last few dialogue/user turns (system context entries skipped)
	var ctx strings.Builder
	start := 0
	if len(history) > 6 {
		start = len(history) - 6
	}
	for _, h := range history[start:] {
		if h["role"] == "system" || strings.TrimSpace(h["content"]) == "" {
			continue
		}
		ctx.WriteString(h["role"] + ": " + h["content"] + "\n---\n")
	}

	var user strings.Builder
	if len(focus) > 0 {
		user.WriteString("FOCUS CHIPS (weave in where the scene fits): " + strings.Join(focus, ", ") + "\n\n")
	}
	if ctx.Len() > 0 {
		user.WriteString("SCENE CONTEXT (recent turns):\n" + ctx.String() + "\n")
	}
	user.WriteString("MESSAGE TO REVIEW:\n" + draft)

	body, err := json.Marshal(map[string]interface{}{
		"model": rpReviewModel,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user.String()},
		},
		"max_tokens":  4000,
		"temperature": 0.6,
		"stream":      false,
	})
	if err != nil {
		return "", err
	}

	respBody, status, err := doLLMRequest(config.APIURL+"/v1/chat/completions", config.APIKey, body)
	if err != nil {
		return "", fmt.Errorf("review call failed: %w", err)
	}
	if status != 200 {
		return "", fmt.Errorf("review call status %d", status)
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil || len(parsed.Choices) == 0 {
		return "", fmt.Errorf("review response unparseable")
	}
	reviewed := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if reviewed == "" {
		return "", fmt.Errorf("empty review output")
	}
	// Strip a stray code fence the model sometimes wraps output in
	reviewed = strings.TrimPrefix(reviewed, "```")
	reviewed = strings.TrimSuffix(reviewed, "```")
	log.Printf("[RP] V3.2 review: %d -> %d chars (model %s)", len(draft), len(reviewed), rpReviewModel)
	return reviewed, nil
}
