package cloud

import (
	"context"
	"testing"
	"time"
)

// ============================================================
// CONCEPT: Testing Struct Validation
// 🎓 These tests validate that Instance struct has all required fields
// and can be properly initialized. This ensures data integrity.
// ============================================================

// TestInstance_String tests the string representation of an instance
func TestInstance_String(t *testing.T) {
	tests := []struct {
		name     string
		instance *Instance
		expected string
	}{
		{
			name: "AWS instance with all fields",
			instance: &Instance{
				ID:      "i-1234567890abcdef0",
				Cloud:   "aws",
				Account: "111111111111",
				Region:  "us-east-1",
			},
			expected: "aws:111111111111:us-east-1:i-1234567890abcdef0",
		},
		{
			name: "GCP instance",
			instance: &Instance{
				ID:      "instance-12345",
				Cloud:   "gcp",
				Account: "my-project-id",
				Region:  "us-central1",
			},
			expected: "gcp:my-project-id:us-central1:instance-12345",
		},
		{
			name: "Azure instance",
			instance: &Instance{
				ID:      "/subscriptions/abc123/resourceGroups/rg1/providers/Microsoft.Compute/virtualMachines/vm1",
				Cloud:   "azure",
				Account: "abc123",
				Region:  "eastus",
			},
			expected: "azure:abc123:eastus:/subscriptions/abc123/resourceGroups/rg1/providers/Microsoft.Compute/virtualMachines/vm1",
		},
		{
			name: "instance with empty fields",
			instance: &Instance{
				ID:      "",
				Cloud:   "",
				Account: "",
				Region:  "",
			},
			expected: ":::",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.instance.String()
			if result != tt.expected {
				t.Errorf("Instance.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestInstance_Metadata tests the metadata field initialization and usage
func TestInstance_Metadata(t *testing.T) {
	t.Run("metadata can be initialized and accessed", func(t *testing.T) {
		instance := &Instance{
			ID:       "i-test",
			Cloud:    "aws",
			Account:  "123456789012",
			Region:   "us-east-1",
			Metadata: make(map[string]string),
		}

		// Add metadata
		instance.Metadata["environment"] = "production"
		instance.Metadata["app_name"] = "web-api"
		instance.Metadata["team"] = "platform"

		// Verify metadata
		if instance.Metadata["environment"] != "production" {
			t.Errorf("Metadata[environment] = %q, want %q", instance.Metadata["environment"], "production")
		}
		if instance.Metadata["app_name"] != "web-api" {
			t.Errorf("Metadata[app_name] = %q, want %q", instance.Metadata["app_name"], "web-api")
		}
		if instance.Metadata["team"] != "platform" {
			t.Errorf("Metadata[team] = %q, want %q", instance.Metadata["team"], "platform")
		}
		if len(instance.Metadata) != 3 {
			t.Errorf("Metadata should have 3 entries, but has %d", len(instance.Metadata))
		}
	})

	t.Run("metadata can be nil", func(t *testing.T) {
		instance := &Instance{
			ID:       "i-test",
			Cloud:    "aws",
			Account:  "123456789012",
			Region:   "us-east-1",
			Metadata: nil,
		}

		// Should not panic when accessing nil metadata
		if instance.Metadata != nil {
			t.Error("Metadata should be nil but is not")
		}
	})

	t.Run("metadata with pre-populated values", func(t *testing.T) {
		instance := &Instance{
			ID:      "i-test",
			Cloud:   "aws",
			Account: "123456789012",
			Region:  "us-east-1",
			Metadata: map[string]string{
				"Name":        "web-server-01",
				"Environment": "staging",
			},
		}

		if instance.Metadata["Name"] != "web-server-01" {
			t.Errorf("Metadata[Name] = %q, want %q", instance.Metadata["Name"], "web-server-01")
		}
		if instance.Metadata["Environment"] != "staging" {
			t.Errorf("Metadata[Environment] = %q, want %q", instance.Metadata["Environment"], "staging")
		}
	})
}

// ============================================================
// CONCEPT: Testing Command Result
// 🎓 CommandResult encapsulates execution results from remote commands.
// We test both success and failure scenarios to ensure correct behavior.
// ============================================================

// TestCommandResult_Success tests the Success method
func TestCommandResult_Success(t *testing.T) {
	tests := []struct {
		name     string
		result   *CommandResult
		expected bool
	}{
		{
			name: "successful command (exit code 0, no error)",
			result: &CommandResult{
				InstanceID: "i-test",
				ExitCode:   0,
				Stdout:     "command output",
				Stderr:     "",
				Duration:   100 * time.Millisecond,
				Error:      nil,
			},
			expected: true,
		},
		{
			name: "failed command (non-zero exit code)",
			result: &CommandResult{
				InstanceID: "i-test",
				ExitCode:   1,
				Stdout:     "",
				Stderr:     "command failed",
				Duration:   50 * time.Millisecond,
				Error:      nil,
			},
			expected: false,
		},
		{
			name: "command with error (even if exit code is 0)",
			result: &CommandResult{
				InstanceID: "i-test",
				ExitCode:   0,
				Stdout:     "",
				Stderr:     "",
				Duration:   10 * time.Millisecond,
				Error:      context.DeadlineExceeded,
			},
			expected: false,
		},
		{
			name: "command with non-zero exit and error",
			result: &CommandResult{
				InstanceID: "i-test",
				ExitCode:   127,
				Stdout:     "",
				Stderr:     "command not found",
				Duration:   5 * time.Millisecond,
				Error:      context.Canceled,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.result.Success()
			if result != tt.expected {
				t.Errorf("CommandResult.Success() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestCommandResult_Failed tests the Failed method
func TestCommandResult_Failed(t *testing.T) {
	tests := []struct {
		name     string
		result   *CommandResult
		expected bool
	}{
		{
			name: "successful command should not be failed",
			result: &CommandResult{
				InstanceID: "i-test",
				ExitCode:   0,
				Stdout:     "output",
				Error:      nil,
			},
			expected: false,
		},
		{
			name: "command with error should be failed",
			result: &CommandResult{
				InstanceID: "i-test",
				ExitCode:   1,
				Stderr:     "error",
				Error:      nil,
			},
			expected: true,
		},
		{
			name: "command with go error should be failed",
			result: &CommandResult{
				InstanceID: "i-test",
				ExitCode:   0,
				Error:      context.Canceled,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.result.Failed()
			if result != tt.expected {
				t.Errorf("CommandResult.Failed() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestCommandResult_SuccessAndFailedAreOpposites ensures Success and Failed are logical opposites
func TestCommandResult_SuccessAndFailedAreOpposites(t *testing.T) {
	testCases := []*CommandResult{
		{InstanceID: "i-1", ExitCode: 0, Error: nil},
		{InstanceID: "i-2", ExitCode: 1, Error: nil},
		{InstanceID: "i-3", ExitCode: 0, Error: context.Canceled},
		{InstanceID: "i-4", ExitCode: 127, Error: context.DeadlineExceeded},
	}

	for _, result := range testCases {
		t.Run(result.InstanceID, func(t *testing.T) {
			success := result.Success()
			failed := result.Failed()

			// Success and Failed should always be opposites
			if success == failed {
				t.Errorf("Success() = %v and Failed() = %v should be opposites", success, failed)
			}

			// Exactly one should be true
			if success && failed {
				t.Error("Both Success() and Failed() returned true - should be mutually exclusive")
			}
			if !success && !failed {
				t.Error("Both Success() and Failed() returned false - one must be true")
			}
		})
	}
}

// TestCommandResult_DurationTracking tests that duration is properly set
func TestCommandResult_DurationTracking(t *testing.T) {
	t.Run("duration is preserved", func(t *testing.T) {
		expectedDuration := 250 * time.Millisecond
		result := &CommandResult{
			InstanceID: "i-test",
			ExitCode:   0,
			Duration:   expectedDuration,
		}

		if result.Duration != expectedDuration {
			t.Errorf("Duration = %v, want %v", result.Duration, expectedDuration)
		}
	})

	t.Run("zero duration is valid", func(t *testing.T) {
		result := &CommandResult{
			InstanceID: "i-test",
			ExitCode:   0,
			Duration:   0,
		}

		if result.Duration != 0 {
			t.Errorf("Duration = %v, want 0", result.Duration)
		}
	})
}

// ============================================================
// CONCEPT: Interface Compliance Testing
// 🎓 This ensures that any type implementing CloudProvider
// actually provides all required methods. This is a compile-time check.
// ============================================================

// mockCloudProvider is a minimal implementation for testing
type mockCloudProvider struct{}

func (*mockCloudProvider) Name() string {
	return "mock"
}

func (*mockCloudProvider) ValidateInstance(_ context.Context, _ *Instance) error {
	return nil
}

func (*mockCloudProvider) ExecuteCommand(_ context.Context, _ *Instance, _ []string, _ time.Duration) (*CommandResult, error) {
	return &CommandResult{ExitCode: 0}, nil
}

func (*mockCloudProvider) TestConnectivity(_ context.Context, _ *Instance, _ string, _ int) error {
	return nil
}

func (*mockCloudProvider) TagInstance(_ context.Context, _ *Instance, _ map[string]string) error {
	return nil
}

func (*mockCloudProvider) HasTag(_ context.Context, _ *Instance, _, _ string) (bool, error) {
	return false, nil
}

// TestCloudProvider_InterfaceCompliance validates that mockCloudProvider implements CloudProvider
// This is a compile-time test - if mockCloudProvider doesn't implement all methods,
// this won't compile
func TestCloudProvider_InterfaceCompliance(t *testing.T) {
	var _ CloudProvider = (*mockCloudProvider)(nil)
	t.Log("mockCloudProvider correctly implements CloudProvider interface")
}

// TestCloudProvider_MockImplementation tests that mock provider works as expected
func TestCloudProvider_MockImplementation(t *testing.T) {
	provider := &mockCloudProvider{}
	ctx := context.Background()
	instance := &Instance{
		ID:      "i-test",
		Cloud:   "mock",
		Account: "123456",
		Region:  "test-region",
	}

	t.Run("Name returns mock", func(t *testing.T) {
		name := provider.Name()
		if name != "mock" {
			t.Errorf("Name() = %q, want %q", name, "mock")
		}
	})

	t.Run("ValidateInstance succeeds", func(t *testing.T) {
		err := provider.ValidateInstance(ctx, instance)
		if err != nil {
			t.Errorf("ValidateInstance() returned error: %v", err)
		}
	})

	t.Run("ExecuteCommand returns success", func(t *testing.T) {
		result, err := provider.ExecuteCommand(ctx, instance, []string{"echo test"}, 5*time.Second)
		if err != nil {
			t.Errorf("ExecuteCommand() returned error: %v", err)
		}
		if result == nil {
			t.Fatal("ExecuteCommand() returned nil result")
		}
		if result.ExitCode != 0 {
			t.Errorf("ExitCode = %d, want 0", result.ExitCode)
		}
	})

	t.Run("TestConnectivity succeeds", func(t *testing.T) {
		err := provider.TestConnectivity(ctx, instance, "example.com", 443)
		if err != nil {
			t.Errorf("TestConnectivity() returned error: %v", err)
		}
	})

	t.Run("TagInstance succeeds", func(t *testing.T) {
		tags := map[string]string{"test": "value"}
		err := provider.TagInstance(ctx, instance, tags)
		if err != nil {
			t.Errorf("TagInstance() returned error: %v", err)
		}
	})

	t.Run("HasTag returns false by default", func(t *testing.T) {
		hasTag, err := provider.HasTag(ctx, instance, "test", "value")
		if err != nil {
			t.Errorf("HasTag() returned error: %v", err)
		}
		if hasTag {
			t.Error("HasTag() = true, want false")
		}
	})
}

// TestInstance_AllFieldsPopulated validates that Instance struct can hold all expected data
func TestInstance_AllFieldsPopulated(t *testing.T) {
	instance := &Instance{
		ID:      "i-0123456789abcdef0",
		Cloud:   "aws",
		Account: "111111111111",
		Region:  "us-east-1",
		Metadata: map[string]string{
			"Name":        "web-server",
			"Environment": "production",
			"Team":        "platform",
			"Application": "api-gateway",
		},
	}

	// Validate required fields
	if instance.ID == "" {
		t.Error("ID should not be empty")
	}
	if instance.Cloud == "" {
		t.Error("Cloud should not be empty")
	}
	if instance.Account == "" {
		t.Error("Account should not be empty")
	}
	if instance.Region == "" {
		t.Error("Region should not be empty")
	}

	// Validate metadata
	if len(instance.Metadata) != 4 {
		t.Errorf("Metadata should have 4 entries, but has %d", len(instance.Metadata))
	}

	// Validate specific metadata values
	expectedMetadata := map[string]string{
		"Name":        "web-server",
		"Environment": "production",
		"Team":        "platform",
		"Application": "api-gateway",
	}

	for key, expectedValue := range expectedMetadata {
		actualValue, exists := instance.Metadata[key]
		if !exists {
			t.Errorf("Metadata missing key: %s", key)
		}
		if actualValue != expectedValue {
			t.Errorf("Metadata[%s] = %q, want %q", key, actualValue, expectedValue)
		}
	}
}
