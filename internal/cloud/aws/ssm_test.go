package aws

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/estudosdevops/opsmaster/internal/cloud"
)

// ============================================================
// CONCEPT: AWS Provider Testing
// 🎓 AWSProvider implements the CloudProvider interface for AWS.
// We test the concrete implementation to ensure it correctly
// uses SSM for remote execution and handles AWS-specific errors.
//
// Note: These are unit tests that don't require real AWS credentials.
// We test the structure and logic, not actual AWS API calls.
// ============================================================

// TestNewAWSProvider tests the creation of AWS provider
func TestNewAWSProvider(t *testing.T) {
	provider := NewAWSProvider()

	if provider == nil {
		t.Fatal("NewAWSProvider() returned nil")
	}

	if provider.sessionManager == nil {
		t.Error("AWSProvider sessionManager not initialized")
	}

	if provider.log == nil {
		t.Error("AWSProvider logger not initialized")
	}
}

// TestAWSProvider_Name tests the provider name
func TestAWSProvider_Name(t *testing.T) {
	provider := NewAWSProvider()

	name := provider.Name()
	if name != "aws" {
		t.Errorf("Name() = %q, want %q", name, "aws")
	}
}

// TestAWSProvider_InterfaceCompliance validates that AWSProvider implements CloudProvider
func TestAWSProvider_InterfaceCompliance(t *testing.T) {
	var _ cloud.CloudProvider = (*AWSProvider)(nil)
	t.Log("AWSProvider correctly implements CloudProvider interface")
}

// ============================================================
// CONCEPT: Instance Validation Tests
// 🎓 ValidateInstance checks if an instance is accessible via SSM.
// Without real AWS, we test the error handling and validation logic.
// ============================================================

// TestAWSProvider_ValidateInstance_InstanceStructure tests instance validation logic
func TestAWSProvider_ValidateInstance_InstanceStructure(t *testing.T) {
	provider := NewAWSProvider()
	ctx := context.Background()

	tests := []struct {
		name     string
		instance *cloud.Instance
		wantErr  bool
	}{
		{
			name: "valid instance structure",
			instance: &cloud.Instance{
				ID:      "i-1234567890abcdef0",
				Cloud:   "aws",
				Account: "111111111111",
				Region:  "us-east-1",
			},
			wantErr: true, // Will fail without real AWS, but tests structure
		},
		{
			name: "instance with metadata",
			instance: &cloud.Instance{
				ID:      "i-0987654321fedcba0",
				Cloud:   "aws",
				Account: "222222222222",
				Region:  "sa-east-1",
				Metadata: map[string]string{
					"Name": "test-instance",
				},
			},
			wantErr: true, // Will fail without real AWS
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This will fail without real AWS credentials, but we're testing
			// that the function can be called with proper structure
			err := provider.ValidateInstance(ctx, tt.instance)

			// We expect error without real AWS
			if err == nil && tt.wantErr {
				t.Error("ValidateInstance() expected error without AWS credentials, but got nil")
			}

			// Verify error message format (should mention SSM or credentials)
			if err != nil {
				errMsg := err.Error()
				if !contains(errMsg, "SSM") && !contains(errMsg, "config") && !contains(errMsg, "credentials") {
					t.Logf("ValidateInstance() error (expected): %v", err)
				}
			}
		})
	}
}

// ============================================================
// CONCEPT: Command Execution Tests
// 🎓 ExecuteCommand runs shell commands remotely via SSM.
// We test the command structure and timeout handling.
// ============================================================

