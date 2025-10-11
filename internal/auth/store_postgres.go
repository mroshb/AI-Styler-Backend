package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// postgresStore implements the Store interface using PostgreSQL
type postgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new PostgreSQL store
func NewPostgresStore(db *sql.DB) Store {
	return &postgresStore{db: db}
}

// CreateOTP creates a new OTP record
func (s *postgresStore) CreateOTP(ctx context.Context, phone, purpose string, digits int, ttl time.Duration) (string, time.Time, error) {
	code := generateOTPCode(digits)
	expiresAt := time.Now().Add(ttl)

	query := `
		INSERT INTO otps (phone, code, purpose, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (phone, purpose) 
		DO UPDATE SET 
			code = EXCLUDED.code,
			expires_at = EXCLUDED.expires_at,
			attempt_count = 0,
			created_at = NOW()
		RETURNING code, expires_at`

	var returnedCode string
	var returnedExpiresAt time.Time
	err := s.db.QueryRowContext(ctx, query, phone, code, purpose, expiresAt).Scan(&returnedCode, &returnedExpiresAt)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to create OTP: %w", err)
	}

	return returnedCode, returnedExpiresAt, nil
}

// VerifyOTP verifies an OTP code
func (s *postgresStore) VerifyOTP(ctx context.Context, phone, code, purpose string) (bool, error) {
	query := `
		UPDATE otps 
		SET consumed_at = NOW(),
		    attempt_count = attempt_count + 1
		WHERE phone = $1 AND code = $2 AND purpose = $3 
		  AND expires_at > NOW() 
		  AND consumed_at IS NULL
		  AND attempt_count < 5
		RETURNING id`

	var otpID string
	err := s.db.QueryRowContext(ctx, query, phone, code, purpose).Scan(&otpID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, ErrOTPInvalid
		}
		return false, fmt.Errorf("failed to verify OTP: %w", err)
	}

	return true, nil
}

// MarkPhoneVerified marks a phone number as verified
func (s *postgresStore) MarkPhoneVerified(ctx context.Context, phone string) error {
	// Update users table
	query := `UPDATE users SET is_phone_verified = true WHERE phone = $1`
	result, err := s.db.ExecContext(ctx, query, phone)
	if err != nil {
		return fmt.Errorf("failed to mark user phone verified: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		// Try vendors table
		query = `UPDATE vendors SET is_phone_verified = true WHERE phone = $1`
		result, err = s.db.ExecContext(ctx, query, phone)
		if err != nil {
			return fmt.Errorf("failed to mark vendor phone verified: %w", err)
		}

		rowsAffected, err = result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}

		if rowsAffected == 0 {
			return errors.New("phone number not found")
		}
	}

	return nil
}

// UserExists checks if a user exists by phone number
func (s *postgresStore) UserExists(ctx context.Context, phone string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE phone = $1)`
	var exists bool
	err := s.db.QueryRowContext(ctx, query, phone).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}
	return exists, nil
}

// IsPhoneVerified checks if a phone number is verified
func (s *postgresStore) IsPhoneVerified(ctx context.Context, phone string) (bool, error) {
	query := `
		SELECT is_phone_verified 
		FROM users 
		WHERE phone = $1
		UNION ALL
		SELECT is_phone_verified 
		FROM vendors 
		WHERE phone = $1
		LIMIT 1`

	var verified bool
	err := s.db.QueryRowContext(ctx, query, phone).Scan(&verified)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, errors.New("phone number not found")
		}
		return false, fmt.Errorf("failed to check phone verification: %w", err)
	}
	return verified, nil
}

// CreateUser creates a new user
func (s *postgresStore) CreateUser(ctx context.Context, phone, passwordHash, role, displayName, companyName string) (string, error) {
	var userID string
	var query string
	var args []interface{}

	if role == "vendor" {
		query = `
			INSERT INTO vendors (phone, password_hash, business_name, profile_info)
			VALUES ($1, $2, $3, $4)
			RETURNING id`
		args = []interface{}{phone, passwordHash, companyName, fmt.Sprintf(`{"display_name": "%s"}`, displayName)}
	} else {
		query = `
			INSERT INTO users (phone, password_hash, name)
			VALUES ($1, $2, $3)
			RETURNING id`
		args = []interface{}{phone, passwordHash, displayName}
	}

	err := s.db.QueryRowContext(ctx, query, args...).Scan(&userID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return "", errors.New("phone number already exists")
		}
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	return userID, nil
}

// GetUserByPhone retrieves a user by phone number
func (s *postgresStore) GetUserByPhone(ctx context.Context, phone string) (User, error) {
	// Try users table first
	query := `
		SELECT id, phone, password_hash, name, avatar_url, bio, 
		       is_phone_verified, is_active, last_login_at, created_at
		FROM users 
		WHERE phone = $1`

	var user User
	err := s.db.QueryRowContext(ctx, query, phone).Scan(
		&user.ID, &user.Phone, &user.PasswordHash, &user.Name, &user.AvatarURL, &user.Bio,
		&user.IsPhoneVerified, &user.IsActive, &user.LastLoginAt, &user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Try vendors table
			query = `
				SELECT id, phone, password_hash, business_name, avatar_url, bio,
				       is_phone_verified, is_active, last_login_at, created_at
				FROM vendors 
				WHERE phone = $1`

			err = s.db.QueryRowContext(ctx, query, phone).Scan(
				&user.ID, &user.Phone, &user.PasswordHash, &user.Name, &user.AvatarURL, &user.Bio,
				&user.IsPhoneVerified, &user.IsActive, &user.LastLoginAt, &user.CreatedAt,
			)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return User{}, errors.New("user not found")
				}
				return User{}, fmt.Errorf("failed to get user: %w", err)
			}
			user.Role = "vendor"
		} else {
			return User{}, fmt.Errorf("failed to get user: %w", err)
		}
	} else {
		user.Role = "user"
	}

	return user, nil
}

// Helper function to generate OTP code
func generateOTPCode(digits int) string {
	// Simple implementation - in production, use crypto/rand
	code := fmt.Sprintf("%0*d", digits, time.Now().UnixNano()%int64(pow10(digits)))
	return code
}

// Helper function to calculate power of 10
func pow10(n int) int {
	result := 1
	for i := 0; i < n; i++ {
		result *= 10
	}
	return result
}
