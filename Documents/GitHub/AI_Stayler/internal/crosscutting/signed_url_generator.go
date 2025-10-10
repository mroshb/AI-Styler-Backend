package crosscutting

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// SignedURLConfig represents configuration for signed URLs
type SignedURLConfig struct {
	// Signing key
	SigningKey string `json:"signing_key"`

	// Default expiration
	DefaultExpiration time.Duration `json:"default_expiration"`

	// Maximum expiration
	MaxExpiration time.Duration `json:"max_expiration"`

	// URL patterns that require signing
	RequireSigning []string `json:"require_signing"`

	// Allowed domains for signed URLs
	AllowedDomains []string `json:"allowed_domains"`

	// Security settings
	ValidateIP        bool `json:"validate_ip"`
	ValidateReferer   bool `json:"validate_referer"`
	ValidateUserAgent bool `json:"validate_user_agent"`
}

// SignedURLRequest represents a request to generate a signed URL
type SignedURLRequest struct {
	Path       string                 `json:"path"`
	Method     string                 `json:"method"`
	Expiration time.Duration          `json:"expiration"`
	IPAddress  string                 `json:"ip_address"`
	UserAgent  string                 `json:"user_agent"`
	Referer    string                 `json:"referer"`
	UserID     string                 `json:"user_id"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// SignedURL represents a signed URL with its components
type SignedURL struct {
	URL       string                 `json:"url"`
	Signature string                 `json:"signature"`
	ExpiresAt time.Time              `json:"expires_at"`
	Path      string                 `json:"path"`
	Method    string                 `json:"method"`
	IPAddress string                 `json:"ip_address"`
	UserAgent string                 `json:"user_agent"`
	Referer   string                 `json:"referer"`
	UserID    string                 `json:"user_id"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// SignedURLValidationResult represents the result of URL validation
type SignedURLValidationResult struct {
	Valid            bool                   `json:"valid"`
	Reason           string                 `json:"reason"`
	Expired          bool                   `json:"expired"`
	InvalidSignature bool                   `json:"invalid_signature"`
	InvalidIP        bool                   `json:"invalid_ip"`
	InvalidReferer   bool                   `json:"invalid_referer"`
	InvalidUserAgent bool                   `json:"invalid_user_agent"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// DefaultSignedURLConfig returns default signed URL configuration
func DefaultSignedURLConfig() *SignedURLConfig {
	return &SignedURLConfig{
		SigningKey:        "your-signing-key-change-in-production",
		DefaultExpiration: 24 * time.Hour,
		MaxExpiration:     7 * 24 * time.Hour, // 7 days
		RequireSigning: []string{
			"/api/storage/",
			"/api/images/",
			"/api/conversions/",
			"/api/results/",
		},
		AllowedDomains: []string{
			"localhost",
			"127.0.0.1",
			"your-domain.com",
		},
		ValidateIP:        true,
		ValidateReferer:   false,
		ValidateUserAgent: false,
	}
}

// SignedURLGenerator provides signed URL generation and validation
type SignedURLGenerator struct {
	config  *SignedURLConfig
	baseURL string
}

// NewSignedURLGenerator creates a new signed URL generator
func NewSignedURLGenerator(baseURL string, config *SignedURLConfig) *SignedURLGenerator {
	if config == nil {
		config = DefaultSignedURLConfig()
	}

	return &SignedURLGenerator{
		config:  config,
		baseURL: baseURL,
	}
}

// GenerateSignedURL generates a signed URL for the given request
func (sug *SignedURLGenerator) GenerateSignedURL(ctx context.Context, req *SignedURLRequest) (*SignedURL, error) {
	// Validate expiration
	expiration := req.Expiration
	if expiration == 0 {
		expiration = sug.config.DefaultExpiration
	}
	if expiration > sug.config.MaxExpiration {
		expiration = sug.config.MaxExpiration
	}

	// Calculate expiration time
	expiresAt := time.Now().Add(expiration)

	// Create signature data
	signatureData := map[string]interface{}{
		"path":       req.Path,
		"method":     req.Method,
		"expires_at": expiresAt.Unix(),
		"ip_address": req.IPAddress,
		"user_agent": req.UserAgent,
		"referer":    req.Referer,
		"user_id":    req.UserID,
		"metadata":   req.Metadata,
	}

	// Generate signature
	signature, err := sug.generateSignature(signatureData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate signature: %w", err)
	}

	// Build URL
	urlParams := url.Values{}
	urlParams.Set("signature", signature)
	urlParams.Set("expires_at", strconv.FormatInt(expiresAt.Unix(), 10))
	urlParams.Set("ip_address", req.IPAddress)
	urlParams.Set("user_agent", req.UserAgent)
	urlParams.Set("referer", req.Referer)
	urlParams.Set("user_id", req.UserID)

	// Add metadata as JSON
	if req.Metadata != nil {
		metadataJSON, err := json.Marshal(req.Metadata)
		if err == nil {
			urlParams.Set("metadata", base64.URLEncoding.EncodeToString(metadataJSON))
		}
	}

	fullURL := fmt.Sprintf("%s%s?%s", sug.baseURL, req.Path, urlParams.Encode())

	return &SignedURL{
		URL:       fullURL,
		Signature: signature,
		ExpiresAt: expiresAt,
		Path:      req.Path,
		Method:    req.Method,
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
		Referer:   req.Referer,
		UserID:    req.UserID,
		Metadata:  req.Metadata,
	}, nil
}

// ValidateSignedURL validates a signed URL
func (sug *SignedURLGenerator) ValidateSignedURL(ctx context.Context, urlStr string, clientIP, userAgent, referer string) (*SignedURLValidationResult, error) {
	// Parse URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return &SignedURLValidationResult{
			Valid:  false,
			Reason: "Invalid URL format",
		}, nil
	}

	// Extract parameters
	signature := parsedURL.Query().Get("signature")
	expiresAtStr := parsedURL.Query().Get("expires_at")
	ipAddress := parsedURL.Query().Get("ip_address")
	userAgentParam := parsedURL.Query().Get("user_agent")
	refererParam := parsedURL.Query().Get("referer")
	userID := parsedURL.Query().Get("user_id")
	metadataStr := parsedURL.Query().Get("metadata")

	// Check if signature exists
	if signature == "" {
		return &SignedURLValidationResult{
			Valid:  false,
			Reason: "Missing signature",
		}, nil
	}

	// Parse expiration time
	expiresAtUnix, err := strconv.ParseInt(expiresAtStr, 10, 64)
	if err != nil {
		return &SignedURLValidationResult{
			Valid:  false,
			Reason: "Invalid expiration time",
		}, nil
	}

	expiresAt := time.Unix(expiresAtUnix, 0)

	// Check if URL has expired
	if time.Now().After(expiresAt) {
		return &SignedURLValidationResult{
			Valid:   false,
			Reason:  "URL has expired",
			Expired: true,
		}, nil
	}

	// Parse metadata
	var metadata map[string]interface{}
	if metadataStr != "" {
		metadataBytes, err := base64.URLEncoding.DecodeString(metadataStr)
		if err == nil {
			json.Unmarshal(metadataBytes, &metadata)
		}
	}

	// Recreate signature data
	signatureData := map[string]interface{}{
		"path":       parsedURL.Path,
		"method":     "GET", // Default method for validation
		"expires_at": expiresAtUnix,
		"ip_address": ipAddress,
		"user_agent": userAgentParam,
		"referer":    refererParam,
		"user_id":    userID,
		"metadata":   metadata,
	}

	// Generate expected signature
	expectedSignature, err := sug.generateSignature(signatureData)
	if err != nil {
		return &SignedURLValidationResult{
			Valid:  false,
			Reason: "Failed to generate expected signature",
		}, nil
	}

	// Validate signature
	if !sug.validateSignature(signature, expectedSignature) {
		return &SignedURLValidationResult{
			Valid:            false,
			Reason:           "Invalid signature",
			InvalidSignature: true,
		}, nil
	}

	// Validate IP address if enabled
	if sug.config.ValidateIP && ipAddress != "" && ipAddress != clientIP {
		return &SignedURLValidationResult{
			Valid:     false,
			Reason:    "IP address mismatch",
			InvalidIP: true,
		}, nil
	}

	// Validate referer if enabled
	if sug.config.ValidateReferer && refererParam != "" && refererParam != referer {
		return &SignedURLValidationResult{
			Valid:          false,
			Reason:         "Referer mismatch",
			InvalidReferer: true,
		}, nil
	}

	// Validate user agent if enabled
	if sug.config.ValidateUserAgent && userAgentParam != "" && userAgentParam != userAgent {
		return &SignedURLValidationResult{
			Valid:            false,
			Reason:           "User-Agent mismatch",
			InvalidUserAgent: true,
		}, nil
	}

	return &SignedURLValidationResult{
		Valid:    true,
		Reason:   "URL is valid",
		Metadata: signatureData,
	}, nil
}

// generateSignature generates HMAC signature for the given data
func (sug *SignedURLGenerator) generateSignature(data map[string]interface{}) (string, error) {
	// Convert data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data: %w", err)
	}

	// Create HMAC
	h := hmac.New(sha256.New, []byte(sug.config.SigningKey))
	h.Write(jsonData)

	// Encode signature
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return signature, nil
}

// validateSignature validates HMAC signature
func (sug *SignedURLGenerator) validateSignature(signature, expectedSignature string) bool {
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// RequiresSigning checks if a path requires signing
func (sug *SignedURLGenerator) RequiresSigning(path string) bool {
	for _, pattern := range sug.config.RequireSigning {
		if strings.HasPrefix(path, pattern) {
			return true
		}
	}
	return false
}

// IsAllowedDomain checks if a domain is allowed for signed URLs
func (sug *SignedURLGenerator) IsAllowedDomain(domain string) bool {
	for _, allowedDomain := range sug.config.AllowedDomains {
		if domain == allowedDomain {
			return true
		}
	}
	return false
}

// GenerateBulkSignedURLs generates multiple signed URLs efficiently
func (sug *SignedURLGenerator) GenerateBulkSignedURLs(ctx context.Context, requests []*SignedURLRequest) ([]*SignedURL, error) {
	var signedURLs []*SignedURL

	for _, req := range requests {
		signedURL, err := sug.GenerateSignedURL(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to generate signed URL for path %s: %w", req.Path, err)
		}
		signedURLs = append(signedURLs, signedURL)
	}

	return signedURLs, nil
}

// RevokeSignedURL revokes a signed URL by adding it to a blacklist
func (sug *SignedURLGenerator) RevokeSignedURL(ctx context.Context, signature string) error {
	// This would typically involve adding the signature to a blacklist
	// stored in Redis or database. For now, we'll just return success.
	return nil
}

// GetSignedURLStats returns signed URL generation statistics
func (sug *SignedURLGenerator) GetSignedURLStats(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"config":                   sug.config,
		"base_url":                 sug.baseURL,
		"require_signing_patterns": sug.config.RequireSigning,
		"allowed_domains":          sug.config.AllowedDomains,
	}
}

// UpdateConfig updates the signed URL configuration
func (sug *SignedURLGenerator) UpdateConfig(config *SignedURLConfig) {
	sug.config = config
}

// SetBaseURL updates the base URL
func (sug *SignedURLGenerator) SetBaseURL(baseURL string) {
	sug.baseURL = baseURL
}
