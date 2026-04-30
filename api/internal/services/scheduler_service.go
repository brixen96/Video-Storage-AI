package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	appconfig "github.com/brixen96/video-storage-ai/internal/config"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// AutoUpdateHandler is a callback function that runs an auto-update check for a scraper type.
// It receives the scraper type and config JSON, and handles the actual scraping logic.
type AutoUpdateHandler func(scraperType string, configJSON string, maxPages int)

// SchedulerService handles automated job scheduling and execution
type SchedulerService struct {
	db                      *sql.DB
	ctx                     context.Context
	cancel                  context.CancelFunc
	scraperService          *ScraperService
	activityService         *ActivityService
	notificationService     *NotificationService
	linkVerificationService *LinkVerificationService
	autoUpdateHandler       AutoUpdateHandler
	running                 bool
	runningJobs             map[int64]*JobExecution
	mu                      sync.RWMutex
}

// JobExecution represents a running job
type JobExecution struct {
	JobID     int64
	StartedAt time.Time
	Cancel    context.CancelFunc
}

// NewSchedulerService creates a new scheduler service
func NewSchedulerService(db *sql.DB, scraperService *ScraperService, activityService *ActivityService) *SchedulerService {
	ctx, cancel := context.WithCancel(context.Background())
	notificationService := NewNotificationService(db)
	return &SchedulerService{
		db:                      db,
		ctx:                     ctx,
		cancel:                  cancel,
		scraperService:          scraperService,
		activityService:         activityService,
		notificationService:     notificationService,
		linkVerificationService: NewLinkVerificationService(db, activityService),
		running:                 false,
		runningJobs:             make(map[int64]*JobExecution),
	}
}

// SetAutoUpdateHandler sets the callback function for running auto-update checks.
// This allows the api package to inject the full implementation that has access to all scraper services.
func (s *SchedulerService) SetAutoUpdateHandler(handler AutoUpdateHandler) {
	s.autoUpdateHandler = handler
}

// Start starts the scheduler
func (s *SchedulerService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler already running")
	}

	s.running = true
	go s.schedulerLoop()
	log.Println("⏰ Scheduler started")
	return nil
}

// Stop stops the scheduler
func (s *SchedulerService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("scheduler not running")
	}

	s.running = false
	s.cancel()

	// Wait for running jobs to complete
	for jobID := range s.runningJobs {
		log.Printf("⏳ Waiting for job %d to complete...\n", jobID)
	}

	log.Println("⏰ Scheduler stopped")
	return nil
}

// schedulerLoop is the main scheduler loop
func (s *SchedulerService) schedulerLoop() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkAndExecuteJobs()
			s.checkAndRunAutoUpdates() // Also check for auto-updates
		}
	}
}

// checkAndRunAutoUpdates checks for auto-updates that are due and runs them
func (s *SchedulerService) checkAndRunAutoUpdates() {
	// Get all enabled auto-update configs that are due
	rows, err := s.db.Query(`
		SELECT id, scraper_type, interval_minutes, max_pages_to_check, config_json, last_checked_at
		FROM scraper_auto_update_configs
		WHERE enabled = 1
		AND (
			last_checked_at IS NULL
			OR datetime(last_checked_at, '+' || interval_minutes || ' minutes') <= datetime('now')
		)
	`)
	if err != nil {
		log.Printf("Error checking for due auto-updates: %v\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var scraperType string
		var intervalMinutes int
		var maxPagesToCheck int
		var configJSON string
		var lastCheckedAt sql.NullTime

		err := rows.Scan(&id, &scraperType, &intervalMinutes, &maxPagesToCheck, &configJSON, &lastCheckedAt)
		if err != nil {
			continue
		}

		// Check if this scraper is already running an auto-update
		s.mu.RLock()
		runningKey := fmt.Sprintf("auto_update_%s", scraperType)
		_, isRunning := s.runningJobs[int64(hash(runningKey))]
		s.mu.RUnlock()

		if isRunning {
			continue
		}

		// Run auto-update in background
		go s.runAutoUpdate(id, scraperType, intervalMinutes, maxPagesToCheck, configJSON)
	}
}

// hash creates a simple hash for tracking running auto-updates
func hash(s string) int64 {
	var h int64
	for _, c := range s {
		h = 31*h + int64(c)
	}
	return h
}

// runAutoUpdate runs an auto-update check for a scraper by delegating to the autoUpdateHandler.
// The handler is set by the api package which has access to all scraper service singletons.
func (s *SchedulerService) runAutoUpdate(configID int64, scraperType string, maxPages int, _ int, configJSON string) {
	runningKey := fmt.Sprintf("auto_update_%s", scraperType)
	keyHash := hash(runningKey)

	// Track as running
	s.mu.Lock()
	s.runningJobs[keyHash] = &JobExecution{
		JobID:     keyHash,
		StartedAt: time.Now(),
	}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.runningJobs, keyHash)
		s.mu.Unlock()
	}()

	log.Printf("🔄 Running scheduled auto-update check for %s\n", scraperType)

	if s.autoUpdateHandler != nil {
		// Delegate to the full implementation in the api package
		s.autoUpdateHandler(scraperType, configJSON, maxPages)
	} else {
		log.Printf("⚠️ No auto-update handler set for scheduler - skipping %s\n", scraperType)
	}

	log.Printf("✅ Scheduled auto-update check completed for %s\n", scraperType)
}

