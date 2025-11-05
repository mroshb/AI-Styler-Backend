package worker

import (
	"time"
)

// JobStatus represents the status of a worker job
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusCancelled  JobStatus = "cancelled"
)

// JobPriority represents the priority of a job
type JobPriority int

const (
	JobPriorityLow    JobPriority = 1
	JobPriorityNormal JobPriority = 5
	JobPriorityHigh   JobPriority = 10
	JobPriorityUrgent JobPriority = 20
)

// WorkerJob represents a job in the worker queue
type WorkerJob struct {
	ID           string      `json:"id"`
	Type         string      `json:"type"` // "image_conversion", "image_processing", etc.
	ConversionID string      `json:"conversionId"`
	UserID       string      `json:"userId"`
	Priority     JobPriority `json:"priority"`
	Status       JobStatus   `json:"status"`
	WorkerID     string      `json:"workerId,omitempty"`
	RetryCount   int         `json:"retryCount"`
	MaxRetries   int         `json:"maxRetries"`
	ErrorMessage string      `json:"errorMessage,omitempty"`
	Payload      JobPayload  `json:"payload"`
	CreatedAt    time.Time   `json:"createdAt"`
	UpdatedAt    time.Time   `json:"updatedAt"`
	StartedAt    *time.Time  `json:"startedAt,omitempty"`
	CompletedAt  *time.Time  `json:"completedAt,omitempty"`
}

// JobPayload represents the data needed to process a job
type JobPayload struct {
	UserImageID  string                 `json:"userImageId"`
	ClothImageID string                 `json:"clothImageId"`
	Options      map[string]interface{} `json:"options,omitempty"`
}

// WorkerConfig represents configuration for the worker service
type WorkerConfig struct {
	MaxWorkers        int           `json:"maxWorkers"`
	JobTimeout        time.Duration `json:"jobTimeout"`
	RetryDelay        time.Duration `json:"retryDelay"`
	MaxRetries        int           `json:"maxRetries"`
	PollInterval      time.Duration `json:"pollInterval"`
	CleanupInterval   time.Duration `json:"cleanupInterval"`
	HealthCheckPort   int           `json:"healthCheckPort"`
	EnableMetrics     bool          `json:"enableMetrics"`
	EnableHealthCheck bool          `json:"enableHealthCheck"`
}

// WorkerStats represents statistics about the worker service
type WorkerStats struct {
	TotalJobs      int64      `json:"totalJobs"`
	PendingJobs    int64      `json:"pendingJobs"`
	ProcessingJobs int64      `json:"processingJobs"`
	CompletedJobs  int64      `json:"completedJobs"`
	FailedJobs     int64      `json:"failedJobs"`
	ActiveWorkers  int        `json:"activeWorkers"`
	AverageJobTime int64      `json:"averageJobTime"` // in milliseconds
	SuccessRate    float64    `json:"successRate"`
	LastJobTime    *time.Time `json:"lastJobTime,omitempty"`
}

// WorkerHealth represents the health status of a worker
type WorkerHealth struct {
	WorkerID      string    `json:"workerId"`
	Status        string    `json:"status"` // "healthy", "unhealthy", "starting"
	LastSeen      time.Time `json:"lastSeen"`
	JobsProcessed int64     `json:"jobsProcessed"`
	CurrentJob    *string   `json:"currentJob,omitempty"`
	Uptime        int64     `json:"uptime"` // in seconds
}

// GeminiConfig represents configuration for Gemini API
type GeminiConfig struct {
	APIKey                string  `json:"apiKey"`
	BaseURL               string  `json:"baseUrl"`
	Model                 string  `json:"model"`
	MaxRetries            int     `json:"maxRetries"`
	Timeout               int     `json:"timeout"`                 // in seconds
	PreprocessNoiseLevel  float64 `json:"preprocess_noise_level"`  // Noise level for image preprocessing (0.0-1.0)
	PreprocessJpegQuality int     `json:"preprocess_jpeg_quality"` // JPEG quality for preprocessing (1-100)
}

// GeminiRequest represents a request to Gemini API
type GeminiRequest struct {
	Contents         []GeminiContent        `json:"contents"`
	GenerationConfig GeminiGenerationConfig `json:"generationConfig"`
	SafetySettings   []SafetySetting        `json:"safetySettings,omitempty"`
}

// SafetySetting represents safety filter configuration
type SafetySetting struct {
	Category  string `json:"category"`  // e.g., "HARM_CATEGORY_SEXUALLY_EXPLICIT", "HARM_CATEGORY_HATE_SPEECH", etc.
	Threshold string `json:"threshold"` // "BLOCK_NONE", "BLOCK_ONLY_HIGH", "BLOCK_MEDIUM_AND_ABOVE", "BLOCK_LOW_AND_ABOVE"
}

// GeminiContent represents content in a Gemini request
type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

// GeminiPart represents a part of content (text or image)
type GeminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *GeminiInlineData `json:"inlineData,omitempty"`
}

// GeminiInlineData represents inline data (base64 image)
type GeminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

// GeminiGenerationConfig represents generation configuration
type GeminiGenerationConfig struct {
	Temperature     float64 `json:"temperature"`
	TopK            int     `json:"topK"`
	TopP            float64 `json:"topP"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}

// GeminiResponse represents a response from Gemini API
type GeminiResponse struct {
	Candidates    []GeminiCandidate   `json:"candidates"`
	UsageMetadata GeminiUsageMetadata `json:"usageMetadata"`
}

// GeminiCandidate represents a candidate response
type GeminiCandidate struct {
	Content       GeminiContent  `json:"content"`
	FinishReason  string         `json:"finishReason"`
	SafetyRatings []SafetyRating `json:"safetyRatings,omitempty"`
}

// SafetyRating represents safety rating information
type SafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
	Blocked     bool   `json:"blocked"`
}

// GeminiUsageMetadata represents usage metadata
type GeminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// RetryConfig represents configuration for retry mechanism
type RetryConfig struct {
	MaxRetries    int           `json:"maxRetries"`
	InitialDelay  time.Duration `json:"initialDelay"`
	MaxDelay      time.Duration `json:"maxDelay"`
	BackoffFactor float64       `json:"backoffFactor"`
	Jitter        bool          `json:"jitter"`
}

// Default configuration values
const (
	DefaultMaxWorkers      = 5
	DefaultJobTimeout      = 10 * time.Minute
	DefaultRetryDelay      = 30 * time.Second
	DefaultMaxRetries      = 1
	DefaultPollInterval    = 5 * time.Second
	DefaultCleanupInterval = 1 * time.Hour
	DefaultHealthCheckPort = 8081
)
