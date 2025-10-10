package worker

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// GeminiClient implements the GeminiAPI interface
type GeminiClient struct {
	config     *GeminiConfig
	httpClient *http.Client
}

// NewGeminiClient creates a new Gemini API client
func NewGeminiClient(config *GeminiConfig) *GeminiClient {
	if config == nil {
		config = getDefaultGeminiConfig()
	}

	return &GeminiClient{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
	}
}

// ConvertImage converts an image using Gemini API with comprehensive error handling
func (c *GeminiClient) ConvertImage(ctx context.Context, userImageData, clothImageData []byte, options map[string]interface{}) ([]byte, error) {
	// Validate input data
	if len(userImageData) == 0 {
		return nil, fmt.Errorf("user image data is empty")
	}
	if len(clothImageData) == 0 {
		return nil, fmt.Errorf("cloth image data is empty")
	}

	// Check file size limits
	maxSize := int64(10 * 1024 * 1024) // 10MB
	if int64(len(userImageData)) > maxSize {
		return nil, fmt.Errorf("user image too large: %d bytes (max: %d)", len(userImageData), maxSize)
	}
	if int64(len(clothImageData)) > maxSize {
		return nil, fmt.Errorf("cloth image too large: %d bytes (max: %d)", len(clothImageData), maxSize)
	}

	// Detect MIME types
	userMimeType, err := c.detectMimeType(userImageData)
	if err != nil {
		return nil, fmt.Errorf("failed to detect user image MIME type: %w", err)
	}

	clothMimeType, err := c.detectMimeType(clothImageData)
	if err != nil {
		return nil, fmt.Errorf("failed to detect cloth image MIME type: %w", err)
	}

	// Validate MIME types
	if !c.isSupportedMimeType(userMimeType) {
		return nil, fmt.Errorf("unsupported user image type: %s", userMimeType)
	}
	if !c.isSupportedMimeType(clothMimeType) {
		return nil, fmt.Errorf("unsupported cloth image type: %s", clothMimeType)
	}

	// Encode images to base64
	userImageBase64 := base64.StdEncoding.EncodeToString(userImageData)
	clothImageBase64 := base64.StdEncoding.EncodeToString(clothImageData)

	// Build the prompt
	prompt := c.buildConversionPrompt(options)

	// Create the request
	request := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{
						Text: prompt,
					},
					{
						InlineData: &GeminiInlineData{
							MimeType: userMimeType,
							Data:     userImageBase64,
						},
					},
					{
						InlineData: &GeminiInlineData{
							MimeType: clothMimeType,
							Data:     clothImageBase64,
						},
					},
				},
			},
		},
		GenerationConfig: GeminiGenerationConfig{
			Temperature:     0.7,
			TopK:            40,
			TopP:            0.95,
			MaxOutputTokens: 1024,
		},
	}

	// Make the API call with retries and timeout handling
	var response *GeminiResponse
	var lastErr error

	for attempt := 0; attempt < c.config.MaxRetries; attempt++ {
		// Create context with timeout for this attempt
		attemptCtx, cancel := context.WithTimeout(ctx, time.Duration(c.config.Timeout)*time.Second)

		response, lastErr = c.makeAPIRequest(attemptCtx, request)
		cancel()

		if lastErr == nil {
			break
		}

		// Log the attempt
		log.Printf("Gemini API attempt %d/%d failed: %v", attempt+1, c.config.MaxRetries, lastErr)

		// Check if error is retryable
		if !c.isRetryableError(lastErr) {
			return nil, fmt.Errorf("non-retryable error: %w", lastErr)
		}

		// Wait before retry with exponential backoff
		if attempt < c.config.MaxRetries-1 {
			delay := c.calculateRetryDelay(attempt)
			log.Printf("Retrying in %v...", delay)
			time.Sleep(delay)
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to call Gemini API after %d attempts: %w", c.config.MaxRetries, lastErr)
	}

	// Extract the result image from the response
	resultImageData, err := c.extractResultImage(response)
	if err != nil {
		return nil, fmt.Errorf("failed to extract result image: %w", err)
	}

	// Validate result image
	if len(resultImageData) == 0 {
		return nil, fmt.Errorf("empty result image received from Gemini API")
	}

	return resultImageData, nil
}

// GetConversionStatus gets the status of a conversion (not applicable for Gemini API)
func (c *GeminiClient) GetConversionStatus(ctx context.Context, jobID string) (string, error) {
	// Gemini API doesn't support status checking for individual conversions
	// This would typically be handled by the job queue system
	return "completed", nil
}

// CancelConversion cancels a conversion (not applicable for Gemini API)
func (c *GeminiClient) CancelConversion(ctx context.Context, jobID string) error {
	// Gemini API doesn't support cancellation of individual conversions
	// This would typically be handled by the job queue system
	return fmt.Errorf("cancellation not supported for Gemini API")
}

// HealthCheck checks the health of the Gemini API
func (c *GeminiClient) HealthCheck(ctx context.Context) error {
	// Make a simple test request to check API availability
	testRequest := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{
						Text: "Hello, are you working?",
					},
				},
			},
		},
		GenerationConfig: GeminiGenerationConfig{
			Temperature:     0.1,
			MaxOutputTokens: 10,
		},
	}

	_, err := c.makeAPIRequest(ctx, testRequest)
	return err
}