// checkAndExecuteJobs checks for jobs that need to run
func (s *SchedulerService) checkAndExecuteJobs() {
	jobs, err := s.GetDueJobs()
	if err != nil {
		log.Printf("Error getting due jobs: %v\n", err)
		return
	}

	for _, job := range jobs {
		// Check if job is already running
		s.mu.RLock()
		_, isRunning := s.runningJobs[job.ID]
		s.mu.RUnlock()

		if isRunning {
			log.Printf("⏭️ Job %d (%s) already running, skipping\n", job.ID, job.JobType)
			continue
		}

		// Execute job in goroutine
		go s.executeJob(job)
	}
}

// executeJob executes a scheduled job
func (s *SchedulerService) executeJob(job *models.ScheduledJob) {
	jobCtx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	// Track running job
	s.mu.Lock()
	s.runningJobs[job.ID] = &JobExecution{
		JobID:     job.ID,
		StartedAt: time.Now(),
		Cancel:    cancel,
	}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.runningJobs, job.ID)
		s.mu.Unlock()
	}()

	log.Printf("▶️ Executing job %d: %s\n", job.ID, job.JobType)

	// Create execution history record
	historyID, err := s.createExecutionHistory(job.ID)
	if err != nil {
		log.Printf("Failed to create execution history: %v\n", err)
		return
	}

	startTime := time.Now()
	var result *models.JobResult

	// Execute based on job type
	switch job.JobType {
	case "scrape_thread":
		result = s.executeScrapeThread(jobCtx, job)
	case "verify_links":
		result = s.executeVerifyLinks(jobCtx, job)
	case "cleanup_old_activities":
		result = s.executeCleanupOldActivities(jobCtx, job)
	case "cleanup_old_activity_logs":
		result = s.executeCleanupOldActivityLogs(jobCtx, job)
	case "cleanup_old_console_logs":
		result = s.executeCleanupOldConsoleLogs(jobCtx, job)
	case "cleanup_old_audit_logs":
		result = s.executeCleanupOldAuditLogs(jobCtx, job)
	case "database_backup":
		result = s.executeDatabaseBackup(jobCtx, job)
	case "cleanup_old_backups":
		result = s.executeCleanupOldBackups(jobCtx, job)
	default:
		result = &models.JobResult{
			Success: false,
			Message: fmt.Sprintf("Unknown job type: %s", job.JobType),
		}
	}

	duration := time.Since(startTime)

	// Update execution history
	s.completeExecutionHistory(historyID, result, duration)

	// Update job
	s.updateJobAfterExecution(job, result.Success)

	// Send notification
	if s.notificationService != nil {
		err := s.notificationService.NotifyJobCompleted(job.ID, job.JobType, result.Success, result.Message)
		if err != nil {
			log.Printf("Failed to send job notification: %v\n", err)
		}
	}

	if result.Success {
		log.Printf("✅ Job %d completed successfully: %s\n", job.ID, result.Message)
	} else {
		log.Printf("❌ Job %d failed: %s\n", job.ID, result.Message)
	}
}

