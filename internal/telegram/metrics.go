package telegram

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds Prometheus metrics for the Telegram bot
var (
	// telegram_updates_total counts total Telegram updates received
	UpdatesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "telegram_updates_total",
			Help: "Total number of Telegram updates received",
		},
		[]string{"type"},
	)

	// telegram_processing_duration_seconds measures processing time
	ProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "telegram_processing_duration_seconds",
			Help:    "Time spent processing Telegram updates",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"handler"},
	)

	// telegram_errors_total counts errors by type
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "telegram_errors_total",
			Help: "Total number of errors",
		},
		[]string{"type", "handler"},
	)

	// telegram_active_users tracks active users
	ActiveUsers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "telegram_active_users",
			Help: "Number of active Telegram users",
		},
	)

	// telegram_conversions_total counts conversions
	ConversionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "telegram_conversions_total",
			Help: "Total number of conversions",
		},
		[]string{"status"},
	)

	// telegram_api_requests_total counts API requests
	APIRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "telegram_api_requests_total",
			Help: "Total number of API requests",
		},
		[]string{"endpoint", "status"},
	)

	// telegram_api_request_duration_seconds measures API request time
	APIRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "telegram_api_request_duration_seconds",
			Help:    "Time spent on API requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint"},
	)

	// telegram_rate_limit_hits_total counts rate limit hits
	RateLimitHitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "telegram_rate_limit_hits_total",
			Help: "Total number of rate limit hits",
		},
		[]string{"type"},
	)
)

// RecordUpdate records a Telegram update
func RecordUpdate(updateType string) {
	UpdatesTotal.WithLabelValues(updateType).Inc()
}

// RecordProcessingDuration records processing duration
func RecordProcessingDuration(handler string, duration float64) {
	ProcessingDuration.WithLabelValues(handler).Observe(duration)
}

// RecordError records an error
func RecordError(errorType, handler string) {
	ErrorsTotal.WithLabelValues(errorType, handler).Inc()
}

// RecordConversion records a conversion
func RecordConversion(status string) {
	ConversionsTotal.WithLabelValues(status).Inc()
}

// RecordAPIRequest records an API request
func RecordAPIRequest(endpoint, status string) {
	APIRequestsTotal.WithLabelValues(endpoint, status).Inc()
}

// RecordAPIRequestDuration records API request duration
func RecordAPIRequestDuration(endpoint string, duration float64) {
	APIRequestDuration.WithLabelValues(endpoint).Observe(duration)
}

// RecordRateLimitHit records a rate limit hit
func RecordRateLimitHit(limitType string) {
	RateLimitHitsTotal.WithLabelValues(limitType).Inc()
}

// SetActiveUsers sets the number of active users
func SetActiveUsers(count float64) {
	ActiveUsers.Set(count)
}

