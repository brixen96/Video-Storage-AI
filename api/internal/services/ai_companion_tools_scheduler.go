package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/brixen96/video-storage-ai/internal/models"
)

// registerSchedulerTools exposes the job scheduler to the AI Companion so it
// can build its own routines ("every night at 3am, verify links").
//
// The scheduler service is a package-level singleton in the api layer — tools
// reach it through the registered handler set by SetSchedulerAccess.

// schedulerJobTypes are the job types the executor actually implements.
var schedulerJobTypes = []string{
	"scrape_thread", "verify_links",
	"cleanup_old_activities", "cleanup_old_activity_logs",
	"cleanup_old_console_logs", "cleanup_old_audit_logs",
	"database_backup", "cleanup_old_backups",
}

// schedulerAccess is set by the api layer (see SetSchedulerAccess) so tools
// can reach the running scheduler instance.
var schedulerAccess func() SchedulerAccess

// SchedulerAccess is the minimal surface tools need from the scheduler.
type SchedulerAccess interface {
	GetAllJobs() ([]*models.ScheduledJob, error)
	CreateJob(job *models.ScheduledJob) (*models.ScheduledJob, error)
	DeleteJob(id int64) error
	UpdateJob(job *models.ScheduledJob) error
	RunJobNow(id int64) error
}

// SetSchedulerAccess wires the scheduler singleton into the tool layer.
// Called once at startup from the api package.
func SetSchedulerAccess(fn func() SchedulerAccess) {
	schedulerAccess = fn
}

func getScheduler() SchedulerAccess {
	if schedulerAccess == nil {
		return nil
	}
	return schedulerAccess()
}