// executeScrapeThread executes a thread scraping job
func (s *SchedulerService) executeScrapeThread(ctx context.Context, job *models.ScheduledJob) *models.JobResult {
	if job.TargetID == nil {
		return &models.JobResult{
			Success: false,
			Message: "No target thread ID specified",
		}
	}

	// Get thread
	thread, err := s.scraperService.GetThreadByID(*job.TargetID)
	if err != nil {
		return &models.JobResult{
			Success: false,
			Message: fmt.Sprintf("Failed to get thread: %v", err),
		}
	}

	// Start scraping using ScrapeThreadComplete
	err = s.scraperService.ScrapeThreadComplete(thread.URL)
	if err != nil {
		return &models.JobResult{
			Success: false,
			Message: fmt.Sprintf("Scraping failed: %v", err),
		}
	}

	return &models.JobResult{
		Success: true,
		Message: fmt.Sprintf("Successfully scraped thread: %s", thread.Title),
	}
}

// executeVerifyLinks executes a link verification job
func (s *SchedulerService) executeVerifyLinks(ctx context.Context, job *models.ScheduledJob) *models.JobResult {
	if job.TargetID == nil {
		return &models.JobResult{
			Success: false,
			Message: "No target thread ID specified",
		}
	}

	threadID := *job.TargetID
	if err := s.linkVerificationService.VerifyThreadLinks(threadID); err != nil {
		return &models.JobResult{
			Success: false,
			Message: fmt.Sprintf("Link verification failed for thread %d: %v", threadID, err),
		}
	}

	return &models.JobResult{
		Success: true,
		Message: fmt.Sprintf("Link verification completed for thread %d", threadID),
	}
}

// executeCleanupOldActivities cleans up old activity records
func (s *SchedulerService) executeCleanupOldActivities(ctx context.Context, job *models.ScheduledJob) *models.JobResult {
	// Parse config
	var config models.ScheduleConfig
	if err := json.Unmarshal([]byte(job.ScheduleConfig), &config); err != nil {
		return &models.JobResult{
			Success: false,
			Message: fmt.Sprintf("Failed to parse config: %v", err),
		}
	}

	daysToKeep := 30
	if config.TimeoutMinutes > 0 {
		daysToKeep = config.TimeoutMinutes // Reuse field for days
	}

	result, err := s.db.Exec(`
		DELETE FROM activities
		WHERE status IN ('completed', 'failed')
		AND created_at < datetime('now', ?)
	`, fmt.Sprintf("-%d days", daysToKeep))

	if err != nil {
		return &models.JobResult{
			Success: false,
			Message: fmt.Sprintf("Cleanup failed: %v", err),
		}
	}

	rowsAffected, _ := result.RowsAffected()
	return &models.JobResult{
		Success:        true,
		ItemsProcessed: int(rowsAffected),
		Message:        fmt.Sprintf("Cleaned up %d old activity records", rowsAffected),
	}
}

// executeCleanupOldActivityLogs cleans up old activity_logs records (completed/failed)
func (s *SchedulerService) executeCleanupOldActivityLogs(ctx context.Context, job *models.ScheduledJob) *models.JobResult {
	var config models.ScheduleConfig
	if err := json.Unmarshal([]byte(job.ScheduleConfig), &config); err != nil {
		return &models.JobResult{
			Success: false,
			Message: fmt.Sprintf("Failed to parse config: %v", err),
		}
	}

	daysToKeep := 14
	if config.TimeoutMinutes > 0 {
		daysToKeep = config.TimeoutMinutes
	}

	result, err := s.db.Exec(`
		DELETE FROM activity_logs
		WHERE status IN ('completed', 'failed')
		AND created_at < datetime('now', ?)
	`, fmt.Sprintf("-%d days", daysToKeep))

	if err != nil {
		return &models.JobResult{
			Success: false,
			Message: fmt.Sprintf("Cleanup failed: %v", err),
		}
	}

	rowsAffected, _ := result.RowsAffected()
	return &models.JobResult{
		Success:        true,
		ItemsProcessed: int(rowsAffected),
		Message:        fmt.Sprintf("Cleaned up %d old activity log records (kept %d days)", rowsAffected, daysToKeep),
	}
}

