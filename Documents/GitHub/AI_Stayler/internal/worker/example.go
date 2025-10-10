package worker

import (
	"context"
	"log"
	"time"
)

// ExampleUsage demonstrates how to use the worker service
// Note: This is a conceptual example. In real usage, dependencies would be injected
func ExampleUsage() {
	log.Println("Worker Service Example")
	log.Println("This example shows how to configure and use the worker service")
	log.Println("In a real application, dependencies would be injected via dependency injection")

	// Example configuration
	config := &WorkerConfig{
		MaxWorkers:        3,
		JobTimeout:        10 * time.Minute,
		RetryDelay:        30 * time.Second,
		MaxRetries:        3,
		PollInterval:      5 * time.Second,
		CleanupInterval:   1 * time.Hour,
		HealthCheckPort:   8081,
		EnableMetrics:     true,
		EnableHealthCheck: true,
	}

	log.Printf("Worker configuration: %+v", config)

	// Example job payload
	payload := JobPayload{
		UserImageID:  "user-img-123",
		ClothImageID: "cloth-img-456",
		Options: map[string]interface{}{
			"style":   "casual",
			"quality": "high",
		},
	}

	log.Printf("Example job payload: %+v", payload)

	log.Println("Worker service example completed")
}

// ExampleJobProcessing demonstrates job processing flow
func ExampleJobProcessing() {
	// Create a simple job
	job := &WorkerJob{
		ID:           "example-job-1",
		Type:         "image_conversion",
		ConversionID: "conv-123",
		UserID:       "user-456",
		Priority:     JobPriorityNormal,
		Status:       JobStatusPending,
		RetryCount:   0,
		MaxRetries:   3,
		Payload: JobPayload{
			UserImageID:  "user-img-1",
			ClothImageID: "cloth-img-1",
			Options: map[string]interface{}{
				"style": "formal",
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	log.Printf("Created job: %s", job.ID)
	log.Printf("Job type: %s", job.Type)
	log.Printf("Job priority: %d", job.Priority)
	log.Printf("Job status: %s", job.Status)
	log.Printf("Job payload: %+v", job.Payload)
}

// ExampleRetryMechanism demonstrates retry logic
func ExampleRetryMechanism() {
	retryService := NewRetryService(nil)

	// Test different error types
	testCases := []struct {
		name string
		err  error
		want bool
	}{
		{"Timeout error", &timeoutError{}, true},
		{"Connection error", &connectionError{}, true},
		{"Invalid input error", &invalidInputError{}, false},
		{"Permission denied error", &permissionDeniedError{}, false},
	}

	for _, tc := range testCases {
		job := &WorkerJob{
			RetryCount: 1,
			MaxRetries: 3,
			Status:     JobStatusPending,
		}

		shouldRetry := retryService.ShouldRetry(context.Background(), job, tc.err)
		log.Printf("Error: %s, Should retry: %v (expected: %v)", tc.name, shouldRetry, tc.want)

		if shouldRetry {
			delay := retryService.GetRetryDelay(context.Background(), job)
			log.Printf("Retry delay: %v", delay)
		}
	}
}

// Example error types for testing
type timeoutError struct{}

func (e *timeoutError) Error() string { return "timeout error" }

type connectionError struct{}

func (e *connectionError) Error() string { return "connection refused" }

type invalidInputError struct{}

func (e *invalidInputError) Error() string { return "invalid input data" }

type permissionDeniedError struct{}

func (e *permissionDeniedError) Error() string { return "permission denied" }
