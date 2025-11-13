package telegram

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Session represents a Telegram user session
type Session struct {
	ID              string    `json:"id"`
	TelegramUserID  int64     `json:"telegram_user_id"`
	BackendUserID   *string   `json:"backend_user_id,omitempty"`
	Phone           *string   `json:"phone,omitempty"`
	AccessToken     *string   `json:"access_token,omitempty"`
	RefreshToken    *string   `json:"refresh_token,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// UserState represents temporary user state (e.g., waiting for image, OTP, etc.)
type UserState struct {
	Action      string    `json:"action"`       // e.g., "waiting_phone", "waiting_otp", "waiting_user_image", "waiting_cloth_image"
	Data        string    `json:"data"`         // JSON-encoded additional data
	ExpiresAt   time.Time `json:"expires_at"`
}

// Storage provides database operations for Telegram bot
type Storage struct {
	db    *sql.DB
	redis *redis.Client
}

// NewStorage creates a new storage instance
func NewStorage(db *sql.DB, redisClient *redis.Client) (*Storage, error) {
	storage := &Storage{
		db:    db,
		redis: redisClient,
	}

	// Create telegram_sessions table if it doesn't exist
	if err := storage.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return storage, nil
}

// createTables creates necessary database tables
func (s *Storage) createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS telegram_sessions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		telegram_user_id BIGINT UNIQUE NOT NULL,
		backend_user_id UUID,
		phone VARCHAR(20),
		access_token TEXT,
		refresh_token TEXT,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_telegram_sessions_telegram_user_id ON telegram_sessions(telegram_user_id);
	CREATE INDEX IF NOT EXISTS idx_telegram_sessions_backend_user_id ON telegram_sessions(backend_user_id);
	`

	_, err := s.db.Exec(query)
	return err
}

// GetOrCreateSession gets or creates a session for a Telegram user
func (s *Storage) GetOrCreateSession(ctx context.Context, telegramUserID int64) (*Session, error) {
	var session Session
	query := `
		SELECT id, telegram_user_id, backend_user_id, phone, access_token, refresh_token, created_at, updated_at
		FROM telegram_sessions
		WHERE telegram_user_id = $1
	`

	err := s.db.QueryRowContext(ctx, query, telegramUserID).Scan(
		&session.ID,
		&session.TelegramUserID,
		&session.BackendUserID,
		&session.Phone,
		&session.AccessToken,
		&session.RefreshToken,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Create new session
		sessionID := uuid.New().String()
		insertQuery := `
			INSERT INTO telegram_sessions (id, telegram_user_id, created_at, updated_at)
			VALUES ($1, $2, NOW(), NOW())
			RETURNING id, telegram_user_id, backend_user_id, phone, access_token, refresh_token, created_at, updated_at
		`

		err = s.db.QueryRowContext(ctx, insertQuery, sessionID, telegramUserID).Scan(
			&session.ID,
			&session.TelegramUserID,
			&session.BackendUserID,
			&session.Phone,
			&session.AccessToken,
			&session.RefreshToken,
			&session.CreatedAt,
			&session.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return &session, nil
}

// UpdateSession updates a session with new data
func (s *Storage) UpdateSession(ctx context.Context, session *Session) error {
	query := `
		UPDATE telegram_sessions
		SET backend_user_id = $1,
		    phone = $2,
		    access_token = $3,
		    refresh_token = $4,
		    updated_at = NOW()
		WHERE telegram_user_id = $5
	`

	_, err := s.db.ExecContext(ctx, query,
		session.BackendUserID,
		session.Phone,
		session.AccessToken,
		session.RefreshToken,
		session.TelegramUserID,
	)

	return err
}

// GetSessionByTelegramID gets a session by Telegram user ID
func (s *Storage) GetSessionByTelegramID(ctx context.Context, telegramUserID int64) (*Session, error) {
	var session Session
	query := `
		SELECT id, telegram_user_id, backend_user_id, phone, access_token, refresh_token, created_at, updated_at
		FROM telegram_sessions
		WHERE telegram_user_id = $1
	`

	err := s.db.QueryRowContext(ctx, query, telegramUserID).Scan(
		&session.ID,
		&session.TelegramUserID,
		&session.BackendUserID,
		&session.Phone,
		&session.AccessToken,
		&session.RefreshToken,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return &session, nil
}

// SetUserState stores temporary user state in Redis
func (s *Storage) SetUserState(ctx context.Context, telegramUserID int64, state *UserState) error {
	if s.redis == nil {
		return fmt.Errorf("redis client not available")
	}

	key := fmt.Sprintf("telegram:state:%d", telegramUserID)
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	ttl := time.Until(state.ExpiresAt)
	if ttl <= 0 {
		ttl = 1 * time.Hour // Default TTL
	}

	return s.redis.Set(ctx, key, data, ttl).Err()
}

// GetUserState retrieves temporary user state from Redis
func (s *Storage) GetUserState(ctx context.Context, telegramUserID int64) (*UserState, error) {
	if s.redis == nil {
		return nil, fmt.Errorf("redis client not available")
	}

	key := fmt.Sprintf("telegram:state:%d", telegramUserID)
	data, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %w", err)
	}

	var state UserState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// Check if expired
	if time.Now().After(state.ExpiresAt) {
		s.DeleteUserState(ctx, telegramUserID)
		return nil, nil
	}

	return &state, nil
}

// DeleteUserState deletes temporary user state from Redis
func (s *Storage) DeleteUserState(ctx context.Context, telegramUserID int64) error {
	if s.redis == nil {
		return nil
	}

	key := fmt.Sprintf("telegram:state:%d", telegramUserID)
	return s.redis.Del(ctx, key).Err()
}

// StoreToken stores JWT token in Redis with TTL
func (s *Storage) StoreToken(ctx context.Context, telegramUserID int64, accessToken, refreshToken string, ttl time.Duration) error {
	if s.redis == nil {
		return fmt.Errorf("redis client not available")
	}

	key := fmt.Sprintf("telegram:token:%d", telegramUserID)
	tokenData := map[string]string{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	}

	data, err := json.Marshal(tokenData)
	if err != nil {
		return fmt.Errorf("failed to marshal token data: %w", err)
	}

	return s.redis.Set(ctx, key, data, ttl).Err()
}

// GetToken retrieves JWT token from Redis
func (s *Storage) GetToken(ctx context.Context, telegramUserID int64) (accessToken, refreshToken string, err error) {
	if s.redis == nil {
		return "", "", fmt.Errorf("redis client not available")
	}

	key := fmt.Sprintf("telegram:token:%d", telegramUserID)
	data, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", "", nil
	}
	if err != nil {
		return "", "", fmt.Errorf("failed to get token: %w", err)
	}

	var tokenData map[string]string
	if err := json.Unmarshal([]byte(data), &tokenData); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal token data: %w", err)
	}

	return tokenData["access_token"], tokenData["refresh_token"], nil
}

// DeleteToken deletes JWT token from Redis
func (s *Storage) DeleteToken(ctx context.Context, telegramUserID int64) error {
	if s.redis == nil {
		return nil
	}

	key := fmt.Sprintf("telegram:token:%d", telegramUserID)
	return s.redis.Del(ctx, key).Err()
}

// Close closes database connections
func (s *Storage) Close() error {
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			return err
		}
	}
	if s.redis != nil {
		if err := s.redis.Close(); err != nil {
			return err
		}
	}
	return nil
}

