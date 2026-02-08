package services

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// JDownloaderService handles JDownloader 2 API communication
type JDownloaderService struct {
	db         *sql.DB
	httpClient *http.Client
}

// NewJDownloaderService creates a new JDownloader service
func NewJDownloaderService() *JDownloaderService {
	return &JDownloaderService{
		db: database.GetDB(),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetConfig retrieves the JDownloader configuration
func (s *JDownloaderService) GetConfig() (*models.JDownloaderConfig, error) {
	config := &models.JDownloaderConfig{}

	var lastConnectedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, connection_type, local_host, local_port, local_password, local_use_auth,
		       cloud_email, cloud_password, cloud_device_id, cloud_device_name,
		       default_package_name, auto_start_downloads, enabled, last_connected_at,
		       connection_status, created_at, updated_at
		FROM jdownloader_config
		WHERE id = 1
	`).Scan(
		&config.ID, &config.ConnectionType, &config.LocalHost, &config.LocalPort,
		&config.LocalPassword, &config.LocalUseAuth, &config.CloudEmail, &config.CloudPassword,
		&config.CloudDeviceID, &config.CloudDeviceName, &config.DefaultPackageName,
		&config.AutoStartDownloads, &config.Enabled, &lastConnectedAt,
		&config.ConnectionStatus, &config.CreatedAt, &config.UpdatedAt,
	)

	if lastConnectedAt.Valid {
		config.LastConnectedAt = &lastConnectedAt.Time
	}

	if err == sql.ErrNoRows {
		// Return default config
		return &models.JDownloaderConfig{
			ID:               1,
			ConnectionType:   "local",
			LocalHost:        "192.168.1.10",
			LocalPort:        3129,
			Enabled:          true,
			ConnectionStatus: "disconnected",
		}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get JDownloader config: %w", err)
	}

	// Don't expose passwords in response (mask them)
	if config.LocalPassword != "" {
		config.LocalPassword = "********"
	}
	if config.CloudPassword != "" {
		config.CloudPassword = "********"
	}

	return config, nil
}

// UpdateConfig updates the JDownloader configuration
func (s *JDownloaderService) UpdateConfig(config *models.JDownloaderConfig) error {
	// If password is masked, don't update it
	var localPassword, cloudPassword string

	if config.LocalPassword != "" && config.LocalPassword != "********" {
		localPassword = config.LocalPassword
	} else {
		// Keep existing password
		err := s.db.QueryRow("SELECT local_password FROM jdownloader_config WHERE id = 1").Scan(&localPassword)
		if err != nil && err != sql.ErrNoRows {
			localPassword = ""
		}
	}

	if config.CloudPassword != "" && config.CloudPassword != "********" {
		cloudPassword = config.CloudPassword
	} else {
		// Keep existing password
		err := s.db.QueryRow("SELECT cloud_password FROM jdownloader_config WHERE id = 1").Scan(&cloudPassword)
		if err != nil && err != sql.ErrNoRows {
			cloudPassword = ""
		}
	}

	_, err := s.db.Exec(`
		INSERT INTO jdownloader_config (id, connection_type, local_host, local_port, local_password, local_use_auth,
		       cloud_email, cloud_password, cloud_device_id, cloud_device_name,
		       default_package_name, auto_start_downloads, enabled, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
		       connection_type = excluded.connection_type,
		       local_host = excluded.local_host,
		       local_port = excluded.local_port,
		       local_password = excluded.local_password,
		       local_use_auth = excluded.local_use_auth,
		       cloud_email = excluded.cloud_email,
		       cloud_password = excluded.cloud_password,
		       cloud_device_id = excluded.cloud_device_id,
		       cloud_device_name = excluded.cloud_device_name,
		       default_package_name = excluded.default_package_name,
		       auto_start_downloads = excluded.auto_start_downloads,
		       enabled = excluded.enabled,
		       updated_at = CURRENT_TIMESTAMP
	`,
		config.ConnectionType, config.LocalHost, config.LocalPort, localPassword, config.LocalUseAuth,
		config.CloudEmail, cloudPassword, config.CloudDeviceID, config.CloudDeviceName,
		config.DefaultPackageName, config.AutoStartDownloads, config.Enabled,
	)

	return err
}

// TestConnection tests the JDownloader connection
func (s *JDownloaderService) TestConnection() (*models.JDownloaderTestResult, error) {
	config, err := s.getFullConfig()
	if err != nil {
		return &models.JDownloaderTestResult{
			Success:      false,
			Message:      "Failed to get configuration",
			ErrorDetails: err.Error(),
		}, nil
	}

	if config.ConnectionType == "local" {
		return s.testLocalConnection(config)
	}

	return &models.JDownloaderTestResult{
		Success: false,
		Message: "Cloud connection not yet implemented",
	}, nil
}

// testLocalConnection tests connection to local JDownloader API
func (s *JDownloaderService) testLocalConnection(config *models.JDownloaderConfig) (*models.JDownloaderTestResult, error) {
	// JDownloader 2 deprecated API runs on port 3128 by default
	baseURL := fmt.Sprintf("http://%s:%d", config.LocalHost, config.LocalPort)

	// Use /jd/version endpoint to verify it's actually JDownloader (not just any HTTP server)
	versionURL := baseURL + "/jd/version"

	req, err := http.NewRequest("POST", versionURL, nil)
	if err != nil {
		return &models.JDownloaderTestResult{
			Success:      false,
			Message:      "Failed to create request",
			ErrorDetails: err.Error(),
		}, nil
	}
	req.Header.Set("Content-Type", "application/json")

	// Add basic auth if configured
	if config.LocalUseAuth && config.LocalPassword != "" {
		req.SetBasicAuth("", config.LocalPassword)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		// Try alternative endpoints
		alternativeResult := s.tryAlternativeEndpoints(config)
		if alternativeResult.Success {
			return alternativeResult, nil
		}

		return &models.JDownloaderTestResult{
			Success:      false,
			Message:      "Cannot connect to JDownloader API",
			ErrorDetails: fmt.Sprintf("Connection failed: %v\n\nMake sure:\n1. JDownloader 2 is running\n2. Deprecated API is enabled in JDownloader: Settings → Advanced Settings → search 'deprecated' → enable it\n3. If using Docker, ensure port %d is exposed", err, config.LocalPort),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		// Read and validate the version response
		body, _ := io.ReadAll(resp.Body)
		versionStr := s.parseJDVersion(body)

		// Check if this looks like a JDownloader version (should be numeric like "48755")
		if versionStr == "" {
			s.updateConnectionStatus("disconnected")
			return &models.JDownloaderTestResult{
				Success:      false,
				Message:      "Invalid response from server",
				ErrorDetails: fmt.Sprintf("The server responded but doesn't appear to be JDownloader. Response: %s", string(body)),
			}, nil
		}

		// Update connection status
		s.updateConnectionStatus("connected")

		return &models.JDownloaderTestResult{
			Success:      true,
			Message:      "Successfully connected to JDownloader 2",
			APIAvailable: true,
			Version:      versionStr,
		}, nil
	}

	if resp.StatusCode == 401 {
		return &models.JDownloaderTestResult{
			Success:      false,
			Message:      "Authentication required",
			ErrorDetails: "The JDownloader API requires a password. Please configure the password in settings.",
		}, nil
	}

	body, _ := io.ReadAll(resp.Body)
	return &models.JDownloaderTestResult{
		Success:      false,
		Message:      fmt.Sprintf("Unexpected response: %d", resp.StatusCode),
		ErrorDetails: string(body),
	}, nil
}

// tryAlternativeEndpoints tries common alternative API endpoints
func (s *JDownloaderService) tryAlternativeEndpoints(config *models.JDownloaderConfig) *models.JDownloaderTestResult {
	// Common ports for JDownloader API
	ports := []int{3128, 3129, 9665, 9666}

	for _, port := range ports {
		if port == config.LocalPort {
			continue
		}

		testURL := fmt.Sprintf("http://%s:%d/help", config.LocalHost, port)
		resp, err := s.httpClient.Get(testURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return &models.JDownloaderTestResult{
					Success:      true,
					Message:      fmt.Sprintf("JDownloader API found on port %d (update your configured port from %d)", port, config.LocalPort),
					APIAvailable: true,
				}
			}
		}
	}

	return &models.JDownloaderTestResult{
		Success: false,
		Message: "API not found on alternative ports",
	}
}

// getJDVersion tries to get JDownloader version information
func (s *JDownloaderService) getJDVersion(config *models.JDownloaderConfig) string {
	baseURL := fmt.Sprintf("http://%s:%d", config.LocalHost, config.LocalPort)

	// Try to call jd/version endpoint
	versionURL := baseURL + "/jd/version"
	req, err := http.NewRequest("POST", versionURL, nil)
	if err != nil {
		return ""
	}

	if config.LocalUseAuth && config.LocalPassword != "" {
		req.SetBasicAuth("", config.LocalPassword)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, _ := io.ReadAll(resp.Body)
		return strings.TrimSpace(string(body))
	}

	return ""
}

// SendLinks sends links to JDownloader
func (s *JDownloaderService) SendLinks(links []models.JDownloaderLinkRequest) ([]models.JDownloaderAddLinkResponse, error) {
	config, err := s.getFullConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get configuration: %w", err)
	}

	if !config.Enabled {
		return nil, fmt.Errorf("JDownloader integration is disabled")
	}

	results := make([]models.JDownloaderAddLinkResponse, 0, len(links))

	for _, link := range links {
		var result models.JDownloaderAddLinkResponse

		if config.ConnectionType == "local" {
			result = s.sendLinkLocal(config, link)
		} else {
			result = models.JDownloaderAddLinkResponse{
				Success: false,
				Message: "Cloud connection not yet implemented",
			}
		}

		// Record in history
		s.addToHistory(link, result)
		results = append(results, result)
	}

	return results, nil
}

// sendLinkLocal sends a link using the local API
func (s *JDownloaderService) sendLinkLocal(config *models.JDownloaderConfig, link models.JDownloaderLinkRequest) models.JDownloaderAddLinkResponse {
	baseURL := fmt.Sprintf("http://%s:%d", config.LocalHost, config.LocalPort)

	// Use linkgrabberv2/addLinks endpoint for the deprecated API
	addLinksURL := baseURL + "/linkgrabberv2/addLinks"

	// Build the request body for the deprecated API
	// The deprecated API expects params as a direct object (not wrapped in arrays)
	requestParams := map[string]interface{}{
		"autostart":   link.AutoStart || config.AutoStartDownloads,
		"links":       link.URL,
		"packageName": s.resolvePackageName(config, link),
	}

	// Add destination folder if specified
	if link.DownloadPath != "" {
		requestParams["destinationFolder"] = link.DownloadPath
	}

	jsonBody, err := json.Marshal(requestParams)
	if err != nil {
		return models.JDownloaderAddLinkResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to marshal request: %v", err),
		}
	}

	log.Printf("[JDownloader] Sending addLinks request to %s with body: %s", addLinksURL, string(jsonBody))

	req, err := http.NewRequest("POST", addLinksURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return models.JDownloaderAddLinkResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create request: %v", err),
		}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "identity") // Disable gzip compression

	if config.LocalUseAuth && config.LocalPassword != "" {
		req.SetBasicAuth("", config.LocalPassword)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("[JDownloader] Failed to connect: %v", err)
		return models.JDownloaderAddLinkResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to connect to JDownloader: %v", err),
		}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("[JDownloader] addLinks response (HTTP %d): %s", resp.StatusCode, string(body))

	if resp.StatusCode == 200 {
		// Update last connected time
		s.updateConnectionStatus("connected")

		return models.JDownloaderAddLinkResponse{
			Success: true,
			Message: "Link added to JDownloader",
		}
	}

	return models.JDownloaderAddLinkResponse{
		Success: false,
		Message: fmt.Sprintf("Failed to add link (HTTP %d): %s", resp.StatusCode, string(body)),
	}
}

// resolvePackageName resolves the package name using template or default
func (s *JDownloaderService) resolvePackageName(config *models.JDownloaderConfig, link models.JDownloaderLinkRequest) string {
	if link.PackageName != "" {
		return link.PackageName
	}

	if config.DefaultPackageName != "" {
		return config.DefaultPackageName
	}

	// Extract package name from URL
	parsedURL, err := url.Parse(link.URL)
	if err == nil {
		// Use host as default package name
		return parsedURL.Host
	}

	return "Downloads"
}

// addToHistory records a link send attempt in history
func (s *JDownloaderService) addToHistory(link models.JDownloaderLinkRequest, result models.JDownloaderAddLinkResponse) {
	status := "sent"
	errorMsg := ""
	if !result.Success {
		status = "failed"
		errorMsg = result.Message
	}

	_, err := s.db.Exec(`
		INSERT INTO jdownloader_history (video_id, link_url, package_name, status, error_message, sent_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`, link.VideoID, link.URL, link.PackageName, status, errorMsg)

	if err != nil {
		// Log but don't fail the operation
		fmt.Printf("Failed to record JDownloader history: %v\n", err)
	}
}

// GetHistory retrieves JDownloader send history
func (s *JDownloaderService) GetHistory(limit, offset int) ([]models.JDownloaderHistory, int, error) {
	// Get total count
	var total int
	err := s.db.QueryRow("SELECT COUNT(*) FROM jdownloader_history").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Query(`
		SELECT h.id, h.video_id, h.link_url, h.package_name, h.status, h.error_message, h.sent_at,
		       COALESCE(v.title, '') as video_title
		FROM jdownloader_history h
		LEFT JOIN videos v ON h.video_id = v.id
		ORDER BY h.sent_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	history := make([]models.JDownloaderHistory, 0)
	for rows.Next() {
		var h models.JDownloaderHistory
		var videoID sql.NullInt64
		var packageName, errorMessage, videoTitle sql.NullString

		err := rows.Scan(&h.ID, &videoID, &h.LinkURL, &packageName, &h.Status, &errorMessage, &h.SentAt, &videoTitle)
		if err != nil {
			return nil, 0, err
		}

		if videoID.Valid {
			h.VideoID = &videoID.Int64
		}
		h.PackageName = packageName.String
		h.ErrorMessage = errorMessage.String
		h.VideoTitle = videoTitle.String

		history = append(history, h)
	}

	return history, total, nil
}

// GetConnectionStatus returns the current connection status
func (s *JDownloaderService) GetConnectionStatus() (*models.JDownloaderConnectionStatus, error) {
	config, err := s.getFullConfig()
	if err != nil {
		return &models.JDownloaderConnectionStatus{
			Connected:    false,
			ErrorMessage: err.Error(),
		}, nil
	}

	status := &models.JDownloaderConnectionStatus{
		Type:       config.ConnectionType,
		DeviceName: config.CloudDeviceName,
	}

	// Verify connection by checking /jd/version (proves it's actually JDownloader)
	if config.ConnectionType == "local" {
		baseURL := fmt.Sprintf("http://%s:%d/jd/version", config.LocalHost, config.LocalPort)
		resp, err := s.httpClient.Post(baseURL, "application/json", nil)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
				body, _ := io.ReadAll(resp.Body)
				versionStr := s.parseJDVersion(body)
				// Verify we got a valid JDownloader version
				if versionStr != "" {
					status.Connected = true
					status.JDVersion = versionStr
					now := time.Now()
					status.LastCheck = &now
				}
			}
		} else {
			status.ErrorMessage = err.Error()
		}
	}

	return status, nil
}

// getFullConfig retrieves the full config including passwords (internal use only)
func (s *JDownloaderService) getFullConfig() (*models.JDownloaderConfig, error) {
	config := &models.JDownloaderConfig{}

	var lastConnectedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, connection_type, local_host, local_port, local_password, local_use_auth,
		       cloud_email, cloud_password, cloud_device_id, cloud_device_name,
		       default_package_name, auto_start_downloads, enabled, last_connected_at,
		       connection_status, created_at, updated_at
		FROM jdownloader_config
		WHERE id = 1
	`).Scan(
		&config.ID, &config.ConnectionType, &config.LocalHost, &config.LocalPort,
		&config.LocalPassword, &config.LocalUseAuth, &config.CloudEmail, &config.CloudPassword,
		&config.CloudDeviceID, &config.CloudDeviceName, &config.DefaultPackageName,
		&config.AutoStartDownloads, &config.Enabled, &lastConnectedAt,
		&config.ConnectionStatus, &config.CreatedAt, &config.UpdatedAt,
	)

	if lastConnectedAt.Valid {
		config.LastConnectedAt = &lastConnectedAt.Time
	}

	if err == sql.ErrNoRows {
		return &models.JDownloaderConfig{
			ID:               1,
			ConnectionType:   "local",
			LocalHost:        "192.168.1.10",
			LocalPort:        3129,
			Enabled:          true,
			ConnectionStatus: "disconnected",
		}, nil
	}

	return config, err
}

// parseJDVersion extracts the version number from JDownloader API response
// Handles both raw numbers (48637) and JSON-wrapped responses ({"data":48637})
func (s *JDownloaderService) parseJDVersion(body []byte) string {
	bodyStr := strings.TrimSpace(string(body))

	// Try to parse as JSON first ({"data":48637} format)
	var jsonResp struct {
		Data interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &jsonResp); err == nil && jsonResp.Data != nil {
		// Convert data to string
		switch v := jsonResp.Data.(type) {
		case float64:
			return fmt.Sprintf("%.0f", v)
		case string:
			return v
		case int:
			return fmt.Sprintf("%d", v)
		}
	}

	// Try raw number format
	bodyStr = strings.Trim(bodyStr, "\" \n\r")
	if bodyStr == "" || bodyStr == "null" {
		return ""
	}

	// Check if it's a valid number
	var num int
	if _, err := fmt.Sscanf(bodyStr, "%d", &num); err == nil {
		return bodyStr
	}

	// Check if it contains a version-like string
	if strings.Contains(bodyStr, ".") {
		return bodyStr
	}

	return ""
}

// parseJDDataArray extracts an array from JDownloader API response
// Handles both raw arrays and JSON-wrapped responses ({"data":[...]})
func (s *JDownloaderService) parseJDDataArray(body []byte) []map[string]interface{} {
	// Try to parse as wrapped JSON first ({"data":[...]} format)
	var wrappedResp struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &wrappedResp); err == nil && wrappedResp.Data != nil {
		return wrappedResp.Data
	}

	// Try raw array format
	var rawArray []map[string]interface{}
	if err := json.Unmarshal(body, &rawArray); err == nil {
		return rawArray
	}

	return []map[string]interface{}{}
}

// parseJDDataInt64 extracts an int64 from JDownloader API response
// Handles both raw numbers and JSON-wrapped responses ({"data":12345})
func (s *JDownloaderService) parseJDDataInt64(body []byte) int64 {
	// Try to parse as wrapped JSON first
	var wrappedResp struct {
		Data interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &wrappedResp); err == nil && wrappedResp.Data != nil {
		switch v := wrappedResp.Data.(type) {
		case float64:
			return int64(v)
		case int:
			return int64(v)
		case int64:
			return v
		}
	}

	// Try raw number
	var num int64
	if err := json.Unmarshal(body, &num); err == nil {
		return num
	}

	return 0
}

// parseJDDataString extracts a string from JDownloader API response
// Handles both raw strings and JSON-wrapped responses ({"data":"value"})
func (s *JDownloaderService) parseJDDataString(body []byte) string {
	// Try to parse as wrapped JSON first
	var wrappedResp struct {
		Data interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &wrappedResp); err == nil && wrappedResp.Data != nil {
		switch v := wrappedResp.Data.(type) {
		case string:
			return v
		case float64:
			return fmt.Sprintf("%.0f", v)
		}
	}

	// Try raw string (remove quotes)
	return strings.Trim(string(body), "\" \n\r")
}

// updateConnectionStatus updates the connection status in the database
func (s *JDownloaderService) updateConnectionStatus(status string) {
	s.db.Exec(`
		UPDATE jdownloader_config
		SET connection_status = ?, last_connected_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`, status)
}

// GetPackageAddedTimes returns a map of package_name → earliest sent_at timestamp (Unix ms)
// for packages that were added via our app, so we can track when they were added for sorting.
func (s *JDownloaderService) GetPackageAddedTimes() (map[string]int64, error) {
	rows, err := s.db.Query(`
		SELECT package_name, MIN(sent_at) as first_added
		FROM jdownloader_history
		WHERE status = 'sent' AND package_name IS NOT NULL AND package_name != ''
		GROUP BY package_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var name string
		var addedAt time.Time
		if err := rows.Scan(&name, &addedAt); err != nil {
			continue
		}
		result[name] = addedAt.UnixMilli()
	}
	return result, nil
}

// ClearHistory clears all JDownloader history
func (s *JDownloaderService) ClearHistory() error {
	_, err := s.db.Exec("DELETE FROM jdownloader_history")
	return err
}

// IsAvailable checks if JDownloader is running and accessible (quick check)
func (s *JDownloaderService) IsAvailable() bool {
	config, err := s.getFullConfig()
	if err != nil || !config.Enabled {
		return false
	}

	// Use /jd/version to verify it's actually JDownloader
	baseURL := fmt.Sprintf("http://%s:%d/jd/version", config.LocalHost, config.LocalPort)
	resp, err := s.httpClient.Post(baseURL, "application/json", nil)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false
	}

	// Verify the response looks like a JDownloader version
	body, _ := io.ReadAll(resp.Body)
	versionStr := s.parseJDVersion(body)
	return versionStr != ""
}

