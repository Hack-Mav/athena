package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/athena/platform-lib/pkg/logger"
)

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxAttempts        int           // Maximum number of retry attempts
	InitialDelay       time.Duration // Initial delay between retries
	MaxDelay           time.Duration // Maximum delay between retries
	BackoffFactor      float64       // Multiplier for exponential backoff
	RetryableErrors    []error       // Errors that should be retried
	NonRetryableErrors []error       // Errors that should not be retried
	Jitter             bool          // Add random jitter to prevent thundering herd
}

// DefaultRetryConfig returns a sensible default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
	}
}

// RetryableFunc represents a function that can be retried
type RetryableFunc func(ctx context.Context) error

// Retryer provides retry functionality with exponential backoff
type Retryer struct {
	config *RetryConfig
	logger logger.Logger
}

// NewRetryer creates a new retryer instance
func NewRetryer(config *RetryConfig, logger logger.Logger) *Retryer {
	if config == nil {
		config = DefaultRetryConfig()
	}

	return &Retryer{
		config: config,
		logger: logger,
	}
}

// Execute executes a function with retry logic
func (r *Retryer) Execute(ctx context.Context, fn RetryableFunc) error {
	var lastErr error

	for attempt := 1; attempt <= r.config.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := fn(ctx)
		if err == nil {
			if attempt > 1 {
				r.logger.Debug("Operation succeeded after retry", "attempt", attempt)
			}
			return nil
		}

		lastErr = err

		// Check if error is non-retryable
		if r.isNonRetryableError(err) {
			r.logger.Error("Non-retryable error encountered", "error", err, "attempt", attempt)
			return err
		}

		// Check if error is retryable
		if !r.isRetryableError(err) {
			r.logger.Error("Non-retryable error encountered", "error", err, "attempt", attempt)
			return err
		}

		// If this is the last attempt, don't wait
		if attempt == r.config.MaxAttempts {
			break
		}

		// Calculate delay for next attempt
		delay := r.calculateDelay(attempt)

		r.logger.Warn("Operation failed, retrying",
			"error", err,
			"attempt", attempt,
			"max_attempts", r.config.MaxAttempts,
			"delay", delay)

		// Wait before next attempt
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	r.logger.Error("Operation failed after all retries",
		"error", lastErr,
		"attempts", r.config.MaxAttempts)

	return fmt.Errorf("operation failed after %d attempts: %w", r.config.MaxAttempts, lastErr)
}

// ExecuteWithResult executes a function that returns a result with retry logic
func ExecuteWithResult[T any](r *Retryer, ctx context.Context, fn func(ctx context.Context) (T, error)) (T, error) {
	var result T
	var lastErr error

	for attempt := 1; attempt <= r.config.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		res, err := fn(ctx)
		if err == nil {
			if attempt > 1 {
				r.logger.Debug("Operation succeeded after retry", "attempt", attempt)
			}
			return res, nil
		}

		lastErr = err

		// Check if error is non-retryable
		if r.isNonRetryableError(err) {
			r.logger.Error("Non-retryable error encountered", "error", err, "attempt", attempt)
			return result, err
		}

		// Check if error is retryable
		if !r.isRetryableError(err) {
			r.logger.Error("Non-retryable error encountered", "error", err, "attempt", attempt)
			return result, err
		}

		// If this is the last attempt, don't wait
		if attempt == r.config.MaxAttempts {
			break
		}

		// Calculate delay for next attempt
		delay := r.calculateDelay(attempt)

		r.logger.Warn("Operation failed, retrying",
			"error", err,
			"attempt", attempt,
			"max_attempts", r.config.MaxAttempts,
			"delay", delay)

		// Wait before next attempt
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(delay):
		}
	}

	r.logger.Error("Operation failed after all retries",
		"error", lastErr,
		"attempts", r.config.MaxAttempts)

	return result, fmt.Errorf("operation failed after %d attempts: %w", r.config.MaxAttempts, lastErr)
}

