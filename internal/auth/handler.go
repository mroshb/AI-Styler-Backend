package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"ai-styler/internal/common"
	"ai-styler/internal/config"
	"ai-styler/internal/security"
	"ai-styler/internal/sms"
)

type Handler struct {
	store       Store
	tokens      TokenService
	rateLimiter RateLimiter
	sms         sms.Provider
	hasher      security.PasswordHasher
}

func NewHandler(store Store, tokens TokenService, rl RateLimiter, smsProvider sms.Provider) *Handler {
	// Load config for password hashing
	cfg, err := config.Load()
	if err != nil {
		// Use default config if loading fails
		cfg = &config.Config{
			Security: config.SecurityConfig{
				BCryptCost:        12,
				Argon2Memory:      65536,
				Argon2Iterations:  3,
				Argon2Parallelism: 2,
				Argon2SaltLength:  16,
				Argon2KeyLength:   32,
			},
		}
	}

	// Create password hasher based on config
	var hasher security.PasswordHasher
	if cfg.Security.Argon2Memory > 0 {
		hasher = security.NewArgon2Hasher(
			cfg.Security.Argon2Memory,
			cfg.Security.Argon2Iterations,
			cfg.Security.Argon2Parallelism,
			cfg.Security.Argon2SaltLength,
			cfg.Security.Argon2KeyLength,
		)
	} else {
		hasher = security.NewBCryptHasher(cfg.Security.BCryptCost)
	}

	return &Handler{
		store:       store,
		tokens:      tokens,
		rateLimiter: rl,
		sms:         smsProvider,
		hasher:      hasher,
	}
}

type sendOtpReq struct {
	Phone   string `json:"phone"`
	Purpose string `json:"purpose"`
	Channel string `json:"channel"`
}

type sendOtpResp struct {
	Sent         bool   `json:"sent"`
	ExpiresInSec int    `json:"expiresInSec"`
	Code         string `json:"code,omitempty"` // Only returned in development/mock mode
	Debug        bool   `json:"debug,omitempty"` // Indicates if this is a debug response
}

func (h *Handler) SendOTP(w http.ResponseWriter, r *http.Request) {
	var req sendOtpReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, http.StatusBadRequest, "bad_request", "invalid json", nil)
		return
	}
	phone := normalizePhone(req.Phone)
	if phone == "" {
		common.WriteError(w, http.StatusBadRequest, "bad_request", "invalid phone", nil)
		return
	}
	ip := clientIP(r)
	if !h.rateLimiter.Allow(r.Context(), "send_otp:phone:"+phone, 3, time.Hour) ||
		!h.rateLimiter.Allow(r.Context(), "send_otp:ip:"+ip, 100, 24*time.Hour) {
		common.WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many requests", nil)
		return
	}
	code, _, err := h.store.CreateOTP(r.Context(), phone, "phone_verify", 6, 5*time.Minute)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "server_error", "could not create otp", nil)
		return
	}
	_ = h.sms.Send(code, phone)
	
	// If SMS provider is mock, include the code in response for development
	resp := sendOtpResp{
		Sent:         true,
		ExpiresInSec: 300,
	}
	if h.sms.IsMock() {
		resp.Code = code
		resp.Debug = true
	}
	
	common.WriteJSON(w, http.StatusOK, resp)
}

type verifyReq struct {
	Phone   string `json:"phone"`
	Code    string `json:"code"`
	Purpose string `json:"purpose"`
}

type verifyResp struct {
	Verified bool `json:"verified"`
}

func (h *Handler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req verifyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, http.StatusBadRequest, "bad_request", "invalid json", nil)
		return
	}
	phone := normalizePhone(req.Phone)
	if phone == "" || len(req.Code) != 6 {
		common.WriteError(w, http.StatusBadRequest, "bad_request", "invalid input", nil)
		return
	}
	ok, err := h.store.VerifyOTP(r.Context(), phone, req.Code, "phone_verify")
	if err != nil {
		if errors.Is(err, ErrOTPExpired) || errors.Is(err, ErrOTPInvalid) {
			common.WriteError(w, http.StatusBadRequest, "invalid_otp", "invalid or expired otp", nil)
			return
		}
		common.WriteError(w, http.StatusInternalServerError, "server_error", "verification failed", nil)
		return
	}
	if ok {
		_ = h.store.MarkPhoneVerified(r.Context(), phone)
		common.WriteJSON(w, http.StatusOK, verifyResp{Verified: true})
		return
	}
	common.WriteJSON(w, http.StatusOK, verifyResp{Verified: false})
}

type registerReq struct {
	Phone       string `json:"phone"`
	Password    string `json:"password"`
	Role        string `json:"role"`
	AutoLogin   bool   `json:"autoLogin"`
	DisplayName string `json:"displayName"`
	CompanyName string `json:"companyName"`
}

