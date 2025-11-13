package telegram

import (
	"context"
	"log"
	"time"
)

// SessionManager manages user sessions and state
type SessionManager struct {
	storage *Storage
}

// GetStorage returns the storage instance (for handlers that need direct access)
func (sm *SessionManager) GetStorage() *Storage {
	return sm.storage
}

// NewSessionManager creates a new session manager
func NewSessionManager(storage *Storage) *SessionManager {
	return &SessionManager{
		storage: storage,
	}
}

// GetSession gets or creates a session for a Telegram user
func (sm *SessionManager) GetSession(ctx context.Context, telegramUserID int64) (*Session, error) {
	return sm.storage.GetOrCreateSession(ctx, telegramUserID)
}

// UpdateSession updates session data
func (sm *SessionManager) UpdateSession(ctx context.Context, session *Session) error {
	return sm.storage.UpdateSession(ctx, session)
}

// SetState sets temporary user state
func (sm *SessionManager) SetState(ctx context.Context, telegramUserID int64, action string, data interface{}) error {
	stateData := ""
	if data != nil {
		// Simple JSON encoding - can be enhanced
		if str, ok := data.(string); ok {
			stateData = str
		}
	}

	state := &UserState{
		Action:    action,
		Data:      stateData,
		ExpiresAt: time.Now().Add(1 * time.Hour), // Default 1 hour TTL
	}

	return sm.storage.SetUserState(ctx, telegramUserID, state)
}

// GetState gets temporary user state
func (sm *SessionManager) GetState(ctx context.Context, telegramUserID int64) (*UserState, error) {
	state, err := sm.storage.GetUserState(ctx, telegramUserID)
	if err != nil {
		log.Printf("GetState error for user %d: %v", telegramUserID, err)
		return nil, err
	}
	if state == nil {
		log.Printf("GetState returned nil for user %d", telegramUserID)
	} else {
		log.Printf("GetState returned state for user %d: action=%s, data=%s", telegramUserID, state.Action, state.Data)
	}
	return state, err
}

// ClearState clears temporary user state
func (sm *SessionManager) ClearState(ctx context.Context, telegramUserID int64) error {
	return sm.storage.DeleteUserState(ctx, telegramUserID)
}

// IsAuthenticated checks if user is authenticated
func (sm *SessionManager) IsAuthenticated(ctx context.Context, telegramUserID int64) (bool, error) {
	session, err := sm.storage.GetSessionByTelegramID(ctx, telegramUserID)
	if err != nil {
		return false, err
	}
	if session == nil {
		return false, nil
	}

	// Check if has backend user ID and access token
	return session.BackendUserID != nil && *session.BackendUserID != "" && 
		   session.AccessToken != nil && *session.AccessToken != "", nil
}

// GetAccessToken gets access token for user
func (sm *SessionManager) GetAccessToken(ctx context.Context, telegramUserID int64) (string, error) {
	// First try Redis
	accessToken, _, err := sm.storage.GetToken(ctx, telegramUserID)
	if err == nil && accessToken != "" {
		return accessToken, nil
	}

	// Fallback to database
	session, err := sm.storage.GetSessionByTelegramID(ctx, telegramUserID)
	if err != nil {
		return "", err
	}
	if session == nil || session.AccessToken == nil {
		return "", nil
	}

	return *session.AccessToken, nil
}