// makeAPIRequest makes an HTTP request to the Gemini API
func (c *GeminiClient) makeAPIRequest(ctx context.Context, request GeminiRequest) (*GeminiResponse, error) {
	// Marshal request to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", c.config.BaseURL, c.config.Model)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	// Make the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var response GeminiResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for errors in response
	if len(response.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	candidate := response.Candidates[0]
	if candidate.FinishReason != "STOP" {
		return nil, fmt.Errorf("unexpected finish reason: %s", candidate.FinishReason)
	}

	return &response, nil
}

// buildConversionPrompt builds the prompt for image conversion
func (c *GeminiClient) buildConversionPrompt(options map[string]interface{}) string {
	basePrompt := `Please convert the first image (person) to wear the clothing from the second image (clothing item). 
	The result should be a realistic image where the person is wearing the clothing item. 
	Maintain the person's pose, facial features, and body proportions while accurately applying the clothing.
	The clothing should fit naturally and look realistic.
	Return the result as a high-quality image.`

	// Add custom options if provided
	if style, ok := options["style"].(string); ok && style != "" {
		basePrompt += fmt.Sprintf(" Apply the style: %s.", style)
	}

	if quality, ok := options["quality"].(string); ok && quality != "" {
		basePrompt += fmt.Sprintf(" Ensure high quality: %s.", quality)
	}

	return basePrompt
}

// extractResultImage extracts the result image from the Gemini response
func (c *GeminiClient) extractResultImage(response *GeminiResponse) ([]byte, error) {
	if len(response.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	candidate := response.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return nil, fmt.Errorf("no parts in candidate content")
	}

	// Look for image data in the response
	for _, part := range candidate.Content.Parts {
		if part.InlineData != nil {
			// Decode base64 image data
			imageData, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
			if err != nil {
				continue // Try next part
			}
			return imageData, nil
		}
	}

	// If no image data found, this might be a text response
	// In a real implementation, you might need to handle this differently
	// or use a different approach to get the image
	return nil, fmt.Errorf("no image data found in response")
}

// getDefaultGeminiConfig returns default Gemini configuration
func getDefaultGeminiConfig() *GeminiConfig {
	return &GeminiConfig{
		APIKey:     "", // Should be set from environment
		BaseURL:    "https://generativelanguage.googleapis.com",
		Model:      "gemini-1.5-pro",
		MaxRetries: 3,
		Timeout:    60,
	}
}

// GeminiImageConverter is a specialized converter for image-to-image tasks
type GeminiImageConverter struct {
	client *GeminiClient
}

// NewGeminiImageConverter creates a new image converter
func NewGeminiImageConverter(config *GeminiConfig) *GeminiImageConverter {
	return &GeminiImageConverter{
		client: NewGeminiClient(config),
	}
}

// Convert performs the actual image conversion
func (converter *GeminiImageConverter) Convert(ctx context.Context, userImageData, clothImageData []byte, options map[string]interface{}) ([]byte, error) {
	return converter.client.ConvertImage(ctx, userImageData, clothImageData, options)
}

// ValidateInputs validates the input images before conversion
func (converter *GeminiImageConverter) ValidateInputs(userImageData, clothImageData []byte) error {
	if len(userImageData) == 0 {
		return fmt.Errorf("user image data is empty")
	}

	if len(clothImageData) == 0 {
		return fmt.Errorf("cloth image data is empty")
	}

	// Add more validation as needed (file size, format, etc.)
	return nil
}

// GetSupportedFormats returns the supported image formats
func (converter *GeminiImageConverter) GetSupportedFormats() []string {
	return []string{"image/jpeg", "image/png", "image/webp"}
}

// detectMimeType detects the MIME type of the file
func (c *GeminiClient) detectMimeType(fileData []byte) (string, error) {
	if len(fileData) < 4 {
		return "", fmt.Errorf("file too small to determine type")
	}

	// JPEG
	if fileData[0] == 0xFF && fileData[1] == 0xD8 {
		return "image/jpeg", nil
	}

	// PNG
	if fileData[0] == 0x89 && fileData[1] == 0x50 && fileData[2] == 0x4E && fileData[3] == 0x47 {
		return "image/png", nil
	}

	// WebP
	if len(fileData) >= 12 &&
		fileData[0] == 0x52 && fileData[1] == 0x49 && fileData[2] == 0x46 && fileData[3] == 0x46 &&
		fileData[8] == 0x57 && fileData[9] == 0x45 && fileData[10] == 0x42 && fileData[11] == 0x50 {
		return "image/webp", nil
	}

	// GIF
	if len(fileData) >= 6 &&
		fileData[0] == 0x47 && fileData[1] == 0x49 && fileData[2] == 0x46 {
		return "image/gif", nil
	}

	return "application/octet-stream", fmt.Errorf("unknown file type")
}

// isSupportedMimeType checks if the MIME type is supported
func (c *GeminiClient) isSupportedMimeType(mimeType string) bool {
	supportedTypes := []string{"image/jpeg", "image/png", "image/webp", "image/gif"}
	for _, supportedType := range supportedTypes {
		if mimeType == supportedType {
			return true
		}
	}
	return false
}

// isRetryableError checks if an error is retryable
func (c *GeminiClient) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Retryable errors
	retryableErrors := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"network unreachable",
		"temporary failure",
		"service unavailable",
		"too many requests",
		"rate limit",
		"server error",
		"internal server error",
		"bad gateway",
		"gateway timeout",
		"request timeout",
		"context deadline exceeded",
	}

	for _, retryableError := range retryableErrors {
		if contains(errStr, retryableError) {
			return true
		}
	}

	return false
}

// calculateRetryDelay calculates retry delay with exponential backoff
func (c *GeminiClient) calculateRetryDelay(attempt int) time.Duration {
	baseDelay := time.Second
	delay := time.Duration(1<<uint(attempt)) * baseDelay

	// Cap at 5 minutes
	maxDelay := 5 * time.Minute
	if delay > maxDelay {
		delay = maxDelay
	}

	return delay
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstring(s, substr)))
}

// containsSubstring performs case-insensitive substring search
func containsSubstring(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return strings.Contains(s, substr)
}