// StartDownloads starts all downloads in JDownloader
func (s *JDownloaderService) StartDownloads() error {
	config, err := s.getFullConfig()
	if err != nil {
		return err
	}

	baseURL := fmt.Sprintf("http://%s:%d", config.LocalHost, config.LocalPort)
	resp, err := s.httpClient.Get(baseURL + "/downloadcontroller/start")
	if err != nil {
		return fmt.Errorf("failed to start downloads: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("JDownloader returned status %d", resp.StatusCode)
	}

	return nil
}

// StopDownloads stops all downloads in JDownloader
func (s *JDownloaderService) StopDownloads() error {
	config, err := s.getFullConfig()
	if err != nil {
		return err
	}

	baseURL := fmt.Sprintf("http://%s:%d", config.LocalHost, config.LocalPort)
	resp, err := s.httpClient.Get(baseURL + "/downloadcontroller/stop")
	if err != nil {
		return fmt.Errorf("failed to stop downloads: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("JDownloader returned status %d", resp.StatusCode)
	}

	return nil
}

// PauseDownloads pauses/resumes all downloads
func (s *JDownloaderService) PauseDownloads(pause bool) error {
	config, err := s.getFullConfig()
	if err != nil {
		return err
	}

	baseURL := fmt.Sprintf("http://%s:%d", config.LocalHost, config.LocalPort)
	endpoint := fmt.Sprintf("/downloadcontroller/pause/%t", pause)
	resp, err := s.httpClient.Get(baseURL + endpoint)
	if err != nil {
		return fmt.Errorf("failed to pause downloads: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// GetDownloadSpeed returns current download speed in bytes per second
func (s *JDownloaderService) GetDownloadSpeed() (int64, error) {
	config, err := s.getFullConfig()
	if err != nil {
		return 0, err
	}

	baseURL := fmt.Sprintf("http://%s:%d", config.LocalHost, config.LocalPort)
	resp, err := s.httpClient.Get(baseURL + "/downloadcontroller/getSpeedInBps")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	// Parse response - handles both raw numbers and {"data":...} wrapped format
	return s.parseJDDataInt64(body), nil
}

// GetDownloadState returns current download controller state
func (s *JDownloaderService) GetDownloadState() (string, error) {
	config, err := s.getFullConfig()
	if err != nil {
		return "", err
	}

	baseURL := fmt.Sprintf("http://%s:%d", config.LocalHost, config.LocalPort)
	resp, err := s.httpClient.Get(baseURL + "/downloadcontroller/getCurrentState")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	// Parse response - handles both raw strings and {"data":...} wrapped format
	return s.parseJDDataString(body), nil
}

// GetLinkGrabberLinks returns all links in the LinkGrabber
func (s *JDownloaderService) GetLinkGrabberLinks() ([]map[string]interface{}, error) {
	config, err := s.getFullConfig()
	if err != nil {
		return nil, err
	}

	baseURL := fmt.Sprintf("http://%s:%d", config.LocalHost, config.LocalPort)

	// Build query params
	params := map[string]interface{}{
		"bytesTotal":   true,
		"comment":      true,
		"status":       true,
		"enabled":      true,
		"maxResults":   -1,
		"host":         true,
		"availability": true,
		"url":          true,
		"priority":     true,
		"variantName":  true,
		"addedDate":    true,
	}

	// JDownloader deprecated API expects params object directly, not wrapped in array
	jsonBody, _ := json.Marshal(params)
	req, err := http.NewRequest("POST", baseURL+"/linkgrabberv2/queryLinks", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "identity") // Disable gzip compression
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return s.parseJDDataArray(body), nil
}

// GetLinkGrabberPackages returns all packages in the LinkGrabber
func (s *JDownloaderService) GetLinkGrabberPackages() ([]map[string]interface{}, error) {
	config, err := s.getFullConfig()
	if err != nil {
		return nil, err
	}

	baseURL := fmt.Sprintf("http://%s:%d", config.LocalHost, config.LocalPort)

	params := map[string]interface{}{
		"bytesTotal":    true,
		"childCount":    true,
		"comment":       true,
		"enabled":       true,
		"hosts":         true,
		"maxResults":    -1,
		"priority":      true,
		"saveTo":        true,
		"status":        true,
		"addedDate":     true,
		"availableOfflineCount": true,
		"availableOnlineCount":  true,
		"availableTempUnknownCount": true,
		"availableUnknownCount": true,
	}

	// JDownloader deprecated API expects params object directly, not wrapped in array
	jsonBody, _ := json.Marshal(params)
	req, err := http.NewRequest("POST", baseURL+"/linkgrabberv2/queryPackages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "identity") // Disable gzip compression
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return s.parseJDDataArray(body), nil
}

// GetDownloadLinks returns all links in the download list
func (s *JDownloaderService) GetDownloadLinks() ([]map[string]interface{}, error) {
	config, err := s.getFullConfig()
	if err != nil {
		return nil, err
	}

	baseURL := fmt.Sprintf("http://%s:%d", config.LocalHost, config.LocalPort)

	params := map[string]interface{}{
		"bytesLoaded":      true,
		"bytesTotal":       true,
		"comment":          true,
		"status":           true,
		"enabled":          true,
		"eta":              true,
		"extractionStatus": true,
		"finished":         true,
		"running":          true,
		"skipped":          true,
		"speed":            true,
		"url":              true,
		"host":             true,
		"priority":         true,
		"maxResults":       -1,
	}

	// JDownloader deprecated API expects params object directly, not wrapped in array
	jsonBody, _ := json.Marshal(params)
	req, err := http.NewRequest("POST", baseURL+"/downloadsV2/queryLinks", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "identity") // Disable gzip compression
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return s.parseJDDataArray(body), nil
}

// GetDownloadPackages returns all packages in the download list
func (s *JDownloaderService) GetDownloadPackages() ([]map[string]interface{}, error) {
	config, err := s.getFullConfig()
	if err != nil {
		return nil, err
	}

	baseURL := fmt.Sprintf("http://%s:%d", config.LocalHost, config.LocalPort)

	params := map[string]interface{}{
		"bytesLoaded":  true,
		"bytesTotal":   true,
		"childCount":   true,
		"comment":      true,
		"enabled":      true,
		"eta":          true,
		"finished":     true,
		"hosts":        true,
		"maxResults":   -1,
		"priority":     true,
		"running":      true,
		"saveTo":       true,
		"speed":        true,
		"status":       true,
		"addedDate":    true,
		"finishedDate": true,
	}

	// JDownloader deprecated API expects params object directly, not wrapped in array
	jsonBody, _ := json.Marshal(params)
	req, err := http.NewRequest("POST", baseURL+"/downloadsV2/queryPackages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "identity") // Disable gzip compression
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return s.parseJDDataArray(body), nil
}

// doJDAction sends a POST request to JDownloader with proper headers and error handling
func (s *JDownloaderService) doJDAction(endpoint string, params interface{}) error {
	config, err := s.getFullConfig()
	if err != nil {
		return err
	}

	baseURL := fmt.Sprintf("http://%s:%d", config.LocalHost, config.LocalPort)
	jsonBody, _ := json.Marshal(params)

	log.Printf("[JDownloader] POST %s body: %s", endpoint, string(jsonBody))

	req, err := http.NewRequest("POST", baseURL+endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "identity")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("[JDownloader] Response %d: %s", resp.StatusCode, string(body))

	if resp.StatusCode >= 400 {
		return fmt.Errorf("JDownloader returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// doJDMultiParamAction sends a multi-parameter action to JDownloader.
// Each parameter is individually JSON-serialized as a string within the outer array.
// This produces ["[1,2,3]","[4,5,6]"] format which JDownloader's parseParameters
// can correctly deserialize (avoids ClassCastException from raw nested arrays).
func (s *JDownloaderService) doJDMultiParamAction(endpoint string, params ...interface{}) error {
	stringParams := make([]string, len(params))
	for i, p := range params {
		jsonBytes, err := json.Marshal(p)
		if err != nil {
			return fmt.Errorf("failed to marshal param %d: %w", i, err)
		}
		stringParams[i] = string(jsonBytes)
	}
	return s.doJDAction(endpoint, stringParams)
}

// MoveToDownloads moves links/packages from LinkGrabber to download list
func (s *JDownloaderService) MoveToDownloads(linkIDs []int64, packageIDs []int64) error {
	return s.doJDMultiParamAction("/linkgrabberv2/moveToDownloadlist", linkIDs, packageIDs)
}

// RemoveLinkGrabberLinks removes links from LinkGrabber
func (s *JDownloaderService) RemoveLinkGrabberLinks(linkIDs []int64, packageIDs []int64) error {
	return s.doJDMultiParamAction("/linkgrabberv2/removeLinks", linkIDs, packageIDs)
}

// RemoveDownloadLinks removes links from download list
func (s *JDownloaderService) RemoveDownloadLinks(linkIDs []int64, packageIDs []int64) error {
	return s.doJDMultiParamAction("/downloadsV2/removeLinks", linkIDs, packageIDs)
}

// ClearLinkGrabber clears all links from LinkGrabber
func (s *JDownloaderService) ClearLinkGrabber() error {
	return s.doJDAction("/linkgrabberv2/clearList", nil)
}

// SetLinkEnabled enables/disables a link in LinkGrabber
func (s *JDownloaderService) SetLinkEnabled(enabled bool, linkIDs []int64, packageIDs []int64, isDownload bool) error {
	endpoint := "/linkgrabberv2/setEnabled"
	if isDownload {
		endpoint = "/downloadsV2/setEnabled"
	}

	return s.doJDMultiParamAction(endpoint, enabled, linkIDs, packageIDs)
}

// ForceDownload forces download of specific links
func (s *JDownloaderService) ForceDownload(linkIDs []int64, packageIDs []int64) error {
	return s.doJDMultiParamAction("/downloadsV2/forceDownload", linkIDs, packageIDs)
}

// SetSpeedLimit sets the download speed limit (0 = unlimited)
func (s *JDownloaderService) SetSpeedLimit(bytesPerSecond int64) error {
	return s.doJDMultiParamAction("/config/set",
		"org.jdownloader.settings.GeneralSettings",
		nil,
		"maxdownloadspeedinbytespersecond",
		bytesPerSecond,
	)
}

// GetSpeedLimit gets the current download speed limit
func (s *JDownloaderService) GetSpeedLimit() (int64, error) {
	config, err := s.getFullConfig()
	if err != nil {
		return 0, err
	}

	baseURL := fmt.Sprintf("http://%s:%d", config.LocalHost, config.LocalPort)

	// String-encode each parameter for JDownloader's parseParameters
	stringParams := []string{
		`"org.jdownloader.settings.GeneralSettings"`,
		"null",
		`"maxdownloadspeedinbytespersecond"`,
	}
	jsonBody, _ := json.Marshal(stringParams)

	req, err := http.NewRequest("POST", baseURL+"/config/get", bytes.NewBuffer(jsonBody))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "identity")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	// Parse response - handles both raw numbers and {"data":...} wrapped format
	return s.parseJDDataInt64(body), nil
}

// GetSystemInfo returns JDownloader system information
func (s *JDownloaderService) GetSystemInfo() (map[string]interface{}, error) {
	config, err := s.getFullConfig()
	if err != nil {
		return nil, err
	}

	baseURL := fmt.Sprintf("http://%s:%d", config.LocalHost, config.LocalPort)
	info := make(map[string]interface{})

	// Get version - handles both raw and {"data":...} wrapped format
	if resp, err := s.httpClient.Post(baseURL+"/jd/version", "application/json", nil); err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		info["version"] = s.parseJDVersion(body)
	}

	// Get uptime - handles both raw and {"data":...} wrapped format
	if resp, err := s.httpClient.Post(baseURL+"/jd/uptime", "application/json", nil); err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		info["uptime"] = s.parseJDDataInt64(body)
	}

	// Get current state
	state, _ := s.GetDownloadState()
	info["state"] = state

	// Get speed
	speed, _ := s.GetDownloadSpeed()
	info["speed"] = speed

	// Get speed limit
	limit, _ := s.GetSpeedLimit()
	info["speedLimit"] = limit

	return info, nil
}

// CleanupFinished removes finished downloads from the list
func (s *JDownloaderService) CleanupFinished() error {
	// cleanup(Action action, Mode mode, SelectionType selectionType, long[] linkIds, long[] packageIds)
	return s.doJDMultiParamAction("/downloadsV2/cleanup",
		"DELETE_FINISHED", "REMOVE_LINKS_ONLY", "ALL", nil, nil,
	)
}

// ResetLinks resets failed/skipped links to allow retry
func (s *JDownloaderService) ResetLinks(linkIDs []int64, packageIDs []int64) error {
	return s.doJDMultiParamAction("/downloadsV2/resetLinks", linkIDs, packageIDs)
}

// SetPriority sets download priority for links
func (s *JDownloaderService) SetPriority(priority string, linkIDs []int64, packageIDs []int64, isDownload bool) error {
	endpoint := "/linkgrabberv2/setPriority"
	if isDownload {
		endpoint = "/downloadsV2/setPriority"
	}

	return s.doJDMultiParamAction(endpoint, priority, linkIDs, packageIDs)
}

// RenamePackage renames a package
func (s *JDownloaderService) RenamePackage(packageID int64, newName string, isDownload bool) error {
	endpoint := "/linkgrabberv2/renamePackage"
	if isDownload {
		endpoint = "/downloadsV2/renamePackage"
	}

	return s.doJDMultiParamAction(endpoint, packageID, newName)
}
