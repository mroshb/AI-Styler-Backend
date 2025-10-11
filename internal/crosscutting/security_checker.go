package crosscutting

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// SecurityConfig represents configuration for security checks
type SecurityConfig struct {
	// File upload limits
	MaxFileSize  int64    `json:"max_file_size"`
	AllowedTypes []string `json:"allowed_types"`
	BlockedTypes []string `json:"blocked_types"`

	// Virus scanning
	VirusScanEnabled bool          `json:"virus_scan_enabled"`
	VirusScanAPI     string        `json:"virus_scan_api"`
	VirusScanTimeout time.Duration `json:"virus_scan_timeout"`

	// Payload inspection
	PayloadInspectionEnabled bool     `json:"payload_inspection_enabled"`
	MaxPayloadSize           int64    `json:"max_payload_size"`
	BlockedPatterns          []string `json:"blocked_patterns"`

	// Image-specific checks
	ImageValidationEnabled bool `json:"image_validation_enabled"`
	MaxImageWidth          int  `json:"max_image_width"`
	MaxImageHeight         int  `json:"max_image_height"`
	MinImageWidth          int  `json:"min_image_width"`
	MinImageHeight         int  `json:"min_image_height"`

	// Security headers
	SecurityHeadersEnabled bool `json:"security_headers_enabled"`

	// Rate limiting for uploads
	UploadRateLimitEnabled bool          `json:"upload_rate_limit_enabled"`
	UploadRateLimitPerIP   int           `json:"upload_rate_limit_per_ip"`
	UploadRateLimitWindow  time.Duration `json:"upload_rate_limit_window"`
}

// SecurityCheckResult represents the result of a security check
type SecurityCheckResult struct {
	Allowed  bool                   `json:"allowed"`
	Reason   string                 `json:"reason"`
	Details  map[string]interface{} `json:"details"`
	Threats  []Threat               `json:"threats"`
	Metadata map[string]interface{} `json:"metadata"`
}

