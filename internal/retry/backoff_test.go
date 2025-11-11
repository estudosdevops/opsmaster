package retry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"
)

// TestNew verifies that New() creates a proper Retryer instance
func TestNew(t *testing.T) {
	config := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   time.Second,
		MaxDelay:    10 * time.Second,
		Jitter:      true,
	}

	retryer := New(config)

	if retryer == nil {
		t.Fatal("New() returned nil")
	}

	// Verify internal configuration (we can access it because it's same package)
	eb, ok := retryer.(*exponentialBackoff)
	if !ok {
		t.Fatal("New() did not return *exponentialBackoff")
	}

	if eb.config.MaxAttempts != 3 {
		t.Errorf("Expected MaxAttempts=3, got %d", eb.config.MaxAttempts)
	}

	if eb.config.BaseDelay != time.Second {
		t.Errorf("Expected BaseDelay=1s, got %v", eb.config.BaseDelay)
	}
}

// TestSuccessOnFirstTry verifies no retry when operation succeeds immediately
func TestSuccessOnFirstTry(t *testing.T) {
	config := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    time.Second,
		Jitter:      false,
	}

	retryer := New(config)
	ctx := context.Background()

	callCount := 0
	err := retryer.Do(ctx, func() error {
		callCount++
		return nil // Success on first try
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
}

// TestSuccessOnSecondTry verifies retry stops when operation succeeds
func TestSuccessOnSecondTry(t *testing.T) {
	config := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    time.Second,
		Jitter:      false,
	}

	retryer := New(config)
	ctx := context.Background()

	callCount := 0
	err := retryer.Do(ctx, func() error {
		callCount++
		if callCount == 1 {
			return errors.New("temporary failure")
		}
		return nil // Success on second try
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 calls, got %d", callCount)
	}
}

// TestMaxAttemptsReached verifies retry stops at MaxAttempts
func TestMaxAttemptsReached(t *testing.T) {
	config := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    time.Second,
		Jitter:      false,
	}

	retryer := New(config)
	ctx := context.Background()

	callCount := 0
	expectedError := errors.New("persistent failure")

	err := retryer.Do(ctx, func() error {
		callCount++
		return expectedError
	})

	if err == nil {
		t.Error("Expected error after max attempts, got nil")
	}

	if callCount != 3 {
		t.Errorf("Expected 3 calls (max attempts), got %d", callCount)
	}

	// Error should be the last attempt's error
	if !errors.Is(err, expectedError) {
		t.Errorf("Expected original error, got %v", err)
	}
}