func (s *AICompanionService) registerSchedulerTools() {
	// ── list_scheduled_jobs ───────────────────────────────────────────────────
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "list_scheduled_jobs",
			Description: "List all scheduled jobs (scrapes, cleanups, backups) with schedule, enabled state, last/next run and success counts. Use when Brix asks what is scheduled or what runs automatically.",
			Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "required": []string{}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		sch := getScheduler()
		if sch == nil {
			return map[string]interface{}{"error": "scheduler not initialized"}, nil
		}
		jobs, err := sch.GetAllJobs()
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"count": len(jobs), "jobs": jobs}, nil
	})

	// ── create_scheduled_job ──────────────────────────────────────────────────
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name: "create_scheduled_job",
			Description: "Create a scheduled job. Valid job types: " +
				"scrape_thread (needs target_id = thread ID), verify_links, cleanup_old_activities, " +
				"cleanup_old_activity_logs, cleanup_old_console_logs, cleanup_old_audit_logs, " +
				"database_backup, cleanup_old_backups. " +
				"Schedule: interval (interval_minutes), cron (cron_expression), or once (run_at RFC3339). " +
				"Use when Brix asks to automate something recurring — confirm the job type and schedule with him before creating.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"job_type":        map[string]interface{}{"type": "string", "description": "One of the valid job types listed in the description"},
				"schedule_type":   map[string]interface{}{"type": "string", "enum": []string{"interval", "cron", "once"}, "description": "How the job repeats"},
				"interval_minutes": map[string]interface{}{"type": "number", "description": "For interval: minutes between runs (e.g. 1440 = daily)"},
				"cron_expression": map[string]interface{}{"type": "string", "description": "For cron: standard 5-field cron expression (e.g. '0 3 * * *' = daily 03:00)"},
				"run_at":          map[string]interface{}{"type": "string", "description": "For once: RFC3339 timestamp (e.g. 2026-11-01T03:00:00)"},
				"target_id":       map[string]interface{}{"type": "number", "description": "Target ID — required for scrape_thread (the thread ID)"},
				"enabled":         map[string]interface{}{"type": "boolean", "description": "Start enabled (default: true)"},
			}, "required": []string{"job_type", "schedule_type"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		sch := getScheduler()
		if sch == nil {
			return map[string]interface{}{"error": "scheduler not initialized"}, nil
		}

		jobType, _ := args["job_type"].(string)
		scheduleType, _ := args["schedule_type"].(string)

		valid := false
		for _, t := range schedulerJobTypes {
			if t == jobType {
				valid = true
				break
			}
		}
		if !valid {
			return map[string]interface{}{"error": "invalid job_type", "valid_types": schedulerJobTypes}, nil
		}
		if scheduleType != "interval" && scheduleType != "cron" && scheduleType != "once" {
			return map[string]interface{}{"error": "schedule_type must be interval, cron or once"}, nil
		}

		cfg := models.ScheduleConfig{}
		var nextRunAt *time.Time
		now := time.Now()

		switch scheduleType {
		case "interval":
			mins := intArg(args, "interval_minutes", 0)
			if mins <= 0 {
				return map[string]interface{}{"error": "interval_minutes required for interval schedule"}, nil
			}
			cfg.IntervalMinutes = mins
			next := now.Add(time.Duration(mins) * time.Minute)
			nextRunAt = &next
		case "cron":
			cron, _ := args["cron_expression"].(string)
			if cron == "" {
				return map[string]interface{}{"error": "cron_expression required for cron schedule"}, nil
			}
			cfg.CronExpression = cron
			// NULL next_run = due on next tick; executor recalculates after run
		case "once":
			runAtStr, _ := args["run_at"].(string)
			if runAtStr == "" {
				nextRunAt = &now // default: run immediately
			} else {
				t, err := time.Parse(time.RFC3339, runAtStr)
				if err != nil {
					return map[string]interface{}{"error": "run_at must be RFC3339 (e.g. 2026-11-01T03:00:00)"}, nil
				}
				cfg.RunAt = &t
				nextRunAt = &t
			}
		}

		cfgJSON, err := json.Marshal(cfg)
		if err != nil {
			return nil, err
		}

		enabled := true
		if v, ok := args["enabled"].(bool); ok {
			enabled = v
		}

		var targetID *int64
		if id := int64(floatArg(args, "target_id", 0)); id > 0 {
			targetID = &id
		}
		if jobType == "scrape_thread" && targetID == nil {
			return map[string]interface{}{"error": "target_id (thread ID) required for scrape_thread"}, nil
		}

		job := &models.ScheduledJob{
			JobType:        jobType,
			ScheduleType:   scheduleType,
			ScheduleConfig: string(cfgJSON),
			TargetType:     "global",
			TargetID:       targetID,
			Enabled:        enabled,
			NextRunAt:      nextRunAt,
		}

		created, err := sch.CreateJob(job)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"created": true,
			"job":     created,
			"note":    "Job will run automatically per its schedule. Use run_scheduled_job_now to test it immediately.",
		}, nil
	})

	// ── manage_scheduled_job ──────────────────────────────────────────────────
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "manage_scheduled_job",
			Description: "Manage an existing scheduled job: enable it, disable it, delete it, or run it immediately (run_now). Use run_now to test a job right away without waiting for its schedule.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"enable", "disable", "delete", "run_now"},
					"description": "Action to perform on the job",
				},
				"job_id": map[string]interface{}{"type": "number", "description": "ID of the job (from list_scheduled_jobs)"},
			}, "required": []string{"action", "job_id"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		sch := getScheduler()
		if sch == nil {
			return map[string]interface{}{"error": "scheduler not initialized"}, nil
		}
		action, _ := args["action"].(string)
		jobID := int64(floatArg(args, "job_id", 0))
		if jobID == 0 {
			return map[string]interface{}{"error": "job_id required"}, nil
		}

		switch action {
		case "delete":
			if err := sch.DeleteJob(jobID); err != nil {
				return nil, err
			}
			return map[string]interface{}{"action": action, "job_id": jobID, "ok": true}, nil
		case "run_now":
			if err := sch.RunJobNow(jobID); err != nil {
				return nil, err
			}
			return map[string]interface{}{"action": action, "job_id": jobID, "ok": true, "note": "Job started in background — check list_scheduled_jobs or the Activity page for results."}, nil
		case "enable", "disable":
			// Fetch, flip enabled, update
			jobs, err := sch.GetAllJobs()
			if err != nil {
				return nil, err
			}
			var target *models.ScheduledJob
			for _, j := range jobs {
				if j.ID == jobID {
					target = j
					break
				}
			}
			if target == nil {
				return map[string]interface{}{"error": fmt.Sprintf("job %d not found", jobID)}, nil
			}
			target.Enabled = action == "enable"
			if err := sch.UpdateJob(target); err != nil {
				return nil, err
			}
			return map[string]interface{}{"action": action, "job_id": jobID, "enabled": target.Enabled, "ok": true}, nil
		default:
			return map[string]interface{}{"error": "unknown action: " + action}, nil
		}
	})
}