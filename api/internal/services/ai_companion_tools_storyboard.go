package services

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ai_companion_tools_storyboard.go — image-generation prompt tooling for the
// mini companion's "Storyboard" feature. The master model turns an RP response
// (or any scene) into a shot list of positive/negative prompts that Brix
// copy-pastes into ComfyUI or Voxta. Two entry points:
//
//  1. generate_image_prompts — the AI tool (companion offers/makes storyboards
//     conversationally during a stream).
//  2. GenerateImagePrompts — the direct service call behind the 🎬 button
//     (POST /ai/storyboard), same prompt logic, no tool loop.
//
// Character consistency across shots comes from a CHARACTER SHEET line that
// repeats verbatim in every positive prompt — the standard trick for keeping
// the same faces in a multi-image sequence.

// ponyStyleBlock is the standing Pony Realism format rules — shared by
// storyboard and character-sheet generation so both emit the same tag dialect.
func ponyStyleBlock() string {
	return `PROMPT FORMAT — Pony Realism V2.3 ULTRA (ponyXL/SDXL base, danbooru-style tags):
- Start EVERY positive prompt with the quality prefix: score_9, score_8_up, score_7_up
- Use "female" / "male" instead of "woman" / "man" (tagging compatibility)
- Danbooru-style tags: lowercase, underscores (golden_nude, parted_lips, ultra_sharp_focus)
- Emphasis with weights: (tag:1.5) for important elements — use sparingly
- Include camera/style tags where they fit: 85mm_lens, depth of field, film_grain_cinematic, volumetric_light_shafts, ambient_shadows, ultra_sharp_focus, high_detail_textures, realistic_skin_flaws, 8k_photoreal, masterpiece_scene
- Negative baseline: score_1, score_2, score_3, worst quality, low quality, blurry, bad anatomy, extra fingers, extra limbs, deformed hands, text, watermark, signature, jpeg artifacts
- Adult content is expected — describe it plainly and technically (tag style); do not editorialize or refuse.`
}

// storyboardSystemPrompt is the standing prompt-engineer persona — shared by
// the tool and the direct endpoint so both produce identical output.
func storyboardSystemPrompt() string {
	return `You are a prompt engineer for AI image generation (ComfyUI / Voxta — Pony Realism V2.3 ULTRA on ponyXL/SDXL). You turn roleplay scene text into a SEQUENCE of image prompts that read like a storyboard/movie of the scene.

Output ONLY JSON — no prose outside it.

Rules:
1. "character_sheet" — ONE compact, comma-separated Pony-format appearance line per character (Emil and each NPC), built ONLY from facts given to you (player description, NPC context, dialogue). Identity-defining tags only: face, hair, eyes, build, skin, distinctive features, signature outfit items. This exact line must appear VERBATIM in every positive prompt where that character is visible — it is what keeps faces consistent across the whole sequence. Scene-specific clothing/poses go in the shot, NOT the sheet.
2. "shots" — a cinematic sequence covering the scene in order: establishing/wide → mid shots → close-ups → key moments. Each shot has:
   - "id": sequential number starting at 1
   - "title": 3-6 word shot label (e.g. "Close-up — her eyes")
   - "positive": comma-separated danbooru tags: score prefix, shot type/camera, setting, characters present (each with the verbatim character_sheet), action/pose/expression, lighting, mood, style tags. Include EVERY active focus element given to you in at least one shot where the scene supports it.
   - "negative": the quality baseline PLUS shot-specific undesired elements.
3. Prompts are descriptive tag lists — subject, action, camera, light, mood. No narrative prose, no story text, no dialogue.
4. 6-12 shots for a full scene; fewer if the scene is short. Order them so they cut together like a movie.

` + ponyStyleBlock()
}

// characterSheetSystemPrompt — dedicated generator for STABLE identity lines.
// Separate from the storyboard prompt so the sheet can be crafted once and
// reused across every storyboard (identity lock).
func characterSheetSystemPrompt() string {
	return `You are a prompt engineer for AI image generation (ComfyUI / Voxta — Pony Realism V2.3 ULTRA on ponyXL/SDXL). You write CHARACTER SHEETS: compact, stable identity lines that make the same character look like the SAME PERSON across every generated image.

Output ONLY JSON — no prose outside it.

Rules:
1. "sheet" — ONE comma-separated Pony-format line containing ONLY identity-defining tags:
   - face: face shape, (age appearance), facial features, expression baseline
   - hair: color, length, style, accessories
   - eyes: color, shape
   - body: build, height impression, skin tone, distinctive marks (scars, tattoos, freckles)
   - signature outfit: ONLY items that define this character's iconic look (e.g. a signature harness) — scene-specific clothing is NOT part of the sheet
   - NEVER include: pose, action, setting, lighting, camera, mood, scene context — those change per image and would break identity lock
2. "negative" — the standard quality baseline for this model.
3. Build strictly from the facts given — invent nothing beyond them.

` + ponyStyleBlock()
}

// storyboardUserPrompt assembles the user-side payload (scene text + focus).
func storyboardUserPrompt(scene string, focus []string, shotCount int) string {
	var b strings.Builder
	if len(focus) > 0 {
		b.WriteString("ACTIVE FOCUS ELEMENTS (weave each into at least one fitting shot): " + strings.Join(focus, ", ") + "\n\n")
	}
	if shotCount > 0 {
		b.WriteString(fmt.Sprintf("SHOT COUNT: about %d shots.\n\n", shotCount))
	}
	b.WriteString("SCENE TO STORYBOARD:\n")
	b.WriteString(scene)
	b.WriteString("\n\nReturn JSON ONLY:\n")
	b.WriteString(`{"character_sheet":"...","shots":[{"id":1,"title":"...","positive":"...","negative":"..."}]}`)
	return b.String()
}