// TestContextCancellation verifies context cancellation stops retry
func TestContextCancellation(t *testing.T) {
	config := RetryConfig{
		MaxAttempts: 10,
		BaseDelay:   50 * time.Millisecond,
		MaxDelay:    time.Second,
		Jitter:      false,
	}

	retryer := New(config)
	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	var err error

	// Cancel context after short delay
	go func() {
		time.Sleep(25 * time.Millisecond)
		cancel()
	}()

	err = retryer.Do(ctx, func() error {
		callCount++
		return errors.New("failure")
	})

	if err == nil {
		t.Error("Expected context cancellation error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}

	// Should have stopped early due to cancellation
	if callCount >= 10 {
		t.Errorf("Expected fewer than 10 calls due to cancellation, got %d", callCount)
	}
}

// TestRetryableErrors verifies retryable errors trigger retry
func TestRetryableErrors(t *testing.T) {
	config := RetryConfig{
		MaxAttempts: 2,
		BaseDelay:   time.Millisecond,
		MaxDelay:    time.Second,
		Jitter:      false,
	}

	retryer := New(config)
	ctx := context.Background()

	retryableErrors := []string{
		"timeout",
		"throttling",
		"rate limit",
		"server error",
		"internal error",
		"service unavailable",
		"connection reset",
		"network unreachable",
	}

	for _, errorMsg := range retryableErrors {
		t.Run(fmt.Sprintf("retryable_%s", strings.ReplaceAll(errorMsg, " ", "_")), func(t *testing.T) {
			callCount := 0
			err := retryer.Do(ctx, func() error {
				callCount++
				return errors.New(errorMsg)
			})

			if err == nil {
				t.Error("Expected error, got nil")
			}

			if callCount != 2 {
				t.Errorf("Expected 2 calls for retryable error '%s', got %d", errorMsg, callCount)
			}
		})
	}
}

// TestNonRetryableErrors verifies non-retryable errors don't trigger retry
func TestNonRetryableErrors(t *testing.T) {
	config := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    time.Second,
		Jitter:      false,
	}

	retryer := New(config)
	ctx := context.Background()

	nonRetryableErrors := []string{
		"permission denied",
		"access denied",
		"unauthorized",
		"forbidden",
		"not found",
		"invalid argument",
		"bad request",
	}

	for _, errorMsg := range nonRetryableErrors {
		t.Run(fmt.Sprintf("non_retryable_%s", strings.ReplaceAll(errorMsg, " ", "_")), func(t *testing.T) {
			callCount := 0
			err := retryer.Do(ctx, func() error {
				callCount++
				return errors.New(errorMsg)
			})

			if err == nil {
				t.Error("Expected error, got nil")
			}

			if callCount != 1 {
				t.Errorf("Expected 1 call for non-retryable error '%s', got %d", errorMsg, callCount)
			}
		})
	}
}

// TestExponentialBackoffProgression verifies delay progression
func TestExponentialBackoffProgression(t *testing.T) {
	config := RetryConfig{
		MaxAttempts: 4,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    time.Second,
		Jitter:      false, // Disable jitter for precise timing
	}

	retryer := New(config)
	ctx := context.Background()

	var delays []time.Duration
	lastTime := time.Now()

	callCount := 0
	err := retryer.Do(ctx, func() error {
		callCount++
		if callCount > 1 {
			// Record delay since last call
			now := time.Now()
			delay := now.Sub(lastTime)
			delays = append(delays, delay)
		}
		lastTime = time.Now()
		return errors.New("retry me")
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}

	if len(delays) != 3 {
		t.Fatalf("Expected 3 delays, got %d: %v", len(delays), delays)
	}

	// Verify exponential progression (with some tolerance for timing)
	expectedDelays := []time.Duration{
		100 * time.Millisecond, // 1st retry: base delay
		200 * time.Millisecond, // 2nd retry: base * 2
		400 * time.Millisecond, // 3rd retry: base * 4
	}

	for i, expected := range expectedDelays {
		actual := delays[i]
		tolerance := 50 * time.Millisecond

		if actual < expected-tolerance || actual > expected+tolerance {
			t.Errorf("Delay %d: expected ~%v, got %v", i+1, expected, actual)
		}
	}
}

// TestBackoffMaxDelay verifies delays don't exceed MaxDelay
func TestBackoffMaxDelay(t *testing.T) {
	config := RetryConfig{
		MaxAttempts: 5,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    250 * time.Millisecond, // Small max delay
		Jitter:      false,
	}

	retryer := New(config)
	ctx := context.Background()

	var delays []time.Duration
	lastTime := time.Now()

	callCount := 0
	err := retryer.Do(ctx, func() error {
		callCount++
		if callCount > 1 {
			now := time.Now()
			delay := now.Sub(lastTime)
			delays = append(delays, delay)
		}
		lastTime = time.Now()
		return errors.New("retry me")
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}

	// All delays should be <= MaxDelay + tolerance
	maxAllowed := 250*time.Millisecond + 50*time.Millisecond
	for i, delay := range delays {
		if delay > maxAllowed {
			t.Errorf("Delay %d exceeded max: got %v, max allowed %v", i+1, delay, maxAllowed)
		}
	}
}

// TestJitterVariation verifies jitter adds randomness
func TestJitterVariation(t *testing.T) {
	config := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    time.Second,
		Jitter:      true,
	}

	// Run multiple times to check for variation
	var delays []time.Duration

	for range 10 {
		retryer := New(config)
		ctx := context.Background()

		lastTime := time.Now()
		callCount := 0

		_ = retryer.Do(ctx, func() error {
			callCount++
			if callCount == 2 {
				// Record first retry delay
				now := time.Now()
				delay := now.Sub(lastTime)
				delays = append(delays, delay)
			}
			lastTime = time.Now()
			return errors.New("retry me")
		})
	}

	// With jitter, we should see variation in delays
	if len(delays) < 5 {
		t.Fatalf("Expected at least 5 delay measurements, got %d", len(delays))
	}

	// Check if delays vary (not all identical)
	firstDelay := delays[0]
	hasVariation := false
	tolerance := 10 * time.Millisecond

	for _, delay := range delays[1:] {
		if math.Abs(float64(delay-firstDelay)) > float64(tolerance) {
			hasVariation = true
			break
		}
	}

	if !hasVariation {
		t.Error("Expected jitter to create delay variation, but all delays were similar")
	}
}

// TestPredefinedPolicies verifies predefined policies have correct values
func TestPredefinedPolicies(t *testing.T) {
	tests := []struct {
		name     string
		policy   RetryConfig
		expected RetryConfig
	}{
		{
			name:   "SSMPolicy",
			policy: SSMPolicy,
			expected: RetryConfig{
				MaxAttempts: 3,
				BaseDelay:   1 * time.Second,
				MaxDelay:    30 * time.Second,
				Jitter:      true,
			},
		},
		{
			name:   "EC2Policy",
			policy: EC2Policy,
			expected: RetryConfig{
				MaxAttempts: 5,
				BaseDelay:   500 * time.Millisecond,
				MaxDelay:    10 * time.Second,
				Jitter:      true,
			},
		},
		{
			name:   "NetworkPolicy",
			policy: NetworkPolicy,
			expected: RetryConfig{
				MaxAttempts: 3,
				BaseDelay:   2 * time.Second,
				MaxDelay:    15 * time.Second,
				Jitter:      true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.policy.MaxAttempts != tt.expected.MaxAttempts {
				t.Errorf("%s.MaxAttempts = %d, want %d", tt.name, tt.policy.MaxAttempts, tt.expected.MaxAttempts)
			}
			if tt.policy.BaseDelay != tt.expected.BaseDelay {
				t.Errorf("%s.BaseDelay = %v, want %v", tt.name, tt.policy.BaseDelay, tt.expected.BaseDelay)
			}
			if tt.policy.MaxDelay != tt.expected.MaxDelay {
				t.Errorf("%s.MaxDelay = %v, want %v", tt.name, tt.policy.MaxDelay, tt.expected.MaxDelay)
			}
			if tt.policy.Jitter != tt.expected.Jitter {
				t.Errorf("%s.Jitter = %v, want %v", tt.name, tt.policy.Jitter, tt.expected.Jitter)
			}
		})
	}
}

// TestCustomConfiguration verifies custom configurations work
func TestCustomConfiguration(t *testing.T) {
	config := RetryConfig{
		MaxAttempts: 5,
		BaseDelay:   200 * time.Millisecond,
		MaxDelay:    2 * time.Second,
		Jitter:      false,
	}

	retryer := New(config)
	ctx := context.Background()

	callCount := 0
	err := retryer.Do(ctx, func() error {
		callCount++
		return errors.New("always fail")
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}

	if callCount != 5 {
		t.Errorf("Expected 5 calls, got %d", callCount)
	}
}

// TestZeroMaxAttempts verifies edge case of zero max attempts
func TestZeroMaxAttempts(t *testing.T) {
	config := RetryConfig{
		MaxAttempts: 0,
		BaseDelay:   time.Millisecond,
		MaxDelay:    time.Second,
		Jitter:      false,
	}

	retryer := New(config)
	ctx := context.Background()

	callCount := 0
	err := retryer.Do(ctx, func() error {
		callCount++
		return errors.New("failure")
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}

	// With MaxAttempts=0, should not call the function at all
	if callCount != 0 {
		t.Errorf("Expected 0 calls with MaxAttempts=0, got %d", callCount)
	}
}

// TestOneMaxAttempt verifies edge case of exactly one attempt
func TestOneMaxAttempt(t *testing.T) {
	config := RetryConfig{
		MaxAttempts: 1,
		BaseDelay:   time.Millisecond,
		MaxDelay:    time.Second,
		Jitter:      false,
	}

	retryer := New(config)
	ctx := context.Background()

	callCount := 0
	err := retryer.Do(ctx, func() error {
		callCount++
		return errors.New("failure")
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}

	// With MaxAttempts=1, should call exactly once (no retries)
	if callCount != 1 {
		t.Errorf("Expected 1 call with MaxAttempts=1, got %d", callCount)
	}
}

// Note: This test is simplified since the actual logging integration
// uses the logger package which gets the global logger.
// In a real implementation, you might need to inject the logger
// or provide a way to set a test logger.
func TestLoggingBehavior(t *testing.T) {
	config := RetryConfig{
		MaxAttempts: 2,
		BaseDelay:   time.Millisecond,
		MaxDelay:    time.Second,
		Jitter:      false,
	}

	retryer := New(config)
	ctx := context.Background()

	// This test verifies the function calls happen in the right order
	// Actual log verification would require dependency injection of logger
	callCount := 0
	err := retryer.Do(ctx, func() error {
		callCount++
		if callCount == 1 {
			return errors.New("temporary failure")
		}
		return nil // Success on second try
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 calls, got %d", callCount)
	}

	// Note: In a production environment, you would verify that the logger
	// received calls with the correct structured data:
	// - retry_attempt_start with attempt number
	// - retry_attempt_failed with error and delay (for failures)
	// - retry_attempt_success (for success)
	// - retry_completed with total duration and attempts
}

// BenchmarkRetryOverhead measures the overhead of retry logic
func BenchmarkRetryOverhead(b *testing.B) {
	config := RetryConfig{
		MaxAttempts: 1, // No retries, just measure overhead
		BaseDelay:   time.Millisecond,
		MaxDelay:    time.Second,
		Jitter:      false,
	}

	retryer := New(config)
	ctx := context.Background()

	b.ResetTimer()

	for range b.N {
		_ = retryer.Do(ctx, func() error {
			return nil // Always succeed
		})
	}
}

// BenchmarkRetryWithFailures measures performance with actual retries
func BenchmarkRetryWithFailures(b *testing.B) {
	config := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   time.Microsecond, // Very small delay for benchmark
		MaxDelay:    time.Millisecond,
		Jitter:      false,
	}

	retryer := New(config)
	ctx := context.Background()

	b.ResetTimer()

	for range b.N {
		attempt := 0
		_ = retryer.Do(ctx, func() error {
			attempt++
			if attempt < 3 {
				return errors.New("retry me")
			}
			return nil // Success on 3rd try
		})
	}
}
