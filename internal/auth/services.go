package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"
)

var (
	ErrOTPExpired = errors.New("otp expired")
	ErrOTPInvalid = errors.New("otp invalid")
)

type Store interface {
	CreateOTP(ctx context.Context, phone, purpose string, digits int, ttl time.Duration) (code string, expiresAt time.Time, err error)
	VerifyOTP(ctx context.Context, phone, code, purpose string) (bool, error)
	MarkPhoneVerified(ctx context.Context, phone string) error
	UserExists(ctx context.Context, phone string) (bool, error)
	IsPhoneVerified(ctx context.Context, phone string) (bool, error)
	CreateUser(ctx context.Context, phone, passwordHash, role, displayName, companyName string) (userID string, err error)
	GetUserByPhone(ctx context.Context, phone string) (User, error)
}

type TokenClaims struct {
	UserID    string
	Phone     string
	Role      string
	SessionID string
	ExpiresAt time.Time
}

type TokenService interface {
	IssueTokens(ctx context.Context, userID, phone, role, userAgent string) (access string, refresh string, refreshExp time.Time, err error)
	ValidateAccess(ctx context.Context, token string) (TokenClaims, error)
	Rotate(ctx context.Context, refresh string) (access string, newRefresh string, refreshExp time.Time, err error)
	RevokeSession(ctx context.Context, sessionID string) error
	RevokeAll(ctx context.Context, userID string) error
}

type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) bool
}

// SMSProvider interface moved to internal/sms package

// In-memory implementations for scaffolding
type inMemoryStore struct {
	users map[string]User
	otps  map[string]struct {
		code    string
		purpose string
		expires time.Time
	}
	verifiedPhones map[string]bool
}

func NewInMemoryStore() Store {
	return &inMemoryStore{
		users: map[string]User{},
		otps: map[string]struct {
			code    string
			purpose string
			expires time.Time
		}{},
		verifiedPhones: map[string]bool{},
	}
}

func (s *inMemoryStore) CreateOTP(ctx context.Context, phone, purpose string, digits int, ttl time.Duration) (string, time.Time, error) {
	code := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	exp := time.Now().Add(ttl)
	s.otps[phone] = struct {
		code    string
		purpose string
		expires time.Time
	}{code: code, purpose: purpose, expires: exp}
	return code, exp, nil
}

func (s *inMemoryStore) VerifyOTP(ctx context.Context, phone, code, purpose string) (bool, error) {
	otp, ok := s.otps[phone]
	if !ok || otp.purpose != purpose {
		return false, ErrOTPInvalid
	}
	if time.Now().After(otp.expires) {
		return false, ErrOTPExpired
	}
	if otp.code != code {
		return false, ErrOTPInvalid
	}
	delete(s.otps, phone)
	return true, nil
}

func (s *inMemoryStore) MarkPhoneVerified(ctx context.Context, phone string) error {
	s.verifiedPhones[phone] = true
	return nil
}

func (s *inMemoryStore) UserExists(ctx context.Context, phone string) (bool, error) {
	_, ok := s.users[phone]
	return ok, nil
}

func (s *inMemoryStore) IsPhoneVerified(ctx context.Context, phone string) (bool, error) {
	return s.verifiedPhones[phone], nil
}

func (s *inMemoryStore) CreateUser(ctx context.Context, phone, passwordHash, role, displayName, companyName string) (string, error) {
	id := randomID()
	s.users[phone] = User{
		ID: id, Phone: phone, PasswordHash: passwordHash,
		Role: role, Name: displayName, IsPhoneVerified: true,
		IsActive: true, CreatedAt: time.Now(),
	}
	return id, nil
}

func (s *inMemoryStore) GetUserByPhone(ctx context.Context, phone string) (User, error) {
	u, ok := s.users[phone]
	if !ok {
		return User{}, errors.New("not found")
	}
	return u, nil
}

