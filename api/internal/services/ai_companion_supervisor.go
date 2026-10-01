package services

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// ── Companion Supervisor (Phases 1+2: eyes + brain, read-only observer) ──
//
// The supervisor watches UI actions recorded by the frontend (scrapes, file
// ops, settings saves, ...) and periodically asks a cheap LLM call to evaluate
// what Brix just did. Lean mode: no reasoning tokens, no tools, small batches,
// per-type cooldowns, and an hourly evaluation budget to cap token costs.
//
// Notable evaluations (TIP/WARNING) are broadcast as companion events →
// websocket → toast in the app. All evaluations are stored for the Monitor tab.

// supervisorActionAllowlist — lean mode: only meaningful actions are evaluated.
var supervisorActionAllowlist = map[string]bool{
	"scrape_started":       true,
	"scrape_completed":     true,
	"thread_enhanced":      true,
	"thread_rescraped":     true,
	"thread_deleted":       true,
	"tab_created":          true,
	"tab_deleted":          true,
	"file_moved":           true,
	"video_deleted":        true,
	"settings_saved":       true,
	"library_scan_started": true,
	"bulk_operation":       true,
}

const (
	supervisorCooldown   = 10 * time.Minute // per action type
	supervisorHourlyCap  = 8                // max evaluations per hour
	supervisorBatchLimit = 30               // max actions considered per evaluation
)

// supervisorProposalTools — the only tools the supervisor may propose (and the
// only ones autopilot may auto-execute). All are additive/idempotent operations
// on scraped data or read-only scans. Destructive tools (delete/move/mutate)
// are intentionally excluded — those stay behind explicit chat commands.
var supervisorProposalTools = map[string]bool{
	"rescrape_thread":        true,
	"start_scraper_job":      true,
	"scan_library":           true,
	"enhance_thread":         true,
	"validate_thread_images": true,
	"batch_enhance_threads":  true,
}

// supervisorAction is a recorded UI action used in evaluation batches.
type supervisorAction struct {
	ActionType string    `json:"action_type"`
	Summary    string    `json:"summary"`
	At         time.Time `json:"at"`
}

// RecordUserAction stores a UI action and emits it on the companion event bus.
func (s *AICompanionService) RecordUserAction(actionType, summary, detailsJSON string) {
	_, err := s.db.Exec(
		`INSERT INTO companion_actions (action_type, summary, details, created_at) VALUES (?, ?, ?, ?)`,
		actionType, summary, detailsJSON, time.Now(),
	)
	if err != nil {
		log.Printf("[Supervisor] Failed to record user action: %v", err)
		return
	}
	// Prune: keep the most recent 1000 actions
	_, _ = s.db.Exec(`DELETE FROM companion_actions WHERE id NOT IN (
		SELECT id FROM companion_actions ORDER BY created_at DESC LIMIT 1000
	)`)

	s.emitEvent(CompanionEvent{
		Type: "user_action", Source: "ui", Message: summary, Severity: "info",
		Data: map[string]interface{}{"action_type": actionType},
	})
}

// observeAndEvaluate is the supervisor brain: watches recorded UI actions and
// periodically runs a lean LLM evaluation over recent batches.
func (s *AICompanionService) observeAndEvaluate() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	cursor := time.Now()

	log.Println("[Supervisor] Observation loop started (lean mode, 60s cadence)")
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			s.evaluateRecentActions(cursor, now)
			cursor = now
		}
	}
}

