package crosscutting

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ServiceType represents different types of services
type ServiceType string

const (
	ServiceTypeAnalytics      ServiceType = "analytics"
	ServiceTypeRecommendation ServiceType = "recommendation"
	ServiceTypeCache          ServiceType = "cache"
	ServiceTypeSearch         ServiceType = "search"
	ServiceTypeML             ServiceType = "ml"
	ServiceTypeExternal       ServiceType = "external"
)

// ServicePriority represents service execution priority
type ServicePriority int

const (
	PriorityLow ServicePriority = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

// ServiceHook represents a hook that can be registered for service events
type ServiceHook struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Type     ServiceType            `json:"type"`
	Priority ServicePriority        `json:"priority"`
	Handler  ServiceHookHandler     `json:"-"`
	Config   map[string]interface{} `json:"config"`
	Enabled  bool                   `json:"enabled"`
	Metadata map[string]interface{} `json:"metadata"`
}

// ServiceHookHandler defines the interface for service hook handlers
type ServiceHookHandler interface {
	Execute(ctx context.Context, event *ServiceEvent) error
	GetName() string
	GetType() ServiceType
	GetPriority() ServicePriority
	IsEnabled() bool
}

// ServiceEvent represents an event that can trigger service hooks
type ServiceEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	UserID    string                 `json:"user_id,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	TraceID   string                 `json:"trace_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// ServicePipeline represents a pipeline of services
type ServicePipeline struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Services []*ServiceHook         `json:"services"`
	Config   map[string]interface{} `json:"config"`
	Enabled  bool                   `json:"enabled"`
	Metadata map[string]interface{} `json:"metadata"`
}

// ExtensibilityConfig represents configuration for the extensibility framework
type ExtensibilityConfig struct {
	// Service registration
	AutoDiscovery bool `json:"auto_discovery"`

	// Pipeline configuration
	MaxPipelines int `json:"max_pipelines"`
	MaxHooks     int `json:"max_hooks"`

	// Execution settings
	ParallelExecution bool `json:"parallel_execution"`
	TimeoutSeconds    int  `json:"timeout_seconds"`

	// Error handling
	ContinueOnError bool `json:"continue_on_error"`
	RetryOnError    bool `json:"retry_on_error"`
	MaxRetries      int  `json:"max_retries"`

	// Monitoring
	EnableMetrics bool `json:"enable_metrics"`
	EnableLogging bool `json:"enable_logging"`
}

// DefaultExtensibilityConfig returns default extensibility configuration
func DefaultExtensibilityConfig() *ExtensibilityConfig {
	return &ExtensibilityConfig{
		AutoDiscovery:     true,
		MaxPipelines:      100,
		MaxHooks:          1000,
		ParallelExecution: true,
		TimeoutSeconds:    30,
		ContinueOnError:   true,
		RetryOnError:      true,
		MaxRetries:        3,
		EnableMetrics:     true,
		EnableLogging:     true,
	}
}

// ExtensibilityFramework provides comprehensive extensibility functionality
type ExtensibilityFramework struct {
	config    *ExtensibilityConfig
	hooks     map[string]*ServiceHook
	pipelines map[string]*ServicePipeline
	mu        sync.RWMutex
	logger    *StructuredLogger
	metrics   *ServiceMetrics
}

// ServiceMetrics provides metrics for service execution
type ServiceMetrics struct {
	HookExecutions     map[string]int64   `json:"hook_executions"`
	HookErrors         map[string]int64   `json:"hook_errors"`
	PipelineExecutions map[string]int64   `json:"pipeline_executions"`
	PipelineErrors     map[string]int64   `json:"pipeline_errors"`
	ExecutionTimes     map[string]float64 `json:"execution_times"`
}

// NewExtensibilityFramework creates a new extensibility framework
func NewExtensibilityFramework(config *ExtensibilityConfig, logger *StructuredLogger) *ExtensibilityFramework {
	if config == nil {
		config = DefaultExtensibilityConfig()
	}

	return &ExtensibilityFramework{
		config:    config,
		hooks:     make(map[string]*ServiceHook),
		pipelines: make(map[string]*ServicePipeline),
		logger:    logger,
		metrics: &ServiceMetrics{
			HookExecutions:     make(map[string]int64),
			HookErrors:         make(map[string]int64),
			PipelineExecutions: make(map[string]int64),
			PipelineErrors:     make(map[string]int64),
			ExecutionTimes:     make(map[string]float64),
		},
	}
}

