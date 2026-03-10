package resilience

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/athena/platform-lib/pkg/logger"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name           string
	maxFailures    int
	resetTimeout   time.Duration
	halfOpenMaxReq int
	logger         *logger.Logger

	state        CircuitState
	failures     int
	lastFailTime time.Time
	halfOpenReq  int
	mu           sync.RWMutex
}

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	Name            string        `json:"name"`
	MaxFailures     int           `json:"max_failures"`
	ResetTimeout    time.Duration `json:"reset_timeout"`
	HalfOpenMaxReq  int           `json:"half_open_max_requests"`
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config *CircuitBreakerConfig, logger *logger.Logger) *CircuitBreaker {
	if config.MaxFailures <= 0 {
		config.MaxFailures = 5
	}
	if config.ResetTimeout <= 0 {
		config.ResetTimeout = 60 * time.Second
	}
	if config.HalfOpenMaxReq <= 0 {
		config.HalfOpenMaxReq = 3
	}

	return &CircuitBreaker{
		name:           config.Name,
		maxFailures:    config.MaxFailures,
		resetTimeout:   config.ResetTimeout,
		halfOpenMaxReq: config.HalfOpenMaxReq,
		logger:         logger,
		state:          StateClosed,
	}
}

// Execute executes a function through the circuit breaker
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	if !cb.allowRequest() {
		return errors.New("circuit breaker is open")
	}

	err := fn()
	cb.recordResult(err == nil)
	return err
}

// allowRequest determines if a request should be allowed
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if reset timeout has passed
		if time.Since(cb.lastFailTime) >= cb.resetTimeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.state = StateHalfOpen
			cb.halfOpenReq = 0
			cb.mu.Unlock()
			cb.mu.RLock()
			return true
		}
		return false
	case StateHalfOpen:
		return cb.halfOpenReq < cb.halfOpenMaxReq
	default:
		return false
	}
}

// recordResult records the result of a request
func (cb *CircuitBreaker) recordResult(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if success {
		cb.onSuccess()
	} else {
		cb.onFailure()
	}
}

// onSuccess handles successful request
func (cb *CircuitBreaker) onSuccess() {
	cb.failures = 0
	cb.lastFailTime = time.Time{}

	switch cb.state {
	case StateClosed:
		// Already in good state
	case StateHalfOpen:
		cb.halfOpenReq++
		if cb.halfOpenReq >= cb.halfOpenMaxReq {
			cb.state = StateClosed
			cb.logger.Infof("Circuit breaker %s closed after successful half-open requests", cb.name)
		}
	case StateOpen:
		// Should not happen, but handle gracefully
		cb.state = StateClosed
		cb.logger.Infof("Circuit breaker %s closed after successful request", cb.name)
	}
}

// onFailure handles failed request
func (cb *CircuitBreaker) onFailure() {
	cb.failures++
	cb.lastFailTime = time.Now()

	switch cb.state {
	case StateClosed:
		if cb.failures >= cb.maxFailures {
			cb.state = StateOpen
			cb.logger.Warnf("Circuit breaker %s opened after %d failures", cb.name, cb.failures)
		}
	case StateHalfOpen:
		cb.state = StateOpen
		cb.logger.Warnf("Circuit breaker %s opened after failure in half-open state", cb.name)
	case StateOpen:
		// Already open, just update failure count
	}
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return map[string]interface{}{
		"name":           cb.name,
		"state":          cb.state.String(),
		"failures":       cb.failures,
		"max_failures":   cb.maxFailures,
		"last_fail_time": cb.lastFailTime,
		"half_open_req":  cb.halfOpenReq,
		"reset_timeout":  cb.resetTimeout,
	}
}

// String returns string representation of circuit state
func (cs CircuitState) String() string {
	switch cs {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries      int           `json:"max_retries"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor"`
	RetryableErrors []error       `json:"-"`
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:    3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
	}
}

// Retry executes a function with retry logic
func Retry(ctx context.Context, config *RetryConfig, fn func() error) error {
	if config == nil {
		config = DefaultRetryConfig()
	}

	var lastErr error
	delay := config.InitialDelay

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err, config.RetryableErrors) {
			break
		}

		// Calculate next delay with exponential backoff
		delay = time.Duration(float64(delay) * config.BackoffFactor)
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}
	}

	return lastErr
}

// isRetryableError checks if an error is retryable
func isRetryableError(err error, retryableErrors []error) bool {
	if len(retryableErrors) == 0 {
		// Default retryable errors
		return true
	}

	for _, retryableErr := range retryableErrors {
		if errors.Is(err, retryableErr) {
			return true
		}
	}

	return false
}

// ResilienceClient combines circuit breaker and retry logic
type ResilienceClient struct {
	circuitBreaker *CircuitBreaker
	retryConfig    *RetryConfig
	logger         *logger.Logger
}

// NewResilienceClient creates a new resilience client
func NewResilienceClient(cbConfig *CircuitBreakerConfig, retryConfig *RetryConfig, logger *logger.Logger) *ResilienceClient {
	if retryConfig == nil {
		retryConfig = DefaultRetryConfig()
	}

	return &ResilienceClient{
		circuitBreaker: NewCircuitBreaker(cbConfig, logger),
		retryConfig:    retryConfig,
		logger:         logger,
	}
}

// Execute executes a function with both circuit breaker and retry logic
func (rc *ResilienceClient) Execute(ctx context.Context, fn func() error) error {
	return rc.circuitBreaker.Execute(ctx, func() error {
		return Retry(ctx, rc.retryConfig, fn)
	})
}

// GetCircuitBreaker returns the underlying circuit breaker
func (rc *ResilienceClient) GetCircuitBreaker() *CircuitBreaker {
	return rc.circuitBreaker
}
