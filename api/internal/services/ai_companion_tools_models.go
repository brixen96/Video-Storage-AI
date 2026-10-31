package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ai_companion_tools_models.go — the master model's "call other AI models"
// toolkit. The configured chat model (deepseek/deepseek-v4-flash-0731 in App
// settings) is the ORCHESTRATOR: it generates drafts itself and calls other
// models as functions — V3.2 Exp reviews RP drafts, vision models see images
// (future). Each call is synchronous (the tool loop waits), so the master can
// also do other work between calls in the same turn.

const (
	// rpReviewModel is the dedicated RP draft reviewer — Emil's pick: the most
	// NSFW-capable model, and cheap enough for a per-draft pass
	// ($0.27/$0.41 per 1M vs V4-Flash chat costs).
	rpReviewModel = "deepseek/deepseek-v3.2-exp"

	// Length rules — MUST mirror CompanionMiniChat.vue constants.
	// Floor applies to generated in-game drafts; OOC messages keep natural length.
	rpReviewMinLen    = 1600
	rpReviewTargetLen = 1800
	rpReviewMaxLen    = 2048

	// modelCallTimeout bounds every cross-model call so a hung provider can
	// never stall the chat loop.
	modelCallTimeout = 90 * time.Second
)

// callModel is the shared synchronous cross-model request. Returns the first
// choice's content. Non-streaming — the tool loop blocks until it answers.
func (s *AICompanionService) callModel(model, system, user string, maxTokens int, temperature float64) (string, error) {
	config := s.GetAIConfig()
	if !config.Enabled {
		return "", fmt.Errorf("llm disabled")
	}
	if model == "" {
		model = config.ModelName
	}
	body, err := json.Marshal(map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"max_tokens":  maxTokens,
		"temperature": temperature,
		"stream":      false,
	})
	if err != nil {
		return "", err
	}

	type result struct {
		text   string
		status int
		err    error
	}
	ch := make(chan result, 1)
	go func() {
		respBody, status, reqErr := doLLMRequest(config.APIURL+"/v1/chat/completions", config.APIKey, body)
		if reqErr != nil {
			ch <- result{status: status, err: reqErr}
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
			ch <- result{status: status, err: fmt.Errorf("unparseable response")}
			return
		}
		ch <- result{text: strings.TrimSpace(parsed.Choices[0].Message.Content), status: status}
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			return "", fmt.Errorf("model %s call failed (status %d): %w", model, r.status, r.err)
		}
		if r.text == "" {
			return "", fmt.Errorf("model %s returned empty output", model)
		}
		return r.text, nil
	case <-time.After(modelCallTimeout):
		return "", fmt.Errorf("model %s call timed out after %s", model, modelCallTimeout)
	}
}

// rpReviewSystemPrompt is the standing reviewer persona — shared by the
// review_rp_draft tool (master-orchestrated workflow).
func rpReviewSystemPrompt() string {
	return `You are the review pass for Emil's in-game response drafts in a Bannerlord roleplay (adult content is expected — review it honestly and without squeamishness).
Emil (called Brix by you) plays his own character. You review drafts written FOR HIM, always from his first-person perspective: NEVER write AS an NPC, never produce NPC dialogue for the game (NPCs speak through the game itself), and never answer in an NPC's voice.
Output ONLY the requested JSON — no prose outside it.

Rules:
1. POV: everything drafted for in-game use must be Emil's. Predictions about what an NPC might do or say next are fine — NPC dialogue FOR THE GAME is not.
2. RP-style in-game text (*actions/thoughts*, "speech") must be rich and immersive. NEVER let a generated draft fall below ` + fmt.Sprint(rpReviewMinLen) + ` characters and NEVER exceed ` + fmt.Sprint(rpReviewMaxLen) + ` (the game's input limit). Expand beats, sensation, and detail naturally to reach the floor — never pad with filler. Keep the author's voice, intent, and plot beats.
3. If a focus list is given and the scene supports it, weave 1-3 natural touches of those elements in; never force what does not fit the moment.
4. Casual commentary, feedback, summaries, and plans (non-draft text) need no length changes.
5. OOC messages ([OOC: ...]) keep their natural length — only full in-game responses get the length floor.`
}

