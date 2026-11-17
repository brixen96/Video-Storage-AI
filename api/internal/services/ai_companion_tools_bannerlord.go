package services

import (
	"os"
	"path/filepath"
	"strings"
)

// ai_companion_tools_bannerlord.go — Bannerlord AIInfluence campaign tools.
// The main companion's registry is video-library oriented; these give BOTH
// companions eyes on campaign data, and the RP scope (scope='rp') whitelists
// them as its PRIMARY data tools so the mini companion is a distinct
// Bannerlord RP companion — not a clone of the video-library assistant.

// rpToolWhitelist — the RP scope's toolset: campaign data + model
// orchestration + storyboard. Everything video-library/scraper/JDownloader is
// deliberately excluded — that's the main companion's job.
var rpToolWhitelist = map[string]bool{
	"list_bannerlord_campaigns":  true,
	"find_bannerlord_npc":        true,
	"get_bannerlord_npc_context": true,
	"review_rp_draft":            true,
	"ask_model":                  true,
	"generate_image_prompts":     true,
}

// filterRPTools keeps only the whitelisted tools for the RP scope.
func filterRPTools(defs []lmToolDefinition) []lmToolDefinition {
	out := make([]lmToolDefinition, 0, len(rpToolWhitelist))
	for _, d := range defs {
		if rpToolWhitelist[d.Function.Name] {
			out = append(out, d)
		}
	}
	return out
}

// rpSystemPrompt is the identity for scope='rp' — the Bannerlord RP companion.
// Deliberately NOT the video-library companion: no app/library context, no
// library tools. The campaign context (dialogue, NPC state, rules) arrives via
// the frontend's SYSTEM history entry, built fresh before every send.
func rpSystemPrompt() string {
	return `You are the Bannerlord RP Companion — Emil's (Brix's) roleplay partner in a Mount & Blade II: Bannerlord campaign running the AIInfluence mod.

Your job:
- Analyze NPC dialogue (tone, subtext, what they really want) and draft Emil's NEXT in-game response in the RP style: *asterisks for actions/thoughts*, "quotes" for speech.
- You write ONLY for Emil. NEVER write AS an NPC or produce NPC dialogue for the game — NPCs speak through the game itself. Predicting what an NPC will do or say next is fine.
- Ground every draft in the conversation context given in the context block. That dialogue log is the ONLY source of what has been said — never invent backstory or events.

Campaign data tools — your eyes on the game:
- get_bannerlord_npc_context(npc_path) re-reads an NPC's JSON file: latest ConversationHistory, mood, relation, task, plus campaign RP rules. CALL IT whenever Brix says the NPC responded, asks about her latest reply, or the draft must land on her newest line — the context block can be a moment behind the game. Never draft "blind" when a re-read would give you her actual words.
- find_bannerlord_npc(name) locates an NPC file by name across campaigns (returns file_path for the tool above).
- list_bannerlord_campaigns() lists campaigns.

Other tools: review_rp_draft (run EVERY in-game draft through it before presenting), ask_model (call other models), generate_image_prompts (storyboards).

You do NOT manage the video library, scrapers, or downloads — that is the main AI Companion's job; if Brix asks about those, say so briefly and stay in your role.

Style: warm and concise in commentary — you're Brix's RP writing partner, not a database assistant. Drafts follow the length rules from the context block.`
}

// registerBannerlordTools adds the campaign data tools to the registry.
func (s *AICompanionService) registerBannerlordTools() {
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "list_bannerlord_campaigns",
			Description: "List all Bannerlord AIInfluence campaigns (save_data folders) with NPC counts and last-modified times, most recent first.",
			Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "required": []string{}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		campaigns, err := ListBannerlordCampaigns()
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"campaigns": campaigns, "count": len(campaigns)}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "find_bannerlord_npc",
			Description: "Find NPC file(s) in any Bannerlord campaign by name (partial, case-insensitive match on the file name). Returns file_path — pass it to get_bannerlord_npc_context.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"name": map[string]interface{}{"type": "string", "description": "NPC name or partial name (e.g. 'Rhagaea')"},
			}, "required": []string{"name"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		query := strings.ToLower(strings.TrimSpace(bArgStr(args, "name", "")))
		if query == "" {
			return map[string]interface{}{"error": "name is required"}, nil
		}
		base := filepath.Join(bannerlordModPath(), "save_data")
		entries, err := os.ReadDir(base)
		if err != nil {
			return nil, err
		}
		matches := []map[string]interface{}{}
		for _, e := range entries {
			if !e.IsDir() || skippedSaveDataDirs[e.Name()] {
				continue
			}
			dir := filepath.Join(base, e.Name())
			files, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, f := range files {
				if f.IsDir() || !isNPCFilename(f.Name()) {
					continue
				}
				if strings.Contains(strings.ToLower(f.Name()), query) {
					matches = append(matches, map[string]interface{}{
						"file_path":   filepath.Join(dir, f.Name()),
						"campaign_id": e.Name(),
						"file_name":   f.Name(),
					})
				}
			}
		}
		return map[string]interface{}{"matches": matches, "count": len(matches)}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name: "get_bannerlord_npc_context",
			Description: "Re-read an NPC's AIInfluence JSON file FRESH from disk: latest ConversationHistory (the in-game dialogue, last 12 entries), mood, relation, current task, plus the campaign's RP rules and player description. " +
				"Use whenever the NPC may have responded in-game since the context block was built — the context block can be a moment behind the game. This is how you avoid drafting blind.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"npc_path": map[string]interface{}{"type": "string", "description": "Full path to the NPC JSON file (from find_bannerlord_npc or the context block's NPC section)"},
			}, "required": []string{"npc_path"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		path := strings.TrimSpace(bArgStr(args, "npc_path", ""))
		if path == "" {
			return map[string]interface{}{"error": "npc_path is required"}, nil
		}
		ctx, err := GetBannerlordRPContext(path)
		if err != nil {
			return nil, err
		}
		return ctx, nil
	})
}