// executeCleanupOldConsoleLogs cleans up old console_logs records
func (s *SchedulerService) executeCleanupOldConsoleLogs(ctx context.Context, job *models.ScheduledJob) *models.JobResult {
	var config models.ScheduleConfig
	if err := json.Unmarshal([]byte(job.ScheduleConfig), &config); err != nil {
		return &models.JobResult{
			Success: false,
			Message: fmt.Sprintf("Failed to parse config: %v", err),
		}
	}

	daysToKeep := 7
	if config.TimeoutMinutes > 0 {
		daysToKeep = config.TimeoutMinutes
	}

	result, err := s.db.Exec(`
		DELETE FROM console_logs
		WHERE created_at < datetime('now', ?)
	`, fmt.Sprintf("-%d days", daysToKeep))

	if err != nil {
		return &models.JobResult{
			Success: false,
			Message: fmt.Sprintf("Cleanup failed: %v", err),
		}
	}

	rowsAffected, _ := result.RowsAffected()
	return &models.JobResult{
		Success:        true,
		ItemsProcessed: int(rowsAffected),
		Message:        fmt.Sprintf("Cleaned up %d old console log records (kept %d days)", rowsAffected, daysToKeep),
	}
}

// executeCleanupOldAuditLogs cleans up old audit logs
func (s *SchedulerService) executeCleanupOldAuditLogs(ctx context.Context, job *models.ScheduledJob) *models.JobResult {
	var config models.ScheduleConfig
	if err := json.Unmarshal([]byte(job.ScheduleConfig), &config); err != nil {
		return &models.JobResult{
			Success: false,
			Message: fmt.Sprintf("Failed to parse config: %v", err),
		}
	}

	daysToKeep := 90
	if config.TimeoutMinutes > 0 {
		daysToKeep = config.TimeoutMinutes
	}

	result, err := s.db.Exec(`
		DELETE FROM ai_audit_logs
		WHERE created_at < datetime('now', ?)
	`, fmt.Sprintf("-%d days", daysToKeep))

	if err != nil {
		return &models.JobResult{
			Success: false,
			Message: fmt.Sprintf("Cleanup failed: %v", err),
		}
	}

	rowsAffected, _ := result.RowsAffected()
	return &models.JobResult{
		Success:        true,
		ItemsProcessed: int(rowsAffected),
		Message:        fmt.Sprintf("Cleaned up %d old audit logs", rowsAffected),
	}
}

// executeDatabaseBackup creates a database backup
func (s *SchedulerService) executeDatabaseBackup(ctx context.Context, job *models.ScheduledJob) *models.JobResult {
	backupSvc := NewBackupService(filepath.Join(appconfig.DataDir(), "backups"))

	backup, err := backupSvc.CreateBackup("automatic")
	if err != nil {
		return &models.JobResult{
			Success: false,
			Message: fmt.Sprintf("Backup failed: %v", err),
		}
	}

	return &models.JobResult{
		Success:        true,
		ItemsProcessed: 1,
		Message:        fmt.Sprintf("Backup created: %s (%.2f MB)", backup.Filename, float64(backup.Size)/1024/1024),
	}
}

// executeCleanupOldBackups cleans up old automatic backups
func (s *SchedulerService) executeCleanupOldBackups(ctx context.Context, job *models.ScheduledJob) *models.JobResult {
	var config models.ScheduleConfig
	if err := json.Unmarshal([]byte(job.ScheduleConfig), &config); err != nil {
		return &models.JobResult{
			Success: false,
			Message: fmt.Sprintf("Failed to parse config: %v", err),
		}
	}

	retentionDays := 30
	keepMinimum := 3
	if config.TimeoutMinutes > 0 {
		retentionDays = config.TimeoutMinutes
	}

	backupSvc := NewBackupService(filepath.Join(appconfig.DataDir(), "backups"))
	deletedCount, err := backupSvc.CleanupOldBackups(retentionDays, keepMinimum)
	if err != nil {
		return &models.JobResult{
			Success: false,
			Message: fmt.Sprintf("Cleanup failed: %v", err),
		}
	}

	return &models.JobResult{
		Success:        true,
		ItemsProcessed: deletedCount,
		Message:        fmt.Sprintf("Cleaned up %d old backups (retention: %d days, minimum kept: %d)", deletedCount, retentionDays, keepMinimum),
	}
}

