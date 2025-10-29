package validator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/estudosdevops/opsmaster/internal/cloud"
)

// ============================================================
// MOCKS - External dependency simulation
// ============================================================

// mockCloudProvider simulates a cloud provider for testing validators.
// Allows configuring specific behaviors for validation scenarios.
type mockCloudProvider struct {
	validateInstanceFunc func(ctx context.Context, instance *cloud.Instance) error
	testConnectivityFunc func(ctx context.Context, instance *cloud.Instance, host string, port int) error
	executeCommandFunc   func(ctx context.Context, instance *cloud.Instance, commands []string, timeout time.Duration) (*cloud.CommandResult, error)
}

// Name implements the cloud.CloudProvider interface
func (_ *mockCloudProvider) Name() string {
	return "mock"
}

// ValidateInstance implements cloud.CloudProvider interface
func (m *mockCloudProvider) ValidateInstance(ctx context.Context, instance *cloud.Instance) error {
	if m.validateInstanceFunc != nil {
		return m.validateInstanceFunc(ctx, instance)
	}
	return nil
}

// TestConnectivity implements cloud.CloudProvider interface
func (m *mockCloudProvider) TestConnectivity(ctx context.Context, instance *cloud.Instance, host string, port int) error {
	if m.testConnectivityFunc != nil {
		return m.testConnectivityFunc(ctx, instance, host, port)
	}
	return nil
}

// ExecuteCommand implements cloud.CloudProvider interface
func (m *mockCloudProvider) ExecuteCommand(ctx context.Context, instance *cloud.Instance, commands []string, timeout time.Duration) (*cloud.CommandResult, error) {
	if m.executeCommandFunc != nil {
		return m.executeCommandFunc(ctx, instance, commands, timeout)
	}
	return &cloud.CommandResult{Stdout: "", ExitCode: 0}, nil
}

// TagInstance implements cloud.CloudProvider interface
func (_ *mockCloudProvider) TagInstance(_ context.Context, _ *cloud.Instance, _ map[string]string) error {
	return nil
}

// HasTag implements cloud.CloudProvider interface
func (_ *mockCloudProvider) HasTag(_ context.Context, _ *cloud.Instance, _, _ string) (bool, error) {
	return false, nil
}

// ============================================================
// HELPER FUNCTIONS - Test utilities
// ============================================================

// createTestInstance creates a standard test instance
func createTestInstance() *cloud.Instance {
	return &cloud.Instance{
		ID:      "i-1234567890abcdef0",
		Account: "123456789012",
		Region:  "us-east-1",
		Cloud:   "aws",
		Metadata: map[string]string{
			"environment": "production",
		},
	}
}

// ============================================================
// CONNECTIVITY VALIDATOR TESTS
// ============================================================

// TestNewConnectivityValidator tests the creation of connectivity validators.
//
// 🎓 CONCEPT: Constructor testing
// - Validate proper initialization with provided values
// - Validate default values when parameters are zero/empty
func TestNewConnectivityValidator(t *testing.T) {
	tests := []struct {
		name            string
		validatorName   string
		host            string
		port            int
		timeout         time.Duration
		expectedTimeout time.Duration
	}{
		{
			name:            "with custom timeout",
			validatorName:   "puppet_server",
			host:            "puppet.example.com",
			port:            8140,
			timeout:         15 * time.Second,
			expectedTimeout: 15 * time.Second,
		},
		{
			name:            "with zero timeout (uses default)",
			validatorName:   "docker_registry",
			host:            "registry.example.com",
			port:            443,
			timeout:         0,
			expectedTimeout: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ACT
			validator := NewConnectivityValidator(tt.validatorName, tt.host, tt.port, tt.timeout)

			// ASSERT
			if validator.Name != tt.validatorName {
				t.Errorf("Name = %q, want %q", validator.Name, tt.validatorName)
			}
			if validator.Host != tt.host {
				t.Errorf("Host = %q, want %q", validator.Host, tt.host)
			}
			if validator.Port != tt.port {
				t.Errorf("Port = %d, want %d", validator.Port, tt.port)
			}
			if validator.Timeout != tt.expectedTimeout {
				t.Errorf("Timeout = %v, want %v", validator.Timeout, tt.expectedTimeout)
			}
		})
	}
}

