package auth

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"AI_Styler/internal/security"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// SessionStore defines the interface for session storage
type SessionStore interface {
	CreateSession(ctx context.Context, sessionID, userID, refreshTokenHash, userAgent, ip string, expiresAt time.Time) error
	GetSession(ctx context.Context, sessionID string) (*Session, error)
	UpdateSession(ctx context.Context, sessionID string, lastUsedAt time.Time) error
	RevokeSession(ctx context.Context, sessionID string) error
	RevokeUserSessions(ctx context.Context, userID string) error
	CleanupExpiredSessions(ctx context.Context) error
}

// PostgresSessionStore implements SessionStore using PostgreSQL
type PostgresSessionStore struct {
	db *sql.DB
}

// NewPostgresSessionStore creates a new PostgreSQL session store
func NewPostgresSessionStore(db *sql.DB) *PostgresSessionStore {
	return &PostgresSessionStore{db: db}
}

// CreateSession creates a new session
func (s *PostgresSessionStore) CreateSession(ctx context.Context, sessionID, userID, refreshTokenHash, userAgent, ip string, expiresAt time.Time) error {
	query := `
		INSERT INTO sessions (id, user_id, refresh_token_hash, user_agent, ip, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := s.db.ExecContext(ctx, query, sessionID, userID, refreshTokenHash, userAgent, ip, expiresAt)
	return err
}

// GetSession retrieves a session by ID
func (s *PostgresSessionStore) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	query := `
		SELECT id, user_id, refresh_token_hash, user_agent, ip, last_used_at, expires_at, revoked_at
		FROM sessions
		WHERE id = $1 AND revoked_at IS NULL
	`

	var session Session
	err := s.db.QueryRowContext(ctx, query, sessionID).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshTokenHash,
		&session.UserAgent,
		&session.IP,
		&session.LastUsedAt,
		&session.ExpiresAt,
		&session.RevokedAt,
	)

	if err != nil {
		return nil, err
	}

	return &session, nil
}

// UpdateSession updates the last used time for a session
func (s *PostgresSessionStore) UpdateSession(ctx context.Context, sessionID string, lastUsedAt time.Time) error {
	query := `
		UPDATE sessions
		SET last_used_at = $1
		WHERE id = $2 AND revoked_at IS NULL
	`

	_, err := s.db.ExecContext(ctx, query, lastUsedAt, sessionID)
	return err
}

// RevokeSession revokes a specific session
func (s *PostgresSessionStore) RevokeSession(ctx context.Context, sessionID string) error {
	query := `
		UPDATE sessions
		SET revoked_at = NOW()
		WHERE id = $1
	`

	_, err := s.db.ExecContext(ctx, query, sessionID)
	return err
}

// RevokeUserSessions revokes all sessions for a user
func (s *PostgresSessionStore) RevokeUserSessions(ctx context.Context, userID string) error {
	query := `
		UPDATE sessions
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`

	_, err := s.db.ExecContext(ctx, query, userID)
	return err
}

// CleanupExpiredSessions removes expired sessions
func (s *PostgresSessionStore) CleanupExpiredSessions(ctx context.Context) error {
	query := `
		DELETE FROM sessions
		WHERE expires_at < NOW()
	`

	_, err := s.db.ExecContext(ctx, query)
	return err
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

// ProductionTokenService implements secure token management
type ProductionTokenService struct {
	jwtSigner    *security.ProductionJWTSigner
	sessionStore SessionStore
	accessTTL    time.Duration
	refreshTTL   time.Duration
}

// NewProductionTokenService creates a new production token service
func NewProductionTokenService(jwtSigner *security.ProductionJWTSigner, sessionStore SessionStore, accessTTL, refreshTTL time.Duration) *ProductionTokenService {
	return &ProductionTokenService{
		jwtSigner:    jwtSigner,
		sessionStore: sessionStore,
		accessTTL:    accessTTL,
		refreshTTL:   refreshTTL,
	}
}

// IssueTokens creates new access and refresh tokens
func (s *ProductionTokenService) IssueTokens(ctx context.Context, userID, phone, role, userAgent string) (string, string, time.Time, error) {
	sessionID := uuid.New().String()
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
func (s *ProductionTokenService) ValidateAccess(ctx context.Context, token string) (*security.JWTClaims, error) {
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
		return nil, fmt.Errorf("session revoked")
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
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
		return "", "", time.Time{}, fmt.Errorf("session revoked")
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