type inMemoryLimiter struct {
	buckets map[string]int
	reset   map[string]time.Time
}

func NewInMemoryLimiter() RateLimiter {
	return &inMemoryLimiter{buckets: map[string]int{}, reset: map[string]time.Time{}}
}

func (l *inMemoryLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) bool {
	now := time.Now()
	if t, ok := l.reset[key]; !ok || now.After(t) {
		l.reset[key] = now.Add(window)
		l.buckets[key] = 0
	}
	if l.buckets[key] >= limit {
		return false
	}
	l.buckets[key]++
	return true
}

// MockSMS moved to internal/sms package

// Token service with unsigned base64 for scaffolding (replace with JWT RS256 in prod)
type simpleTokenService struct {
	sessions map[string]struct {
		userID, phone, role string
		exp                 time.Time
	}
}

func NewSimpleTokenService() TokenService {
	return &simpleTokenService{sessions: map[string]struct {
		userID, phone, role string
		exp                 time.Time
	}{}}
}

func (t *simpleTokenService) IssueTokens(ctx context.Context, userID, phone, role, userAgent string) (string, string, time.Time, error) {
	sid := randomID()
	t.sessions[sid] = struct {
		userID, phone, role string
		exp                 time.Time
	}{userID: userID, phone: phone, role: role, exp: time.Now().Add(30 * 24 * time.Hour)}
	access := base64.StdEncoding.EncodeToString([]byte(userID + "|" + role + "|" + sid))
	refresh := base64.StdEncoding.EncodeToString([]byte(sid))
	return access, refresh, t.sessions[sid].exp, nil
}

func (t *simpleTokenService) ValidateAccess(ctx context.Context, token string) (TokenClaims, error) {
	b, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return TokenClaims{}, err
	}
	// naive split
	var uid, role, sid string
	// simplified parsing for scaffolding
	ss := string(b)
	idx1 := indexOf(ss, "|")
	idx2 := indexOf(ss[idx1+1:], "|") + idx1 + 1
	if idx1 <= 0 || idx2 <= idx1 {
		return TokenClaims{}, errors.New("invalid")
	}
	uid = ss[:idx1]
	role = ss[idx1+1 : idx2]
	sid = ss[idx2+1:]
	sess, ok := t.sessions[sid]
	if !ok || time.Now().After(sess.exp) {
		return TokenClaims{}, errors.New("expired")
	}
	return TokenClaims{UserID: uid, Role: role, SessionID: sid, ExpiresAt: time.Now().Add(15 * time.Minute)}, nil
}

func (t *simpleTokenService) Rotate(ctx context.Context, refresh string) (string, string, time.Time, error) {
	b, err := base64.StdEncoding.DecodeString(refresh)
	if err != nil {
		return "", "", time.Time{}, err
	}
	sid := string(b)
	sess, ok := t.sessions[sid]
	if !ok || time.Now().After(sess.exp) {
		return "", "", time.Time{}, errors.New("invalid")
	}
	// rotate by replacing session id
	delete(t.sessions, sid)
	newSid := randomID()
	t.sessions[newSid] = sess
	access := base64.StdEncoding.EncodeToString([]byte(sess.userID + "|" + sess.role + "|" + newSid))
	newRefresh := base64.StdEncoding.EncodeToString([]byte(newSid))
	return access, newRefresh, sess.exp, nil
}

func (t *simpleTokenService) RevokeSession(ctx context.Context, sessionID string) error {
	delete(t.sessions, sessionID)
	return nil
}

func (t *simpleTokenService) RevokeAll(ctx context.Context, userID string) error {
	for sid, s := range t.sessions {
		if s.userID == userID {
			delete(t.sessions, sid)
		}
	}
	return nil
}

func randomID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func indexOf(s, sep string) int {
	for i := 0; i+len(sep) <= len(s); i++ {
		if s[i:i+len(sep)] == sep {
			return i
		}
	}
	return -1
}