// TestConnectivityValidator_Validate tests connectivity validation scenarios.
//
// 🎓 CONCEPT: Behavior testing with mocks
// - Test success path (connectivity works)
// - Test failure path (connectivity fails)
// - Validate result structure and messages
func TestConnectivityValidator_Validate(t *testing.T) {
	tests := []struct {
		name            string
		host            string
		port            int
		mockError       error
		expectedSuccess bool
		messageContains string
	}{
		{
			name:            "successful connection",
			host:            "puppet.example.com",
			port:            8140,
			mockError:       nil,
			expectedSuccess: true,
			messageContains: "Successfully connected",
		},
		{
			name:            "connection refused",
			host:            "puppet.example.com",
			port:            8140,
			mockError:       errors.New("connection refused"),
			expectedSuccess: false,
			messageContains: "Cannot reach",
		},
		{
			name:            "timeout error",
			host:            "unreachable.example.com",
			port:            443,
			mockError:       context.DeadlineExceeded,
			expectedSuccess: false,
			messageContains: "Cannot reach",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			ctx := context.Background()
			instance := createTestInstance()

			mockProvider := &mockCloudProvider{
				testConnectivityFunc: func(_ context.Context, _ *cloud.Instance, _ string, _ int) error {
					return tt.mockError
				},
			}

			validator := NewConnectivityValidator("test_connectivity", tt.host, tt.port, 5*time.Second)

			// ACT
			result := validator.Validate(ctx, instance, mockProvider)

			// ASSERT
			if result.Success != tt.expectedSuccess {
				t.Errorf("Success = %v, want %v", result.Success, tt.expectedSuccess)
			}

			if result.Name != "test_connectivity" {
				t.Errorf("Name = %q, want %q", result.Name, "test_connectivity")
			}

			if !contains(result.Message, tt.messageContains) {
				t.Errorf("Message = %q, want to contain %q", result.Message, tt.messageContains)
			}

			if tt.expectedSuccess && result.Error != nil {
				t.Errorf("Expected no error on success, got %v", result.Error)
			}

			if !tt.expectedSuccess && result.Error == nil {
				t.Error("Expected error on failure, got nil")
			}
		})
	}
}

// TestConnectivityValidator_Validate_ContextCancellation tests behavior when context is canceled.
//
// 🎓 CONCEPT: Context cancellation testing
// - Validate proper handling of canceled contexts
// - Ensure resources are cleaned up
func TestConnectivityValidator_Validate_ContextCancellation(t *testing.T) {
	// ARRANGE
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	instance := createTestInstance()
	mockProvider := &mockCloudProvider{
		testConnectivityFunc: func(ctx context.Context, _ *cloud.Instance, _ string, _ int) error {
			return ctx.Err()
		},
	}

	validator := NewConnectivityValidator("test", "puppet.example.com", 8140, 5*time.Second)

	// ACT
	result := validator.Validate(ctx, instance, mockProvider)

	// ASSERT
	if result.Success {
		t.Error("Expected validation to fail with canceled context")
	}

	if result.Error == nil {
		t.Error("Expected error with canceled context")
	}
}

// ============================================================
// SSM VALIDATOR TESTS
// ============================================================

// TestNewSSMValidator tests SSM validator creation.
func TestNewSSMValidator(t *testing.T) {
	tests := []struct {
		name            string
		timeout         time.Duration
		expectedTimeout time.Duration
	}{
		{
			name:            "with custom timeout",
			timeout:         20 * time.Second,
			expectedTimeout: 20 * time.Second,
		},
		{
			name:            "with zero timeout (uses default)",
			timeout:         0,
			expectedTimeout: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ACT
			validator := NewSSMValidator(tt.timeout)

			// ASSERT
			if validator.Name != "ssm_connectivity" {
				t.Errorf("Name = %q, want %q", validator.Name, "ssm_connectivity")
			}
			if validator.Timeout != tt.expectedTimeout {
				t.Errorf("Timeout = %v, want %v", validator.Timeout, tt.expectedTimeout)
			}
		})
	}
}

