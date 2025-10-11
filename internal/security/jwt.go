package security

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// JWTClaims represents the claims in a JWT token
type JWTClaims struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	Role      string `json:"role"`
	Phone     string `json:"phone"`
	jwt.RegisteredClaims
}

// ProductionJWTSigner implements secure JWT signing using golang-jwt/jwt/v5
type ProductionJWTSigner struct {
	secretKey []byte
	issuer    string
}

// NewProductionJWTSigner creates a new production-ready JWT signer
func NewProductionJWTSigner(secretKey, issuer string) *ProductionJWTSigner {
	return &ProductionJWTSigner{
		secretKey: []byte(secretKey),
		issuer:    issuer,
	}
}

// Sign creates a signed JWT token with proper claims
func (s *ProductionJWTSigner) Sign(userID, sessionID, role, phone string, expiresAt time.Time) (string, error) {
	now := time.Now()

	claims := JWTClaims{
		UserID:    userID,
		SessionID: sessionID,
		Role:      role,
		Phone:     phone,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID,
			Audience:  []string{"ai-styler-api"},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        sessionID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

// Verify verifies a JWT token and returns the claims
func (s *ProductionJWTSigner) Verify(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// RefreshTokenClaims represents refresh token claims
type RefreshTokenClaims struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

// SignRefreshToken creates a signed refresh token
func (s *ProductionJWTSigner) SignRefreshToken(userID, sessionID string, expiresAt time.Time) (string, error) {
	now := time.Now()

	claims := RefreshTokenClaims{
		UserID:    userID,
		SessionID: sessionID,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID,
			Audience:  []string{"ai-styler-refresh"},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        sessionID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

// VerifyRefreshToken verifies a refresh token
func (s *ProductionJWTSigner) VerifyRefreshToken(tokenString string) (*RefreshTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &RefreshTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse refresh token: %w", err)
	}

	if claims, ok := token.Claims.(*RefreshTokenClaims); ok && token.Valid {
		if claims.TokenType != "refresh" {
			return nil, errors.New("invalid token type")
		}
		return claims, nil
	}

	return nil, errors.New("invalid refresh token")
}

// GenerateSessionID generates a secure session ID
func GenerateSessionID() string {
	return uuid.New().String()
}

// TokenService interface for token operations
type TokenService interface {
	IssueTokens(ctx context.Context, userID, phone, role, userAgent string) (accessToken, refreshToken string, refreshExpiresAt time.Time, err error)
	ValidateAccess(ctx context.Context, token string) (*JWTClaims, error)
	Rotate(ctx context.Context, refreshToken string) (accessToken, newRefreshToken string, refreshExpiresAt time.Time, err error)
	RevokeSession(ctx context.Context, sessionID string) error
	RevokeAll(ctx context.Context, userID string) error
}

// ProductionTokenService implements secure token management
type ProductionTokenService struct {
	jwtSigner    *ProductionJWTSigner
	sessionStore SessionStore
	accessTTL    time.Duration
	refreshTTL   time.Duration
}

// SessionStore interface for session management
type SessionStore interface {
	CreateSession(ctx context.Context, sessionID, userID, refreshTokenHash, userAgent, ip string, expiresAt time.Time) error
	GetSession(ctx context.Context, sessionID string) (*Session, error)
	UpdateSession(ctx context.Context, sessionID string, lastUsedAt time.Time) error
	RevokeSession(ctx context.Context, sessionID string) error
	RevokeUserSessions(ctx context.Context, userID string) error
	CleanupExpiredSessions(ctx context.Context) error
}

// Session represents a user session
type Session struct {
	ID               string
	UserID           string
	RefreshTokenHash string
	UserAgent        string
	IP               string
	LastUsedAt       time.Time
	ExpiresAt        time.Time
	RevokedAt        *time.Time
}

// NewProductionTokenService creates a new production token service
func NewProductionTokenService(jwtSigner *ProductionJWTSigner, sessionStore SessionStore, accessTTL, refreshTTL time.Duration) *ProductionTokenService {
	return &ProductionTokenService{
		jwtSigner:    jwtSigner,
		sessionStore: sessionStore,
		accessTTL:    accessTTL,
		refreshTTL:   refreshTTL,
	}
}

// IssueTokens creates new access and refresh tokens
func (s *ProductionTokenService) IssueTokens(ctx context.Context, userID, phone, role, userAgent string) (string, string, time.Time, error) {
	sessionID := GenerateSessionID()
	now := time.Now()

	// Create refresh token
	refreshExpiresAt := now.Add(s.refreshTTL)
	refreshToken, err := s.jwtSigner.SignRefreshToken(userID, sessionID, refreshExpiresAt)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("failed to create refresh token: %w", err)
	}

	// Hash refresh token for storage
	refreshTokenHash, err := s.hashToken(refreshToken)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("failed to hash refresh token: %w", err)
	}

	// Create access token
	accessExpiresAt := now.Add(s.accessTTL)
	accessToken, err := s.jwtSigner.Sign(userID, sessionID, role, phone, accessExpiresAt)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("failed to create access token: %w", err)
	}

	// Store session
	err = s.sessionStore.CreateSession(ctx, sessionID, userID, refreshTokenHash, userAgent, "", refreshExpiresAt)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("failed to create session: %w", err)
	}

	return accessToken, refreshToken, refreshExpiresAt, nil
}

// ValidateAccess validates an access token
func (s *ProductionTokenService) ValidateAccess(ctx context.Context, token string) (*JWTClaims, error) {
	claims, err := s.jwtSigner.Verify(token)
	if err != nil {
		return nil, err
	}

	// Check if session is still valid
	session, err := s.sessionStore.GetSession(ctx, claims.SessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	if session.RevokedAt != nil {
		return nil, errors.New("session revoked")
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, errors.New("session expired")
	}

	// Update last used time
	_ = s.sessionStore.UpdateSession(ctx, claims.SessionID, time.Now())

	return claims, nil
}

// Rotate rotates refresh token and creates new access token
func (s *ProductionTokenService) Rotate(ctx context.Context, refreshToken string) (string, string, time.Time, error) {
	claims, err := s.jwtSigner.VerifyRefreshToken(refreshToken)
	if err != nil {
		return "", "", time.Time{}, err
	}

	// Verify session
	session, err := s.sessionStore.GetSession(ctx, claims.SessionID)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("session not found: %w", err)
	}

	if session.RevokedAt != nil {
		return "", "", time.Time{}, errors.New("session revoked")
	}

	// Revoke old session
	err = s.sessionStore.RevokeSession(ctx, claims.SessionID)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("failed to revoke old session: %w", err)
	}

	// Create new tokens (this will create a new session)
	// Note: We need user details from the old session, but for now we'll use the claims
	// In a real implementation, you'd fetch user details from the database
	return s.IssueTokens(ctx, claims.UserID, "", "", session.UserAgent)
}

// RevokeSession revokes a specific session
func (s *ProductionTokenService) RevokeSession(ctx context.Context, sessionID string) error {
	return s.sessionStore.RevokeSession(ctx, sessionID)
}

// RevokeAll revokes all sessions for a user
func (s *ProductionTokenService) RevokeAll(ctx context.Context, userID string) error {
	return s.sessionStore.RevokeUserSessions(ctx, userID)
}

// hashToken hashes a token for secure storage
func (s *ProductionTokenService) hashToken(token string) (string, error) {
	// Use bcrypt to hash the token
	hash, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
