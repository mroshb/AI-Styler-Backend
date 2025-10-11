package security

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"time"
)

// TLSConfig holds TLS configuration
type TLSConfig struct {
	// Certificate and key files
	CertFile string
	KeyFile  string

	// TLS version settings
	MinVersion uint16
	MaxVersion uint16

	// Cipher suites
	CipherSuites []uint16

	// Security settings
	InsecureSkipVerify       bool
	PreferServerCipherSuites bool

	// Session settings
	SessionTicketsDisabled bool
	SessionTicketKey       []byte

	// OCSP settings
	OCSPStapling bool

	// HSTS settings
	HSTSMaxAge            int
	HSTSIncludeSubdomains bool
	HSTSPreload           bool
}

// DefaultTLSConfig returns default TLS configuration
func DefaultTLSConfig() *TLSConfig {
	return &TLSConfig{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
		PreferServerCipherSuites: true,
		SessionTicketsDisabled:   false,
		OCSPStapling:             true,
		HSTSMaxAge:               31536000, // 1 year
		HSTSIncludeSubdomains:    true,
		HSTSPreload:              false,
	}
}

// SecureTLSConfig returns a more secure TLS configuration
func SecureTLSConfig() *TLSConfig {
	config := DefaultTLSConfig()
	config.MinVersion = tls.VersionTLS13
	config.MaxVersion = tls.VersionTLS13
	config.CipherSuites = []uint16{
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,
		tls.TLS_AES_128_GCM_SHA256,
	}
	config.SessionTicketsDisabled = true
	config.HSTSPreload = true
	return config
}

// CreateTLSConfig creates a TLS configuration from the config
func (tc *TLSConfig) CreateTLSConfig() (*tls.Config, error) {
	config := &tls.Config{
		MinVersion:               tc.MinVersion,
		MaxVersion:               tc.MaxVersion,
		CipherSuites:             tc.CipherSuites,
		PreferServerCipherSuites: tc.PreferServerCipherSuites,
		InsecureSkipVerify:       tc.InsecureSkipVerify,
		SessionTicketsDisabled:   tc.SessionTicketsDisabled,
	}

	// Set session ticket key if provided
	if len(tc.SessionTicketKey) > 0 && len(tc.SessionTicketKey) == 32 {
		var key [32]byte
		copy(key[:], tc.SessionTicketKey)
		config.SessionTicketKey = key
	}

	// Validate configuration
	if tc.MinVersion > tc.MaxVersion {
		return nil, fmt.Errorf("min TLS version cannot be greater than max version")
	}

	if tc.MinVersion < tls.VersionTLS12 {
		return nil, fmt.Errorf("minimum TLS version must be at least 1.2")
	}

	return config, nil
}

// HTTPSRedirectMiddleware redirects HTTP requests to HTTPS
func HTTPSRedirectMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if request is already HTTPS
			if r.TLS != nil {
				next.ServeHTTP(w, r)
				return
			}

			// Check X-Forwarded-Proto header for load balancers
			if r.Header.Get("X-Forwarded-Proto") == "https" {
				next.ServeHTTP(w, r)
				return
			}

			// Redirect to HTTPS
			httpsURL := fmt.Sprintf("https://%s%s", r.Host, r.RequestURI)
			http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
		})
	}
}