// createExecutionHistory creates a new execution history record
func (s *SchedulerService) createExecutionHistory(jobID int64) (int64, error) {
	result, err := s.db.Exec(`
		INSERT INTO job_execution_history (job_id, started_at, status)
		VALUES (?, ?, 'running')
	`, jobID, time.Now())

	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// completeExecutionHistory updates the execution history record
func (s *SchedulerService) completeExecutionHistory(historyID int64, result *models.JobResult, duration time.Duration) {
	resultJSON, _ := json.Marshal(result)
	completedAt := time.Now()
	status := "completed"
	errorMsg := ""

	if !result.Success {
		status = "failed"
		errorMsg = result.Message
	}

	_, err := s.db.Exec(`
		UPDATE job_execution_history
		SET completed_at = ?,
		    status = ?,
		    result_data = ?,
		    error_message = ?,
		    duration_ms = ?
		WHERE id = ?
	`, completedAt, status, string(resultJSON), errorMsg, duration.Milliseconds(), historyID)

	if err != nil {
		log.Printf("Failed to update execution history: %v\n", err)
	}
}

// updateJobAfterExecution updates job statistics after execution
func (s *SchedulerService) updateJobAfterExecution(job *models.ScheduledJob, success bool) {
	now := time.Now()
	var nextRunAt *time.Time

	// Calculate next run time based on schedule type
	var config models.ScheduleConfig
	if err := json.Unmarshal([]byte(job.ScheduleConfig), &config); err == nil {
		switch job.ScheduleType {
		case "interval":
			if config.IntervalMinutes > 0 {
				next := now.Add(time.Duration(config.IntervalMinutes) * time.Minute)
				nextRunAt = &next
			}
		case "once":
			// Disable one-time jobs after execution
			_, _ = s.db.Exec(`UPDATE scheduled_jobs SET enabled = 0 WHERE id = ?`, job.ID)
		}
	}

	successIncrement := 0
	failureIncrement := 0
	if success {
		successIncrement = 1
	} else {
		failureIncrement = 1
	}

	_, err := s.db.Exec(`
		UPDATE scheduled_jobs
		SET last_run_at = ?,
		    next_run_at = ?,
		    run_count = run_count + 1,
		    success_count = success_count + ?,
		    failure_count = failure_count + ?,
		    updated_at = ?
		WHERE id = ?
	`, now, nextRunAt, successIncrement, failureIncrement, now, job.ID)

	if err != nil {
		log.Printf("Failed to update job after execution: %v\n", err)
	}
}

// GetDueJobs returns jobs that are due to run
func (s *SchedulerService) GetDueJobs() ([]*models.ScheduledJob, error) {
	rows, err := s.db.Query(`
		SELECT
			id, job_type, schedule_type, schedule_config, target_type, target_id,
			enabled, last_run_at, next_run_at, run_count, success_count, failure_count,
			created_at, updated_at
		FROM scheduled_jobs
		WHERE enabled = 1
		AND (next_run_at IS NULL OR next_run_at <= ?)
		ORDER BY next_run_at ASC
	`, time.Now())

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*models.ScheduledJob
	for rows.Next() {
		var job models.ScheduledJob
		if err := rows.Scan(
			&job.ID, &job.JobType, &job.ScheduleType, &job.ScheduleConfig, &job.TargetType, &job.TargetID,
			&job.Enabled, &job.LastRunAt, &job.NextRunAt, &job.RunCount, &job.SuccessCount, &job.FailureCount,
			&job.CreatedAt, &job.UpdatedAt,
		); err != nil {
			continue
		}
		jobs = append(jobs, &job)
	}

	return jobs, nil
}

// CreateJob creates a new scheduled job
func (s *SchedulerService) CreateJob(job *models.ScheduledJob) (*models.ScheduledJob, error) {
	configJSON, err := json.Marshal(job.ScheduleConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	now := time.Now()
	result, err := s.db.Exec(`
		INSERT INTO scheduled_jobs (
			job_type, schedule_type, schedule_config, target_type, target_id,
			enabled, next_run_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, job.JobType, job.ScheduleType, string(configJSON), job.TargetType, job.TargetID,
		job.Enabled, job.NextRunAt, now, now)

	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return s.GetJobByID(id)
}

// GetJobByID retrieves a job by ID
func (s *SchedulerService) GetJobByID(id int64) (*models.ScheduledJob, error) {
	var job models.ScheduledJob
	err := s.db.QueryRow(`
		SELECT
			id, job_type, schedule_type, schedule_config, target_type, target_id,
			enabled, last_run_at, next_run_at, run_count, success_count, failure_count,
			created_at, updated_at
		FROM scheduled_jobs
		WHERE id = ?
	`, id).Scan(
		&job.ID, &job.JobType, &job.ScheduleType, &job.ScheduleConfig, &job.TargetType, &job.TargetID,
		&job.Enabled, &job.LastRunAt, &job.NextRunAt, &job.RunCount, &job.SuccessCount, &job.FailureCount,
		&job.CreatedAt, &job.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &job, nil
}

// GetAllJobs retrieves all jobs
func (s *SchedulerService) GetAllJobs() ([]*models.ScheduledJob, error) {
	rows, err := s.db.Query(`
		SELECT
			id, job_type, schedule_type, schedule_config, target_type, target_id,
			enabled, last_run_at, next_run_at, run_count, success_count, failure_count,
			created_at, updated_at
		FROM scheduled_jobs
		ORDER BY created_at DESC
	`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*models.ScheduledJob
	for rows.Next() {
		var job models.ScheduledJob
		if err := rows.Scan(
			&job.ID, &job.JobType, &job.ScheduleType, &job.ScheduleConfig, &job.TargetType, &job.TargetID,
			&job.Enabled, &job.LastRunAt, &job.NextRunAt, &job.RunCount, &job.SuccessCount, &job.FailureCount,
			&job.CreatedAt, &job.UpdatedAt,
		); err != nil {
			continue
		}
		jobs = append(jobs, &job)
	}

	return jobs, nil
}

// UpdateJob updates a job
func (s *SchedulerService) UpdateJob(job *models.ScheduledJob) error {
	now := time.Now()
	_, err := s.db.Exec(`
		UPDATE scheduled_jobs
		SET job_type = ?,
		    schedule_type = ?,
		    schedule_config = ?,
		    target_type = ?,
		    target_id = ?,
		    enabled = ?,
		    next_run_at = ?,
		    updated_at = ?
		WHERE id = ?
	`, job.JobType, job.ScheduleType, job.ScheduleConfig, job.TargetType, job.TargetID,
		job.Enabled, job.NextRunAt, now, job.ID)

	return err
}

// DeleteJob deletes a job
func (s *SchedulerService) DeleteJob(id int64) error {
	// Cancel if running
	s.mu.Lock()
	if exec, exists := s.runningJobs[id]; exists {
		exec.Cancel()
	}
	s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM scheduled_jobs WHERE id = ?`, id)
	return err
}

// GetExecutionHistory retrieves execution history for a job
func (s *SchedulerService) GetExecutionHistory(jobID int64, limit int) ([]*models.JobExecutionHistory, error) {
	rows, err := s.db.Query(`
		SELECT
			id, job_id, started_at, completed_at, status, result_data, error_message, duration_ms
		FROM job_execution_history
		WHERE job_id = ?
		ORDER BY started_at DESC
		LIMIT ?
	`, jobID, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []*models.JobExecutionHistory
	for rows.Next() {
		var h models.JobExecutionHistory
		if err := rows.Scan(
			&h.ID, &h.JobID, &h.StartedAt, &h.CompletedAt, &h.Status, &h.ResultData, &h.ErrorMessage, &h.DurationMs,
		); err != nil {
			continue
		}
		history = append(history, &h)
	}

	return history, nil
}