// RegisterHook registers a service hook
func (ef *ExtensibilityFramework) RegisterHook(hook *ServiceHook) error {
	ef.mu.Lock()
	defer ef.mu.Unlock()

	if len(ef.hooks) >= ef.config.MaxHooks {
		return fmt.Errorf("maximum number of hooks (%d) reached", ef.config.MaxHooks)
	}

	if hook.ID == "" {
		hook.ID = fmt.Sprintf("hook_%d", len(ef.hooks))
	}

	ef.hooks[hook.ID] = hook

	if ef.logger != nil {
		ef.logger.Info(context.Background(), "Service hook registered", map[string]interface{}{
			"hook_id":   hook.ID,
			"hook_name": hook.Name,
			"hook_type": hook.Type,
			"priority":  hook.Priority,
		})
	}

	return nil
}

// UnregisterHook unregisters a service hook
func (ef *ExtensibilityFramework) UnregisterHook(hookID string) error {
	ef.mu.Lock()
	defer ef.mu.Unlock()

	if _, exists := ef.hooks[hookID]; !exists {
		return fmt.Errorf("hook %s not found", hookID)
	}

	delete(ef.hooks, hookID)

	if ef.logger != nil {
		ef.logger.Info(context.Background(), "Service hook unregistered", map[string]interface{}{
			"hook_id": hookID,
		})
	}

	return nil
}

// CreatePipeline creates a new service pipeline
func (ef *ExtensibilityFramework) CreatePipeline(pipeline *ServicePipeline) error {
	ef.mu.Lock()
	defer ef.mu.Unlock()

	if len(ef.pipelines) >= ef.config.MaxPipelines {
		return fmt.Errorf("maximum number of pipelines (%d) reached", ef.config.MaxPipelines)
	}

	if pipeline.ID == "" {
		pipeline.ID = fmt.Sprintf("pipeline_%d", len(ef.pipelines))
	}

	// Validate hooks exist
	for _, hook := range pipeline.Services {
		if _, exists := ef.hooks[hook.ID]; !exists {
			return fmt.Errorf("hook %s not found", hook.ID)
		}
	}

	ef.pipelines[pipeline.ID] = pipeline

	if ef.logger != nil {
		ef.logger.Info(context.Background(), "Service pipeline created", map[string]interface{}{
			"pipeline_id":   pipeline.ID,
			"pipeline_name": pipeline.Name,
			"hook_count":    len(pipeline.Services),
		})
	}

	return nil
}

// ExecutePipeline executes a service pipeline
func (ef *ExtensibilityFramework) ExecutePipeline(ctx context.Context, pipelineID string, event *ServiceEvent) error {
	ef.mu.RLock()
	pipeline, exists := ef.pipelines[pipelineID]
	ef.mu.RUnlock()

	if !exists {
		return fmt.Errorf("pipeline %s not found", pipelineID)
	}

	if !pipeline.Enabled {
		return fmt.Errorf("pipeline %s is disabled", pipelineID)
	}

	// Update metrics
	ef.metrics.PipelineExecutions[pipelineID]++

	if ef.logger != nil {
		ef.logger.Info(ctx, "Executing service pipeline", map[string]interface{}{
			"pipeline_id": pipelineID,
			"event_type":  event.Type,
			"hook_count":  len(pipeline.Services),
		})
	}

	// Execute hooks
	if ef.config.ParallelExecution {
		return ef.executePipelineParallel(ctx, pipeline, event)
	} else {
		return ef.executePipelineSequential(ctx, pipeline, event)
	}
}

