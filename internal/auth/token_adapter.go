package auth

import (
	"context"
	"time"
)

// TokenServiceAdapter adapts ProductionTokenService to implement auth.TokenService interface
type TokenServiceAdapter struct {
	service *ProductionTokenService
}

// NewTokenServiceAdapter creates a new adapter that wraps ProductionTokenService
func NewTokenServiceAdapter(service *ProductionTokenService) TokenService {
	return &TokenServiceAdapter{service: service}
}

// IssueTokens implements TokenService interface
func (a *TokenServiceAdapter) IssueTokens(ctx context.Context, userID, phone, role, userAgent string) (string, string, time.Time, error) {
	return a.service.IssueTokens(ctx, userID, phone, role, userAgent)
}

// ValidateAccess implements TokenService interface by converting security.JWTClaims to auth.TokenClaims
func (a *TokenServiceAdapter) ValidateAccess(ctx context.Context, token string) (TokenClaims, error) {
	claims, err := a.service.ValidateAccess(ctx, token)
	if err != nil {
		return TokenClaims{}, err
	}

	// Convert security.JWTClaims to auth.TokenClaims
	return TokenClaims{
		UserID:    claims.UserID,
		Phone:     claims.Phone,
		Role:      claims.Role,
		SessionID: claims.SessionID,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}

// Rotate implements TokenService interface
func (a *TokenServiceAdapter) Rotate(ctx context.Context, refreshToken string) (string, string, time.Time, error) {
	return a.service.Rotate(ctx, refreshToken)
}

// RevokeSession implements TokenService interface
func (a *TokenServiceAdapter) RevokeSession(ctx context.Context, sessionID string) error {
	return a.service.RevokeSession(ctx, sessionID)
}

// RevokeAll implements TokenService interface
func (a *TokenServiceAdapter) RevokeAll(ctx context.Context, userID string) error {
	return a.service.RevokeAll(ctx, userID)
}