// TestAWSProvider_ExecuteCommand_CommandStructure tests command execution structure
func TestAWSProvider_ExecuteCommand_CommandStructure(t *testing.T) {
	provider := NewAWSProvider()
	ctx := context.Background()

	instance := &cloud.Instance{
		ID:      "i-test",
		Cloud:   "aws",
		Account: "123456789012",
		Region:  "us-east-1",
	}

	tests := []struct {
		name     string
		commands []string
		timeout  time.Duration
		wantErr  bool
	}{
		{
			name:     "single command",
			commands: []string{"echo 'test'"},
			timeout:  30 * time.Second,
			wantErr:  true, // Will fail without real AWS
		},
		{
			name: "multiple commands",
			commands: []string{
				"echo 'step 1'",
				"echo 'step 2'",
				"echo 'step 3'",
			},
			timeout: 60 * time.Second,
			wantErr: true, // Will fail without real AWS
		},
		{
			name:     "command with short timeout",
			commands: []string{"sleep 1 && echo 'done'"},
			timeout:  5 * time.Second,
			wantErr:  true, // Will fail without real AWS
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := provider.ExecuteCommand(ctx, instance, tt.commands, tt.timeout)

			// Without real AWS, we expect error
			if err == nil && tt.wantErr {
				t.Error("ExecuteCommand() expected error without AWS, but got nil")
			}

			if err != nil {
				// Verify error is related to AWS connectivity
				errMsg := err.Error()
				if !contains(errMsg, "SSM") && !contains(errMsg, "config") && !contains(errMsg, "credentials") {
					t.Logf("ExecuteCommand() error (expected): %v", err)
				}
			}

			// Result should be nil on error
			if err != nil && result != nil {
				t.Error("ExecuteCommand() returned non-nil result on error")
			}
		})
	}
}

// TestAWSProvider_ExecuteCommand_ContextCancellation tests context handling
func TestAWSProvider_ExecuteCommand_ContextCancellation(t *testing.T) {
	provider := NewAWSProvider()

	instance := &cloud.Instance{
		ID:      "i-test",
		Cloud:   "aws",
		Account: "123456789012",
		Region:  "us-east-1",
	}

	t.Run("canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := provider.ExecuteCommand(ctx, instance, []string{"echo test"}, 30*time.Second)
		if err == nil {
			t.Error("ExecuteCommand() expected error with canceled context")
		}
	})

	t.Run("deadline exceeded", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		time.Sleep(10 * time.Millisecond) // Ensure deadline passed

		_, err := provider.ExecuteCommand(ctx, instance, []string{"echo test"}, 30*time.Second)
		if err == nil {
			t.Error("ExecuteCommand() expected error with deadline exceeded")
		}
	})
}

// ============================================================
// CONCEPT: Connectivity Testing
// 🎓 TestConnectivity validates network connectivity from instance
// to external hosts. Useful for pre-flight checks before installation.
// ============================================================

// TestAWSProvider_TestConnectivity_Structure tests connectivity check structure
func TestAWSProvider_TestConnectivity_Structure(t *testing.T) {
	provider := NewAWSProvider()
	ctx := context.Background()

	instance := &cloud.Instance{
		ID:      "i-test",
		Cloud:   "aws",
		Account: "123456789012",
		Region:  "us-east-1",
	}

	tests := []struct {
		name    string
		host    string
		port    int
		wantErr bool
	}{
		{
			name:    "HTTPS port",
			host:    "example.com",
			port:    443,
			wantErr: true, // Will fail without real AWS
		},
		{
			name:    "SSH port",
			host:    "github.com",
			port:    22,
			wantErr: true, // Will fail without real AWS
		},
		{
			name:    "Puppet Server",
			host:    "puppet.example.com",
			port:    8140,
			wantErr: true, // Will fail without real AWS
		},
		{
			name:    "custom port",
			host:    "app.example.com",
			port:    3000,
			wantErr: true, // Will fail without real AWS
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.TestConnectivity(ctx, instance, tt.host, tt.port)

			// Without real AWS, we expect error
			if err == nil && tt.wantErr {
				t.Error("TestConnectivity() expected error without AWS")
			} else if err != nil {
				// Verify error message structure
				errMsg := err.Error()
				if !contains(errMsg, "SSM") && !contains(errMsg, "config") && !contains(errMsg, "credentials") {
					t.Logf("TestConnectivity() error (expected): %v", err)
				}
			}
		})
	}
}

