package models

import (
	"encoding/json"
	"time"
)

// ScrapedThread represents a forum thread
type ScrapedThread struct {
	ID              int64                  `json:"id" db:"id"`
	ExternalID      string                 `json:"external_id" db:"external_id"`         // Thread ID from source site
	Source          string                 `json:"source" db:"source"`                   // e.g., "simpcity"
	Title           string                 `json:"title" db:"title"`
	URL             string                 `json:"url" db:"url"`
	Author          string                 `json:"author" db:"author"`
	Category        string                 `json:"category" db:"category"`               // Forum category
	ViewCount       int                    `json:"view_count" db:"view_count"`
	ReplyCount      int                    `json:"reply_count" db:"reply_count"`
	PostCount       int                    `json:"post_count" db:"post_count"`           // Total posts in thread
	DownloadCount   int                    `json:"download_count" db:"download_count"`   // Number of download links found
	Metadata        string                 `json:"-" db:"metadata"`                      // JSON string
	MetadataObj     *ScrapedThreadMetadata `json:"metadata,omitempty" db:"-"`
	FirstScrapedAt  time.Time              `json:"first_scraped_at" db:"first_scraped_at"`
	LastScrapedAt   time.Time              `json:"last_scraped_at" db:"last_scraped_at"`
	LastUpdatedAt   time.Time              `json:"last_updated_at" db:"last_updated_at"` // Thread last updated on source
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
}

// ScrapedThreadMetadata represents additional thread information
type ScrapedThreadMetadata struct {
	Tags            []string               `json:"tags,omitempty"`
	FirstPostDate   string                 `json:"first_post_date,omitempty"`
	LastPostDate    string                 `json:"last_post_date,omitempty"`
	IsPinned        bool                   `json:"is_pinned,omitempty"`
	IsLocked        bool                   `json:"is_locked,omitempty"`
	ThumbnailURL    string                 `json:"thumbnail_url,omitempty"`     // Primary thumbnail
	ThumbnailURLs   []string               `json:"thumbnail_urls,omitempty"`    // All available thumbnails (fallback support)
	PerformerNames  []string               `json:"performer_names,omitempty"`  // Extracted from title/content
	StudioNames     []string               `json:"studio_names,omitempty"`     // Extracted from title/content
	RawData         map[string]interface{} `json:"raw_data,omitempty"`        // Store full scraped data
}

// ScrapedPost represents a forum post
type ScrapedPost struct {
	ID             int64                 `json:"id" db:"id"`
	ThreadID       int64                 `json:"thread_id" db:"thread_id"`
	ExternalID     string                `json:"external_id" db:"external_id"`   // Post ID from source site
	Source         string                `json:"source" db:"source"`
	Author         string                `json:"author" db:"author"`
	Content        string                `json:"content" db:"content"`           // HTML content
	PlainText      string                `json:"plain_text" db:"plain_text"`     // Stripped text
	PostNumber     int                   `json:"post_number" db:"post_number"`   // Position in thread
	LikeCount      int                   `json:"like_count" db:"like_count"`
	Metadata       string                `json:"-" db:"metadata"`                // JSON string
	MetadataObj    *ScrapedPostMetadata  `json:"metadata,omitempty" db:"-"`
	PostedAt       time.Time             `json:"posted_at" db:"posted_at"`       // Original post date
	ScrapedAt      time.Time             `json:"scraped_at" db:"scraped_at"`
	CreatedAt      time.Time             `json:"created_at" db:"created_at"`
}

// ScrapedPostMetadata represents additional post information
type ScrapedPostMetadata struct {
	Attachments     []Attachment           `json:"attachments,omitempty"`
	QuotedPosts     []string               `json:"quoted_posts,omitempty"`      // IDs of quoted posts
	EditHistory     []Edit                 `json:"edit_history,omitempty"`
	RawHTML         string                 `json:"raw_html,omitempty"`
	ExtractedLinks  []string               `json:"extracted_links,omitempty"`
	RawData         map[string]interface{} `json:"raw_data,omitempty"`
}