// TestSSMValidator_Validate tests SSM validation scenarios.
func TestSSMValidator_Validate(t *testing.T) {
	tests := []struct {
		name            string
		mockError       error
		expectedSuccess bool
		messageContains string
	}{
		{
			name:            "instance is accessible",
			mockError:       nil,
			expectedSuccess: true,
			messageContains: "online and accessible",
		},
		{
			name:            "instance not accessible",
			mockError:       errors.New("instance not managed by SSM"),
			expectedSuccess: false,
			messageContains: "not accessible",
		},
		{
			name:            "instance offline",
			mockError:       errors.New("instance status: offline"),
			expectedSuccess: false,
			messageContains: "not accessible",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			ctx := context.Background()
			instance := createTestInstance()

			mockProvider := &mockCloudProvider{
				validateInstanceFunc: func(_ context.Context, _ *cloud.Instance) error {
					return tt.mockError
				},
			}

			validator := NewSSMValidator(5 * time.Second)

			// ACT
			result := validator.Validate(ctx, instance, mockProvider)

			// ASSERT
			if result.Success != tt.expectedSuccess {
				t.Errorf("Success = %v, want %v", result.Success, tt.expectedSuccess)
			}

			if !contains(result.Message, tt.messageContains) {
				t.Errorf("Message = %q, want to contain %q", result.Message, tt.messageContains)
			}

			if tt.expectedSuccess && result.Error != nil {
				t.Errorf("Expected no error on success, got %v", result.Error)
			}

			if !tt.expectedSuccess && result.Error == nil {
				t.Error("Expected error on failure, got nil")
			}
		})
	}
}

// ============================================================
// COMPOSITE VALIDATOR TESTS
// ============================================================

// TestNewCompositeValidator tests composite validator creation.
func TestNewCompositeValidator(t *testing.T) {
	// ARRANGE
	validators := []Validator{
		NewSSMValidator(5 * time.Second),
		NewConnectivityValidator("puppet", "puppet.example.com", 8140, 5*time.Second),
	}

	// ACT
	composite := NewCompositeValidator(validators, true)

	// ASSERT
	if len(composite.Validators) != 2 {
		t.Errorf("len(Validators) = %d, want 2", len(composite.Validators))
	}

	if !composite.StopOnFail {
		t.Error("StopOnFail = false, want true")
	}
}

// TestCompositeValidator_Validate_AllPass tests composite validator when all validations pass.
//
// 🎓 CONCEPT: Composite pattern testing
// - Multiple validators run in sequence
// - All results are collected
// - Success when all pass
func TestCompositeValidator_Validate_AllPass(t *testing.T) {
	// ARRANGE
	ctx := context.Background()
	instance := createTestInstance()

	mockProvider := &mockCloudProvider{
		validateInstanceFunc: func(_ context.Context, _ *cloud.Instance) error {
			return nil // SSM check passes
		},
		testConnectivityFunc: func(_ context.Context, _ *cloud.Instance, _ string, _ int) error {
			return nil // Connectivity check passes
		},
	}

	validators := []Validator{
		NewSSMValidator(5 * time.Second),
		NewConnectivityValidator("puppet", "puppet.example.com", 8140, 5*time.Second),
	}

	composite := NewCompositeValidator(validators, false)

	// ACT
	results := composite.Validate(ctx, instance, mockProvider)

	// ASSERT
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}

	for i, result := range results {
		if !result.Success {
			t.Errorf("results[%d].Success = false, want true. Error: %v", i, result.Error)
		}
	}

	if !AllPassed(results) {
		t.Error("AllPassed(results) = false, want true")
	}
}