// ExecuteHooksByType executes all hooks of a specific type
func (ef *ExtensibilityFramework) ExecuteHooksByType(ctx context.Context, serviceType ServiceType, event *ServiceEvent) error {
	ef.mu.RLock()
	var hooks []*ServiceHook
	for _, hook := range ef.hooks {
		if hook.Type == serviceType && hook.Enabled {
			hooks = append(hooks, hook)
		}
	}
	ef.mu.RUnlock()

	if ef.logger != nil {
		ef.logger.Info(ctx, "Executing hooks by type", map[string]interface{}{
			"service_type": serviceType,
			"hook_count":   len(hooks),
			"event_type":   event.Type,
		})
	}

	// Sort by priority
	ef.sortHooksByPriority(hooks)

	// Execute hooks
	if ef.config.ParallelExecution {
		return ef.executeHooksParallel(ctx, hooks, event)
	} else {
		return ef.executeHooksSequential(ctx, hooks, event)
	}
}

// executePipelineSequential executes pipeline hooks sequentially
func (ef *ExtensibilityFramework) executePipelineSequential(ctx context.Context, pipeline *ServicePipeline, event *ServiceEvent) error {
	for _, hook := range pipeline.Services {
		if !hook.Enabled {
			continue
		}

		err := ef.executeHook(ctx, hook, event)
		if err != nil {
			ef.metrics.PipelineErrors[pipeline.ID]++

			if !ef.config.ContinueOnError {
				return fmt.Errorf("pipeline execution failed at hook %s: %w", hook.ID, err)
			}

			if ef.logger != nil {
				ef.logger.Error(ctx, "Pipeline hook execution failed", map[string]interface{}{
					"pipeline_id": pipeline.ID,
					"hook_id":     hook.ID,
					"error":       err.Error(),
				})
			}
		}
	}

	return nil
}

// executePipelineParallel executes pipeline hooks in parallel
func (ef *ExtensibilityFramework) executePipelineParallel(ctx context.Context, pipeline *ServicePipeline, event *ServiceEvent) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(pipeline.Services))

	for _, hook := range pipeline.Services {
		if !hook.Enabled {
			continue
		}

		wg.Add(1)
		go func(h *ServiceHook) {
			defer wg.Done()
			err := ef.executeHook(ctx, h, event)
			if err != nil {
				errChan <- fmt.Errorf("hook %s failed: %w", h.ID, err)
			}
		}(hook)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		ef.metrics.PipelineErrors[pipeline.ID]++

		if !ef.config.ContinueOnError {
			return fmt.Errorf("pipeline execution failed with %d errors", len(errors))
		}

		if ef.logger != nil {
			ef.logger.Error(ctx, "Pipeline execution completed with errors", map[string]interface{}{
				"pipeline_id": pipeline.ID,
				"error_count": len(errors),
			})
		}
	}

	return nil
}

// executeHooksSequential executes hooks sequentially
func (ef *ExtensibilityFramework) executeHooksSequential(ctx context.Context, hooks []*ServiceHook, event *ServiceEvent) error {
	for _, hook := range hooks {
		err := ef.executeHook(ctx, hook, event)
		if err != nil {
			ef.metrics.HookErrors[hook.ID]++

			if !ef.config.ContinueOnError {
				return fmt.Errorf("hook execution failed at %s: %w", hook.ID, err)
			}

			if ef.logger != nil {
				ef.logger.Error(ctx, "Hook execution failed", map[string]interface{}{
					"hook_id": hook.ID,
					"error":   err.Error(),
				})
			}
		}
	}

	return nil
}

// executeHooksParallel executes hooks in parallel
func (ef *ExtensibilityFramework) executeHooksParallel(ctx context.Context, hooks []*ServiceHook, event *ServiceEvent) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(hooks))

	for _, hook := range hooks {
		wg.Add(1)
		go func(h *ServiceHook) {
			defer wg.Done()
			err := ef.executeHook(ctx, h, event)
			if err != nil {
				errChan <- fmt.Errorf("hook %s failed: %w", h.ID, err)
			}
		}(hook)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 && !ef.config.ContinueOnError {
		return fmt.Errorf("hook execution failed with %d errors", len(errors))
	}

	return nil
}