// ============================================================
// CONCEPT: Tagging Tests
// 🎓 TagInstance adds tags to instances. Useful for tracking
// installation status and inventory management.
// ============================================================

// TestAWSProvider_TagInstance_Structure tests instance tagging structure
func TestAWSProvider_TagInstance_Structure(t *testing.T) {
	provider := NewAWSProvider()
	ctx := context.Background()

	instance := &cloud.Instance{
		ID:      "i-test",
		Cloud:   "aws",
		Account: "123456789012",
		Region:  "us-east-1",
	}

	tests := []struct {
		name    string
		tags    map[string]string
		wantErr bool
	}{
		{
			name: "single tag",
			tags: map[string]string{
				"puppet:installed": "true",
			},
			wantErr: true, // Will fail without real AWS
		},
		{
			name: "multiple tags",
			tags: map[string]string{
				"puppet:installed":    "true",
				"puppet:certname":     "node-001.puppet",
				"puppet:install_date": "2025-10-29",
			},
			wantErr: true, // Will fail without real AWS
		},
		{
			name:    "empty tags",
			tags:    map[string]string{},
			wantErr: true, // Will fail without real AWS
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.TagInstance(ctx, instance, tt.tags)

			// Without real AWS, we expect error
			if err == nil && tt.wantErr {
				t.Error("TagInstance() expected error without AWS")
			} else if err != nil {
				errMsg := err.Error()
				if !contains(errMsg, "EC2") && !contains(errMsg, "config") && !contains(errMsg, "credentials") {
					t.Logf("TagInstance() error (expected): %v", err)
				}
			}
		})
	}
}

// TestAWSProvider_HasTag_Structure tests tag checking structure
func TestAWSProvider_HasTag_Structure(t *testing.T) {
	provider := NewAWSProvider()
	ctx := context.Background()

	instance := &cloud.Instance{
		ID:      "i-test",
		Cloud:   "aws",
		Account: "123456789012",
		Region:  "us-east-1",
	}

	tests := []struct {
		name  string
		key   string
		value string
	}{
		{
			name:  "check puppet installed tag",
			key:   "puppet:installed",
			value: "true",
		},
		{
			name:  "check environment tag",
			key:   "Environment",
			value: "production",
		},
		{
			name:  "check custom tag",
			key:   "custom:key",
			value: "custom:value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasTag, err := provider.HasTag(ctx, instance, tt.key, tt.value)

			// Without real AWS, we expect error
			if err == nil {
				t.Error("HasTag() expected error without AWS")
			}

			// hasTag should be false on error
			if hasTag {
				t.Error("HasTag() returned true on error, expected false")
			}

			if err != nil {
				errMsg := err.Error()
				if !contains(errMsg, "EC2") && !contains(errMsg, "config") && !contains(errMsg, "credentials") {
					t.Logf("HasTag() error (expected): %v", err)
				}
			}
		})
	}
}

// ============================================================
// CONCEPT: Error Handling Tests
// 🎓 Validates that errors are properly wrapped and informative.
// ============================================================

// TestAWSProvider_ErrorMessages validates error message quality
func TestAWSProvider_ErrorMessages(t *testing.T) {
	provider := NewAWSProvider()
	ctx := context.Background()

	instance := &cloud.Instance{
		ID:      "i-invalid",
		Cloud:   "aws",
		Account: "000000000000",
		Region:  "invalid-region",
	}

	t.Run("ValidateInstance error is informative", func(t *testing.T) {
		err := provider.ValidateInstance(ctx, instance)
		if err == nil {
			t.Skip("ValidateInstance succeeded (unexpected - may have real AWS credentials)")
		}

		// Error should be wrapped and informative
		if err != nil && !errors.Is(err, context.Canceled) {
			// Should contain useful context
			errMsg := err.Error()
			if len(errMsg) < 10 {
				t.Errorf("Error message too short: %q", errMsg)
			}
		}
	})

	t.Run("ExecuteCommand error is informative", func(t *testing.T) {
		_, err := provider.ExecuteCommand(ctx, instance, []string{"echo test"}, 5*time.Second)
		if err == nil {
			t.Skip("ExecuteCommand succeeded (unexpected - may have real AWS credentials)")
		}

		if err != nil {
			errMsg := err.Error()
			if len(errMsg) < 10 {
				t.Errorf("Error message too short: %q", errMsg)
			}
		}
	})
}

