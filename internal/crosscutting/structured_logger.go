package crosscutting

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

// LogLevel represents different log levels
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
)

// LogConfig represents configuration for structured logging
type LogConfig struct {
	// Output configuration
	OutputFormat string `json:"output_format"` // json, text
	OutputFile   string `json:"output_file"`
	OutputStdout bool   `json:"output_stdout"`

	// Log levels
	MinLevel LogLevel `json:"min_level"`

	// Fields to include
	IncludeTimestamp bool `json:"include_timestamp"`
	IncludeLevel     bool `json:"include_level"`
	IncludeCaller    bool `json:"include_caller"`
	IncludeStack     bool `json:"include_stack"`

	// Performance settings
	BufferSize int `json:"buffer_size"`

	// Rotation settings
	MaxSize    int `json:"max_size"` // MB
	MaxAge     int `json:"max_age"`  // days
	MaxBackups int `json:"max_backups"`

	// Context fields
	DefaultFields map[string]interface{} `json:"default_fields"`
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields"`
	Caller    string                 `json:"caller,omitempty"`
	Stack     string                 `json:"stack,omitempty"`
	TraceID   string                 `json:"trace_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
}

// DefaultLogConfig returns default logging configuration
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		OutputFormat:     "json",
		OutputStdout:     true,
		MinLevel:         LogLevelInfo,
		IncludeTimestamp: true,
		IncludeLevel:     true,
		IncludeCaller:    true,
		IncludeStack:     false,
		BufferSize:       1000,
		MaxSize:          100, // 100MB
		MaxAge:           30,  // 30 days
		MaxBackups:       5,
		DefaultFields: map[string]interface{}{
			"service": "ai-styler",
			"version": "1.0.0",
		},
	}
}

// StructuredLogger provides comprehensive structured logging
type StructuredLogger struct {
	config *LogConfig
	logger *log.Logger
	file   *os.File
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(config *LogConfig) *StructuredLogger {
	if config == nil {
		config = DefaultLogConfig()
	}

	sl := &StructuredLogger{
		config: config,
	}

	// Setup output
	if config.OutputFile != "" {
		file, err := os.OpenFile(config.OutputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Printf("Failed to open log file: %v", err)
		} else {
			sl.file = file
		}
	}

	// Create logger
	if sl.file != nil && config.OutputStdout {
		// Write to both file and stdout
		sl.logger = log.New(os.Stdout, "", 0)
	} else if sl.file != nil {
		// Write only to file
		sl.logger = log.New(sl.file, "", 0)
	} else {
		// Write only to stdout
		sl.logger = log.New(os.Stdout, "", 0)
	}

	return sl
}

// Debug logs a debug message
func (sl *StructuredLogger) Debug(ctx context.Context, message string, fields map[string]interface{}) {
	sl.log(LogLevelDebug, ctx, message, fields)
}

// Info logs an info message
func (sl *StructuredLogger) Info(ctx context.Context, message string, fields map[string]interface{}) {
	sl.log(LogLevelInfo, ctx, message, fields)
}

// Warn logs a warning message
func (sl *StructuredLogger) Warn(ctx context.Context, message string, fields map[string]interface{}) {
	sl.log(LogLevelWarn, ctx, message, fields)
}

// Error logs an error message
func (sl *StructuredLogger) Error(ctx context.Context, message string, fields map[string]interface{}) {
	sl.log(LogLevelError, ctx, message, fields)
}

// Fatal logs a fatal message and exits
func (sl *StructuredLogger) Fatal(ctx context.Context, message string, fields map[string]interface{}) {
	sl.log(LogLevelFatal, ctx, message, fields)
	os.Exit(1)
}

// log performs the actual logging
func (sl *StructuredLogger) log(level LogLevel, ctx context.Context, message string, fields map[string]interface{}) {
	// Check if we should log this level
	if !sl.shouldLog(level) {
		return
	}

	// Create log entry
	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Fields:    make(map[string]interface{}),
	}

	// Add default fields
	for k, v := range sl.config.DefaultFields {
		entry.Fields[k] = v
	}

	// Add provided fields
	for k, v := range fields {
		entry.Fields[k] = v
	}

	// Add context fields
	if ctx != nil {
		if traceID := ctx.Value("trace_id"); traceID != nil {
			entry.TraceID = fmt.Sprintf("%v", traceID)
		}
		if userID := ctx.Value("user_id"); userID != nil {
			entry.UserID = fmt.Sprintf("%v", userID)
		}
		if requestID := ctx.Value("request_id"); requestID != nil {
			entry.RequestID = fmt.Sprintf("%v", requestID)
		}
	}

	// Add caller information
	if sl.config.IncludeCaller {
		if pc, file, line, ok := runtime.Caller(3); ok {
			funcName := runtime.FuncForPC(pc).Name()
			entry.Caller = fmt.Sprintf("%s:%d:%s", file, line, funcName)
		}
	}

	// Add stack trace for errors and fatals
	if sl.config.IncludeStack && (level == LogLevelError || level == LogLevelFatal) {
		entry.Stack = sl.getStackTrace()
	}

	// Format and output
	sl.output(entry)
}

// shouldLog checks if we should log at the given level
func (sl *StructuredLogger) shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		LogLevelDebug: 0,
		LogLevelInfo:  1,
		LogLevelWarn:  2,
		LogLevelError: 3,
		LogLevelFatal: 4,
	}

	return levels[level] >= levels[sl.config.MinLevel]
}

// output formats and outputs the log entry
func (sl *StructuredLogger) output(entry *LogEntry) {
	var output string

	if sl.config.OutputFormat == "json" {
		jsonData, err := json.Marshal(entry)
		if err != nil {
			output = fmt.Sprintf("{\"error\":\"failed to marshal log entry: %v\"}", err)
		} else {
			output = string(jsonData)
		}
	} else {
		// Text format
		var parts []string

		if sl.config.IncludeTimestamp {
			parts = append(parts, fmt.Sprintf("[%s]", entry.Timestamp.Format("2006-01-02 15:04:05")))
		}

		if sl.config.IncludeLevel {
			parts = append(parts, fmt.Sprintf("[%s]", strings.ToUpper(string(entry.Level))))
		}

		if entry.Caller != "" && sl.config.IncludeCaller {
			parts = append(parts, fmt.Sprintf("[%s]", entry.Caller))
		}

		if entry.TraceID != "" {
			parts = append(parts, fmt.Sprintf("[trace:%s]", entry.TraceID))
		}

		if entry.UserID != "" {
			parts = append(parts, fmt.Sprintf("[user:%s]", entry.UserID))
		}

		parts = append(parts, entry.Message)

		if len(entry.Fields) > 0 {
			fieldsStr := make([]string, 0, len(entry.Fields))
			for k, v := range entry.Fields {
				fieldsStr = append(fieldsStr, fmt.Sprintf("%s=%v", k, v))
			}
			parts = append(parts, fmt.Sprintf("{%s}", strings.Join(fieldsStr, ", ")))
		}

		output = strings.Join(parts, " ")
	}

	// Output to logger
	sl.logger.Println(output)

	// Also write to file if configured
	if sl.file != nil && !sl.config.OutputStdout {
		sl.file.WriteString(output + "\n")
	}
}

// getStackTrace returns the current stack trace
func (sl *StructuredLogger) getStackTrace() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// WithFields creates a new logger with additional fields
func (sl *StructuredLogger) WithFields(fields map[string]interface{}) *StructuredLogger {
	newLogger := &StructuredLogger{
		config: sl.config,
		logger: sl.logger,
		file:   sl.file,
	}

	// Merge default fields with provided fields
	newLogger.config.DefaultFields = make(map[string]interface{})
	for k, v := range sl.config.DefaultFields {
		newLogger.config.DefaultFields[k] = v
	}
	for k, v := range fields {
		newLogger.config.DefaultFields[k] = v
	}

	return newLogger
}

// WithContext creates a new logger with context fields
func (sl *StructuredLogger) WithContext(ctx context.Context) *StructuredLogger {
	fields := make(map[string]interface{})

	if traceID := ctx.Value("trace_id"); traceID != nil {
		fields["trace_id"] = traceID
	}
	if userID := ctx.Value("user_id"); userID != nil {
		fields["user_id"] = userID
	}
	if requestID := ctx.Value("request_id"); requestID != nil {
		fields["request_id"] = requestID
	}

	return sl.WithFields(fields)
}

// LogAPIRequest logs an API request
func (sl *StructuredLogger) LogAPIRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration, fields map[string]interface{}) {
	requestFields := map[string]interface{}{
		"method":      method,
		"path":        path,
		"status_code": statusCode,
		"duration_ms": float64(duration.Nanoseconds()) / 1e6,
		"type":        "api_request",
	}

	// Merge with provided fields
	for k, v := range fields {
		requestFields[k] = v
	}

	level := LogLevelInfo
	if statusCode >= 500 {
		level = LogLevelError
	} else if statusCode >= 400 {
		level = LogLevelWarn
	}

	sl.log(level, ctx, fmt.Sprintf("API request: %s %s", method, path), requestFields)
}

// LogConversion logs a conversion operation
func (sl *StructuredLogger) LogConversion(ctx context.Context, conversionID, userID string, status string, fields map[string]interface{}) {
	conversionFields := map[string]interface{}{
		"conversion_id": conversionID,
		"user_id":       userID,
		"status":        status,
		"type":          "conversion",
	}

	// Merge with provided fields
	for k, v := range fields {
		conversionFields[k] = v
	}

	level := LogLevelInfo
	if status == "failed" || status == "error" {
		level = LogLevelError
	}

	sl.log(level, ctx, fmt.Sprintf("Conversion %s: %s", conversionID, status), conversionFields)
}

// LogPayment logs a payment operation
func (sl *StructuredLogger) LogPayment(ctx context.Context, paymentID, userID string, amount float64, currency, status string, fields map[string]interface{}) {
	paymentFields := map[string]interface{}{
		"payment_id": paymentID,
		"user_id":    userID,
		"amount":     amount,
		"currency":   currency,
		"status":     status,
		"type":       "payment",
	}

	// Merge with provided fields
	for k, v := range fields {
		paymentFields[k] = v
	}

	level := LogLevelInfo
	if status == "failed" || status == "error" {
		level = LogLevelError
	}

	sl.log(level, ctx, fmt.Sprintf("Payment %s: %s %.2f %s", paymentID, status, amount, currency), paymentFields)
}

// LogStorage logs a storage operation
func (sl *StructuredLogger) LogStorage(ctx context.Context, operation, path string, size int64, fields map[string]interface{}) {
	storageFields := map[string]interface{}{
		"operation": operation,
		"path":      path,
		"size":      size,
		"type":      "storage",
	}

	// Merge with provided fields
	for k, v := range fields {
		storageFields[k] = v
	}

	sl.log(LogLevelInfo, ctx, fmt.Sprintf("Storage %s: %s", operation, path), storageFields)
}

// LogSecurity logs a security event
func (sl *StructuredLogger) LogSecurity(ctx context.Context, event, severity string, fields map[string]interface{}) {
	securityFields := map[string]interface{}{
		"event":    event,
		"severity": severity,
		"type":     "security",
	}

	// Merge with provided fields
	for k, v := range fields {
		securityFields[k] = v
	}

	level := LogLevelWarn
	if severity == "high" || severity == "critical" {
		level = LogLevelError
	}

	sl.log(level, ctx, fmt.Sprintf("Security event: %s", event), securityFields)
}

// Close closes the logger and any open files
func (sl *StructuredLogger) Close() error {
	if sl.file != nil {
		return sl.file.Close()
	}
	return nil
}

// GetLogStats returns logging statistics
func (sl *StructuredLogger) GetLogStats(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"config":        sl.config,
		"output_format": sl.config.OutputFormat,
		"min_level":     sl.config.MinLevel,
		"output_file":   sl.config.OutputFile,
		"output_stdout": sl.config.OutputStdout,
	}
}
