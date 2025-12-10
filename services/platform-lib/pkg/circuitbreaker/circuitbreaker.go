package circuitbreaker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/athena/platform-lib/pkg/logger"
)

// State represents the circuit breaker state
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// Config defines circuit breaker behavior
type Config struct {
	MaxFailures      int           // Number of failures before opening
	ResetTimeout     time.Duration // Time to wait before trying half-open
	SuccessThreshold int           // Number of successes in half-open to close
	FailureThreshold int           // Number of failures in half-open to open again
	Timeout          time.Duration // Operation timeout
	MonitoringPeriod time.Duration // Period for monitoring metrics
	Enabled          bool          // Whether circuit breaker is enabled
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() *Config {
	return &Config{
		MaxFailures:      5,
		ResetTimeout:     60 * time.Second,
		SuccessThreshold: 3,
		FailureThreshold: 1,
		Timeout:          30 * time.Second,
		MonitoringPeriod: 10 * time.Second,
		Enabled:          true,
	}
}

// Metrics tracks circuit breaker statistics
type Metrics struct {
	TotalRequests      int64     `json:"total_requests"`
	SuccessfulReqs     int64     `json:"successful_requests"`
	FailedReqs         int64     `json:"failed_requests"`
	TimeoutReqs        int64     `json:"timeout_requests"`
	RejectedReqs       int64     `json:"rejected_requests"`
	LastFailureTime    time.Time `json:"last_failure_time"`
	LastSuccessTime    time.Time `json:"last_success_time"`
	ConsecutiveFails   int       `json:"consecutive_failures"`
	ConsecutiveSuccess int       `json:"consecutive_successes"`
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name    string
	config  *Config
	logger  logger.Logger
	mu      sync.RWMutex
	state   State
	metrics Metrics
}

// NewCircuitBreaker creates a new circuit breaker instance
func NewCircuitBreaker(name string, config *Config, logger logger.Logger) *CircuitBreaker {
	if config == nil {
		config = DefaultConfig()
	}

	cb := &CircuitBreaker{
		name:   name,
		config: config,
		logger: logger,
		state:  StateClosed,
	}

	// Start monitoring goroutine
	go cb.monitor()

	logger.Info("Circuit breaker created", "name", name, "config", config)
	return cb
}

// Execute executes a function through the circuit breaker
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	if !cb.config.Enabled {
		return fn(ctx)
	}

	// Check if circuit is open
	if !cb.canExecute() {
		cb.recordRejected()
		return fmt.Errorf("circuit breaker '%s' is OPEN", cb.name)
	}

	// Execute with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, cb.config.Timeout)
	defer cancel()

	resultChan := make(chan error, 1)
	go func() {
		resultChan <- fn(timeoutCtx)
	}()

	select {
	case err := <-resultChan:
		if err != nil {
			cb.recordFailure()
			return err
		}
		cb.recordSuccess()
		return nil

	case <-timeoutCtx.Done():
		cb.recordTimeout()
		return fmt.Errorf("circuit breaker '%s' operation timeout", cb.name)
	}
}

// ExecuteWithResult executes a function that returns a result through the circuit breaker
func ExecuteWithResult[T any](cb *CircuitBreaker, ctx context.Context, fn func(ctx context.Context) (T, error)) (T, error) {
	var result T

	if !cb.config.Enabled {
		return fn(ctx)
	}

	// Check if circuit is open
	if !cb.canExecute() {
		cb.recordRejected()
		return result, fmt.Errorf("circuit breaker '%s' is OPEN", cb.name)
	}

	// Execute with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, cb.config.Timeout)
	defer cancel()

	resultChan := make(chan struct {
		value T
		err   error
	}, 1)

	go func() {
		value, err := fn(timeoutCtx)
		resultChan <- struct {
			value T
			err   error
		}{value, err}
	}()

	select {
	case res := <-resultChan:
		if res.err != nil {
			cb.recordFailure()
			return result, res.err
		}
		cb.recordSuccess()
		return res.value, nil

	case <-timeoutCtx.Done():
		cb.recordTimeout()
		return result, fmt.Errorf("circuit breaker '%s' operation timeout", cb.name)
	}
}

// canExecute determines if the circuit breaker should allow execution
func (cb *CircuitBreaker) canExecute() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if reset timeout has passed
		if time.Since(cb.metrics.LastFailureTime) > cb.config.ResetTimeout {
			cb.mu.RUnlock()
			cb.transitionToHalfOpen()
			cb.mu.RLock()
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// recordSuccess records a successful operation
func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.metrics.TotalRequests++
	cb.metrics.SuccessfulReqs++
	cb.metrics.LastSuccessTime = time.Now()
	cb.metrics.ConsecutiveFails = 0
	cb.metrics.ConsecutiveSuccess++

	switch cb.state {
	case StateHalfOpen:
		// Check if we should close the circuit
		if cb.metrics.ConsecutiveSuccess >= cb.config.SuccessThreshold {
			cb.transitionToClosed()
		}
	}

	cb.logger.Debug("Circuit breaker success recorded",
		"name", cb.name,
		"state", cb.state,
		"consecutive_successes", cb.metrics.ConsecutiveSuccess)
}