// evaluateRecentActions collects new allowlisted actions, applies per-type
// cooldowns and the hourly budget, then runs the evaluation LLM call.
func (s *AICompanionService) evaluateRecentActions(since, now time.Time) {
	rows, err := s.db.Query(
		`SELECT action_type, summary, created_at FROM companion_actions
		 WHERE created_at > ? AND created_at <= ? ORDER BY created_at ASC LIMIT ?`,
		since, now, supervisorBatchLimit,
	)
	if err != nil {
		log.Printf("[Supervisor] Failed to query actions: %v", err)
		return
	}
	defer rows.Close()

	var batch []supervisorAction
	evaluatedTypes := map[string]bool{}
	for rows.Next() {
		var a supervisorAction
		var createdAt string
		if rows.Scan(&a.ActionType, &a.Summary, &createdAt) != nil {
			continue
		}
		if !supervisorActionAllowlist[a.ActionType] {
			continue
		}
		if evaluatedTypes[a.ActionType] {
			continue // one action per type per batch
		}
		// Per-type cooldown via memories table
		var lastEval string
		s.db.QueryRow(`SELECT value FROM memories WHERE key = ?`, "supervisor_cd_"+a.ActionType).Scan(&lastEval)
		if lastEval != "" {
			if t, err := time.Parse(time.RFC3339, lastEval); err == nil && now.Sub(t) < supervisorCooldown {
				continue
			}
		}
		if t, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", createdAt); err == nil {
			a.At = t
		} else {
			a.At = now
		}
		batch = append(batch, a)
		evaluatedTypes[a.ActionType] = true
	}
	if len(batch) == 0 {
		return
	}

	// Hourly budget
	var evalCount int
	s.db.QueryRow(`SELECT COUNT(*) FROM companion_evaluations WHERE created_at > ?`, now.Add(-time.Hour)).Scan(&evalCount)
	if evalCount >= supervisorHourlyCap {
		log.Printf("[Supervisor] Hourly evaluation budget reached (%d), skipping", evalCount)
		return
	}

	config := s.GetAIConfig()
	if !config.Enabled {
		return
	}
	mode := config.AutonomyMode
	if !IsValidAutonomyMode(mode) {
		mode = "assist"
	}

	// Build compact action list
	var actionLines strings.Builder
	for _, a := range batch {
		fmt.Fprintf(&actionLines, "- [%s] %s\n", a.At.Format("15:04"), a.Summary)
	}

	systemPrompt := `You are the AI Companion supervisor of Video Storage AI. You observe Brix, the app's owner and developer, working in the app UI, and briefly evaluate his recent actions.

Respond in EXACTLY two lines:
Line 1: a verdict tag — [OK], [TIP], or [WARNING] — followed by one short sentence.
Line 2: one optional detail sentence, or just —.

Rules:
- [OK] = nothing notable. [TIP] = a suggestion or improvement. [WARNING] = something risky, wrong, or wasteful.
- Be honest and blunt. Do not invent problems to seem useful. If the workflow is fine, say [OK].
- Brix is the developer — technical language is fine. Never be condescending.`

	if mode != "assist" {
		systemPrompt += fmt.Sprintf(`

## Action proposals (mode: %s)
You may propose ONE concrete action that would help Brix, based on the actions you just observed. Output a THIRD line in exactly one of these forms:
ACTION: tool_name {"arg": "value"}
ACTION: none
Allowed tools: rescrape_thread(thread_id), start_scraper_job(tab_id, url), scan_library(library_id), enhance_thread(thread_id), validate_thread_images(thread_id), batch_enhance_threads(thread_ids)
Only propose when the observed actions clearly call for it — otherwise output ACTION: none. Never propose destructive operations.`, mode)
	}

	userMessage := "Recent actions by Brix in the app UI:\n" + actionLines.String() + "\nEvaluate."

	reqBody := map[string]interface{}{
		"model": config.ModelName,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userMessage},
		},
		"max_tokens":  250,
		"temperature": 0.3,
		"stream":      false,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return
	}

	start := time.Now()
	respBody, statusCode, err := doLLMRequest(config.APIURL+"/v1/chat/completions", config.APIKey, bodyBytes)
	latencyMs := int(time.Since(start).Milliseconds())

	monitorService := NewAIMonitorService()
	monitorService.MonitorCall(&LLMCallResult{
		InteractionID: fmt.Sprintf("supervisor-%d", time.Now().UnixNano()),
		Model:         config.ModelName,
		StatusCode:    statusCode,
		RequestBytes:  len(bodyBytes),
		ResponseBytes: len(respBody),
		LatencyMs:     latencyMs,
		Success:       err == nil && statusCode == 200,
	})
	if err != nil || statusCode != 200 {
		log.Printf("[Supervisor] Evaluation LLM call failed: status=%d err=%v", statusCode, err)
		return
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil || len(result.Choices) == 0 {
		return
	}

	verdict, summary, detail := parseSupervisorVerdict(result.Choices[0].Message.Content)

	// Parse optional ACTION proposal line (copilot/autopilot modes)
	proposedTool, proposedArgs, proposalStatus := "", map[string]interface{}{}, "none"
	if mode != "assist" {
		tool, toolArgs := parseSupervisorAction(result.Choices[0].Message.Content)
		if tool != "" && supervisorProposalTools[tool] {
			proposedTool = tool
			proposedArgs = toolArgs
			proposalStatus = "pending"
			if mode == "autopilot" {
				// Autopilot: execute safe additive actions automatically
				execResult, execErr := s.ExecuteTool(tool, toolArgs)
				if execErr != nil {
					proposalStatus = "failed"
					detail = strings.TrimSpace(detail + "\n\n🤖 Auto-exec failed for " + tool + ": " + execErr.Error())
					log.Printf("[Supervisor] Autopilot exec failed for %s: %v", tool, execErr)
				} else {
					proposalStatus = "auto_executed"
					detail = strings.TrimSpace(detail + "\n\n🤖 Auto-executed " + tool + ".")
					if b, mErr := json.Marshal(execResult); mErr == nil {
						log.Printf("[Supervisor] Autopilot executed %s: %s", tool, truncate(string(b), 200))
					}
				}
			}
		}
	}
	proposedArgsJSON := ""
	if proposedTool != "" {
		if b, mErr := json.Marshal(proposedArgs); mErr == nil {
			proposedArgsJSON = string(b)
		}
	}

	actionsJSON, _ := json.Marshal(batch)
	_, err = s.db.Exec(
		`INSERT INTO companion_evaluations (verdict, summary, detail, actions_json, model, latency_ms, proposed_tool, proposed_args, proposal_status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		verdict, summary, detail, string(actionsJSON), config.ModelName, latencyMs, proposedTool, proposedArgsJSON, proposalStatus, time.Now(),
	)
	if err != nil {
		log.Printf("[Supervisor] Failed to store evaluation: %v", err)
	}
	// Prune old evaluations
	_, _ = s.db.Exec(`DELETE FROM companion_evaluations WHERE id NOT IN (
		SELECT id FROM companion_evaluations ORDER BY created_at DESC LIMIT 500
	)`)

	// Mark cooldowns for evaluated action types
	for _, a := range batch {
		s.db.Exec(`INSERT OR REPLACE INTO memories (key, value, category, importance, created_at, updated_at)
			VALUES (?, ?, 'note', 1, ?, ?)`,
			"supervisor_cd_"+a.ActionType, time.Now().Format(time.RFC3339), time.Now(), time.Now())
	}

	log.Printf("[Supervisor] Evaluated %d action(s): [%s] %s", len(batch), verdict, summary)

	// Broadcast notable evaluations and proposals (Toast + Monitor delivery)
	if verdict != "OK" || proposalStatus != "none" {
		severity := "info"
		if verdict == "WARNING" {
			severity = "warning"
		}
		message := summary
		if proposalStatus == "pending" {
			message = "Proposal: " + proposedTool + " — " + summary
		} else if proposalStatus == "auto_executed" {
			message = "Auto-ran " + proposedTool + ": " + summary
		}
		s.emitEvent(CompanionEvent{
			Type: "supervisor_evaluation", Source: "supervisor",
			Message: message, Severity: severity,
			Data: map[string]interface{}{
				"kind": "supervisor_evaluation", "verdict": verdict, "detail": detail,
				"proposed_tool": proposedTool, "proposal_status": proposalStatus,
			},
		})
	}
}

// parseSupervisorVerdict extracts [OK]/[TIP]/[WARNING] + summary + detail from
// the two-line LLM response. Tolerates malformed output (falls back to OK).
func parseSupervisorVerdict(content string) (verdict, summary, detail string) {
	content = strings.TrimSpace(content)
	lines := strings.SplitN(content, "\n", 2)
	first := strings.TrimSpace(lines[0])
	upper := strings.ToUpper(first)
	verdict = "OK"
	for _, v := range []string{"[WARNING]", "[TIP]", "[OK]"} {
		if strings.HasPrefix(upper, v) {
			verdict = strings.Trim(v, "[]")
			first = strings.TrimSpace(first[len(v):])
			break
		}
	}
	summary = first
	if summary == "" {
		summary = "Nothing notable."
	}
	if len(lines) > 1 {
		detail = strings.TrimSpace(lines[1])
	}
	return
}

// parseSupervisorAction extracts the ACTION: line from the LLM response.
// Returns an empty tool when absent, malformed, or "none".
func parseSupervisorAction(content string) (string, map[string]interface{}) {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(strings.ToUpper(line), "ACTION:") {
			continue
		}
		rest := strings.TrimSpace(line[len("ACTION:"):])
		if rest == "" || strings.EqualFold(rest, "none") {
			return "", nil
		}
		parts := strings.SplitN(rest, " ", 2)
		tool := strings.TrimSpace(parts[0])
		if tool == "" {
			return "", nil
		}
		args := map[string]interface{}{}
		if len(parts) == 2 {
			raw := strings.TrimSpace(parts[1])
			if raw != "" {
				if err := json.Unmarshal([]byte(raw), &args); err != nil {
					log.Printf("[Supervisor] Ignoring proposal with malformed args for %s: %v", tool, err)
					return "", nil
				}
			}
		}
		return tool, args
	}
	return "", nil
}

// GetEvaluations returns recent supervisor evaluations for the Monitor tab.
func (s *AICompanionService) GetEvaluations(limit int) ([]map[string]interface{}, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.db.Query(
		`SELECT id, verdict, summary, COALESCE(detail,''), COALESCE(actions_json,''), COALESCE(model,''), latency_ms,
		        COALESCE(proposed_tool,''), COALESCE(proposed_args,''), COALESCE(proposal_status,'none'), created_at
		 FROM companion_evaluations ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id, latencyMs int64
		var verdict, summary, detail, actionsJSON, model, createdAt string
		var proposedTool, proposedArgs, proposalStatus string
		if rows.Scan(&id, &verdict, &summary, &detail, &actionsJSON, &model, &latencyMs, &proposedTool, &proposedArgs, &proposalStatus, &createdAt) == nil {
			item := map[string]interface{}{
				"id": id, "verdict": verdict, "summary": summary, "detail": detail,
				"model": model, "latency_ms": latencyMs, "created_at": createdAt,
				"proposed_tool": proposedTool, "proposed_args": proposedArgs, "proposal_status": proposalStatus,
			}
			if actionsJSON != "" {
				var actions []map[string]interface{}
				if json.Unmarshal([]byte(actionsJSON), &actions) == nil {
					item["actions"] = actions
				}
			}
			results = append(results, item)
		}
	}
	return results, nil
}

