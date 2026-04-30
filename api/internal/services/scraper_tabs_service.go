package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ScraperTabsService handles scraper tab configurations
type ScraperTabsService struct {
	db *sql.DB
}

// NewScraperTabsService creates a new scraper tabs service
func NewScraperTabsService(db *sql.DB) *ScraperTabsService {
	svc := &ScraperTabsService{db: db}
	// Reset any jobs that were left in running/pending state from a previous session (e.g. service crash/restart)
	db.Exec(`UPDATE scraper_tab_jobs SET status = 'failed', completed_at = CURRENT_TIMESTAMP WHERE status IN ('running', 'pending')`)
	return svc
}

// ScraperTab represents a saved scraper configuration
type ScraperTab struct {
	ID            int64                  `json:"id"`
	Name          string                 `json:"name"`
	TabType       string                 `json:"tabType"`
	BaseURL       string                 `json:"baseUrl"`
	SiteType      string                 `json:"siteType"`
	Icon          string                 `json:"icon"`
	Color         string                 `json:"color"`
	ContentType   string                 `json:"contentType"` // regular, zoo, 3d, etc.
	ScrapeOptions map[string]interface{} `json:"scrapeOptions"`
	IsActive      bool                   `json:"isActive"`
	Position      int                    `json:"position"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
}

// ScraperJob represents a background scraping job
type ScraperJob struct {
	ID             int64     `json:"id"`
	TabID          *int64    `json:"tabId,omitempty"`    // Nullable for source-specific jobs (like PornHub)
	JobType        string    `json:"jobType"`            // "tab", "pornhub", etc.
	Status         string    `json:"status"`             // pending, running, completed, failed, cancelled
	URL            string    `json:"url"`
	TotalPages     int       `json:"totalPages"`
	PagesCompleted int       `json:"pagesCompleted"`
	VideosFound    int       `json:"videosFound"`
	VideosSaved    int       `json:"videosSaved"`
	ErrorsCount    int       `json:"errorsCount"`
	ErrorMessages  []string  `json:"errorMessages"`
	StartedAt      *time.Time `json:"startedAt,omitempty"`
	CompletedAt    *time.Time `json:"completedAt,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// CreateScraperTab creates a new scraper tab configuration
func (s *ScraperTabsService) CreateScraperTab(tab *ScraperTab) (*ScraperTab, error) {
	// Get the highest position
	var maxPosition int
	err := s.db.QueryRow("SELECT COALESCE(MAX(position), -1) FROM scraper_tabs").Scan(&maxPosition)
	if err != nil {
		return nil, fmt.Errorf("failed to get max position: %w", err)
	}
	tab.Position = maxPosition + 1

	// Serialize scrape options
	optionsJSON, err := json.Marshal(tab.ScrapeOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal scrape options: %w", err)
	}

	// Default content type if not set
	if tab.ContentType == "" {
		tab.ContentType = "regular"
	}

	query := `INSERT INTO scraper_tabs (name, tab_type, base_url, site_type, icon, color, content_type, scrape_options, is_active, position, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`

	result, err := s.db.Exec(query, tab.Name, tab.TabType, tab.BaseURL, tab.SiteType, tab.Icon, tab.Color, tab.ContentType, string(optionsJSON), tab.IsActive, tab.Position)
	if err != nil {
		return nil, fmt.Errorf("failed to create scraper tab: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get insert id: %w", err)
	}

	tab.ID = id
	tab.CreatedAt = time.Now()
	tab.UpdatedAt = time.Now()

	return tab, nil
}

// GetScraperTabs retrieves all scraper tabs
func (s *ScraperTabsService) GetScraperTabs() ([]*ScraperTab, error) {
	query := `SELECT id, name, tab_type, base_url, site_type, icon, color, COALESCE(content_type, 'regular'), scrape_options, is_active, position, created_at, updated_at
			  FROM scraper_tabs
			  WHERE is_active = 1
			  ORDER BY position ASC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query scraper tabs: %w", err)
	}
	defer rows.Close()

	tabs := make([]*ScraperTab, 0)
	for rows.Next() {
		tab := &ScraperTab{}
		var optionsJSON string

		err := rows.Scan(&tab.ID, &tab.Name, &tab.TabType, &tab.BaseURL, &tab.SiteType, &tab.Icon, &tab.Color, &tab.ContentType, &optionsJSON, &tab.IsActive, &tab.Position, &tab.CreatedAt, &tab.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan scraper tab: %w", err)
		}

		// Deserialize scrape options
		if err := json.Unmarshal([]byte(optionsJSON), &tab.ScrapeOptions); err != nil {
			tab.ScrapeOptions = make(map[string]interface{})
		}

		tabs = append(tabs, tab)
	}

	return tabs, nil
}

// GetScraperTabByID retrieves a single scraper tab by ID
func (s *ScraperTabsService) GetScraperTabByID(id int64) (*ScraperTab, error) {
	query := `SELECT id, name, tab_type, base_url, site_type, icon, color, COALESCE(content_type, 'regular'), scrape_options, is_active, position, created_at, updated_at
			  FROM scraper_tabs
			  WHERE id = ?`

	tab := &ScraperTab{}
	var optionsJSON string

	err := s.db.QueryRow(query, id).Scan(&tab.ID, &tab.Name, &tab.TabType, &tab.BaseURL, &tab.SiteType, &tab.Icon, &tab.Color, &tab.ContentType, &optionsJSON, &tab.IsActive, &tab.Position, &tab.CreatedAt, &tab.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("scraper tab not found")
		}
		return nil, fmt.Errorf("failed to get scraper tab: %w", err)
	}

	// Deserialize scrape options
	if err := json.Unmarshal([]byte(optionsJSON), &tab.ScrapeOptions); err != nil {
		tab.ScrapeOptions = make(map[string]interface{})
	}

	return tab, nil
}

// UpdateScraperTab updates an existing scraper tab
func (s *ScraperTabsService) UpdateScraperTab(id int64, tab *ScraperTab) (*ScraperTab, error) {
	// Serialize scrape options
	optionsJSON, err := json.Marshal(tab.ScrapeOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal scrape options: %w", err)
	}

	// Default content type if not set
	if tab.ContentType == "" {
		tab.ContentType = "regular"
	}

	query := `UPDATE scraper_tabs
			  SET name = ?, tab_type = ?, base_url = ?, site_type = ?, icon = ?, color = ?, content_type = ?, scrape_options = ?, is_active = ?, position = ?, updated_at = CURRENT_TIMESTAMP
			  WHERE id = ?`

	_, err = s.db.Exec(query, tab.Name, tab.TabType, tab.BaseURL, tab.SiteType, tab.Icon, tab.Color, tab.ContentType, string(optionsJSON), tab.IsActive, tab.Position, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update scraper tab: %w", err)
	}

	return s.GetScraperTabByID(id)
}

// DeleteScraperTab deletes a scraper tab
func (s *ScraperTabsService) DeleteScraperTab(id int64) error {
	_, err := s.db.Exec("DELETE FROM scraper_tabs WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete scraper tab: %w", err)
	}
	return nil
}

// CreateScraperJob creates a new background scraping job
func (s *ScraperTabsService) CreateScraperJob(job *ScraperJob) (*ScraperJob, error) {
	errorsJSON, _ := json.Marshal(job.ErrorMessages)

	// Default job type to "tab" if not specified
	if job.JobType == "" {
		job.JobType = "tab"
	}

	query := `INSERT INTO scraper_tab_jobs (tab_id, job_type, status, url, total_pages, pages_completed, videos_found, videos_saved, errors_count, error_messages, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`

	result, err := s.db.Exec(query, job.TabID, job.JobType, job.Status, job.URL, job.TotalPages, job.PagesCompleted, job.VideosFound, job.VideosSaved, job.ErrorsCount, string(errorsJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create scraper job: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get insert id: %w", err)
	}

	job.ID = id
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()

	return job, nil
}

// UpdateScraperJob updates a scraper job's progress
func (s *ScraperTabsService) UpdateScraperJob(id int64, job *ScraperJob) error {
	errorsJSON, _ := json.Marshal(job.ErrorMessages)

	query := `UPDATE scraper_tab_jobs
			  SET status = ?, total_pages = ?, pages_completed = ?, videos_found = ?, videos_saved = ?, errors_count = ?, error_messages = ?, started_at = ?, completed_at = ?, updated_at = CURRENT_TIMESTAMP
			  WHERE id = ?`

	_, err := s.db.Exec(query, job.Status, job.TotalPages, job.PagesCompleted, job.VideosFound, job.VideosSaved, job.ErrorsCount, string(errorsJSON), job.StartedAt, job.CompletedAt, id)
	if err != nil {
		return fmt.Errorf("failed to update scraper job: %w", err)
	}

	return nil
}

// GetScraperJobsByTabID gets all jobs for a specific tab
func (s *ScraperTabsService) GetScraperJobsByTabID(tabID int64) ([]*ScraperJob, error) {
	query := `SELECT id, tab_id, COALESCE(job_type, 'tab'), status, url, total_pages, pages_completed, videos_found, videos_saved, errors_count, error_messages, started_at, completed_at, created_at, updated_at
			  FROM scraper_tab_jobs
			  WHERE tab_id = ?
			  ORDER BY created_at DESC`

	rows, err := s.db.Query(query, tabID)
	if err != nil {
		return nil, fmt.Errorf("failed to query scraper jobs: %w", err)
	}
	defer rows.Close()

	jobs := make([]*ScraperJob, 0)
	for rows.Next() {
		job := &ScraperJob{}
		var errorsJSON string

		err := rows.Scan(&job.ID, &job.TabID, &job.JobType, &job.Status, &job.URL, &job.TotalPages, &job.PagesCompleted, &job.VideosFound, &job.VideosSaved, &job.ErrorsCount, &errorsJSON, &job.StartedAt, &job.CompletedAt, &job.CreatedAt, &job.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan scraper job: %w", err)
		}

		// Deserialize error messages
		if err := json.Unmarshal([]byte(errorsJSON), &job.ErrorMessages); err != nil {
			job.ErrorMessages = []string{}
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

// GetActiveScraperJobs gets all currently running jobs
func (s *ScraperTabsService) GetActiveScraperJobs() ([]*ScraperJob, error) {
	query := `SELECT id, tab_id, COALESCE(job_type, 'tab'), status, url, total_pages, pages_completed, videos_found, videos_saved, errors_count, error_messages, started_at, completed_at, created_at, updated_at
			  FROM scraper_tab_jobs
			  WHERE status IN ('pending', 'running')
			  ORDER BY created_at ASC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active scraper jobs: %w", err)
	}
	defer rows.Close()

	jobs := make([]*ScraperJob, 0)
	for rows.Next() {
		job := &ScraperJob{}
		var errorsJSON string

		err := rows.Scan(&job.ID, &job.TabID, &job.JobType, &job.Status, &job.URL, &job.TotalPages, &job.PagesCompleted, &job.VideosFound, &job.VideosSaved, &job.ErrorsCount, &errorsJSON, &job.StartedAt, &job.CompletedAt, &job.CreatedAt, &job.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan scraper job: %w", err)
		}

		// Deserialize error messages
		if err := json.Unmarshal([]byte(errorsJSON), &job.ErrorMessages); err != nil {
			job.ErrorMessages = []string{}
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

// GetScraperJobByID retrieves a single scraper job by ID
func (s *ScraperTabsService) GetScraperJobByID(id int64) (*ScraperJob, error) {
	query := `SELECT id, tab_id, COALESCE(job_type, 'tab'), status, url, total_pages, pages_completed, videos_found, videos_saved, errors_count, error_messages, started_at, completed_at, created_at, updated_at
			  FROM scraper_tab_jobs
			  WHERE id = ?`

	job := &ScraperJob{}
	var errorsJSON string

	err := s.db.QueryRow(query, id).Scan(&job.ID, &job.TabID, &job.JobType, &job.Status, &job.URL, &job.TotalPages, &job.PagesCompleted, &job.VideosFound, &job.VideosSaved, &job.ErrorsCount, &errorsJSON, &job.StartedAt, &job.CompletedAt, &job.CreatedAt, &job.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("scraper job not found")
		}
		return nil, fmt.Errorf("failed to get scraper job: %w", err)
	}

	// Deserialize error messages
	if err := json.Unmarshal([]byte(errorsJSON), &job.ErrorMessages); err != nil {
		job.ErrorMessages = []string{}
	}

	return job, nil
}
