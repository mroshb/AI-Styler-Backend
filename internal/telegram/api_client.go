package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/sony/gobreaker"
)

// APIClient handles communication with the backend API
type APIClient struct {
	baseURL     string
	apiKey      string
	httpClient  *http.Client
	circuitBreaker *gobreaker.CircuitBreaker
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL, apiKey string, timeout time.Duration) *APIClient {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "backend-api",
		MaxRequests: 3,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 5
		},
	})

	return &APIClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		circuitBreaker: cb,
	}
}

// APIResponse represents a generic API response
type APIResponse struct {
	Data  interface{} `json:"data,omitempty"`
	Error *APIError   `json:"error,omitempty"`
}

// APIError represents an API error
type APIError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// SendOTPRequest represents send OTP request
type SendOTPRequest struct {
	Phone   string `json:"phone"`
	Purpose string `json:"purpose"`
	Channel string `json:"channel"`
}

// SendOTPResponse represents send OTP response
type SendOTPResponse struct {
	Sent         bool   `json:"sent"`
	ExpiresInSec int    `json:"expiresInSec"`
	Code         string `json:"code,omitempty"`
}

// VerifyOTPRequest represents verify OTP request
type VerifyOTPRequest struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

// VerifyOTPResponse represents verify OTP response
type VerifyOTPResponse struct {
	Verified bool `json:"verified"`
}

// CheckUserRequest represents check user request
type CheckUserRequest struct {
	Phone string `json:"phone"`
}

// CheckUserResponse represents check user response
type CheckUserResponse struct {
	Registered bool `json:"registered"`
}

// RegisterRequest represents registration request
type RegisterRequest struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Role     string `json:"role"`
}

// RegisterResponse represents registration response
type RegisterResponse struct {
	UserID          string `json:"userId"`
	Role            string `json:"role"`
	IsPhoneVerified bool   `json:"isPhoneVerified"`
	AccessToken     string `json:"accessToken,omitempty"`
	AccessExpiresIn int    `json:"accessTokenExpiresIn,omitempty"`
	RefreshToken    string `json:"refreshToken,omitempty"`
	RefreshExpires  string `json:"refreshTokenExpiresAt,omitempty"`
}

// LoginRequest represents login request
type LoginRequest struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

// LoginResponse represents login response
type LoginResponse struct {
	AccessToken     string `json:"accessToken"`
	AccessExpiresIn int    `json:"accessTokenExpiresIn"`
	RefreshToken    string `json:"refreshToken"`
	RefreshExpires  string `json:"refreshTokenExpiresAt"`
	User            struct {
		ID              string `json:"id"`
		Role            string `json:"role"`
		IsPhoneVerified bool   `json:"isPhoneVerified"`
	} `json:"user"`
}

// ImageUploadResponse represents image upload response
type ImageUploadResponse struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Type     string `json:"type"`
	FileName string `json:"fileName"`
	FileSize int64  `json:"fileSize"`
}

// ConversionRequest represents conversion creation request
type ConversionRequest struct {
	UserImageID string `json:"userImageId"`
	ClothImageID string `json:"clothImageId"`
	StyleName   string `json:"styleName,omitempty"`
}

