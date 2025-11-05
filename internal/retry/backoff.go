package retry

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/estudosdevops/opsmaster/internal/logger"
)

// RetryConfig defines how retry behavior should work.
type RetryConfig struct {
	MaxAttempts int           // Maximum number of attempts (e.g., 3)
	BaseDelay   time.Duration // Initial delay (e.g., 1s)
	MaxDelay    time.Duration // Maximum delay (e.g., 30s)
	Jitter      bool          // Add randomness? (e.g., true)
}

// Retryer interface defines the contract for retry implementations.
// NOTE: Interfaces cannot have generic methods in Go, so we only include non-generic methods.
type Retryer interface {
	// Do executes a function with retry (no return value)
	Do(ctx context.Context, fn func() error) error
}

// exponentialBackoff is our concrete implementation of the Retryer interface.
type exponentialBackoff struct {
	config RetryConfig
	log    *slog.Logger
}

// New creates a new Retryer instance with the given configuration.
func New(config RetryConfig) Retryer {
	return &exponentialBackoff{
		config: config,
		log:    logger.Get(),
	}
}

// NewWithConfig creates a new exponentialBackoff instance directly.
// Use this when you need access to generic methods.
func NewWithConfig(config RetryConfig) *exponentialBackoff {
	return &exponentialBackoff{
		config: config,
		log:    logger.Get(),
	}
}

// Do executes a function with retry (no return value).
func (e *exponentialBackoff) Do(ctx context.Context, fn func() error) error {
	startTime := time.Now()

	// Log operation start
	e.log.Debug("Starting retry operation",
		"max_attempts", e.config.MaxAttempts,
		"base_delay", e.config.BaseDelay,
		"max_delay", e.config.MaxDelay,
		"jitter", e.config.Jitter)

	// Retry loop - implemented directly without generics
	for attempt := 1; attempt <= e.config.MaxAttempts; attempt++ {
		// CONTEXT: Check if canceled/timeout
		select {
		case <-ctx.Done():
			e.log.Warn("Retry operation canceled",
				"attempt", attempt,
				"max_attempts", e.config.MaxAttempts,
				"duration_seconds", time.Since(startTime).Seconds(),
				"reason", ctx.Err().Error())
			return ctx.Err() // Canceled by user or timeout
		default:
			// Continue execution
		}

		// Log attempt start
		attemptStart := time.Now()
		if attempt > 1 {
			e.log.Debug("Starting retry attempt",
				"attempt", attempt,
				"max_attempts", e.config.MaxAttempts)
		}

		// Execute the function
		err := fn()
		attemptDuration := time.Since(attemptStart)

		if err == nil {
			// Success! Log and return
			if attempt == 1 {
				e.log.Debug("Operation succeeded on first attempt",
					"duration_seconds", attemptDuration.Seconds())
			} else {
				e.log.Info("Operation succeeded after retry",
					"attempt", attempt,
					"max_attempts", e.config.MaxAttempts,
					"total_duration_seconds", time.Since(startTime).Seconds(),
					"attempt_duration_seconds", attemptDuration.Seconds())
			}
			return nil // ✅ Success!
		}

		// Check if we should retry
		if !isRetryableError(err) {
			e.log.Error("Operation failed with non-retryable error",
				"attempt", attempt,
				"max_attempts", e.config.MaxAttempts,
				"duration_seconds", time.Since(startTime).Seconds(),
				"error", err.Error())
			return fmt.Errorf("non-retryable error: %w", err)
		}

		// If this was the last attempt, return the error
		if attempt == e.config.MaxAttempts {
			e.log.Error("Operation failed after max attempts",
				"attempt", attempt,
				"max_attempts", e.config.MaxAttempts,
				"total_duration_seconds", time.Since(startTime).Seconds(),
				"final_error", err.Error())
			return fmt.Errorf("max attempts reached (%d): %w", e.config.MaxAttempts, err)
		}

		// Calculate delay for next attempt
		delay := e.calculateDelay(attempt)

		// Log retry warning
		e.log.Warn("Operation attempt failed, retrying",
			"attempt", attempt,
			"max_attempts", e.config.MaxAttempts,
			"delay_seconds", delay.Seconds(),
			"error", err.Error(),
			"retryable", isRetryableError(err))

		// Wait for the delay (with possibility of cancellation)
		select {
		case <-ctx.Done():
			e.log.Warn("Retry operation canceled during delay",
				"attempt", attempt,
				"max_attempts", e.config.MaxAttempts,
				"duration_seconds", time.Since(startTime).Seconds(),
				"reason", ctx.Err().Error())
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// Should never reach here, but Go requires return in all paths
	e.log.Error("Unexpected retry loop exit",
		"max_attempts", e.config.MaxAttempts,
		"duration_seconds", time.Since(startTime).Seconds())
	return fmt.Errorf("unexpected retry loop exit")
}

// calculateDelay calculates exponential delay with optional jitter.
func (e *exponentialBackoff) calculateDelay(attempt int) time.Duration {
	// Exponential backoff formula: baseDelay * 2^(attempt-1)
	// attempt=1: 1s * 2^0 = 1s
	// attempt=2: 1s * 2^1 = 2s
	// attempt=3: 1s * 2^2 = 4s
	exponentialDelay := float64(e.config.BaseDelay) * math.Pow(2, float64(attempt-1))

	// Apply maximum limit
	if exponentialDelay > float64(e.config.MaxDelay) {
		exponentialDelay = float64(e.config.MaxDelay)
	}

	delay := time.Duration(exponentialDelay)

	// JITTER: Add randomness to avoid "thundering herd"
	if e.config.Jitter {
		// Add up to 25% random variation
		jitterRange := float64(delay) * 0.25
		// #nosec G404 - Using math/rand for jitter is acceptable (not cryptographic)
		jitter := time.Duration(rand.Float64() * jitterRange)
		delay += jitter
	}

	return delay
}

// isRetryableError determines which errors should trigger retry and which are permanent.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Simple error analysis based on error message
	errStr := strings.ToLower(err.Error())

	// Errors that should trigger retry (temporary problems)
	retryableErrors := []string{
		"timeout",
		"connection refused",
		"network is unreachable",
		"temporary failure",
		"rate limit",
		"throttling",
		"service unavailable",
		"internal server error",
		"bad gateway",
		"gateway timeout",
	}

	for _, retryable := range retryableErrors {
		if strings.Contains(errStr, retryable) {
			return true
		}
	}

	// Errors that should NOT trigger retry (permanent problems)
	nonRetryableErrors := []string{
		"permission denied",
		"access denied",
		"unauthorized",
		"forbidden",
		"not found",
		"invalid argument",
		"bad request",
		"invalid credentials",
	}

	for _, nonRetryable := range nonRetryableErrors {
		if strings.Contains(errStr, nonRetryable) {
			return false
		}
	}

	// By default, attempt retry (safety principle)
	return true
}

// Package-level variables - Predefined policies.
var (
	// SSMPolicy is the retry policy for SSM operations (remote commands)
	SSMPolicy = RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Second,
		MaxDelay:    30 * time.Second,
		Jitter:      true,
	}

	// EC2Policy is the retry policy for EC2 APIs (metadata, tags)
	EC2Policy = RetryConfig{
		MaxAttempts: 5,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    10 * time.Second,
		Jitter:      true,
	}

	// NetworkPolicy is the retry policy for network operations
	NetworkPolicy = RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   2 * time.Second,
		MaxDelay:    15 * time.Second,
		Jitter:      true,
	}
)