// TestCompositeValidator_Validate_OneFails tests composite validator when one validation fails.
func TestCompositeValidator_Validate_OneFails(t *testing.T) {
	// ARRANGE
	ctx := context.Background()
	instance := createTestInstance()

	mockProvider := &mockCloudProvider{
		validateInstanceFunc: func(_ context.Context, _ *cloud.Instance) error {
			return nil // SSM check passes
		},
		testConnectivityFunc: func(_ context.Context, _ *cloud.Instance, _ string, _ int) error {
			return errors.New("connection refused") // Connectivity check fails
		},
	}

	validators := []Validator{
		NewSSMValidator(5 * time.Second),
		NewConnectivityValidator("puppet", "puppet.example.com", 8140, 5*time.Second),
	}

	composite := NewCompositeValidator(validators, false) // Don't stop on failure

	// ACT
	results := composite.Validate(ctx, instance, mockProvider)

	// ASSERT
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}

	// First should pass
	if !results[0].Success {
		t.Errorf("results[0].Success = false, want true")
	}

	// Second should fail
	if results[1].Success {
		t.Errorf("results[1].Success = true, want false")
	}

	if AllPassed(results) {
		t.Error("AllPassed(results) = true, want false")
	}

	failed := GetFailedValidations(results)
	if len(failed) != 1 {
		t.Errorf("len(failed) = %d, want 1", len(failed))
	}
}

// TestCompositeValidator_Validate_StopOnFail tests stop-on-failure behavior.
//
// 🎓 CONCEPT: Short-circuit evaluation
// - When StopOnFail=true, stop after first failure
// - Remaining validators are not executed
func TestCompositeValidator_Validate_StopOnFail(t *testing.T) {
	// ARRANGE
	ctx := context.Background()
	instance := createTestInstance()

	mockProvider := &mockCloudProvider{
		validateInstanceFunc: func(_ context.Context, _ *cloud.Instance) error {
			return errors.New("ssm not available") // First check fails
		},
		testConnectivityFunc: func(_ context.Context, _ *cloud.Instance, _ string, _ int) error {
			t.Error("TestConnectivity should not be called when StopOnFail=true and first validator fails")
			return nil
		},
	}

	validators := []Validator{
		NewSSMValidator(5 * time.Second),
		NewConnectivityValidator("puppet", "puppet.example.com", 8140, 5*time.Second),
	}

	composite := NewCompositeValidator(validators, true) // Stop on first failure

	// ACT
	results := composite.Validate(ctx, instance, mockProvider)

	// ASSERT
	// Should only have 1 result (stopped after first failure)
	if len(results) != 1 {
		t.Errorf("len(results) = %d, want 1 (should stop after first failure)", len(results))
	}

	if results[0].Success {
		t.Error("results[0].Success = true, want false")
	}
}

// TestCompositeValidator_Validate_ContextCancellation tests context cancellation during validation.
func TestCompositeValidator_Validate_ContextCancellation(t *testing.T) {
	// ARRANGE
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately before validation starts

	instance := createTestInstance()

	mockProvider := &mockCloudProvider{
		validateInstanceFunc: func(ctx context.Context, _ *cloud.Instance) error {
			// Check if context is already canceled
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return nil
			}
		},
		testConnectivityFunc: func(_ context.Context, _ *cloud.Instance, _ string, _ int) error {
			return nil
		},
	}

	validators := []Validator{
		NewSSMValidator(5 * time.Second),
		NewConnectivityValidator("puppet", "puppet.example.com", 8140, 5*time.Second),
	}

	composite := NewCompositeValidator(validators, false)

	// ACT
	results := composite.Validate(ctx, instance, mockProvider)

	// ASSERT
	// When context is canceled before validation, should get cancellation result
	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}

	// Should detect context cancellation
	foundCancellation := false
	for _, result := range results {
		if result.Name == "validation_canceled" || !result.Success {
			foundCancellation = true
			break
		}
	}

	if !foundCancellation {
		t.Error("Expected to detect context cancellation in results")
	}
}

// ============================================================
// HELPER FUNCTION TESTS
// ============================================================

