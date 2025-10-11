package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"time"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// PasswordHasher interface for different hashing algorithms
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, hash string) bool
	GetAlgorithm() string
}

// BCryptHasher implements bcrypt password hashing
type BCryptHasher struct {
	cost int
}

// NewBCryptHasher creates a new bcrypt hasher with specified cost
func NewBCryptHasher(cost int) *BCryptHasher {
	if cost < 4 || cost > 31 {
		cost = 12 // Default cost
	}
	return &BCryptHasher{cost: cost}
}

// Hash hashes a password using bcrypt
func (h *BCryptHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// Verify verifies a password against a bcrypt hash
func (h *BCryptHasher) Verify(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GetAlgorithm returns the algorithm name
func (h *BCryptHasher) GetAlgorithm() string {
	return "bcrypt"
}

// Argon2Hasher implements Argon2 password hashing
type Argon2Hasher struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  uint32
	keyLength   uint32
}

// NewArgon2Hasher creates a new Argon2 hasher with specified parameters
func NewArgon2Hasher(memory, iterations uint32, parallelism uint8, saltLength, keyLength uint32) *Argon2Hasher {
	return &Argon2Hasher{
		memory:      memory,
		iterations:  iterations,
		parallelism: parallelism,
		saltLength:  saltLength,
		keyLength:   keyLength,
	}
}

// Hash hashes a password using Argon2id
func (h *Argon2Hasher) Hash(password string) (string, error) {
	// Generate random salt
	salt := make([]byte, h.saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Hash password
	hash := argon2.IDKey([]byte(password), salt, h.iterations, h.memory, h.parallelism, h.keyLength)

	// Encode hash and salt
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Format: $argon2id$v=19$m=memory,t=time,p=parallelism$salt$hash
	encodedHash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, h.memory, h.iterations, h.parallelism, b64Salt, b64Hash)

	return encodedHash, nil
}

// Verify verifies a password against an Argon2 hash
func (h *Argon2Hasher) Verify(password, encodedHash string) bool {
	// Parse the encoded hash
	salt, hash, params, err := h.decodeHash(encodedHash)
	if err != nil {
		return false
	}

	// Hash the password with the same parameters
	otherHash := argon2.IDKey([]byte(password), salt, params.iterations, params.memory, params.parallelism, params.keyLength)

	// Compare hashes using constant time comparison
	return subtle.ConstantTimeCompare(hash, otherHash) == 1
}

// GetAlgorithm returns the algorithm name
func (h *Argon2Hasher) GetAlgorithm() string {
	return "argon2id"
}

// Argon2Params represents Argon2 parameters
type Argon2Params struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  uint32
	keyLength   uint32
}

// decodeHash decodes an Argon2 hash string
func (h *Argon2Hasher) decodeHash(encodedHash string) (salt, hash []byte, params *Argon2Params, err error) {
	// Expected format: $argon2id$v=19$m=memory,t=time,p=parallelism$salt$hash
	parts := make([]string, 6)
	count := 0
	part := ""

	for _, char := range encodedHash {
		if char == '$' {
			if count < 6 {
				parts[count] = part
				part = ""
				count++
			}
		} else {
			part += string(char)
		}
	}
	if count < 6 {
		parts[count] = part
		count++
	}

	if count != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return nil, nil, nil, fmt.Errorf("invalid hash format")
	}

	// Parse version
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return nil, nil, nil, fmt.Errorf("invalid version: %w", err)
	}
	if version != argon2.Version {
		return nil, nil, nil, fmt.Errorf("incompatible version")
	}

	// Parse parameters
	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return nil, nil, nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Decode salt
	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid salt: %w", err)
	}

	// Decode hash
	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid hash: %w", err)
	}

	params = &Argon2Params{
		memory:      memory,
		iterations:  iterations,
		parallelism: parallelism,
		saltLength:  uint32(len(salt)),
		keyLength:   uint32(len(hash)),
	}

	return salt, hash, params, nil
}

// RateLimiter interface for rate limiting
type RateLimiter interface {
	Allow(key string, limit int, window time.Duration) bool
	GetRemaining(key string, limit int, window time.Duration) int
	Reset(key string) error
}

// InMemoryRateLimiter implements rate limiting using in-memory storage
type InMemoryRateLimiter struct {
	requests map[string][]time.Time
}

// NewInMemoryRateLimiter creates a new in-memory rate limiter
func NewInMemoryRateLimiter() *InMemoryRateLimiter {
	return &InMemoryRateLimiter{
		requests: make(map[string][]time.Time),
	}
}

// Allow checks if a request is allowed based on rate limiting rules
func (rl *InMemoryRateLimiter) Allow(key string, limit int, window time.Duration) bool {
	now := time.Now()
	cutoff := now.Add(-window)

	// Get existing requests for this key
	requests, exists := rl.requests[key]
	if !exists {
		requests = make([]time.Time, 0)
	}

	// Remove old requests outside the window
	var validRequests []time.Time
	for _, reqTime := range requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}

	// Check if we're under the limit
	if len(validRequests) >= limit {
		return false
	}

	// Add current request
	validRequests = append(validRequests, now)
	rl.requests[key] = validRequests

	return true
}