// Attachment represents a file attachment in a post
type Attachment struct {
	Type        string `json:"type"`         // image, video, file
	URL         string `json:"url"`
	Filename    string `json:"filename,omitempty"`
	Size        int64  `json:"size,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
}

// Edit represents an edit in post history
type Edit struct {
	EditedAt time.Time `json:"edited_at"`
	EditedBy string    `json:"edited_by,omitempty"`
}

// ScrapedDownloadLink represents a download link found in posts
type ScrapedDownloadLink struct {
	ID             int64                        `json:"id" db:"id"`
	ThreadID       int64                        `json:"thread_id" db:"thread_id"`
	PostID         int64                        `json:"post_id" db:"post_id"`
	Source         string                       `json:"source" db:"source"`           // simpcity
	Provider       string                       `json:"provider" db:"provider"`       // gofile, pixeldrain, bunkr
	URL            string                       `json:"url" db:"url"`
	OriginalURL    string                       `json:"original_url" db:"original_url"` // Before any redirects
	Filename       string                       `json:"filename" db:"filename"`
	FileSize       int64                        `json:"file_size" db:"file_size"`
	FileType       string                       `json:"file_type" db:"file_type"`     // video/mp4, image/jpeg, etc.
	Status         string                       `json:"status" db:"status"`           // active, dead, expired
	Metadata       string                       `json:"-" db:"metadata"`
	MetadataObj    *DownloadLinkMetadata        `json:"metadata,omitempty" db:"-"`
	DiscoveredAt   time.Time                    `json:"discovered_at" db:"discovered_at"`
	LastCheckedAt  *time.Time                   `json:"last_checked_at" db:"last_checked_at"`
	CreatedAt      time.Time                    `json:"created_at" db:"created_at"`
}

// DownloadLinkMetadata represents additional download link information
type DownloadLinkMetadata struct {
	IsPasswordProtected bool                   `json:"is_password_protected,omitempty"`
	Password            string                 `json:"password,omitempty"`
	ExpiresAt           string                 `json:"expires_at,omitempty"`
	DownloadCount       int                    `json:"download_count,omitempty"`
	PreviewURL          string                 `json:"preview_url,omitempty"`
	Checksum            string                 `json:"checksum,omitempty"`
	AlternativeURLs     []string               `json:"alternative_urls,omitempty"`
	RawData             map[string]interface{} `json:"raw_data,omitempty"`
}

// ScraperJob represents a scraping task
type ScraperJob struct {
	ID             int64              `json:"id" db:"id"`
	JobType        string             `json:"job_type" db:"job_type"`         // thread, category, search, full_site
	Source         string             `json:"source" db:"source"`
	TargetURL      string             `json:"target_url" db:"target_url"`
	Status         string             `json:"status" db:"status"`             // pending, running, completed, failed
	Progress       int                `json:"progress" db:"progress"`         // 0-100
	ItemsProcessed int                `json:"items_processed" db:"items_processed"`
	ItemsTotal     int                `json:"items_total" db:"items_total"`
	ErrorMessage   string             `json:"error_message" db:"error_message"`
	Result         string             `json:"-" db:"result"`                  // JSON string
	ResultObj      map[string]interface{} `json:"result,omitempty" db:"-"`
	StartedAt      *time.Time         `json:"started_at" db:"started_at"`
	CompletedAt    *time.Time         `json:"completed_at" db:"completed_at"`
	CreatedAt      time.Time          `json:"created_at" db:"created_at"`
}

// ScraperStats represents scraper statistics
type ScraperStats struct {
	TotalThreads        int       `json:"total_threads"`
	TotalPosts          int       `json:"total_posts"`
	TotalDownloadLinks  int       `json:"total_download_links"`
	ActiveLinks         int       `json:"active_links"`
	DeadLinks           int       `json:"dead_links"`
	LastScrapedAt       time.Time `json:"last_scraped_at"`
	SourceBreakdown     map[string]int `json:"source_breakdown"`
	ProviderBreakdown   map[string]int `json:"provider_breakdown"`
}

// ThreadCreate represents data needed to create a thread
type ThreadCreate struct {
	ExternalID      string                 `json:"external_id" binding:"required"`
	Source          string                 `json:"source" binding:"required"`
	Title           string                 `json:"title" binding:"required"`
	URL             string                 `json:"url" binding:"required"`
	Author          string                 `json:"author"`
	Category        string                 `json:"category"`
	ViewCount       int                    `json:"view_count"`
	ReplyCount      int                    `json:"reply_count"`
	Metadata        *ScrapedThreadMetadata `json:"metadata,omitempty"`
	LastUpdatedAt   time.Time              `json:"last_updated_at"`
}

// PostCreate represents data needed to create a post
type PostCreate struct {
	ThreadID       int64                `json:"thread_id" binding:"required"`
	ExternalID     string               `json:"external_id" binding:"required"`
	Source         string               `json:"source" binding:"required"`
	Author         string               `json:"author"`
	Content        string               `json:"content"`
	PostNumber     int                  `json:"post_number"`
	Metadata       *ScrapedPostMetadata `json:"metadata,omitempty"`
	PostedAt       time.Time            `json:"posted_at"`
}

// DownloadLinkCreate represents data needed to create a download link
type DownloadLinkCreate struct {
	ThreadID       int64                 `json:"thread_id" binding:"required"`
	PostID         int64                 `json:"post_id" binding:"required"`
	Source         string                `json:"source" binding:"required"`
	Provider       string                `json:"provider" binding:"required"`
	URL            string                `json:"url" binding:"required"`
	Filename       string                `json:"filename"`
	FileType       string                `json:"file_type"`
	Metadata       *DownloadLinkMetadata `json:"metadata,omitempty"`
}

// ScraperJobCreate represents data needed to create a scraper job
type ScraperJobCreate struct {
	JobType    string `json:"job_type" binding:"required"`
	Source     string `json:"source" binding:"required"`
	TargetURL  string `json:"target_url" binding:"required"`
}

// UnmarshalMetadata parses the metadata JSON string
func (t *ScrapedThread) UnmarshalMetadata() error {
	if t.Metadata == "" || t.Metadata == "{}" {
		return nil
	}
	return json.Unmarshal([]byte(t.Metadata), &t.MetadataObj)
}

// MarshalMetadata converts MetadataObj to a JSON string
func (t *ScrapedThread) MarshalMetadata() error {
	bytes, err := json.Marshal(t.MetadataObj)
	t.Metadata = string(bytes)
	return err
}

// UnmarshalMetadata parses the metadata JSON string
func (p *ScrapedPost) UnmarshalMetadata() error {
	if p.Metadata == "" || p.Metadata == "{}" {
		return nil
	}
	return json.Unmarshal([]byte(p.Metadata), &p.MetadataObj)
}

// MarshalMetadata converts MetadataObj to a JSON string
func (p *ScrapedPost) MarshalMetadata() error {
	bytes, err := json.Marshal(p.MetadataObj)
	p.Metadata = string(bytes)
	return err
}

// UnmarshalMetadata parses the metadata JSON string
func (d *ScrapedDownloadLink) UnmarshalMetadata() error {
	if d.Metadata == "" || d.Metadata == "{}" {
		return nil
	}
	return json.Unmarshal([]byte(d.Metadata), &d.MetadataObj)
}

// MarshalMetadata converts MetadataObj to a JSON string
func (d *ScrapedDownloadLink) MarshalMetadata() error {
	bytes, err := json.Marshal(d.MetadataObj)
	d.Metadata = string(bytes)
	return err
}

// UnmarshalResult parses the result JSON string
func (j *ScraperJob) UnmarshalResult() error {
	if j.Result == "" || j.Result == "{}" {
		return nil
	}
	return json.Unmarshal([]byte(j.Result), &j.ResultObj)
}

// MarshalResult converts ResultObj to a JSON string
func (j *ScraperJob) MarshalResult() error {
	bytes, err := json.Marshal(j.ResultObj)
	j.Result = string(bytes)
	return err
}