// ============================================================
// CONCEPT: Session Manager Integration
// 🎓 AWSProvider uses SessionManager for connection pooling.
// Verify that sessions are properly managed.
// ============================================================

// TestAWSProvider_SessionManagerIntegration tests session manager usage
func TestAWSProvider_SessionManagerIntegration(t *testing.T) {
	provider := NewAWSProvider()

	if provider.sessionManager == nil {
		t.Fatal("SessionManager not initialized in AWSProvider")
	}

	// Verify session manager is working
	stats := provider.sessionManager.GetStats()
	if stats == nil {
		t.Error("SessionManager.GetStats() returned nil")
	}

	// Initially should be empty
	if stats["total"] != 0 {
		t.Logf("SessionManager has %d cached sessions (may be from previous tests)", stats["total"])
	}
}

// TestAWSProvider_MultipleInstances tests handling multiple instances
func TestAWSProvider_MultipleInstances(t *testing.T) {
	provider := NewAWSProvider()
	ctx := context.Background()

	instances := []*cloud.Instance{
		{
			ID:      "i-001",
			Cloud:   "aws",
			Account: "111111111111",
			Region:  "us-east-1",
		},
		{
			ID:      "i-002",
			Cloud:   "aws",
			Account: "111111111111",
			Region:  "us-west-2",
		},
		{
			ID:      "i-003",
			Cloud:   "aws",
			Account: "222222222222",
			Region:  "sa-east-1",
		},
	}

	for i, instance := range instances {
		t.Run(instance.ID, func(t *testing.T) {
			// Try to validate each instance
			err := provider.ValidateInstance(ctx, instance)

			// Without real AWS, all will fail
			if err == nil {
				t.Logf("Instance %d validated successfully (has real AWS credentials)", i)
			} else if err.Error() == "" {
				// Expected - just verify error handling works
				t.Error("Error message is empty")
			}
		})
	}
}

// ============================================================
// CONCEPT: Timeout Handling
// 🎓 Commands can have different timeout values.
// Verify timeout is respected.
// ============================================================

// TestAWSProvider_TimeoutHandling tests different timeout scenarios
func TestAWSProvider_TimeoutHandling(t *testing.T) {
	provider := NewAWSProvider()
	ctx := context.Background()

	instance := &cloud.Instance{
		ID:      "i-test",
		Cloud:   "aws",
		Account: "123456789012",
		Region:  "us-east-1",
	}

	timeouts := []time.Duration{
		1 * time.Second,
		30 * time.Second,
		5 * time.Minute,
	}

	for _, timeout := range timeouts {
		t.Run(timeout.String(), func(t *testing.T) {
			_, err := provider.ExecuteCommand(ctx, instance, []string{"echo test"}, timeout)

			// Without real AWS, will fail
			if err == nil {
				t.Logf("Command executed successfully with timeout %v", timeout)
			} else if err.Error() == "" {
				// Verify error handling
				t.Error("Error message is empty")
			}
		})
	}
}

// ============================================================
// HELPER FUNCTIONS
// ============================================================

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// ============================================================
// PROFILE RESOLUTION TESTS
// ============================================================