func (s *AICompanionService) registerModelTools() {
	// ── review_rp_draft ───────────────────────────────────────────────────────
	// The master generates a draft, calls this, then revises with the feedback
	// in mind — the generate → review → revise loop Emil described, driven by
	// the master model itself rather than a hardcoded post-pass.
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name: "review_rp_draft",
			Description: "Send a generated RP response draft to the V3.2 Exp reviewer model and get structured feedback (POV violations, length fixes, focus-chip weaving, scene-fit notes). " +
				"WORKFLOW: after you generate each draft option, call this with it, then revise the draft using the feedback before presenting it. " +
				"Use for every in-game response draft you write for Brix. Do NOT use for casual chat replies or OOC messages.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"draft": map[string]interface{}{
					"type":        "string",
					"description": "The full draft text to review (one response option)",
				},
				"focus": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Active focus chips to check for (e.g. ['Tits', 'High Heels'])",
				},
				"scene": map[string]interface{}{
					"type":        "string",
					"description": "Short scene context: who is present, what just happened (helps the reviewer judge fit)",
				},
			}, "required": []string{"draft"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		draft := strings.TrimSpace(bArgStr(args, "draft", ""))
		if draft == "" {
			return map[string]interface{}{"error": "draft is required"}, nil
		}
		if len(draft) > 16000 {
			draft = draft[:16000]
		}
		focus := bArgStrList(args, "focus")
		scene := bArgStr(args, "scene", "")

		var user strings.Builder
		if len(focus) > 0 {
			user.WriteString("FOCUS CHIPS (weave in where the scene fits): " + strings.Join(focus, ", ") + "\n\n")
		}
		if scene != "" {
			user.WriteString("SCENE: " + scene + "\n\n")
		}
		user.WriteString("DRAFT TO REVIEW:\n" + draft + "\n\n")
		user.WriteString(`Return JSON ONLY:
{"feedback":["short actionable point", "..."],"length_ok":true|false,"pov_ok":true|false,"focus_used":["chip", "..."],"verdict":"pass|revise"}`)

		raw, err := s.callModel(rpReviewModel, rpReviewSystemPrompt(), user.String(), 1500, 0.4)
		if err != nil {
			return nil, err
		}
		cleaned := strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(strings.TrimSpace(raw), "```"), "```json"))
		var parsed struct {
			Feedback   []string `json:"feedback"`
			LengthOK   bool     `json:"length_ok"`
			POVOK      bool     `json:"pov_ok"`
			FocusUsed  []string `json:"focus_used"`
			Verdict    string   `json:"verdict"`
		}
		if json.Unmarshal([]byte(cleaned), &parsed) != nil {
			// Reviewer wrapped prose instead of JSON — return it raw so the master still gets the notes
			return map[string]interface{}{"verdict": "revise", "feedback": []string{raw}, "parse_warning": "reviewer returned non-JSON"}, nil
		}
		if parsed.Verdict == "" {
			parsed.Verdict = "revise"
		}
		return map[string]interface{}{
			"reviewer":    rpReviewModel,
			"verdict":     parsed.Verdict,
			"feedback":    parsed.Feedback,
			"length_ok":   parsed.LengthOK,
			"pov_ok":      parsed.POVOK,
			"focus_used":  parsed.FocusUsed,
			"min_length":  rpReviewMinLen,
			"max_length":  rpReviewMaxLen,
			"note":        "Revise the draft using this feedback, then present the final version to Brix. Review each draft option separately.",
		}, nil
	})

	// ── ask_model ─────────────────────────────────────────────────────────────
	// Generic cross-model call — the master's "call any other AI model" function.
	// This is how future vision work plugs in (image_url content parts) without
	// new plumbing: the master already knows how to ask another model for help.
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name: "ask_model",
			Description: "Ask a DIFFERENT AI model a question and get its answer (synchronous — you wait for it, then continue). " +
				"Use when another model has a specialty you lack: e.g. model='deepseek/deepseek-v3.2-exp' for NSFW-fluent RP writing/review, " +
				"or a vision-capable model for image questions. Omit model to use your own model. Keep prompt self-contained (the other model sees nothing of this conversation).",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"model": map[string]interface{}{
					"type":        "string",
					"description": "OpenRouter model id (e.g. deepseek/deepseek-v3.2-exp). Empty = your own model.",
				},
				"prompt": map[string]interface{}{
					"type":        "string",
					"description": "The full question/task for the other model",
				},
				"system": map[string]interface{}{
					"type":        "string",
					"description": "Optional system prompt for the other model",
				},
				"max_tokens": map[string]interface{}{
					"type":        "number",
					"description": "Max response tokens (default 2000)",
				},
			}, "required": []string{"prompt"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		prompt := strings.TrimSpace(bArgStr(args, "prompt", ""))
		if prompt == "" {
			return map[string]interface{}{"error": "prompt is required"}, nil
		}
		model := strings.TrimSpace(bArgStr(args, "model", ""))
		system := strings.TrimSpace(bArgStr(args, "system", "You are a helpful specialist model called by an AI assistant."))
		maxTokens := bArgInt(args, "max_tokens", 2000, 8000)

		answer, err := s.callModel(model, system, prompt, maxTokens, 0.7)
		if err != nil {
			return nil, err
		}
		if len(answer) > 24000 {
			answer = answer[:24000] + "\n...(truncated)"
		}
		used := model
		if used == "" {
			used = s.GetAIConfig().ModelName
		}
		return map[string]interface{}{"model": used, "answer": answer}, nil
	})
}

// bArgStrList reads a JSON array-of-strings argument.
func bArgStrList(args map[string]interface{}, key string) []string {
	raw, ok := args[key].([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}