// Threat represents a security threat detected
type Threat struct {
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	Location    string                 `json:"location"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		MaxFileSize: 50 * 1024 * 1024, // 50MB
		AllowedTypes: []string{
			"image/jpeg",
			"image/jpg",
			"image/png",
			"image/gif",
			"image/webp",
			"image/bmp",
			"image/tiff",
		},
		BlockedTypes: []string{
			"application/x-executable",
			"application/x-msdownload",
			"application/x-msdos-program",
			"application/x-winexe",
			"application/x-javascript",
			"text/javascript",
			"application/javascript",
			"application/x-sh",
			"application/x-csh",
			"text/x-script",
		},
		VirusScanEnabled:         true,
		VirusScanAPI:             "https://api.virustotal.com/v2/file/scan",
		VirusScanTimeout:         30 * time.Second,
		PayloadInspectionEnabled: true,
		MaxPayloadSize:           10 * 1024 * 1024, // 10MB
		BlockedPatterns: []string{
			"<script",
			"javascript:",
			"vbscript:",
			"onload=",
			"onerror=",
			"onclick=",
			"eval(",
			"document.cookie",
			"document.write",
			"window.location",
		},
		ImageValidationEnabled: true,
		MaxImageWidth:          4096,
		MaxImageHeight:         4096,
		MinImageWidth:          100,
		MinImageHeight:         100,
		SecurityHeadersEnabled: true,
		UploadRateLimitEnabled: true,
		UploadRateLimitPerIP:   10,
		UploadRateLimitWindow:  time.Hour,
	}
}

// SecurityChecker provides comprehensive security checking functionality
type SecurityChecker struct {
	config         *SecurityConfig
	scanner        VirusScanner
	inspector      PayloadInspector
	imageValidator ImageValidator
}

// VirusScanner interface for virus scanning
type VirusScanner interface {
	ScanFile(ctx context.Context, file io.Reader, filename string) ([]Threat, error)
	ScanContent(ctx context.Context, content []byte) ([]Threat, error)
}

// PayloadInspector interface for payload inspection
type PayloadInspector interface {
	InspectPayload(ctx context.Context, payload []byte) ([]Threat, error)
	InspectFormData(ctx context.Context, formData map[string]interface{}) ([]Threat, error)
}

// ImageValidator interface for image validation
type ImageValidator interface {
	ValidateImage(ctx context.Context, file io.Reader, filename string) ([]Threat, error)
	GetImageDimensions(ctx context.Context, file io.Reader) (width, height int, err error)
}

// NewSecurityChecker creates a new security checker
func NewSecurityChecker(config *SecurityConfig, scanner VirusScanner, inspector PayloadInspector, imageValidator ImageValidator) *SecurityChecker {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	return &SecurityChecker{
		config:         config,
		scanner:        scanner,
		inspector:      inspector,
		imageValidator: imageValidator,
	}
}

// CheckFileUpload performs comprehensive security checks on file uploads
func (sc *SecurityChecker) CheckFileUpload(ctx context.Context, fileHeader *multipart.FileHeader) (*SecurityCheckResult, error) {
	var threats []Threat

	// Check file size
	if fileHeader.Size > sc.config.MaxFileSize {
		threats = append(threats, Threat{
			Type:        "file_size_exceeded",
			Severity:    "high",
			Description: fmt.Sprintf("File size %d exceeds maximum allowed size %d", fileHeader.Size, sc.config.MaxFileSize),
			Location:    "file_header",
		})
	}

	// Check file type
	contentType := fileHeader.Header.Get("Content-Type")
	if !sc.isAllowedType(contentType) {
		threats = append(threats, Threat{
			Type:        "blocked_file_type",
			Severity:    "high",
			Description: fmt.Sprintf("File type %s is not allowed", contentType),
			Location:    "file_header",
		})
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if sc.isBlockedExtension(ext) {
		threats = append(threats, Threat{
			Type:        "blocked_extension",
			Severity:    "high",
			Description: fmt.Sprintf("File extension %s is blocked", ext),
			Location:    "filename",
		})
	}

	// Check filename for suspicious patterns
	if sc.hasSuspiciousFilename(fileHeader.Filename) {
		threats = append(threats, Threat{
			Type:        "suspicious_filename",
			Severity:    "medium",
			Description: "Filename contains suspicious patterns",
			Location:    "filename",
		})
	}

	// Open file for content inspection
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Virus scan if enabled
	if sc.config.VirusScanEnabled && sc.scanner != nil {
		virusThreats, err := sc.scanner.ScanFile(ctx, file, fileHeader.Filename)
		if err != nil {
			// Log error but don't fail the check
			threats = append(threats, Threat{
				Type:        "virus_scan_error",
				Severity:    "medium",
				Description: fmt.Sprintf("Virus scan failed: %v", err),
				Location:    "virus_scanner",
			})
		} else {
			threats = append(threats, virusThreats...)
		}
	}

	// Image validation if it's an image
	if sc.config.ImageValidationEnabled && sc.imageValidator != nil && sc.isImageType(contentType) {
		// Reset file position
		file.Seek(0, 0)

		imageThreats, err := sc.imageValidator.ValidateImage(ctx, file, fileHeader.Filename)
		if err != nil {
			threats = append(threats, Threat{
				Type:        "image_validation_error",
				Severity:    "medium",
				Description: fmt.Sprintf("Image validation failed: %v", err),
				Location:    "image_validator",
			})
		} else {
			threats = append(threats, imageThreats...)
		}
	}

	// Determine if file is allowed
	allowed := len(threats) == 0 || sc.hasOnlyLowSeverityThreats(threats)

	result := &SecurityCheckResult{
		Allowed: allowed,
		Threats: threats,
		Details: map[string]interface{}{
			"filename":     fileHeader.Filename,
			"size":         fileHeader.Size,
			"content_type": contentType,
			"threat_count": len(threats),
		},
	}

	if !allowed {
		result.Reason = "Security threats detected"
		if len(threats) > 0 {
			result.Reason = threats[0].Description
		}
	} else {
		result.Reason = "File passed security checks"
	}

	return result, nil
}

// CheckPayload performs security checks on request payloads
func (sc *SecurityChecker) CheckPayload(ctx context.Context, payload []byte) (*SecurityCheckResult, error) {
	var threats []Threat

	// Check payload size
	if int64(len(payload)) > sc.config.MaxPayloadSize {
		threats = append(threats, Threat{
			Type:        "payload_size_exceeded",
			Severity:    "high",
			Description: fmt.Sprintf("Payload size %d exceeds maximum allowed size %d", len(payload), sc.config.MaxPayloadSize),
			Location:    "payload",
		})
	}

	// Payload inspection if enabled
	if sc.config.PayloadInspectionEnabled && sc.inspector != nil {
		inspectionThreats, err := sc.inspector.InspectPayload(ctx, payload)
		if err != nil {
			threats = append(threats, Threat{
				Type:        "payload_inspection_error",
				Severity:    "medium",
				Description: fmt.Sprintf("Payload inspection failed: %v", err),
				Location:    "payload_inspector",
			})
		} else {
			threats = append(threats, inspectionThreats...)
		}
	}

	// Check for blocked patterns
	payloadStr := string(payload)
	for _, pattern := range sc.config.BlockedPatterns {
		if strings.Contains(strings.ToLower(payloadStr), strings.ToLower(pattern)) {
			threats = append(threats, Threat{
				Type:        "blocked_pattern",
				Severity:    "high",
				Description: fmt.Sprintf("Payload contains blocked pattern: %s", pattern),
				Location:    "payload",
			})
		}
	}

	// Determine if payload is allowed
	allowed := len(threats) == 0 || sc.hasOnlyLowSeverityThreats(threats)

	result := &SecurityCheckResult{
		Allowed: allowed,
		Threats: threats,
		Details: map[string]interface{}{
			"payload_size": len(payload),
			"threat_count": len(threats),
		},
	}

	if !allowed {
		result.Reason = "Security threats detected in payload"
		if len(threats) > 0 {
			result.Reason = threats[0].Description
		}
	} else {
		result.Reason = "Payload passed security checks"
	}

	return result, nil
}

// CheckRequest performs security checks on HTTP requests
func (sc *SecurityChecker) CheckRequest(ctx context.Context, req *http.Request) (*SecurityCheckResult, error) {
	var threats []Threat

	// Check request size
	if req.ContentLength > sc.config.MaxPayloadSize {
		threats = append(threats, Threat{
			Type:        "request_size_exceeded",
			Severity:    "high",
			Description: fmt.Sprintf("Request size %d exceeds maximum allowed size %d", req.ContentLength, sc.config.MaxPayloadSize),
			Location:    "request_header",
		})
	}

	// Check for suspicious headers
	if sc.hasSuspiciousHeaders(req) {
		threats = append(threats, Threat{
			Type:        "suspicious_headers",
			Severity:    "medium",
			Description: "Request contains suspicious headers",
			Location:    "request_headers",
		})
	}

	// Check User-Agent
	if sc.hasSuspiciousUserAgent(req.UserAgent()) {
		threats = append(threats, Threat{
			Type:        "suspicious_user_agent",
			Severity:    "medium",
			Description: "Request contains suspicious User-Agent",
			Location:    "user_agent",
		})
	}

	// Check Referer
	if sc.hasSuspiciousReferer(req.Referer()) {
		threats = append(threats, Threat{
			Type:        "suspicious_referer",
			Severity:    "medium",
			Description: "Request contains suspicious Referer",
			Location:    "referer",
		})
	}

	// Determine if request is allowed
	allowed := len(threats) == 0 || sc.hasOnlyLowSeverityThreats(threats)

	result := &SecurityCheckResult{
		Allowed: allowed,
		Threats: threats,
		Details: map[string]interface{}{
			"method":       req.Method,
			"url":          req.URL.String(),
			"user_agent":   req.UserAgent(),
			"referer":      req.Referer(),
			"threat_count": len(threats),
		},
	}

	if !allowed {
		result.Reason = "Security threats detected in request"
		if len(threats) > 0 {
			result.Reason = threats[0].Description
		}
	} else {
		result.Reason = "Request passed security checks"
	}

	return result, nil
}

// isAllowedType checks if a content type is allowed
func (sc *SecurityChecker) isAllowedType(contentType string) bool {
	for _, allowedType := range sc.config.AllowedTypes {
		if strings.EqualFold(contentType, allowedType) {
			return true
		}
	}
	return false
}

// isBlockedExtension checks if a file extension is blocked
func (sc *SecurityChecker) isBlockedExtension(ext string) bool {
	blockedExts := []string{
		".exe", ".bat", ".cmd", ".com", ".pif", ".scr", ".vbs", ".js",
		".jar", ".war", ".ear", ".class", ".sh", ".ps1", ".php", ".asp",
		".jsp", ".py", ".rb", ".pl", ".cgi", ".htaccess", ".htpasswd",
	}

	for _, blockedExt := range blockedExts {
		if ext == blockedExt {
			return true
		}
	}
	return false
}

// hasSuspiciousFilename checks if filename contains suspicious patterns
func (sc *SecurityChecker) hasSuspiciousFilename(filename string) bool {
	suspiciousPatterns := []string{
		"..", "\\", "/", "<", ">", ":", "\"", "|", "?", "*",
		"script", "cmd", "exec", "eval", "javascript", "vbscript",
	}

	filenameLower := strings.ToLower(filename)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(filenameLower, pattern) {
			return true
		}
	}
	return false
}

// isImageType checks if content type is an image
func (sc *SecurityChecker) isImageType(contentType string) bool {
	return strings.HasPrefix(strings.ToLower(contentType), "image/")
}

// hasOnlyLowSeverityThreats checks if all threats are low severity
func (sc *SecurityChecker) hasOnlyLowSeverityThreats(threats []Threat) bool {
	for _, threat := range threats {
		if threat.Severity != "low" {
			return false
		}
	}
	return true
}

// hasSuspiciousHeaders checks for suspicious headers
func (sc *SecurityChecker) hasSuspiciousHeaders(req *http.Request) bool {
	suspiciousHeaders := []string{
		"X-Forwarded-For", "X-Real-IP", "X-Originating-IP",
		"X-Remote-IP", "X-Remote-Addr", "X-Client-IP",
	}

	for _, header := range suspiciousHeaders {
		if req.Header.Get(header) != "" {
			return true
		}
	}
	return false
}

// hasSuspiciousUserAgent checks for suspicious User-Agent
func (sc *SecurityChecker) hasSuspiciousUserAgent(userAgent string) bool {
	suspiciousPatterns := []string{
		"bot", "crawler", "spider", "scraper", "scanner",
		"sqlmap", "nikto", "nmap", "masscan", "zap",
		"curl", "wget", "python-requests", "java",
	}

	userAgentLower := strings.ToLower(userAgent)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(userAgentLower, pattern) {
			return true
		}
	}
	return false
}

// hasSuspiciousReferer checks for suspicious Referer
func (sc *SecurityChecker) hasSuspiciousReferer(referer string) bool {
	if referer == "" {
		return false
	}

	suspiciousPatterns := []string{
		"javascript:", "data:", "vbscript:", "file:",
		"ftp:", "gopher:", "news:", "nntp:",
	}

	refererLower := strings.ToLower(referer)
	for _, pattern := range suspiciousPatterns {
		if strings.HasPrefix(refererLower, pattern) {
			return true
		}
	}
	return false
}

// GetSecurityStats returns security checking statistics
func (sc *SecurityChecker) GetSecurityStats(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"config":                     sc.config,
		"virus_scan_enabled":         sc.config.VirusScanEnabled,
		"payload_inspection_enabled": sc.config.PayloadInspectionEnabled,
		"image_validation_enabled":   sc.config.ImageValidationEnabled,
	}
}
