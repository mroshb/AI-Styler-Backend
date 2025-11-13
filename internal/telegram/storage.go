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
	ID              string     `json:"id"`
	TelegramUserID  int64      `json:"telegram_user_id"`
	BackendUserID   *string    `json:"backend_user_id,omitempty"`
	Phone           *string    `json:"phone,omitempty"`
	AccessToken     *string    `json:"access_token,omitempty"`
	RefreshToken    *string    `json:"refresh_token,omitempty"`
	TokenExpiresAt  *time.Time `json:"token_expires_at,omitempty"`
	FirstName       *string    `json:"first_name,omitempty"`
	LastName        *string    `json:"last_name,omitempty"`
	Username        *string    `json:"username,omitempty"`
	LanguageCode    *string    `json:"language_code,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
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
	// Create table if it doesn't exist
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS telegram_sessions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		telegram_user_id BIGINT UNIQUE NOT NULL,
		backend_user_id UUID,
		phone VARCHAR(20),
		access_token TEXT,
		refresh_token TEXT,
		token_expires_at TIMESTAMP,
		first_name VARCHAR(255),
		last_name VARCHAR(255),
		username VARCHAR(255),
		language_code VARCHAR(10),
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);
	`

	if _, err := s.db.Exec(createTableQuery); err != nil {
		return fmt.Errorf("failed to create telegram_sessions table: %w", err)
	}

	// Add missing columns if table already exists
	alterQueries := []string{
		`DO $$ 
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
			               WHERE table_name = 'telegram_sessions' AND column_name = 'token_expires_at') THEN
				ALTER TABLE telegram_sessions ADD COLUMN token_expires_at TIMESTAMP;
			END IF;
		END $$;`,
	}

	for _, query := range alterQueries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("failed to alter telegram_sessions table: %w", err)
		}
	}

	// Create indexes
	indexQueries := []string{
		`CREATE INDEX IF NOT EXISTS idx_telegram_sessions_telegram_user_id ON telegram_sessions(telegram_user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_telegram_sessions_backend_user_id ON telegram_sessions(backend_user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_telegram_sessions_phone ON telegram_sessions(phone);`,
		`CREATE INDEX IF NOT EXISTS idx_telegram_sessions_token_expires_at ON telegram_sessions(token_expires_at);`,
	}

	for _, query := range indexQueries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// GetOrCreateSession gets or creates a session for a Telegram user
func (s *Storage) GetOrCreateSession(ctx context.Context, telegramUserID int64) (*Session, error) {
	var session Session
	var tokenExpiresAt sql.NullTime
	query := `
		SELECT id, telegram_user_id, backend_user_id, phone, access_token, refresh_token, 
		       token_expires_at, first_name, last_name, username, language_code, 
		       created_at, updated_at
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
		&tokenExpiresAt,
		&session.FirstName,
		&session.LastName,
		&session.Username,
		&session.LanguageCode,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Create new session
		var newTokenExpiresAt sql.NullTime
		sessionID := uuid.New().String()
		insertQuery := `
			INSERT INTO telegram_sessions (id, telegram_user_id, created_at, updated_at)
			VALUES ($1, $2, NOW(), NOW())
			RETURNING id, telegram_user_id, backend_user_id, phone, access_token, refresh_token, 
			          token_expires_at, first_name, last_name, username, language_code, 
			          created_at, updated_at
		`

		err = s.db.QueryRowContext(ctx, insertQuery, sessionID, telegramUserID).Scan(
			&session.ID,
			&session.TelegramUserID,
			&session.BackendUserID,
			&session.Phone,
			&session.AccessToken,
			&session.RefreshToken,
			&newTokenExpiresAt,
			&session.FirstName,
			&session.LastName,
			&session.Username,
			&session.LanguageCode,
			&session.CreatedAt,
			&session.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
		if newTokenExpiresAt.Valid {
			session.TokenExpiresAt = &newTokenExpiresAt.Time
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if tokenExpiresAt.Valid {
		session.TokenExpiresAt = &tokenExpiresAt.Time
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
		    token_expires_at = $5,
		    first_name = $6,
		    last_name = $7,
		    username = $8,
		    language_code = $9,
		    updated_at = NOW()
		WHERE telegram_user_id = $10
	`

	_, err := s.db.ExecContext(ctx, query,
		session.BackendUserID,
		session.Phone,
		session.AccessToken,
		session.RefreshToken,
		session.TokenExpiresAt,
		session.FirstName,
		session.LastName,
		session.Username,
		session.LanguageCode,
		session.TelegramUserID,
	)

	return err
}

// GetSessionByTelegramID gets a session by Telegram user ID
func (s *Storage) GetSessionByTelegramID(ctx context.Context, telegramUserID int64) (*Session, error) {
	var session Session
	var tokenExpiresAt sql.NullTime
	query := `
		SELECT id, telegram_user_id, backend_user_id, phone, access_token, refresh_token, 
		       token_expires_at, first_name, last_name, username, language_code, 
		       created_at, updated_at
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
		&tokenExpiresAt,
		&session.FirstName,
		&session.LastName,
		&session.Username,
		&session.LanguageCode,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if tokenExpiresAt.Valid {
		session.TokenExpiresAt = &tokenExpiresAt.Time
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