// HSTSMiddleware adds HTTP Strict Transport Security headers
func HSTSMiddleware(config *TLSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only add HSTS header for HTTPS requests
			if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
				hstsValue := fmt.Sprintf("max-age=%d", config.HSTSMaxAge)
				if config.HSTSIncludeSubdomains {
					hstsValue += "; includeSubDomains"
				}
				if config.HSTSPreload {
					hstsValue += "; preload"
				}
				w.Header().Set("Strict-Transport-Security", hstsValue)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// TLSCertificateManager manages TLS certificates
type TLSCertificateManager struct {
	certFile string
	keyFile  string
	cert     *tls.Certificate
	lastMod  time.Time
}

// NewTLSCertificateManager creates a new certificate manager
func NewTLSCertificateManager(certFile, keyFile string) *TLSCertificateManager {
	return &TLSCertificateManager{
		certFile: certFile,
		keyFile:  keyFile,
	}
}

// LoadCertificate loads the TLS certificate
func (tcm *TLSCertificateManager) LoadCertificate() error {
	cert, err := tls.LoadX509KeyPair(tcm.certFile, tcm.keyFile)
	if err != nil {
		return fmt.Errorf("failed to load certificate: %w", err)
	}

	tcm.cert = &cert
	tcm.lastMod = time.Now()

	return nil
}

// GetCertificate returns the current certificate
func (tcm *TLSCertificateManager) GetCertificate() *tls.Certificate {
	return tcm.cert
}

// ReloadCertificate reloads the certificate if it has been modified
func (tcm *TLSCertificateManager) ReloadCertificate() error {
	// Check if certificate files have been modified
	// This is a simplified implementation
	// In production, use file system watchers

	return tcm.LoadCertificate()
}

// CertificateInfo holds information about a certificate
type CertificateInfo struct {
	Subject         string
	Issuer          string
	NotBefore       time.Time
	NotAfter        time.Time
	SerialNumber    string
	DNSNames        []string
	IsValid         bool
	DaysUntilExpiry int
}

// GetCertificateInfo extracts information from a certificate
func GetCertificateInfo(cert *tls.Certificate) (*CertificateInfo, error) {
	if len(cert.Certificate) == 0 {
		return nil, fmt.Errorf("no certificate data")
	}

	// Parse the first certificate in the chain
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	now := time.Now()
	daysUntilExpiry := int(x509Cert.NotAfter.Sub(now).Hours() / 24)

	info := &CertificateInfo{
		Subject:         x509Cert.Subject.String(),
		Issuer:          x509Cert.Issuer.String(),
		NotBefore:       x509Cert.NotBefore,
		NotAfter:        x509Cert.NotAfter,
		SerialNumber:    x509Cert.SerialNumber.String(),
		DNSNames:        x509Cert.DNSNames,
		IsValid:         now.After(x509Cert.NotBefore) && now.Before(x509Cert.NotAfter),
		DaysUntilExpiry: daysUntilExpiry,
	}

	return info, nil
}

// TLSServerConfig holds configuration for a TLS server
type TLSServerConfig struct {
	Addr         string
	TLSConfig    *TLSConfig
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DefaultTLSServerConfig returns default TLS server configuration
func DefaultTLSServerConfig() *TLSServerConfig {
	return &TLSServerConfig{
		Addr:         ":443",
		TLSConfig:    DefaultTLSConfig(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

// CreateTLSServer creates a configured TLS server
func CreateTLSServer(config *TLSServerConfig, handler http.Handler) (*http.Server, error) {
	tlsConfig, err := config.TLSConfig.CreateTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS config: %w", err)
	}

	server := &http.Server{
		Addr:         config.Addr,
		Handler:      handler,
		TLSConfig:    tlsConfig,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}

	return server, nil
}

// ValidateTLSConfig validates TLS configuration
func ValidateTLSConfig(config *TLSConfig) error {
	if config.MinVersion < tls.VersionTLS12 {
		return fmt.Errorf("minimum TLS version must be at least 1.2")
	}

	if config.MinVersion > config.MaxVersion {
		return fmt.Errorf("minimum TLS version cannot be greater than maximum version")
	}

	if len(config.CipherSuites) == 0 {
		return fmt.Errorf("at least one cipher suite must be specified")
	}

	return nil
}

// GetRecommendedCipherSuites returns recommended cipher suites for different TLS versions
func GetRecommendedCipherSuites(version uint16) []uint16 {
	switch version {
	case tls.VersionTLS13:
		return []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_AES_128_GCM_SHA256,
		}
	case tls.VersionTLS12:
		return []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		}
	default:
		return []uint16{}
	}
}
