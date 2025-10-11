package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// Job represents a background job
type Job struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Data       map[string]interface{} `json:"data"`
	CreatedAt  time.Time              `json:"created_at"`
	Retries    int                    `json:"retries"`
	MaxRetries int                    `json:"max_retries"`
}

// RedisJobQueue manages background jobs using Redis
type RedisJobQueue struct {
	redisClient *redis.Client
	queueName   string
}

// NewRedisJobQueue creates a new Redis job queue
func NewRedisJobQueue(redisClient *redis.Client, queueName string) *RedisJobQueue {
	return &RedisJobQueue{
		redisClient: redisClient,
		queueName:   queueName,
	}
}

// Enqueue adds a job to the queue
func (q *RedisJobQueue) Enqueue(ctx context.Context, job *Job) error {
	job.CreatedAt = time.Now()
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	return q.redisClient.LPush(ctx, q.queueName, jobData).Err()
}

// Dequeue removes and returns a job from the queue
func (q *RedisJobQueue) Dequeue(ctx context.Context, timeout time.Duration) (*Job, error) {
	result, err := q.redisClient.BRPop(ctx, timeout, q.queueName).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // No jobs available
		}
		return nil, fmt.Errorf("failed to dequeue job: %w", err)
	}

	var job Job
	err = json.Unmarshal([]byte(result[1]), &job)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

// RetryJob adds a job back to the queue for retry
func (q *RedisJobQueue) RetryJob(ctx context.Context, job *Job) error {
	job.Retries++
	if job.Retries > job.MaxRetries {
		// Move to dead letter queue
		return q.MoveToDeadLetterQueue(ctx, job)
	}

	// Add delay before retry (exponential backoff)
	delay := time.Duration(job.Retries*job.Retries) * time.Second
	time.Sleep(delay)

	return q.Enqueue(ctx, job)
}

// MoveToDeadLetterQueue moves failed jobs to dead letter queue
func (q *RedisJobQueue) MoveToDeadLetterQueue(ctx context.Context, job *Job) error {
	deadLetterQueue := q.queueName + ":dead"
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job for dead letter queue: %w", err)
	}

	return q.redisClient.LPush(ctx, deadLetterQueue, jobData).Err()
}

// GetQueueLength returns the number of jobs in the queue
func (q *RedisJobQueue) GetQueueLength(ctx context.Context) (int64, error) {
	return q.redisClient.LLen(ctx, q.queueName).Result()
}

// GetDeadLetterQueueLength returns the number of jobs in the dead letter queue
func (q *RedisJobQueue) GetDeadLetterQueueLength(ctx context.Context) (int64, error) {
	deadLetterQueue := q.queueName + ":dead"
	return q.redisClient.LLen(ctx, deadLetterQueue).Result()
}

// WorkerPool manages multiple workers
type WorkerPool struct {
	queue       *RedisJobQueue
	workers     []*QueueWorker
	numWorkers  int
	stopChan    chan struct{}
	jobHandlers map[string]JobHandler
}

// JobHandler processes a specific type of job
type JobHandler func(ctx context.Context, job *Job) error

// NewWorkerPool creates a new worker pool
func NewWorkerPool(queue *RedisJobQueue, numWorkers int) *WorkerPool {
	return &WorkerPool{
		queue:       queue,
		numWorkers:  numWorkers,
		stopChan:    make(chan struct{}),
		jobHandlers: make(map[string]JobHandler),
	}
}

// RegisterHandler registers a handler for a specific job type
func (p *WorkerPool) RegisterHandler(jobType string, handler JobHandler) {
	p.jobHandlers[jobType] = handler
}

// Start starts the worker pool
func (p *WorkerPool) Start(ctx context.Context) error {
	for i := 0; i < p.numWorkers; i++ {
		worker := NewQueueWorker(p.queue, p.jobHandlers, p.stopChan)
		p.workers = append(p.workers, worker)
		go worker.Start(ctx)
	}
	return nil
}

// Stop stops the worker pool
func (p *WorkerPool) Stop(ctx context.Context) error {
	close(p.stopChan)

	// Wait for all workers to stop
	for _, worker := range p.workers {
		worker.Wait()
	}

	return nil
}

// QueueWorker processes jobs from the queue
type QueueWorker struct {
	queue    *RedisJobQueue
	handlers map[string]JobHandler
	stopChan chan struct{}
}

// NewQueueWorker creates a new queue worker
func NewQueueWorker(queue *RedisJobQueue, handlers map[string]JobHandler, stopChan chan struct{}) *QueueWorker {
	return &QueueWorker{
		queue:    queue,
		handlers: handlers,
		stopChan: stopChan,
	}
}

// Start starts the queue worker
func (w *QueueWorker) Start(ctx context.Context) {
	for {
		select {
		case <-w.stopChan:
			return
		default:
			job, err := w.queue.Dequeue(ctx, 5*time.Second)
			if err != nil {
				continue
			}
			if job == nil {
				continue
			}

			w.processJob(ctx, job)
		}
	}
}

// processJob processes a single job
func (w *QueueWorker) processJob(ctx context.Context, job *Job) {
	handler, exists := w.handlers[job.Type]
	if !exists {
		// Unknown job type, move to dead letter queue
		w.queue.MoveToDeadLetterQueue(ctx, job)
		return
	}

	err := handler(ctx, job)
	if err != nil {
		// Job failed, retry or move to dead letter queue
		w.queue.RetryJob(ctx, job)
	}
}

// Wait waits for the queue worker to stop
func (w *QueueWorker) Wait() {
	// Worker stops when stopChan is closed
}

// ConversionJobHandler handles conversion jobs
func ConversionJobHandler(ctx context.Context, job *Job) error {
	conversionID, ok := job.Data["conversion_id"].(string)
	if !ok {
		return fmt.Errorf("missing conversion_id in job data")
	}

	// Mock conversion processing
	fmt.Printf("Processing conversion %s\n", conversionID)

	// Simulate processing time
	time.Sleep(2 * time.Second)

	// Update conversion status in database
	// This would be implemented with actual database calls

	return nil
}

// ImageProcessingJobHandler handles image processing jobs
func ImageProcessingJobHandler(ctx context.Context, job *Job) error {
	imageID, ok := job.Data["image_id"].(string)
	if !ok {
		return fmt.Errorf("missing image_id in job data")
	}

	// Mock image processing
	fmt.Printf("Processing image %s\n", imageID)

	// Simulate processing time
	time.Sleep(1 * time.Second)

	return nil
}

// NotificationJobHandler handles notification jobs
func NotificationJobHandler(ctx context.Context, job *Job) error {
	userID, ok := job.Data["user_id"].(string)
	if !ok {
		return fmt.Errorf("missing user_id in job data")
	}

	message, ok := job.Data["message"].(string)
	if !ok {
		return fmt.Errorf("missing message in job data")
	}

	// Mock notification sending
	fmt.Printf("Sending notification to user %s: %s\n", userID, message)

	return nil
}