// TestAllPassed tests the AllPassed helper function.
func TestAllPassed(t *testing.T) {
	tests := []struct {
		name     string
		results  []*ValidationResult
		expected bool
	}{
		{
			name:     "empty results",
			results:  []*ValidationResult{},
			expected: true,
		},
		{
			name: "all pass",
			results: []*ValidationResult{
				{Name: "test1", Success: true},
				{Name: "test2", Success: true},
			},
			expected: true,
		},
		{
			name: "one fails",
			results: []*ValidationResult{
				{Name: "test1", Success: true},
				{Name: "test2", Success: false},
			},
			expected: false,
		},
		{
			name: "all fail",
			results: []*ValidationResult{
				{Name: "test1", Success: false},
				{Name: "test2", Success: false},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AllPassed(tt.results)
			if result != tt.expected {
				t.Errorf("AllPassed() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestGetFailedValidations tests the GetFailedValidations helper function.
func TestGetFailedValidations(t *testing.T) {
	tests := []struct {
		name          string
		results       []*ValidationResult
		expectedCount int
	}{
		{
			name: "no failures",
			results: []*ValidationResult{
				{Name: "test1", Success: true},
				{Name: "test2", Success: true},
			},
			expectedCount: 0,
		},
		{
			name: "one failure",
			results: []*ValidationResult{
				{Name: "test1", Success: true},
				{Name: "test2", Success: false},
			},
			expectedCount: 1,
		},
		{
			name: "all failures",
			results: []*ValidationResult{
				{Name: "test1", Success: false},
				{Name: "test2", Success: false},
			},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failed := GetFailedValidations(tt.results)
			if len(failed) != tt.expectedCount {
				t.Errorf("len(failed) = %d, want %d", len(failed), tt.expectedCount)
			}
		})
	}
}

// TestFormatValidationResults tests the formatting of validation results.
func TestFormatValidationResults(t *testing.T) {
	tests := []struct {
		name            string
		results         []*ValidationResult
		expectedContain string
	}{
		{
			name:            "empty results",
			results:         []*ValidationResult{},
			expectedContain: "No validations run",
		},
		{
			name: "single success",
			results: []*ValidationResult{
				{Name: "test1", Success: true, Message: "all good"},
			},
			expectedContain: "✓ test1",
		},
		{
			name: "single failure",
			results: []*ValidationResult{
				{Name: "test1", Success: false, Message: "failed"},
			},
			expectedContain: "✗ test1",
		},
		{
			name: "mixed results",
			results: []*ValidationResult{
				{Name: "test1", Success: true, Message: "passed"},
				{Name: "test2", Success: false, Message: "failed"},
			},
			expectedContain: "✓ test1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := FormatValidationResults(tt.results)
			if !contains(output, tt.expectedContain) {
				t.Errorf("FormatValidationResults() = %q, want to contain %q", output, tt.expectedContain)
			}
		})
	}
}

// TestValidatePuppetPrerequisites tests the convenience function for Puppet validation.
//
// 🎓 CONCEPT: Integration testing
// - Test multiple validators working together
// - Validate convenience wrapper functions
func TestValidatePuppetPrerequisites(t *testing.T) {
	tests := []struct {
		name         string
		ssmError     error
		connectError error
		expectError  bool
	}{
		{
			name:         "all validations pass",
			ssmError:     nil,
			connectError: nil,
			expectError:  false,
		},
		{
			name:         "ssm validation fails",
			ssmError:     errors.New("ssm offline"),
			connectError: nil,
			expectError:  true,
		},
		{
			name:         "connectivity validation fails",
			ssmError:     nil,
			connectError: errors.New("cannot reach puppet server"),
			expectError:  true,
		},
		{
			name:         "both validations fail",
			ssmError:     errors.New("ssm offline"),
			connectError: errors.New("cannot reach puppet server"),
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			ctx := context.Background()
			instance := createTestInstance()

			mockProvider := &mockCloudProvider{
				validateInstanceFunc: func(_ context.Context, _ *cloud.Instance) error {
					return tt.ssmError
				},
				testConnectivityFunc: func(_ context.Context, _ *cloud.Instance, _ string, _ int) error {
					return tt.connectError
				},
			}

			// ACT
			results, err := ValidatePuppetPrerequisites(ctx, instance, mockProvider, "puppet.example.com", 8140)

			// ASSERT
			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Should always return results (even on error)
			if len(results) != 2 {
				t.Errorf("len(results) = %d, want 2", len(results))
			}
		})
	}
}

// ============================================================
// UTILITY FUNCTIONS
// ============================================================

// contains checks if a string contains a substring (case-insensitive helper)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || substr == "" ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
