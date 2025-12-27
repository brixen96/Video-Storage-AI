package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/brixen96/video-storage-ai/internal/config"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// AdultDataLinkService handles fetching metadata from the AdultDataLink API
type AdultDataLinkService struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewAdultDataLinkService creates a new AdultDataLink API service
func NewAdultDataLinkService(cfg *config.Config) *AdultDataLinkService {
	return &AdultDataLinkService{
		apiKey:  cfg.API.AdultDataLinkAPIKey,
		baseURL: "https://api.adultdatalink.com",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchPerformerData fetches performer metadata from AdultDataLink API
func (s *AdultDataLinkService) FetchPerformerData(performerName string) (*models.PerformerMetadata, error) {
	// Replace spaces with %20 for URL encoding
	encodedName := url.QueryEscape(performerName)
	apiURL := fmt.Sprintf("%s/pornstar/pornstar-data?name=%s", s.baseURL, encodedName)

	// Create request
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authorization header
	req.Header.Set("Authorization", "Bearer " + s.apiKey)

	// Make request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data from API: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response into a generic map first
	var apiResponse map[string]interface{}
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	// Convert API response to our metadata structure
	metadata := s.convertAPIResponseToMetadata(apiResponse)

	return metadata, nil
}

// convertAPIResponseToMetadata converts the AdultDataLink API response to our metadata structure
func (s *AdultDataLinkService) convertAPIResponseToMetadata(apiResponse map[string]interface{}) *models.PerformerMetadata {
	metadata := &models.PerformerMetadata{
		AdultDataLinkResponse: apiResponse,
	}

	// Extract commonly used fields from the API response
	if name, ok := apiResponse["name"].(string); ok {
		// Name is at root level
		_ = name
	}

	if dob, ok := apiResponse["date_of_birth"].(string); ok {
		metadata.Birthdate = dob
	}

	if pob, ok := apiResponse["place_of_birth"].(string); ok {
		metadata.Birthplace = pob
	}

	if age, ok := apiResponse["age"].(string); ok {
		// Age might be a string, store as-is in bio for now
		if metadata.Bio == "" {
			metadata.Bio = fmt.Sprintf("Age: %s", age)
		}
	}

	// Extract appearance data
	if appearance, ok := apiResponse["appearance"].(map[string]interface{}); ok {
		if height, ok := appearance["height"].(string); ok {
			metadata.Height = height
		}
		if weight, ok := appearance["weight"].(string); ok {
			metadata.Weight = weight
		}
		if ethnicity, ok := appearance["ethnicity"].(string); ok {
			metadata.Ethnicity = ethnicity
		}
		if hair, ok := appearance["hair_color"].(string); ok {
			metadata.HairColor = hair
		}
		if eyes, ok := appearance["eye_color"].(string); ok {
			metadata.EyeColor = eyes
		}
		if measurements, ok := appearance["measurements"].(string); ok {
			metadata.Measurements = measurements
		}
		if tattoos, ok := appearance["tattoos"].(string); ok {
			metadata.Tattoos = tattoos
		}
		if piercings, ok := appearance["piercings"].(string); ok {
			metadata.Piercings = piercings
		}
	}

	// Extract career info
	if careerStart, ok := apiResponse["career_start"].(string); ok {
		// Try to parse as int
		var year int
		if _, err := fmt.Sscanf(careerStart, "%d", &year); err != nil {
			log.Printf("failed to parse career_start: %v", err)
		}
		if year > 0 {
			metadata.CareerStart = year
		}
	}

	if careerEnd, ok := apiResponse["career_end"].(string); ok {
		var year int
		if _, err := fmt.Sscanf(careerEnd, "%d", &year); err != nil {
			log.Printf("failed to parse career_end: %v", err)
		}
		if year > 0 {
			metadata.CareerEnd = year
		}
	}

	// Extract aliases
	if aliases, ok := apiResponse["aliases"].(string); ok {
		if aliases != "" {
			// Split by comma or semicolon
			aliasList := strings.Split(aliases, ",")
			for i, alias := range aliasList {
				aliasList[i] = strings.TrimSpace(alias)
			}
			metadata.Aliases = aliasList
		}
	}

	// Extract URLs from external_links array
	if externalLinks, ok := apiResponse["external_links"].([]interface{}); ok {
		urls := []string{}
		for _, link := range externalLinks {
			if linkMap, ok := link.(map[string]interface{}); ok {
				if url, ok := linkMap["url"].(string); ok {
					urls = append(urls, url)
				}
			}
		}
		metadata.URLs = urls
	}

	// Extract image URL
	if imageURL, ok := apiResponse["image_url"].(string); ok {
		metadata.ImageURL = imageURL
	}

	// Extract bios
	if bios, ok := apiResponse["bios"].(map[string]interface{}); ok {
		// Combine all bios into one
		bioTexts := []string{}
		for source, text := range bios {
			if bioText, ok := text.(string); ok && bioText != "" {
				bioTexts = append(bioTexts, fmt.Sprintf("[%s] %s", source, bioText))
			}
		}
		if len(bioTexts) > 0 {
			metadata.Bio = strings.Join(bioTexts, "\n\n")
		}
	}

	return metadata
}