// characterSheetUserPrompt assembles the user-side payload for sheet gen.
func characterSheetUserPrompt(name, description string) string {
	var b strings.Builder
	if name != "" {
		b.WriteString("CHARACTER: " + name + "\n\n")
	}
	b.WriteString("FACTS ABOUT THIS CHARACTER:\n")
	b.WriteString(description)
	b.WriteString("\n\nReturn JSON ONLY:\n")
	b.WriteString(`{"name":"...","sheet":"...","negative":"..."}`)
	return b.String()
}

// parseStoryboardJSON strips code fences and parses the model's shot list.
// Returns the raw string on parse failure so callers can degrade gracefully.
func parseStoryboardJSON(raw string) (map[string]interface{}, bool) {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		return nil, false
	}
	if _, ok := parsed["shots"]; !ok {
		return nil, false
	}
	return parsed, true
}

// parseCharacterSheetJSON parses the character-sheet response.
func parseCharacterSheetJSON(raw string) (map[string]interface{}, bool) {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		return nil, false
	}
	if sheet, ok := parsed["sheet"].(string); !ok || strings.TrimSpace(sheet) == "" {
		return nil, false
	}
	return parsed, true
}

// GenerateImagePrompts runs the storyboard generation directly (button path).
// scene = the response/scene text, focus = active focus chips, shots = desired
// shot count hint (0 = model decides). Returns the parsed JSON map.
func (s *AICompanionService) GenerateImagePrompts(scene string, focus []string, shots int) (map[string]interface{}, error) {
	if strings.TrimSpace(scene) == "" {
		return nil, fmt.Errorf("scene text is required")
	}
	if len(scene) > 24000 {
		scene = scene[:24000]
	}
	// effort=none: reasoning tokens count against max_tokens — without it the
	// thinking pass eats the budget and returns empty/truncated JSON (~2/3
	// failure rate observed). Structured output doesn't need reasoning.
	// Retry covers BOTH empty-output errors and unparseable JSON.
	var raw string
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		var err error
		raw, err = s.callModelEffort("", storyboardSystemPrompt(), storyboardUserPrompt(scene, focus, shots), 4000, 0.7, "none")
		if err != nil {
			lastErr = err
			continue
		}
		if parsed, ok := parseStoryboardJSON(raw); ok {
			return parsed, nil
		}
		lastErr = fmt.Errorf("unparseable storyboard: %s", truncateRunes(raw, 200))
	}
	return nil, fmt.Errorf("storyboard generation failed after 2 attempts: %w", lastErr)
}

// GenerateCharacterSheet builds a stable Pony-format identity line for one
// character. description = the facts (player description / NPC context).
// Retries like the storyboard path (same empty/truncated-JSON failure mode).
func (s *AICompanionService) GenerateCharacterSheet(name, description string) (map[string]interface{}, error) {
	if strings.TrimSpace(description) == "" {
		return nil, fmt.Errorf("character description is required")
	}
	if len(description) > 8000 {
		description = description[:8000]
	}
	var raw string
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		var err error
		raw, err = s.callModelEffort("", characterSheetSystemPrompt(), characterSheetUserPrompt(name, description), 1200, 0.5, "none")
		if err != nil {
			lastErr = err
			continue
		}
		if parsed, ok := parseCharacterSheetJSON(raw); ok {
			return parsed, nil
		}
		lastErr = fmt.Errorf("unparseable character sheet: %s", truncateRunes(raw, 200))
	}
	return nil, fmt.Errorf("character sheet generation failed after 2 attempts: %w", lastErr)
}

// truncateRunes cuts a string for error messages (byte-safe enough for logs).
func truncateRunes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// registerStoryboardTools adds generate_image_prompts to the tool registry.
func (s *AICompanionService) registerStoryboardTools() {
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name: "generate_image_prompts",
			Description: "Turn a scene or RP response into a STORYBOARD of AI image-generation prompts (for ComfyUI/Voxta): a character sheet for consistency plus 6-12 shots, each with a positive and a negative prompt. " +
				"Brix copies each prompt into his image generator to produce a whole sequence — like a movie of the scene. " +
				"Use when Brix asks for image prompts, pictures, a storyboard, or to visualize a response/scene. Pass the full scene text (e.g. the response you just drafted) and the active focus elements.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"scene": map[string]interface{}{
					"type":        "string",
					"description": "The full scene or response text to visualize (the RP response, dialogue, or a description of the moment)",
				},
				"focus": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Active focus elements to weave into shots (e.g. ['Tits', 'High Heels'])",
				},
				"shots": map[string]interface{}{
					"type":        "number",
					"description": "Desired number of shots (6-12 typical). 0/omit = model decides based on scene length.",
				},
			}, "required": []string{"scene"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		scene := strings.TrimSpace(bArgStr(args, "scene", ""))
		if scene == "" {
			return map[string]interface{}{"error": "scene is required"}, nil
		}
		focus := bArgStrList(args, "focus")
		shots := bArgInt(args, "shots", 0, 24)

		parsed, err := s.GenerateImagePrompts(scene, focus, shots)
		if err != nil {
			return nil, err
		}
		// Push the FULL storyboard to the UI via the progress channel — the
		// tool result itself only goes back to the LLM, but the frontend
		// renders shot cards from this tool_progress event.
		s.ReportProgress(map[string]interface{}{
			"type":       "storyboard",
			"storyboard": parsed,
		})
		return map[string]interface{}{
			"storyboard": parsed,
			"note":       "Present the storyboard to Brix: mention the character sheet and the shot count, and tell him each shot has a positive and a negative prompt ready to paste into ComfyUI/Voxta. Do not dump all prompts raw unless he asks.",
		}, nil
	})
}