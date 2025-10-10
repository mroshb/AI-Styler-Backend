package monitoring

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TracingService provides distributed tracing capabilities
type TracingService struct {
	tracer trace.Tracer
}

// NewTracingService creates a new tracing service
func NewTracingService(serviceName string) *TracingService {
	tracer := otel.Tracer(serviceName)
	return &TracingService{
		tracer: tracer,
	}
}

// StartSpan starts a new span
func (ts *TracingService) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return ts.tracer.Start(ctx, name, opts...)
}

// AddSpanEvent adds an event to the current span
func (ts *TracingService) AddSpanEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetSpanAttributes sets attributes on the current span
func (ts *TracingService) SetSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// RecordSpanError records an error on the current span
func (ts *TracingService) RecordSpanError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// MetricsService provides metrics collection capabilities
type MetricsService struct {
	counters   map[string]int64
	histograms map[string][]float64
	gauges     map[string]float64
}

// NewMetricsService creates a new metrics service
func NewMetricsService() *MetricsService {
	return &MetricsService{
		counters:   make(map[string]int64),
		histograms: make(map[string][]float64),
		gauges:     make(map[string]float64),
	}
}

// IncrementCounter increments a counter metric
func (ms *MetricsService) IncrementCounter(name string, labels map[string]string) {
	key := ms.buildKey(name, labels)
	ms.counters[key]++
}

// RecordHistogram records a histogram metric
func (ms *MetricsService) RecordHistogram(name string, value float64, labels map[string]string) {
	key := ms.buildKey(name, labels)
	ms.histograms[key] = append(ms.histograms[key], value)
}

// SetGauge sets a gauge metric
func (ms *MetricsService) SetGauge(name string, value float64, labels map[string]string) {
	key := ms.buildKey(name, labels)
	ms.gauges[key] = value
}

// GetCounter returns the current counter value
func (ms *MetricsService) GetCounter(name string, labels map[string]string) int64 {
	key := ms.buildKey(name, labels)
	return ms.counters[key]
}

// GetHistogramStats returns histogram statistics
func (ms *MetricsService) GetHistogramStats(name string, labels map[string]string) map[string]float64 {
	key := ms.buildKey(name, labels)
	values := ms.histograms[key]

	if len(values) == 0 {
		return map[string]float64{
			"count": 0,
			"sum":   0,
			"avg":   0,
			"min":   0,
			"max":   0,
		}
	}

	sum := 0.0
	min := values[0]
	max := values[0]

	for _, v := range values {
		sum += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	return map[string]float64{
		"count": float64(len(values)),
		"sum":   sum,
		"avg":   sum / float64(len(values)),
		"min":   min,
		"max":   max,
	}
}

// GetGauge returns the current gauge value
func (ms *MetricsService) GetGauge(name string, labels map[string]string) float64 {
	key := ms.buildKey(name, labels)
	return ms.gauges[key]
}

// buildKey builds a key for metrics storage
func (ms *MetricsService) buildKey(name string, labels map[string]string) string {
	key := name
	for k, v := range labels {
		key += fmt.Sprintf(":%s=%s", k, v)
	}
	return key
}

// PerformanceMonitor provides performance monitoring capabilities
type PerformanceMonitor struct {
	tracingService *TracingService
	metricsService *MetricsService
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(serviceName string) *PerformanceMonitor {
	return &PerformanceMonitor{
		tracingService: NewTracingService(serviceName),
		metricsService: NewMetricsService(),
	}
}

// MonitorRequest monitors an HTTP request
func (pm *PerformanceMonitor) MonitorRequest(ctx context.Context, method, path string, handler func(context.Context) error) error {
	start := time.Now()

	// Start span
	ctx, span := pm.tracingService.StartSpan(ctx, fmt.Sprintf("%s %s", method, path))
	defer span.End()

	// Set span attributes
	pm.tracingService.SetSpanAttributes(ctx,
		attribute.String("http.method", method),
		attribute.String("http.path", path),
	)

	// Record metrics
	pm.metricsService.IncrementCounter("http_requests_total", map[string]string{
		"method": method,
		"path":   path,
	})

	// Execute handler
	err := handler(ctx)

	// Record duration
	duration := time.Since(start)
	pm.metricsService.RecordHistogram("http_request_duration_seconds", duration.Seconds(), map[string]string{
		"method": method,
		"path":   path,
	})

	// Record error if any
	if err != nil {
		pm.tracingService.RecordSpanError(ctx, err)
		pm.metricsService.IncrementCounter("http_requests_errors_total", map[string]string{
			"method": method,
			"path":   path,
		})
	}

	return err
}

// MonitorDatabaseQuery monitors a database query
func (pm *PerformanceMonitor) MonitorDatabaseQuery(ctx context.Context, query string, handler func(context.Context) error) error {
	start := time.Now()

	// Start span
	ctx, span := pm.tracingService.StartSpan(ctx, "database.query")
	defer span.End()

	// Set span attributes
	pm.tracingService.SetSpanAttributes(ctx,
		attribute.String("db.query", query),
	)

	// Record metrics
	pm.metricsService.IncrementCounter("database_queries_total", map[string]string{
		"query": query,
	})

	// Execute handler
	err := handler(ctx)

	// Record duration
	duration := time.Since(start)
	pm.metricsService.RecordHistogram("database_query_duration_seconds", duration.Seconds(), map[string]string{
		"query": query,
	})

	// Record error if any
	if err != nil {
		pm.tracingService.RecordSpanError(ctx, err)
		pm.metricsService.IncrementCounter("database_queries_errors_total", map[string]string{
			"query": query,
		})
	}

	return err
}

// MonitorExternalAPI monitors an external API call
func (pm *PerformanceMonitor) MonitorExternalAPI(ctx context.Context, service, endpoint string, handler func(context.Context) error) error {
	start := time.Now()

	// Start span
	ctx, span := pm.tracingService.StartSpan(ctx, fmt.Sprintf("external.%s", service))
	defer span.End()

	// Set span attributes
	pm.tracingService.SetSpanAttributes(ctx,
		attribute.String("external.service", service),
		attribute.String("external.endpoint", endpoint),
	)

	// Record metrics
	pm.metricsService.IncrementCounter("external_api_calls_total", map[string]string{
		"service":  service,
		"endpoint": endpoint,
	})

	// Execute handler
	err := handler(ctx)

	// Record duration
	duration := time.Since(start)
	pm.metricsService.RecordHistogram("external_api_duration_seconds", duration.Seconds(), map[string]string{
		"service":  service,
		"endpoint": endpoint,
	})

	// Record error if any
	if err != nil {
		pm.tracingService.RecordSpanError(ctx, err)
		pm.metricsService.IncrementCounter("external_api_errors_total", map[string]string{
			"service":  service,
			"endpoint": endpoint,
		})
	}

	return err
}

// GetMetrics returns all collected metrics
func (pm *PerformanceMonitor) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"counters":   pm.metricsService.counters,
		"histograms": pm.metricsService.histograms,
		"gauges":     pm.metricsService.gauges,
	}
}

// TracingService returns the tracing service
func (pm *PerformanceMonitor) TracingService() *TracingService {
	return pm.tracingService
}

// MetricsService returns the metrics service
func (pm *PerformanceMonitor) MetricsService() *MetricsService {
	return pm.metricsService
}
