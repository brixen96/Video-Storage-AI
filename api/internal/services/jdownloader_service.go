package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// JDownloaderService handles integration with JDownloader2 via its Direct Connection API
type JDownloaderService struct {
	baseURL    string
	httpClient *http.Client
}

// NewJDownloaderService creates a new JDownloader service
// Default JDownloader Direct Connection API runs on http://localhost:3128
func NewJDownloaderService(baseURL string) *JDownloaderService {
	if baseURL == "" {
		baseURL = "http://localhost:3128"
	}

	return &JDownloaderService{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// LinkCandidate represents a link to be added to JDownloader
type LinkCandidate struct {
	URL      string `json:"url"`
	Filename string `json:"filename,omitempty"`
	Comment  string `json:"comment,omitempty"`
}

// AddLinksRequest represents a request to add links to JDownloader
type AddLinksRequest struct {
	Links           []string `json:"links"`
	PackageName     string   `json:"packageName,omitempty"`
	DestinationDir  string   `json:"destinationDir,omitempty"`
	DownloadPassword string  `json:"downloadPassword,omitempty"`
	AutoStart       bool     `json:"autostart"`
}

// JDownloaderResponse represents a generic JDownloader API response
type JDownloaderResponse struct {
	Data interface{} `json:"data"`
}

// IsAvailable checks if JDownloader is running and accessible
func (s *JDownloaderService) IsAvailable() bool {
	resp, err := s.httpClient.Get(s.baseURL + "/flash/get/version")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

// GetVersion returns the JDownloader version
func (s *JDownloaderService) GetVersion() (string, error) {
	resp, err := s.httpClient.Get(s.baseURL + "/flash/get/version")
	if err != nil {
		return "", fmt.Errorf("failed to connect to JDownloader: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("JDownloader returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// AddLinks adds download links to JDownloader
func (s *JDownloaderService) AddLinks(links []string, packageName string, destinationDir string) error {
	if len(links) == 0 {
		return fmt.Errorf("no links provided")
	}

	// Build the request payload
	payload := map[string]interface{}{
		"links":      links,
		"autostart":  false,
		"autoExtract": false,
	}

	if packageName != "" {
		payload["packageName"] = packageName
	}

	if destinationDir != "" {
		payload["destinationFolder"] = destinationDir
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make the API call
	resp, err := s.httpClient.Post(
		s.baseURL+"/linkgrabberv2/addLinks",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to add links: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("JDownloader returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// StartDownloads starts all downloads in JDownloader
func (s *JDownloaderService) StartDownloads() error {
	resp, err := s.httpClient.Get(s.baseURL + "/downloadcontroller/start")
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
	resp, err := s.httpClient.Get(s.baseURL + "/downloadcontroller/stop")
	if err != nil {
		return fmt.Errorf("failed to stop downloads: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("JDownloader returned status %d", resp.StatusCode)
	}

	return nil
}

// GetDownloadStatus returns the current download status
func (s *JDownloaderService) GetDownloadStatus() (map[string]interface{}, error) {
	resp, err := s.httpClient.Get(s.baseURL + "/downloadcontroller/getCurrentState")
	if err != nil {
		return nil, fmt.Errorf("failed to get download status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("JDownloader returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var status map[string]interface{}
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, err
	}

	return status, nil
}

// GetDownloadList returns all downloads in JDownloader
func (s *JDownloaderService) GetDownloadList() ([]map[string]interface{}, error) {
	payload := map[string]interface{}{
		"bytesLoaded":       true,
		"bytesTotal":        true,
		"comment":           true,
		"status":            true,
		"enabled":           true,
		"eta":               true,
		"extractionStatus":  true,
		"finished":          true,
		"running":           true,
		"speed":             true,
		"url":               true,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := s.httpClient.Post(
		s.baseURL+"/downloadsV2/queryLinks",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get download list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("JDownloader returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var downloads []map[string]interface{}
	if err := json.Unmarshal(body, &downloads); err != nil {
		return nil, err
	}

	return downloads, nil
}

// AddLinksAndStart adds links and immediately starts downloading
func (s *JDownloaderService) AddLinksAndStart(links []string, packageName string, destinationDir string) error {
	if err := s.AddLinks(links, packageName, destinationDir); err != nil {
		return err
	}

	// Give JDownloader a moment to process the links
	time.Sleep(500 * time.Millisecond)

	return s.StartDownloads()
}

// ClearFinishedLinks removes completed downloads from the list
func (s *JDownloaderService) ClearFinishedLinks() error {
	resp, err := s.httpClient.Get(s.baseURL + "/downloadcontroller/cleanup/action/DELETE_ALL/mode/REMOVE_LINKS_AND_DELETE_FILES/selection/FINISHED")
	if err != nil {
		return fmt.Errorf("failed to clear finished links: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("JDownloader returned status %d", resp.StatusCode)
	}

	return nil
}
