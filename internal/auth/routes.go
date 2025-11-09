package auth

import (
	"net/http"

	"ai-styler/internal/sms"
)

// ProvideStdMux mounts handlers on a standard library ServeMux.
func ProvideStdMux(mux *http.ServeMux) {
	store := NewInMemoryStore()
	limiter := NewInMemoryLimiter()
	tokens := NewSimpleTokenService()
	sms := &sms.MockSMSProvider{}
	h := NewHandler(store, tokens, limiter, sms)

	mux.HandleFunc("/auth/send-otp", h.SendOTP)
	mux.HandleFunc("/auth/verify-otp", h.VerifyOTP)
	mux.HandleFunc("/auth/check-user", h.CheckUser)
	mux.HandleFunc("/auth/register", h.Register)
	mux.HandleFunc("/auth/login", h.Login)
	mux.HandleFunc("/auth/refresh", h.Refresh)
	mux.HandleFunc("/auth/logout", h.Authenticate(h.Logout))
	mux.HandleFunc("/auth/logout-all", h.Authenticate(h.LogoutAll))
}