type registerResp struct {
	UserID          string `json:"userId"`
	Role            string `json:"role"`
	IsPhoneVerified bool   `json:"isPhoneVerified"`
	AccessToken     string `json:"accessToken,omitempty"`
	AccessExpiresIn int    `json:"accessTokenExpiresIn,omitempty"`
	RefreshToken    string `json:"refreshToken,omitempty"`
	RefreshExpires  string `json:"refreshTokenExpiresAt,omitempty"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, http.StatusBadRequest, "bad_request", "invalid json", nil)
		return
	}
	phone := normalizePhone(req.Phone)
	if phone == "" || len(req.Password) < 10 || (req.Role != "user" && req.Role != "vendor") {
		common.WriteError(w, http.StatusBadRequest, "bad_request", "invalid input", nil)
		return
	}
	if exists, _ := h.store.UserExists(r.Context(), phone); exists {
		common.WriteError(w, http.StatusConflict, "conflict", "account exists", nil)
		return
	}
	verified, _ := h.store.IsPhoneVerified(r.Context(), phone)
	if !verified {
		common.WriteError(w, http.StatusForbidden, "unverified", "phone not verified", nil)
		return
	}
	hash, err := h.hasher.Hash(req.Password)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "server_error", "could not hash password", nil)
		return
	}
	userID, err := h.store.CreateUser(r.Context(), phone, hash, req.Role, req.DisplayName, req.CompanyName)
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "server_error", "could not create user", nil)
		return
	}
	resp := registerResp{UserID: userID, Role: req.Role, IsPhoneVerified: true}
	if req.AutoLogin {
		at, rt, expAt, err := h.tokens.IssueTokens(r.Context(), userID, phone, req.Role, "")
		if err == nil {
			resp.AccessToken = at
			resp.AccessExpiresIn = 900
			resp.RefreshToken = rt
			resp.RefreshExpires = expAt.Format(time.RFC3339)
		}
	}
	common.WriteJSON(w, http.StatusCreated, resp)
}

type loginReq struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type loginResp struct {
	AccessToken           string    `json:"accessToken"`
	AccessTokenExpiresIn  int       `json:"accessTokenExpiresIn"`
	RefreshToken          string    `json:"refreshToken"`
	RefreshTokenExpiresAt time.Time `json:"refreshTokenExpiresAt"`
	User                  struct {
		ID              string `json:"id"`
		Role            string `json:"role"`
		IsPhoneVerified bool   `json:"isPhoneVerified"`
	} `json:"user"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteError(w, http.StatusBadRequest, "bad_request", "invalid json", nil)
		return
	}
	phone := normalizePhone(req.Phone)
	if phone == "" || len(req.Password) == 0 {
		common.WriteError(w, http.StatusBadRequest, "bad_request", "invalid input", nil)
		return
	}
	user, err := h.store.GetUserByPhone(r.Context(), phone)
	if err != nil {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid credentials", nil)
		return
	}
	if !h.hasher.Verify(req.Password, user.PasswordHash) {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid credentials", nil)
		return
	}
	if !user.IsPhoneVerified {
		common.WriteError(w, http.StatusForbidden, "forbidden", "phone not verified", nil)
		return
	}
	at, rt, expAt, err := h.tokens.IssueTokens(r.Context(), user.ID, user.Phone, user.Role, r.UserAgent())
	if err != nil {
		common.WriteError(w, http.StatusInternalServerError, "server_error", "could not issue tokens", nil)
		return
	}
	var resp loginResp
	resp.AccessToken = at
	resp.AccessTokenExpiresIn = 900
	resp.RefreshToken = rt
	resp.RefreshTokenExpiresAt = expAt
	resp.User.ID = user.ID
	resp.User.Role = user.Role
	resp.User.IsPhoneVerified = user.IsPhoneVerified
	common.WriteJSON(w, http.StatusOK, resp)
}

type refreshReq struct {
	RefreshToken string `json:"refreshToken"`
}
type refreshResp struct {
	AccessToken           string `json:"accessToken"`
	AccessTokenExpiresIn  int    `json:"accessTokenExpiresIn"`
	RefreshToken          string `json:"refreshToken"`
	RefreshTokenExpiresAt string `json:"refreshTokenExpiresAt"`
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		common.WriteError(w, http.StatusBadRequest, "bad_request", "invalid input", nil)
		return
	}
	at, rt, expAt, err := h.tokens.Rotate(r.Context(), req.RefreshToken)
	if err != nil {
		common.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid refresh", nil)
		return
	}
	common.WriteJSON(w, http.StatusOK, refreshResp{AccessToken: at, AccessTokenExpiresIn: 900, RefreshToken: rt, RefreshTokenExpiresAt: expAt.Format(time.RFC3339)})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	sid := r.Context().Value(ctxSessionID{}).(string)
	_ = h.tokens.RevokeSession(r.Context(), sid)
	common.WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	uid := r.Context().Value(ctxUserID{}).(string)
	_ = h.tokens.RevokeAll(r.Context(), uid)
	common.WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// Middleware
func (h *Handler) Authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			common.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing token", nil)
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		claims, err := h.tokens.ValidateAccess(r.Context(), token)
		if err != nil {
			common.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid token", nil)
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserID{}, claims.UserID)
		ctx = context.WithValue(ctx, ctxSessionID{}, claims.SessionID)
		next(w, r.WithContext(ctx))
	}
}

// Helpers and models
type ctxUserID struct{}
type ctxSessionID struct{}

type User struct {
	ID              string
	Phone           string
	PasswordHash    string
	Role            string
	Name            string
	AvatarURL       string
	Bio             string
	IsPhoneVerified bool
	IsActive        bool
	LastLoginAt     *time.Time
	CreatedAt       time.Time
}

func normalizePhone(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if !strings.HasPrefix(p, "+") {
		return ""
	}
	return p
}

func clientIP(r *http.Request) string {
	if x := r.Header.Get("X-Forwarded-For"); x != "" {
		return x
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}

// Test helper functions
func TestHashPassword(password string) string {
	// Create a test hasher for testing purposes
	hasher := security.NewBCryptHasher(4) // Lower cost for faster tests
	hash, _ := hasher.Hash(password)
	return hash
}

func TestVerifyPassword(password, hash string) bool {
	hasher := security.NewBCryptHasher(4)
	return hasher.Verify(password, hash)
}