// calculateDelay calculates the delay for the next retry attempt
func (r *Retryer) calculateDelay(attempt int) time.Duration {
	// Exponential backoff: delay = initial_delay * (backoff_factor ^ (attempt - 1))
	delay := float64(r.config.InitialDelay) * math.Pow(r.config.BackoffFactor, float64(attempt-1))

	// Apply maximum delay limit
	if delay > float64(r.config.MaxDelay) {
		delay = float64(r.config.MaxDelay)
	}

	// Add jitter if enabled
	if r.config.Jitter {
		// Add up to 25% random jitter
		jitter := delay * 0.25 * (rand.Float64()*2 - 1) // Random value between -0.25 and +0.25
		delay += jitter
	}

	return time.Duration(delay)
}

// isRetryableError checks if an error should be retried
func (r *Retryer) isRetryableError(err error) bool {
	// If no specific retryable errors are defined, retry all errors
	if len(r.config.RetryableErrors) == 0 {
		return true
	}

	for _, retryableErr := range r.config.RetryableErrors {
		if err.Error() == retryableErr.Error() {
			return true
		}
	}

	return false
}

// isNonRetryableError checks if an error should not be retried
func (r *Retryer) isNonRetryableError(err error) bool {
	for _, nonRetryableErr := range r.config.NonRetryableErrors {
		if err.Error() == nonRetryableErr.Error() {
			return true
		}
	}

	return false
}

// Common retry configurations
var (
	// NetworkRetryConfig for network operations
	NetworkRetryConfig = &RetryConfig{
		MaxAttempts:   5,
		InitialDelay:  200 * time.Millisecond,
		MaxDelay:      10 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
	}

	// DatabaseRetryConfig for database operations
	DatabaseRetryConfig = &RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
	}

	// ExternalServiceRetryConfig for external API calls
	ExternalServiceRetryConfig = &RetryConfig{
		MaxAttempts:   4,
		InitialDelay:  500 * time.Millisecond,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.5,
		Jitter:        true,
	}
)

// Convenience functions for common retry scenarios

// RetryNetwork retries a network operation with standard network retry config
func RetryNetwork(ctx context.Context, logger logger.Logger, fn RetryableFunc) error {
	retryer := NewRetryer(NetworkRetryConfig, logger)
	return retryer.Execute(ctx, fn)
}

// RetryDatabase retries a database operation with standard database retry config
func RetryDatabase(ctx context.Context, logger logger.Logger, fn RetryableFunc) error {
	retryer := NewRetryer(DatabaseRetryConfig, logger)
	return retryer.Execute(ctx, fn)
}

// RetryExternalService retries an external service call with standard external service retry config
func RetryExternalService(ctx context.Context, logger logger.Logger, fn RetryableFunc) error {
	retryer := NewRetryer(ExternalServiceRetryConfig, logger)
	return retryer.Execute(ctx, fn)
}

// RetryNetworkWithResult retries a network operation that returns a result
func RetryNetworkWithResult[T any](ctx context.Context, logger logger.Logger, fn func(ctx context.Context) (T, error)) (T, error) {
	retryer := NewRetryer(NetworkRetryConfig, logger)
	return ExecuteWithResult(retryer, ctx, fn)
}

// RetryDatabaseWithResult retries a database operation that returns a result
func RetryDatabaseWithResult[T any](ctx context.Context, logger logger.Logger, fn func(ctx context.Context) (T, error)) (T, error) {
	retryer := NewRetryer(DatabaseRetryConfig, logger)
	return ExecuteWithResult(retryer, ctx, fn)
}

// RetryExternalServiceWithResult retries an external service call that returns a result
func RetryExternalServiceWithResult[T any](ctx context.Context, logger logger.Logger, fn func(ctx context.Context) (T, error)) (T, error) {
	retryer := NewRetryer(ExternalServiceRetryConfig, logger)
	return ExecuteWithResult(retryer, ctx, fn)
}