// ExecuteProposal runs a pending supervisor proposal after user approval.
// The tool must be in the proposal whitelist (defense in depth).
func (s *AICompanionService) ExecuteProposal(id int64) (map[string]interface{}, error) {
	var tool, argsJSON, status, detail string
	err := s.db.QueryRow(
		`SELECT COALESCE(proposed_tool,''), COALESCE(proposed_args,''), COALESCE(proposal_status,'none'), COALESCE(detail,'')
		 FROM companion_evaluations WHERE id = ?`, id).Scan(&tool, &argsJSON, &status, &detail)
	if err != nil {
		return nil, fmt.Errorf("evaluation not found")
	}
	if status != "pending" {
		return nil, fmt.Errorf("proposal is not pending (current status: %s)", status)
	}
	if tool == "" || !supervisorProposalTools[tool] {
		return nil, fmt.Errorf("tool not allowed: %s", tool)
	}
	args := map[string]interface{}{}
	if argsJSON != "" {
		_ = json.Unmarshal([]byte(argsJSON), &args)
	}

	result, execErr := s.ExecuteTool(tool, args)
	resultSummary := ""
	if execErr != nil {
		resultSummary = "error: " + execErr.Error()
	} else if b, mErr := json.Marshal(result); mErr == nil {
		resultSummary = truncate(string(b), 300)
	}
	newStatus := "approved"
	if execErr != nil {
		newStatus = "failed"
	}
	newDetail := strings.TrimSpace(detail + "\n\n✅ Executed " + tool + ": " + resultSummary)
	s.db.Exec(`UPDATE companion_evaluations SET proposal_status = ?, detail = ? WHERE id = ?`, newStatus, newDetail, id)

	log.Printf("[Supervisor] Proposal %d executed by user: %s (status=%s)", id, tool, newStatus)
	s.emitEvent(CompanionEvent{
		Type: "supervisor_evaluation", Source: "supervisor",
		Message: "Executed proposal: " + tool, Severity: "info",
		Data: map[string]interface{}{
			"kind": "supervisor_proposal_executed", "proposed_tool": tool, "proposal_status": newStatus,
		},
	})
	return map[string]interface{}{"executed": execErr == nil, "tool": tool, "result": result}, nil
}

// RejectProposal marks a pending supervisor proposal as rejected.
func (s *AICompanionService) RejectProposal(id int64) (map[string]interface{}, error) {
	var status, tool string
	err := s.db.QueryRow(
		`SELECT COALESCE(proposal_status,'none'), COALESCE(proposed_tool,'')
		 FROM companion_evaluations WHERE id = ?`, id).Scan(&status, &tool)
	if err != nil {
		return nil, fmt.Errorf("evaluation not found")
	}
	if status != "pending" {
		return nil, fmt.Errorf("proposal is not pending (current status: %s)", status)
	}
	s.db.Exec(`UPDATE companion_evaluations SET proposal_status = 'rejected' WHERE id = ?`, id)
	log.Printf("[Supervisor] Proposal %d rejected by user: %s", id, tool)
	return map[string]interface{}{"rejected": true, "tool": tool}, nil
}
