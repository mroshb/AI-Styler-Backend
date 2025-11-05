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

	// First, invalidate any existing unconsumed OTPs for this phone and purpose
	_, err := s.db.ExecContext(ctx, `
		UPDATE otps 
		SET consumed_at = NOW() 
		WHERE phone = $1 AND purpose = $2 AND consumed_at IS NULL`,
		phone, purpose)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to invalidate existing OTP: %w", err)
	}

	// Then insert the new OTP
	query := `
		INSERT INTO otps (phone, code, purpose, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING code, expires_at`

	var returnedCode string
	var returnedExpiresAt time.Time
	err = s.db.QueryRowContext(ctx, query, phone, code, purpose, expiresAt).Scan(&returnedCode, &returnedExpiresAt)
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
// This can be called before user registration, so we track verification in otps table
// Note: vendors table doesn't have phone field - all users (including vendors) are in users table
func (s *postgresStore) MarkPhoneVerified(ctx context.Context, phone string) error {
	// Update users table (includes both regular users and vendors)
	// Vendors are stored in users table with role='vendor', and linked via vendors.user_id
	query := `UPDATE users SET is_phone_verified = true WHERE phone = $1`
	_, err := s.db.ExecContext(ctx, query, phone)
	if err != nil {
		return fmt.Errorf("failed to mark user phone verified: %w", err)
	}

	// Note: It's OK if no rows affected - user may not exist yet
	// The verification status is tracked via consumed OTPs in IsPhoneVerified()
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
// First checks if user exists and is verified, otherwise checks for consumed OTP
func (s *postgresStore) IsPhoneVerified(ctx context.Context, phone string) (bool, error) {
	// First, check if user exists and is verified
	query := `
		SELECT is_phone_verified 
		FROM users 
		WHERE phone = $1
		LIMIT 1`

	var verified bool
	err := s.db.QueryRowContext(ctx, query, phone).Scan(&verified)
	if err == nil {
		// User exists, return their verification status
		return verified, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("failed to check phone verification: %w", err)
	}

	// User doesn't exist yet, check if there's a consumed OTP for phone_verify
	// This allows registration after OTP verification
	otpQuery := `
		SELECT EXISTS(
			SELECT 1 
			FROM otps 
			WHERE phone = $1 
			  AND purpose = 'phone_verify' 
			  AND consumed_at IS NOT NULL
			  AND consumed_at > NOW() - INTERVAL '24 hours'
		)`

	err = s.db.QueryRowContext(ctx, otpQuery, phone).Scan(&verified)
	if err != nil {
		return false, fmt.Errorf("failed to check OTP verification: %w", err)
	}

	return verified, nil
}

// CreateUser creates a new user
// Assumes phone is already verified (checked before calling this function)
func (s *postgresStore) CreateUser(ctx context.Context, phone, passwordHash, role, displayName, companyName string) (string, error) {
	var userID string
	var query string
	var args []interface{}

	// All users (including vendors) are created in users table
	// Vendors get an additional record in vendors table linked via user_id
		query = `
		INSERT INTO users (phone, password_hash, role, name, is_phone_verified, is_active)
		VALUES ($1, $2, $3, $4, true, true)
			RETURNING id`
	args = []interface{}{phone, passwordHash, role, displayName}

	err := s.db.QueryRowContext(ctx, query, args...).Scan(&userID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return "", errors.New("phone number already exists")
		}
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	// If vendor, create vendor record
	if role == "vendor" {
		vendorQuery := `
			INSERT INTO vendors (user_id, company_name, display_name)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id) DO NOTHING`
		_, err = s.db.ExecContext(ctx, vendorQuery, userID, companyName, displayName)
		if err != nil {
			// Log error but don't fail user creation
			_ = err
		}
	}

	return userID, nil
}

// GetUserByPhone retrieves a user by phone number
func (s *postgresStore) GetUserByPhone(ctx context.Context, phone string) (User, error) {
	// Users table contains all users (including vendors with role = 'vendor')
	query := `
		SELECT id, phone, password_hash, name, avatar_url, bio, 
		       is_phone_verified, is_active, last_login_at, created_at, role
		FROM users 
		WHERE phone = $1`

	var user User
	var name sql.NullString
	var avatarURL sql.NullString
	var bio sql.NullString
	var role sql.NullString
	var lastLoginAt sql.NullTime
	err := s.db.QueryRowContext(ctx, query, phone).Scan(
		&user.ID, &user.Phone, &user.PasswordHash, &name, &avatarURL, &bio,
		&user.IsPhoneVerified, &user.IsActive, &lastLoginAt, &user.CreatedAt, &role,
			)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return User{}, errors.New("user not found")
				}
				return User{}, fmt.Errorf("failed to get user: %w", err)
			}

	// Handle nullable fields
	if name.Valid {
		user.Name = name.String
	}
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}
	if bio.Valid {
		user.Bio = bio.String
	}
	if role.Valid {
		user.Role = role.String
	} else {
		user.Role = "user" // Default role
	}
	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
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
