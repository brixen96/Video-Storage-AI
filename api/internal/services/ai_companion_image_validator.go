package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// ImageValidator handles AI-powered image validation using Qwen3.8-omni-flash
type ImageValidator struct {
	apiKey     string
	modelID    string
	httpClient *http.Client
}

// NewImageValidator creates a new image validator service
func NewImageValidator(apiKey string) *ImageValidator {
	return &ImageValidator{
		apiKey:  apiKey,
		modelID: "qwen/qwen3.8-omni-flash",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ImageValidationRequest represents a request to validate an image
type ImageValidationRequest struct {
	ImageURL    string `json:"image_url"`
	ThreadTitle string `json:"thread_title,omitempty"`
	Context     string `json:"context,omitempty"` // Additional context about the image
}

// ImageValidationResult represents the result of image validation
type ImageValidationResult struct {
	ImageURL      string   `json:"image_url"`
	IsValid       bool     `json:"is_valid"`
	Description   string   `json:"description"`
	QualityScore  int      `json:"quality_score"` // 1-10
	ContentTags   []string `json:"content_tags"`
	IsNSFW        bool     `json:"is_nsfw"`
	Confidence    float64  `json:"confidence"`
	Error         string   `json:"error,omitempty"`
	AnalyzedAt    string   `json:"analyzed_at"`
}

// BatchValidationRequest represents a batch validation request
type BatchValidationRequest struct {
	Images      []ImageValidationRequest `json:"images"`
	ThreadTitle string                   `json:"thread_title,omitempty"`
}

// BatchValidationResult represents batch validation results
type BatchValidationResult struct {
	Results    []ImageValidationResult `json:"results"`
	TotalCount int                     `json:"total_count"`
	ValidCount int                     `json:"valid_count"`
	InvalidCount int                   `json:"invalid_count"`
	Duration   string                  `json:"duration"`
}

// ValidateImage validates a single image using Qwen3.8-omni-flash
func (v *ImageValidator) ValidateImage(req ImageValidationRequest) (*ImageValidationResult, error) {
	startTime := time.Now()

	// Build the prompt for image analysis
	prompt := v.buildAnalysisPrompt(req)

	// Prepare the multimodal request
	requestBody, err := v.buildMultimodalRequest(prompt, req.ImageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	// Call OpenRouter API
	response, err := v.callOpenRouter(requestBody)
	if err != nil {
		return &ImageValidationResult{
			ImageURL:   req.ImageURL,
			IsValid:    false,
			Error:      err.Error(),
			AnalyzedAt: time.Now().Format(time.RFC3339),
		}, nil
	}

	// Parse the response
	result := v.parseValidationResponse(req.ImageURL, response)
	result.AnalyzedAt = time.Now().Format(time.RFC3339)

	log.Printf("Image validation completed in %v: %s", time.Since(startTime), req.ImageURL)
	return result, nil
}

// ValidateBatch validates multiple images in parallel
func (v *ImageValidator) ValidateBatch(req BatchValidationRequest) (*BatchValidationResult, error) {
	startTime := time.Now()
	results := make([]ImageValidationResult, 0, len(req.Images))

	// Process images concurrently with rate limiting
	resultChan := make(chan ImageValidationResult, len(req.Images))
 semaphore := make(chan struct{}, 5) // Max 5 concurrent requests

	for _, img := range req.Images {
		go func(imageReq ImageValidationRequest) {
			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release

			result, err := v.ValidateImage(imageReq)
			if err != nil {
				result = &ImageValidationResult{
					ImageURL:   imageReq.ImageURL,
					IsValid:    false,
					Error:      err.Error(),
					AnalyzedAt: time.Now().Format(time.RFC3339),
				}
			}
			resultChan <- *result
		}(img)
	}

	// Collect results
	for i := 0; i < len(req.Images); i++ {
		results = append(results, <-resultChan)
	}

	// Calculate statistics
	validCount := 0
	invalidCount := 0
	for _, r := range results {
		if r.IsValid {
			validCount++
		} else {
			invalidCount++
		}
	}

	return &BatchValidationResult{
		Results:      results,
		TotalCount:   len(results),
		ValidCount:   validCount,
		InvalidCount: invalidCount,
		Duration:     time.Since(startTime).String(),
	}, nil
}

// buildAnalysisPrompt creates the prompt for image analysis
func (v *ImageValidator) buildAnalysisPrompt(req ImageValidationRequest) string {
	prompt := `Analyze this image and provide the following information in JSON format:

{
  "description": "Brief description of what's in the image",
  "quality_score": <1-10 rating of image quality>,
  "content_tags": ["list", "of", "relevant", "tags"],
  "is_nsfw": <true/false if the content is NSFW>,
  "confidence": <0.0-1.0 confidence in your analysis>
}

Focus on:
1. Image quality (resolution, clarity, lighting)
2. Content relevance to the thread context
3. Whether the image appears to be valid and loadable
4. Any obvious issues (broken images, placeholders, errors)`

	if req.ThreadTitle != "" {
		prompt += fmt.Sprintf("\n\nThread context: %s", req.ThreadTitle)
	}

	if req.Context != "" {
		prompt += fmt.Sprintf("\nAdditional context: %s", req.Context)
	}

	return prompt
}

// buildMultimodalRequest creates the multimodal API request
func (v *ImageValidator) buildMultimodalRequest(prompt, imageURL string) ([]byte, error) {
	// Determine if this is a direct image URL or needs to be fetched
	imageContent := map[string]interface{}{
		"type": "image_url",
		"image_url": map[string]string{
			"url": imageURL,
		},
	}

	textContent := map[string]interface{}{
		"type": "text",
		"text": prompt,
	}

	messages := []map[string]interface{}{
		{
			"role": "user",
			"content": []interface{}{imageContent, textContent},
		},
	}

	requestBody := map[string]interface{}{
		"model":       v.modelID,
		"messages":    messages,
		"max_tokens":  1024,
		"temperature": 0.3,
	}

	return json.Marshal(requestBody)
}

// callOpenRouter makes the API call to OpenRouter
func (v *ImageValidator) callOpenRouter(body []byte) (string, error) {
	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.apiKey)
	req.Header.Set("HTTP-Referer", "https://video-storage-ai.com")
	req.Header.Set("X-Title", "Video Storage AI")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return response.Choices[0].Message.Content, nil
}

// parseValidationResponse parses the AI response into a structured result
func (v *ImageValidator) parseValidationResponse(imageURL, response string) *ImageValidationResult {
	result := &ImageValidationResult{
		ImageURL: imageURL,
	}

	// Try to extract JSON from the response
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart >= 0 && jsonEnd > jsonStart {
		jsonStr := response[jsonStart : jsonEnd+1]

		var parsed struct {
			Description  string   `json:"description"`
			QualityScore int      `json:"quality_score"`
			ContentTags  []string `json:"content_tags"`
			IsNSFW       bool     `json:"is_nsfw"`
			Confidence   float64  `json:"confidence"`
		}

		if err := json.Unmarshal([]byte(jsonStr), &parsed); err == nil {
			result.Description = parsed.Description
			result.QualityScore = parsed.QualityScore
			result.ContentTags = parsed.ContentTags
			result.IsNSFW = parsed.IsNSFW
			result.Confidence = parsed.Confidence
			result.IsValid = parsed.QualityScore >= 3 && parsed.Confidence >= 0.5
			return result
		}
	}

	// Fallback: parse response text
	result.Description = response
	result.QualityScore = 5
	result.IsValid = true
	result.Confidence = 0.3
	result.ContentTags = []string{}

	return result
}

// ValidateThreadImages validates all images in a thread
func (v *ImageValidator) ValidateThreadImages(threadTitle string, imageURLs []string) (*BatchValidationResult, error) {
	images := make([]ImageValidationRequest, len(imageURLs))
	for i, url := range imageURLs {
		images[i] = ImageValidationRequest{
			ImageURL:    url,
			ThreadTitle: threadTitle,
		}
	}

	return v.ValidateBatch(BatchValidationRequest{
		Images:      images,
		ThreadTitle: threadTitle,
	})
}

// DownloadAndValidateImage downloads an image and validates it
func (v *ImageValidator) DownloadAndValidateImage(imageURL string) (*ImageValidationResult, error) {
	// For local files or data URLs, we might need to convert to base64
	if strings.HasPrefix(imageURL, "data:") {
		return v.ValidateImage(ImageValidationRequest{ImageURL: imageURL})
	}

	// For remote URLs, validate directly
	return v.ValidateImage(ImageValidationRequest{ImageURL: imageURL})
}

// Helper function to encode image to base64 (for future use if needed)
func imageToBase64(imageData []byte) string {
	return base64.StdEncoding.EncodeToString(imageData)
}