// executeHook executes a single hook
func (ef *ExtensibilityFramework) executeHook(ctx context.Context, hook *ServiceHook, event *ServiceEvent) error {
	ef.metrics.HookExecutions[hook.ID]++

	if ef.logger != nil {
		ef.logger.Debug(ctx, "Executing service hook", map[string]interface{}{
			"hook_id":    hook.ID,
			"hook_name":  hook.Name,
			"event_type": event.Type,
		})
	}

	// Execute hook handler
	err := hook.Handler.Execute(ctx, event)
	if err != nil {
		ef.metrics.HookErrors[hook.ID]++
		return err
	}

	return nil
}

// sortHooksByPriority sorts hooks by priority (highest first)
func (ef *ExtensibilityFramework) sortHooksByPriority(hooks []*ServiceHook) {
	// Simple bubble sort for priority
	for i := 0; i < len(hooks)-1; i++ {
		for j := 0; j < len(hooks)-i-1; j++ {
			if hooks[j].Priority < hooks[j+1].Priority {
				hooks[j], hooks[j+1] = hooks[j+1], hooks[j]
			}
		}
	}
}

// GetHook returns a hook by ID
func (ef *ExtensibilityFramework) GetHook(hookID string) (*ServiceHook, error) {
	ef.mu.RLock()
	defer ef.mu.RUnlock()

	hook, exists := ef.hooks[hookID]
	if !exists {
		return nil, fmt.Errorf("hook %s not found", hookID)
	}

	return hook, nil
}

// GetPipeline returns a pipeline by ID
func (ef *ExtensibilityFramework) GetPipeline(pipelineID string) (*ServicePipeline, error) {
	ef.mu.RLock()
	defer ef.mu.RUnlock()

	pipeline, exists := ef.pipelines[pipelineID]
	if !exists {
		return nil, fmt.Errorf("pipeline %s not found", pipelineID)
	}

	return pipeline, nil
}

// ListHooks returns all registered hooks
func (ef *ExtensibilityFramework) ListHooks() []*ServiceHook {
	ef.mu.RLock()
	defer ef.mu.RUnlock()

	hooks := make([]*ServiceHook, 0, len(ef.hooks))
	for _, hook := range ef.hooks {
		hooks = append(hooks, hook)
	}

	return hooks
}

// ListPipelines returns all registered pipelines
func (ef *ExtensibilityFramework) ListPipelines() []*ServicePipeline {
	ef.mu.RLock()
	defer ef.mu.RUnlock()

	pipelines := make([]*ServicePipeline, 0, len(ef.pipelines))
	for _, pipeline := range ef.pipelines {
		pipelines = append(pipelines, pipeline)
	}

	return pipelines
}

// GetMetrics returns service execution metrics
func (ef *ExtensibilityFramework) GetMetrics() *ServiceMetrics {
	return ef.metrics
}

// GetStats returns extensibility framework statistics
func (ef *ExtensibilityFramework) GetStats(ctx context.Context) map[string]interface{} {
	ef.mu.RLock()
	defer ef.mu.RUnlock()

	return map[string]interface{}{
		"config":             ef.config,
		"hook_count":         len(ef.hooks),
		"pipeline_count":     len(ef.pipelines),
		"metrics":            ef.metrics,
		"auto_discovery":     ef.config.AutoDiscovery,
		"parallel_execution": ef.config.ParallelExecution,
	}
}

// CreateEvent creates a new service event
func (ef *ExtensibilityFramework) CreateEvent(eventType, source string, data map[string]interface{}) *ServiceEvent {
	return &ServiceEvent{
		ID:        fmt.Sprintf("event_%d", time.Now().UnixNano()),
		Type:      eventType,
		Source:    source,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
}

// EnableHook enables a hook
func (ef *ExtensibilityFramework) EnableHook(hookID string) error {
	ef.mu.Lock()
	defer ef.mu.Unlock()

	hook, exists := ef.hooks[hookID]
	if !exists {
		return fmt.Errorf("hook %s not found", hookID)
	}

	hook.Enabled = true
	return nil
}

// DisableHook disables a hook
func (ef *ExtensibilityFramework) DisableHook(hookID string) error {
	ef.mu.Lock()
	defer ef.mu.Unlock()

	hook, exists := ef.hooks[hookID]
	if !exists {
		return fmt.Errorf("hook %s not found", hookID)
	}

	hook.Enabled = false
	return nil
}