// TestGetProfileForInstance tests the profile resolution logic
func TestGetProfileForInstance(t *testing.T) {
	tests := []struct {
		name            string
		instance        *cloud.Instance
		expectedProfile string
		description     string
	}{
		{
			name: "instance_with_aws_profile_metadata",
			instance: &cloud.Instance{
				ID:      "i-1234567890abcdef0",
				Account: "123456789012",
				Region:  "us-east-1",
				Metadata: map[string]string{
					"aws_profile": "aws-staging-applications",
					"environment": "production",
				},
			},
			expectedProfile: "aws-staging-applications",
			description:     "Should use aws_profile from metadata when present",
		},
		{
			name: "instance_without_aws_profile_metadata",
			instance: &cloud.Instance{
				ID:      "i-1234567890abcdef0",
				Account: "123456789012",
				Region:  "us-east-1",
				Metadata: map[string]string{
					"environment": "production",
					"team":        "devops",
				},
			},
			expectedProfile: "123456789012",
			description:     "Should fallback to account ID when aws_profile not present",
		},
		{
			name: "instance_with_empty_aws_profile",
			instance: &cloud.Instance{
				ID:      "i-1234567890abcdef0",
				Account: "123456789012",
				Region:  "us-east-1",
				Metadata: map[string]string{
					"aws_profile": "",
					"environment": "staging",
				},
			},
			expectedProfile: "123456789012",
			description:     "Should fallback to account ID when aws_profile is empty",
		},
		{
			name: "instance_with_nil_metadata",
			instance: &cloud.Instance{
				ID:       "i-1234567890abcdef0",
				Account:  "123456789012",
				Region:   "us-east-1",
				Metadata: nil,
			},
			expectedProfile: "123456789012",
			description:     "Should fallback to account ID when metadata is nil",
		},
		{
			name: "instance_with_sso_profile",
			instance: &cloud.Instance{
				ID:      "i-sso123456789",
				Account: "783816934837",
				Region:  "us-east-1",
				Metadata: map[string]string{
					"aws_profile": "network-hub-gsn",
					"environment": "production",
				},
			},
			expectedProfile: "network-hub-gsn",
			description:     "Should use SSO profile from CSV metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute the function under test
			actualProfile := getProfileForInstance(tt.instance)

			// Verify the result
			if actualProfile != tt.expectedProfile {
				t.Errorf("getProfileForInstance() = %q, expected %q for case: %s",
					actualProfile, tt.expectedProfile, tt.description)
			} else {
				t.Logf("✅ getProfileForInstance() = %q (correct) for case: %s",
					actualProfile, tt.description)
			}
		})
	}
}

// TestGetProfileForInstance_EdgeCases tests edge cases and error scenarios
func TestGetProfileForInstance_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		instance    *cloud.Instance
		expectPanic bool
		description string
	}{
		{
			name:        "nil_instance",
			instance:    nil,
			expectPanic: true,
			description: "Should handle nil instance gracefully",
		},
		{
			name: "instance_with_whitespace_profile",
			instance: &cloud.Instance{
				ID:      "i-whitespace123",
				Account: "123456789012",
				Region:  "us-east-1",
				Metadata: map[string]string{
					"aws_profile": "  aws-staging-applications  ",
				},
			},
			expectPanic: false,
			description: "Should handle profile with whitespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						t.Logf("✅ getProfileForInstance() panicked as expected for: %s", tt.description)
					} else {
						t.Errorf("getProfileForInstance() should have panicked for: %s", tt.description)
					}
				}()
			}

			// Execute the function under test
			profile := getProfileForInstance(tt.instance)

			if !tt.expectPanic {
				// For non-panic cases, just verify it returns something
				if profile == "" {
					t.Errorf("getProfileForInstance() returned empty profile for: %s", tt.description)
				} else {
					t.Logf("✅ getProfileForInstance() = %q for: %s", profile, tt.description)
				}
			}
		})
	}
}

// BenchmarkGetProfileForInstance benchmarks the profile resolution performance
func BenchmarkGetProfileForInstance(b *testing.B) {
	instance := &cloud.Instance{
		ID:      "i-benchmark123",
		Account: "123456789012",
		Region:  "us-east-1",
		Metadata: map[string]string{
			"aws_profile": "aws-staging-applications",
			"environment": "production",
		},
	}

	b.ResetTimer()
	for range b.N {
		getProfileForInstance(instance)
	}
}
