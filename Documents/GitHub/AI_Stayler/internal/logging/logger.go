package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// LogLevel represents the logging level
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
)

// LoggerConfig represents the logger configuration
type LoggerConfig struct {
	Level       LogLevel
	Format      string // json or text
	Output      string // stdout, stderr, or file path
	Service     string
	Version     string
	Environment string
}

// StructuredLogger provides structured logging capabilities
type StructuredLogger struct {
	logger *logrus.Logger
	config LoggerConfig
}

// LogEntry represents a log entry with context
type LogEntry struct {
	Timestamp    time.Time              `json:"timestamp"`
	Level        string                 `json:"level"`
	Service      string                 `json:"service"`
	Version      string                 `json:"version"`
	Environment  string                 `json:"environment"`
	Message      string                 `json:"message"`
	Fields       map[string]interface{} `json:"fields,omitempty"`
	Context      map[string]interface{} `json:"context,omitempty"`
	Caller       string                 `json:"caller,omitempty"`
	TraceID      string                 `json:"trace_id,omitempty"`
	UserID       string                 `json:"user_id,omitempty"`
	VendorID     string                 `json:"vendor_id,omitempty"`
	ConversionID string                 `json:"conversion_id,omitempty"`
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(config LoggerConfig) *StructuredLogger {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(string(config.Level))
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set formatter
	if config.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
			},
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: time.RFC3339,
			FullTimestamp:   true,
		})
	}

	// Set output
	if config.Output == "stderr" {
		logger.SetOutput(os.Stderr)
	} else if config.Output != "stdout" && config.Output != "" {
		file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			logger.Warnf("Failed to open log file %s: %v", config.Output, err)
		} else {
			logger.SetOutput(file)
		}
	}

	// Add hooks for additional functionality
	logger.AddHook(&ContextHook{})
	logger.AddHook(&CallerHook{})

	return &StructuredLogger{
		logger: logger,
		config: config,
	}
}

// Debug logs a debug message
func (l *StructuredLogger) Debug(ctx context.Context, msg string, fields map[string]interface{}) {
	l.logWithContext(ctx, logrus.DebugLevel, msg, fields)
}

// Info logs an info message
func (l *StructuredLogger) Info(ctx context.Context, msg string, fields map[string]interface{}) {
	l.logWithContext(ctx, logrus.InfoLevel, msg, fields)
}

// Warn logs a warning message
func (l *StructuredLogger) Warn(ctx context.Context, msg string, fields map[string]interface{}) {
	l.logWithContext(ctx, logrus.WarnLevel, msg, fields)
}

// Error logs an error message
func (l *StructuredLogger) Error(ctx context.Context, msg string, fields map[string]interface{}) {
	l.logWithContext(ctx, logrus.ErrorLevel, msg, fields)
}

// Fatal logs a fatal message and exits
func (l *StructuredLogger) Fatal(ctx context.Context, msg string, fields map[string]interface{}) {
	l.logWithContext(ctx, logrus.FatalLevel, msg, fields)
	os.Exit(1)
}

// WithFields creates a new logger with additional fields
func (l *StructuredLogger) WithFields(fields map[string]interface{}) *StructuredLogger {
	return &StructuredLogger{
		logger: l.logger.WithFields(fields).Logger,
		config: l.config,
	}
}

// WithContext creates a new logger with context information
func (l *StructuredLogger) WithContext(ctx context.Context) *StructuredLogger {
	fields := l.extractContextFields(ctx)
	return l.WithFields(fields)
}

// logWithContext logs a message with context information
func (l *StructuredLogger) logWithContext(ctx context.Context, level logrus.Level, msg string, fields map[string]interface{}) {
	entry := l.logger.WithFields(logrus.Fields{
		"service":     l.config.Service,
		"version":     l.config.Version,
		"environment": l.config.Environment,
	})

	// Add context fields
	contextFields := l.extractContextFields(ctx)
	for k, v := range contextFields {
		entry = entry.WithField(k, v)
	}

	// Add custom fields
	for k, v := range fields {
		entry = entry.WithField(k, v)
	}

	entry.Log(level, msg)
}

// extractContextFields extracts relevant fields from context
func (l *StructuredLogger) extractContextFields(ctx context.Context) map[string]interface{} {
	fields := make(map[string]interface{})

	// Extract trace ID
	if traceID := ctx.Value("trace_id"); traceID != nil {
		if id, ok := traceID.(string); ok {
			fields["trace_id"] = id
		}
	}

	// Extract user ID
	if userID := ctx.Value("user_id"); userID != nil {
		if id, ok := userID.(string); ok {
			fields["user_id"] = id
		}
	}

	// Extract vendor ID
	if vendorID := ctx.Value("vendor_id"); vendorID != nil {
		if id, ok := vendorID.(string); ok {
			fields["vendor_id"] = id
		}
	}

	// Extract conversion ID
	if conversionID := ctx.Value("conversion_id"); conversionID != nil {
		if id, ok := conversionID.(string); ok {
			fields["conversion_id"] = id
		}
	}

	// Extract request ID
	if requestID := ctx.Value("request_id"); requestID != nil {
		if id, ok := requestID.(string); ok {
			fields["request_id"] = id
		}
	}

	return fields
}

// ContextHook adds context information to log entries
type ContextHook struct{}

func (hook *ContextHook) Fire(entry *logrus.Entry) error {
	// Add service information
	entry.Data["service"] = "ai-stayler"
	entry.Data["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
	return nil
}

func (hook *ContextHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// CallerHook adds caller information to log entries
type CallerHook struct{}

func (hook *CallerHook) Fire(entry *logrus.Entry) error {
	// Get caller information
	pc := make([]uintptr, 1)
	n := runtime.Callers(6, pc)
	if n > 0 {
		frames := runtime.CallersFrames(pc[:n])
		frame, _ := frames.Next()
		entry.Data["caller"] = fmt.Sprintf("%s:%d", frame.File, frame.Line)
		entry.Data["function"] = frame.Function
	}
	return nil
}

func (hook *CallerHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// LogEntryToJSON converts a LogEntry to JSON
func LogEntryToJSON(entry LogEntry) ([]byte, error) {
	return json.Marshal(entry)
}

// ParseLogLevel parses a string log level
func ParseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return LogLevelDebug
	case "info":
		return LogLevelInfo
	case "warn", "warning":
		return LogLevelWarn
	case "error":
		return LogLevelError
	case "fatal":
		return LogLevelFatal
	default:
		return LogLevelInfo
	}
}

// GetDefaultLoggerConfig returns default logger configuration
func GetDefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Level:       LogLevelInfo,
		Format:      "json",
		Output:      "stdout",
		Service:     "ai-stayler",
		Version:     "1.0.0",
		Environment: "development",
	}
}