// recordFailure records a failed operation
func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.metrics.TotalRequests++
	cb.metrics.FailedReqs++
	cb.metrics.LastFailureTime = time.Now()
	cb.metrics.ConsecutiveFails++
	cb.metrics.ConsecutiveSuccess = 0

	switch cb.state {
	case StateClosed:
		// Check if we should open the circuit
		if cb.metrics.ConsecutiveFails >= cb.config.MaxFailures {
			cb.transitionToOpen()
		}
	case StateHalfOpen:
		// Any failure in half-open opens the circuit
		cb.transitionToOpen()
	}

	cb.logger.Debug("Circuit breaker failure recorded",
		"name", cb.name,
		"state", cb.state,
		"consecutive_failures", cb.metrics.ConsecutiveFails)
}

// recordTimeout records a timeout operation
func (cb *CircuitBreaker) recordTimeout() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.metrics.TotalRequests++
	cb.metrics.TimeoutReqs++
	cb.metrics.LastFailureTime = time.Now()
	cb.metrics.ConsecutiveFails++
	cb.metrics.ConsecutiveSuccess = 0

	switch cb.state {
	case StateClosed:
		// Check if we should open the circuit
		if cb.metrics.ConsecutiveFails >= cb.config.MaxFailures {
			cb.transitionToOpen()
		}
	case StateHalfOpen:
		// Any failure in half-open opens the circuit
		cb.transitionToOpen()
	}

	cb.logger.Debug("Circuit breaker timeout recorded",
		"name", cb.name,
		"state", cb.state,
		"consecutive_failures", cb.metrics.ConsecutiveFails)
}

// recordRejected records a rejected operation
func (cb *CircuitBreaker) recordRejected() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.metrics.RejectedReqs++
}

// transitionToOpen transitions the circuit breaker to OPEN state
func (cb *CircuitBreaker) transitionToOpen() {
	cb.state = StateOpen
	cb.logger.Warn("Circuit breaker opened",
		"name", cb.name,
		"failures", cb.metrics.ConsecutiveFails,
		"max_failures", cb.config.MaxFailures)
}

// transitionToHalfOpen transitions the circuit breaker to HALF_OPEN state
func (cb *CircuitBreaker) transitionToHalfOpen() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateHalfOpen
	cb.metrics.ConsecutiveFails = 0
	cb.metrics.ConsecutiveSuccess = 0

	cb.logger.Info("Circuit breaker transitioned to half-open", "name", cb.name)
}

// transitionToClosed transitions the circuit breaker to CLOSED state
func (cb *CircuitBreaker) transitionToClosed() {
	cb.state = StateClosed
	cb.metrics.ConsecutiveFails = 0
	cb.metrics.ConsecutiveSuccess = 0

	cb.logger.Info("Circuit breaker closed",
		"name", cb.name,
		"successes", cb.metrics.ConsecutiveSuccess)
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetMetrics returns the current metrics
func (cb *CircuitBreaker) GetMetrics() Metrics {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.metrics
}

// Reset resets the circuit breaker to CLOSED state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.metrics = Metrics{}

	cb.logger.Info("Circuit breaker reset", "name", cb.name)
}

// monitor periodically logs circuit breaker metrics
func (cb *CircuitBreaker) monitor() {
	ticker := time.NewTicker(cb.config.MonitoringPeriod)
	defer ticker.Stop()

	for range ticker.C {
		cb.mu.RLock()
		metrics := cb.metrics
		state := cb.state
		cb.mu.RUnlock()

		if metrics.TotalRequests > 0 {
			successRate := float64(metrics.SuccessfulReqs) / float64(metrics.TotalRequests) * 100
			cb.logger.Info("Circuit breaker metrics",
				"name", cb.name,
				"state", state,
				"total_requests", metrics.TotalRequests,
				"success_rate", fmt.Sprintf("%.2f%%", successRate),
				"consecutive_failures", metrics.ConsecutiveFails,
				"consecutive_successes", metrics.ConsecutiveSuccess)
		}
	}
}

// CircuitBreakerManager manages multiple circuit breakers
type CircuitBreakerManager struct {
	breakers map[string]*CircuitBreaker
	logger   logger.Logger
	mu       sync.RWMutex
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager(logger logger.Logger) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
		logger:   logger,
	}
}

// GetOrCreate gets or creates a circuit breaker
func (m *CircuitBreakerManager) GetOrCreate(name string, config *Config) *CircuitBreaker {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cb, exists := m.breakers[name]; exists {
		return cb
	}

	cb := NewCircuitBreaker(name, config, m.logger)
	m.breakers[name] = cb
	return cb
}

// GetAllMetrics returns metrics for all circuit breakers
func (m *CircuitBreakerManager) GetAllMetrics() map[string]Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]Metrics)
	for name, cb := range m.breakers {
		result[name] = cb.GetMetrics()
	}
	return result
}

// ResetAll resets all circuit breakers
func (m *CircuitBreakerManager) ResetAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, cb := range m.breakers {
		cb.Reset()
		m.logger.Info("Circuit breaker reset", "name", name)
	}
}