// ConversionResponse represents conversion response
type ConversionResponse struct {
	ID               string     `json:"id"`
	UserID           string     `json:"userId"`
	UserImageID      string     `json:"userImageId"`
	ClothImageID     string     `json:"clothImageId"`
	Status           string     `json:"status"`
	ResultImageID    *string    `json:"resultImageId,omitempty"`
	ErrorMessage     *string    `json:"errorMessage,omitempty"`
	ProcessingTimeMs *int       `json:"processingTimeMs,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
	CompletedAt      *time.Time `json:"completedAt,omitempty"`
}

// ConversionsListResponse represents conversions list response
type ConversionsListResponse struct {
	Conversions []ConversionResponse `json:"conversions"`
	Total       int                  `json:"total"`
	Page        int                  `json:"page"`
	PageSize    int                  `json:"pageSize"`
	TotalPages  int                  `json:"totalPages"`
}

// doRequest performs an HTTP request with circuit breaker
func (c *APIClient) doRequest(ctx context.Context, method, endpoint string, body interface{}, headers map[string]string) (*http.Response, error) {
	url := c.baseURL + endpoint

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	// Set custom headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Execute with circuit breaker
	var resp *http.Response
	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		httpResp, httpErr := c.httpClient.Do(req)
		if httpErr != nil {
			return nil, httpErr
		}
		if httpResp.StatusCode >= 500 {
			httpResp.Body.Close()
			return nil, fmt.Errorf("server error: %d", httpResp.StatusCode)
		}
		return httpResp, nil
	})

	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Type assert the result
	var ok bool
	resp, ok = result.(*http.Response)
	if !ok {
		return nil, fmt.Errorf("unexpected response type from circuit breaker")
	}

	return resp, nil
}

// SendOTP sends OTP to phone number
func (c *APIClient) SendOTP(ctx context.Context, phone string) (*SendOTPResponse, error) {
	req := SendOTPRequest{
		Phone:   phone,
		Purpose: "phone_verify",
		Channel: "sms",
	}

	resp, err := c.doRequest(ctx, "POST", "/auth/send-otp", req, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result SendOTPResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	return &result, nil
}

// VerifyOTP verifies OTP code
func (c *APIClient) VerifyOTP(ctx context.Context, phone, code string) (*VerifyOTPResponse, error) {
	req := VerifyOTPRequest{
		Phone: phone,
		Code:  code,
	}

	resp, err := c.doRequest(ctx, "POST", "/auth/verify-otp", req, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result VerifyOTPResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	return &result, nil
}

// CheckUser checks if a user exists
func (c *APIClient) CheckUser(ctx context.Context, phone string) (bool, error) {
	req := CheckUserRequest{
		Phone: phone,
	}

	resp, err := c.doRequest(ctx, "POST", "/auth/check-user", req, nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result CheckUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	return result.Registered, nil
}

// Register registers a new user
func (c *APIClient) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	resp, err := c.doRequest(ctx, "POST", "/auth/register", req, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	return &result, nil
}

// Login logs in a user
func (c *APIClient) Login(ctx context.Context, phone, password string) (*LoginResponse, error) {
	req := LoginRequest{
		Phone:    phone,
		Password: password,
	}

	resp, err := c.doRequest(ctx, "POST", "/auth/login", req, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	return &result, nil
}

// UploadImage uploads an image to the backend
func (c *APIClient) UploadImage(ctx context.Context, accessToken string, fileData []byte, fileName, mimeType, imageType string) (*ImageUploadResponse, error) {
	url := c.baseURL + "/api/images"

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add file
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(fileData); err != nil {
		return nil, fmt.Errorf("failed to write file data: %w", err)
	}

	// Add type
	if err := writer.WriteField("type", imageType); err != nil {
		return nil, fmt.Errorf("failed to write type field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var result ImageUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	return &result, nil
}

// CreateConversion creates a new conversion
func (c *APIClient) CreateConversion(ctx context.Context, accessToken string, req ConversionRequest) (*ConversionResponse, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}

	resp, err := c.doRequest(ctx, "POST", "/api/convert", req, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result ConversionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	return &result, nil
}

// GetConversion gets conversion details
func (c *APIClient) GetConversion(ctx context.Context, accessToken, conversionID string) (*ConversionResponse, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}

	resp, err := c.doRequest(ctx, "GET", "/api/conversion/"+conversionID, nil, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result ConversionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	return &result, nil
}

// ListConversions lists user conversions
func (c *APIClient) ListConversions(ctx context.Context, accessToken string, page, pageSize int, status string) (*ConversionsListResponse, error) {
	endpoint := fmt.Sprintf("/api/conversions?page=%d&pageSize=%d", page, pageSize)
	if status != "" {
		endpoint += "&status=" + status
	}

	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}

	resp, err := c.doRequest(ctx, "GET", endpoint, nil, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result ConversionsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	return &result, nil
}

// GetImageURL gets image URL (if available)
func (c *APIClient) GetImageURL(ctx context.Context, accessToken, imageID string) (string, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}

	resp, err := c.doRequest(ctx, "GET", "/api/images/"+imageID, nil, headers)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Backend returns Image struct with originalUrl field
	var result struct {
		ID          string `json:"id"`
		OriginalURL string `json:"originalUrl"`
		ThumbnailURL *string `json:"thumbnailUrl,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error: %d", resp.StatusCode)
	}

	// Prefer originalUrl, fallback to thumbnailUrl if available
	if result.OriginalURL != "" {
		return result.OriginalURL, nil
	}
	if result.ThumbnailURL != nil && *result.ThumbnailURL != "" {
		return *result.ThumbnailURL, nil
	}

	return "", fmt.Errorf("no URL found in image response")
}