// GetRemaining returns the number of remaining requests allowed
func (rl *InMemoryRateLimiter) GetRemaining(key string, limit int, window time.Duration) int {
	now := time.Now()
	cutoff := now.Add(-window)

	requests, exists := rl.requests[key]
	if !exists {
		return limit
	}

	// Count valid requests
	validCount := 0
	for _, reqTime := range requests {
		if reqTime.After(cutoff) {
			validCount++
		}
	}

	remaining := limit - validCount
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Reset clears all requests for a key
func (rl *InMemoryRateLimiter) Reset(key string) error {
	delete(rl.requests, key)
	return nil
}

// JWTSigner interface for JWT operations
type JWTSigner interface {
	Sign(claims map[string]interface{}) (string, error)
	Verify(token string) (map[string]interface{}, error)
	GetAlgorithm() string
}

// SimpleJWTSigner implements basic JWT signing (for development)
// In production, use a proper JWT library like github.com/golang-jwt/jwt/v5
type SimpleJWTSigner struct {
	secret []byte
}

// NewSimpleJWTSigner creates a new simple JWT signer
func NewSimpleJWTSigner(secret string) *SimpleJWTSigner {
	return &SimpleJWTSigner{
		secret: []byte(secret),
	}
}

// Sign creates a signed JWT token
func (s *SimpleJWTSigner) Sign(claims map[string]interface{}) (string, error) {
	// This is a simplified implementation for development
	// In production, use a proper JWT library
	header := `{"alg":"HS256","typ":"JWT"}`
	payload := `{"sub":"user","iat":` + fmt.Sprintf("%d", time.Now().Unix()) + `}`

	// Simple base64 encoding (not proper JWT)
	token := base64.StdEncoding.EncodeToString([]byte(header + "." + payload))
	return token, nil
}

// Verify verifies a JWT token
func (s *SimpleJWTSigner) Verify(token string) (map[string]interface{}, error) {
	// This is a simplified implementation for development
	// In production, use a proper JWT library
	_, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Simple verification (not proper JWT)
	claims := map[string]interface{}{
		"sub": "user",
		"iat": time.Now().Unix(),
	}

	return claims, nil
}

// GetAlgorithm returns the algorithm name
func (s *SimpleJWTSigner) GetAlgorithm() string {
	return "HS256"
}

// ImageScanner interface for scanning uploaded images
type ImageScanner interface {
	ScanImage(imageData []byte, filename string) (*ScanResult, error)
	IsMalicious(result *ScanResult) bool
}

// ScanResult represents the result of an image scan
type ScanResult struct {
	IsClean    bool              `json:"is_clean"`
	Threats    []string          `json:"threats,omitempty"`
	Confidence float64           `json:"confidence"`
	ScanTime   time.Time         `json:"scan_time"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// MockImageScanner implements a mock image scanner for development
type MockImageScanner struct{}

// NewMockImageScanner creates a new mock image scanner
func NewMockImageScanner() *MockImageScanner {
	return &MockImageScanner{}
}

// ScanImage scans an image for threats
func (s *MockImageScanner) ScanImage(imageData []byte, filename string) (*ScanResult, error) {
	// Mock implementation - in production, integrate with real security services
	// like VirusTotal, Google Safe Browsing, or AWS GuardDuty

	// Simulate scan time
	time.Sleep(100 * time.Millisecond)

	// Mock scan result
	result := &ScanResult{
		IsClean:    true,
		Threats:    []string{},
		Confidence: 0.95,
		ScanTime:   time.Now(),
		Metadata: map[string]string{
			"scanner": "mock",
			"version": "1.0.0",
		},
	}

	// Simulate some threats for testing
	if len(imageData) > 10*1024*1024 { // Files larger than 10MB
		result.IsClean = false
		result.Threats = append(result.Threats, "file_too_large")
		result.Confidence = 0.8
	}

	return result, nil
}

// IsMalicious checks if a scan result indicates malicious content
func (s *MockImageScanner) IsMalicious(result *ScanResult) bool {
	return !result.IsClean || len(result.Threats) > 0
}

// SignedURLGenerator interface for generating signed URLs
type SignedURLGenerator interface {
	GenerateSignedURL(bucket, key string, expiration time.Duration) (string, error)
	VerifySignedURL(url string) (bool, error)
}

// MockSignedURLGenerator implements a mock signed URL generator
type MockSignedURLGenerator struct {
	baseURL string
	secret  string
}

// NewMockSignedURLGenerator creates a new mock signed URL generator
func NewMockSignedURLGenerator(baseURL, secret string) *MockSignedURLGenerator {
	return &MockSignedURLGenerator{
		baseURL: baseURL,
		secret:  secret,
	}
}

// GenerateSignedURL generates a signed URL for accessing a resource
func (g *MockSignedURLGenerator) GenerateSignedURL(bucket, key string, expiration time.Duration) (string, error) {
	// Mock implementation - in production, use cloud provider SDKs
	// like AWS S3, Google Cloud Storage, or Azure Blob Storage

	expiresAt := time.Now().Add(expiration)
	signature := fmt.Sprintf("mock_signature_%d", expiresAt.Unix())

	signedURL := fmt.Sprintf("%s/%s/%s?signature=%s&expires=%d",
		g.baseURL, bucket, key, signature, expiresAt.Unix())

	return signedURL, nil
}

// VerifySignedURL verifies a signed URL
func (g *MockSignedURLGenerator) VerifySignedURL(url string) (bool, error) {
	// Mock implementation - in production, verify the signature
	return true, nil
}
