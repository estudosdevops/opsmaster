package installer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/estudosdevops/opsmaster/internal/cloud"
)

// ============================================================
// MOCKS - Interface implementation testing
// ============================================================

// mockPackageInstaller is a mock implementation of PackageInstaller interface.
// Used to verify that interface contracts are correctly defined.
type mockPackageInstaller struct {
	name                      string
	generateScriptFunc        func(os string, options map[string]string) ([]string, error)
	validatePrerequisitesFunc func(ctx context.Context, instance *cloud.Instance, provider cloud.CloudProvider) error
	verifyInstallationFunc    func(ctx context.Context, instance *cloud.Instance, provider cloud.CloudProvider) error
	successTags               map[string]string
	failureTags               map[string]string
	metadata                  map[string]string
}

// Name implements PackageInstaller interface
func (m *mockPackageInstaller) Name() string {
	return m.name
}

// GenerateInstallScript implements PackageInstaller interface
func (m *mockPackageInstaller) GenerateInstallScript(os string, options map[string]string) ([]string, error) {
	if m.generateScriptFunc != nil {
		return m.generateScriptFunc(os, options)
	}
	return []string{"echo 'mock install'"}, nil
}

// ValidatePrerequisites implements PackageInstaller interface
func (m *mockPackageInstaller) ValidatePrerequisites(ctx context.Context, instance *cloud.Instance, provider cloud.CloudProvider) error {
	if m.validatePrerequisitesFunc != nil {
		return m.validatePrerequisitesFunc(ctx, instance, provider)
	}
	return nil
}

// VerifyInstallation implements PackageInstaller interface
func (m *mockPackageInstaller) VerifyInstallation(ctx context.Context, instance *cloud.Instance, provider cloud.CloudProvider) error {
	if m.verifyInstallationFunc != nil {
		return m.verifyInstallationFunc(ctx, instance, provider)
	}
	return nil
}

// GetSuccessTags implements PackageInstaller interface
func (m *mockPackageInstaller) GetSuccessTags() map[string]string {
	if m.successTags != nil {
		return m.successTags
	}
	return map[string]string{"installed": "true"}
}

// GetFailureTags implements PackageInstaller interface
func (m *mockPackageInstaller) GetFailureTags(_ error) map[string]string {
	if m.failureTags != nil {
		return m.failureTags
	}
	return map[string]string{"installed": "failed"}
}

// GetInstallMetadata implements PackageInstaller interface
func (m *mockPackageInstaller) GetInstallMetadata() map[string]string {
	if m.metadata != nil {
		return m.metadata
	}
	return map[string]string{}
}

// ============================================================
// PACKAGE INSTALLER INTERFACE TESTS
// ============================================================

// TestPackageInstaller_InterfaceCompliance verifies that mock properly implements interface.
//
// 🎓 CONCEPT: Interface compliance testing
// - Verify that implementations satisfy the interface contract
// - This test will fail at compile time if interface changes
func TestPackageInstaller_InterfaceCompliance(t *testing.T) {
	// ARRANGE
	var _ PackageInstaller = (*mockPackageInstaller)(nil)

	mock := &mockPackageInstaller{
		name: "test-installer",
	}

	// ACT & ASSERT - Verify all interface methods are callable
	if mock.Name() != "test-installer" {
		t.Errorf("Name() = %q, want %q", mock.Name(), "test-installer")
	}

	scripts, err := mock.GenerateInstallScript("ubuntu", nil)
	if err != nil {
		t.Errorf("GenerateInstallScript() error = %v", err)
	}
	if len(scripts) == 0 {
		t.Error("GenerateInstallScript() returned empty script")
	}

	ctx := context.Background()
	instance := &cloud.Instance{ID: "i-test"}

	if err := mock.ValidatePrerequisites(ctx, instance, nil); err != nil {
		t.Errorf("ValidatePrerequisites() error = %v", err)
	}

	if err := mock.VerifyInstallation(ctx, instance, nil); err != nil {
		t.Errorf("VerifyInstallation() error = %v", err)
	}

	successTags := mock.GetSuccessTags()
	if len(successTags) == 0 {
		t.Error("GetSuccessTags() returned empty map")
	}

	failureTags := mock.GetFailureTags(errors.New("test error"))
	if len(failureTags) == 0 {
		t.Error("GetFailureTags() returned empty map")
	}

	metadata := mock.GetInstallMetadata()
	if metadata == nil {
		t.Error("GetInstallMetadata() returned nil")
	}
}

// TestPackageInstaller_GenerateInstallScript tests script generation behavior.
func TestPackageInstaller_GenerateInstallScript(t *testing.T) {
	tests := []struct {
		name          string
		os            string
		options       map[string]string
		expectedError bool
		validateFunc  func(t *testing.T, scripts []string)
	}{
		{
			name:          "successful script generation",
			os:            "ubuntu",
			options:       map[string]string{"version": "7.0"},
			expectedError: false,
			validateFunc: func(t *testing.T, scripts []string) {
				if len(scripts) == 0 {
					t.Error("Expected non-empty script")
				}
			},
		},
		{
			name:          "script generation with error",
			os:            "unsupported-os",
			options:       nil,
			expectedError: true,
			validateFunc:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			mock := &mockPackageInstaller{
				name: "test",
				generateScriptFunc: func(os string, _ map[string]string) ([]string, error) {
					if os == "unsupported-os" {
						return nil, errors.New("unsupported OS")
					}
					return []string{"apt-get update", "apt-get install -y package"}, nil
				},
			}

			// ACT
			scripts, err := mock.GenerateInstallScript(tt.os, tt.options)

			// ASSERT
			if tt.expectedError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.validateFunc != nil && !tt.expectedError {
				tt.validateFunc(t, scripts)
			}
		})
	}
}

// TestPackageInstaller_ValidatePrerequisites tests prerequisite validation.
func TestPackageInstaller_ValidatePrerequisites(t *testing.T) {
	tests := []struct {
		name          string
		mockFunc      func(ctx context.Context, instance *cloud.Instance, provider cloud.CloudProvider) error
		expectedError bool
	}{
		{
			name: "all prerequisites pass",
			mockFunc: func(_ context.Context, _ *cloud.Instance, _ cloud.CloudProvider) error {
				return nil
			},
			expectedError: false,
		},
		{
			name: "prerequisite validation fails",
			mockFunc: func(_ context.Context, _ *cloud.Instance, _ cloud.CloudProvider) error {
				return errors.New("connectivity check failed")
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			mock := &mockPackageInstaller{
				name:                      "test",
				validatePrerequisitesFunc: tt.mockFunc,
			}
			ctx := context.Background()
			instance := &cloud.Instance{ID: "i-test"}

			// ACT
			err := mock.ValidatePrerequisites(ctx, instance, nil)

			// ASSERT
			if tt.expectedError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestPackageInstaller_VerifyInstallation tests installation verification.
func TestPackageInstaller_VerifyInstallation(t *testing.T) {
	tests := []struct {
		name          string
		mockFunc      func(ctx context.Context, instance *cloud.Instance, provider cloud.CloudProvider) error
		expectedError bool
	}{
		{
			name: "verification succeeds",
			mockFunc: func(_ context.Context, _ *cloud.Instance, _ cloud.CloudProvider) error {
				return nil
			},
			expectedError: false,
		},
		{
			name: "verification fails",
			mockFunc: func(_ context.Context, _ *cloud.Instance, _ cloud.CloudProvider) error {
				return errors.New("package not found")
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			mock := &mockPackageInstaller{
				name:                   "test",
				verifyInstallationFunc: tt.mockFunc,
			}
			ctx := context.Background()
			instance := &cloud.Instance{ID: "i-test"}

			// ACT
			err := mock.VerifyInstallation(ctx, instance, nil)

			// ASSERT
			if tt.expectedError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestPackageInstaller_GetSuccessTags tests success tag generation.
func TestPackageInstaller_GetSuccessTags(t *testing.T) {
	// ARRANGE
	expectedTags := map[string]string{
		"puppet":              "true",
		"puppet_server":       "puppet.example.com",
		"puppet_installed_at": "2025-10-28",
	}

	mock := &mockPackageInstaller{
		name:        "puppet",
		successTags: expectedTags,
	}

	// ACT
	tags := mock.GetSuccessTags()

	// ASSERT
	if len(tags) != len(expectedTags) {
		t.Errorf("len(tags) = %d, want %d", len(tags), len(expectedTags))
	}

	for key, expectedValue := range expectedTags {
		actualValue, exists := tags[key]
		if !exists {
			t.Errorf("Expected tag %q not found", key)
		}
		if actualValue != expectedValue {
			t.Errorf("tags[%q] = %q, want %q", key, actualValue, expectedValue)
		}
	}
}

// TestPackageInstaller_GetFailureTags tests failure tag generation.
func TestPackageInstaller_GetFailureTags(t *testing.T) {
	// ARRANGE
	installError := errors.New("connection timeout")
	expectedTags := map[string]string{
		"puppet":       "failed",
		"puppet_error": "connection timeout",
	}

	mock := &mockPackageInstaller{
		name:        "puppet",
		failureTags: expectedTags,
	}

	// ACT
	tags := mock.GetFailureTags(installError)

	// ASSERT
	if len(tags) != len(expectedTags) {
		t.Errorf("len(tags) = %d, want %d", len(tags), len(expectedTags))
	}

	for key, expectedValue := range expectedTags {
		actualValue, exists := tags[key]
		if !exists {
			t.Errorf("Expected tag %q not found", key)
		}
		if actualValue != expectedValue {
			t.Errorf("tags[%q] = %q, want %q", key, actualValue, expectedValue)
		}
	}
}

// TestPackageInstaller_GetInstallMetadata tests metadata retrieval.
func TestPackageInstaller_GetInstallMetadata(t *testing.T) {
	tests := []struct {
		name             string
		metadata         map[string]string
		expectedNotEmpty bool
	}{
		{
			name: "with metadata",
			metadata: map[string]string{
				"os":                 "ubuntu",
				"certname":           "abc123.puppet",
				"certname_preserved": "false",
			},
			expectedNotEmpty: true,
		},
		{
			name:             "empty metadata",
			metadata:         map[string]string{},
			expectedNotEmpty: false,
		},
		{
			name:             "nil metadata (no installation yet)",
			metadata:         nil,
			expectedNotEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			mock := &mockPackageInstaller{
				name:     "test",
				metadata: tt.metadata,
			}

			// ACT
			metadata := mock.GetInstallMetadata()

			// ASSERT
			if metadata == nil {
				t.Fatal("GetInstallMetadata() returned nil")
			}

			if tt.expectedNotEmpty && len(metadata) == 0 {
				t.Error("Expected non-empty metadata")
			}

			if !tt.expectedNotEmpty && len(metadata) > 0 {
				t.Errorf("Expected empty metadata, got %v", metadata)
			}
		})
	}
}

// ============================================================
// INSTALL OPTIONS TESTS
// ============================================================

// TestInstallOptions_DefaultValues tests default option values.
func TestInstallOptions_DefaultValues(t *testing.T) {
	// ACT
	opts := InstallOptions{}

	// ASSERT - Zero values should be valid defaults
	if opts.DryRun {
		t.Error("DryRun should default to false")
	}
	if opts.SkipValidation {
		t.Error("SkipValidation should default to false")
	}
	if opts.SkipTagging {
		t.Error("SkipTagging should default to false")
	}
	if opts.MaxConcurrency != 0 {
		t.Errorf("MaxConcurrency should default to 0, got %d", opts.MaxConcurrency)
	}
	if opts.Timeout != 0 {
		t.Errorf("Timeout should default to 0, got %v", opts.Timeout)
	}
	if opts.CustomOptions != nil {
		t.Error("CustomOptions should default to nil")
	}
}

// TestInstallOptions_CustomValues tests setting custom values.
func TestInstallOptions_CustomValues(t *testing.T) {
	// ARRANGE & ACT
	opts := InstallOptions{
		DryRun:         true,
		SkipValidation: true,
		SkipTagging:    false,
		MaxConcurrency: 10,
		Timeout:        5 * time.Minute,
		CustomOptions: map[string]string{
			"server":      "puppet.example.com",
			"environment": "production",
		},
	}

	// ASSERT
	if !opts.DryRun {
		t.Error("DryRun should be true")
	}
	if !opts.SkipValidation {
		t.Error("SkipValidation should be true")
	}
	if opts.SkipTagging {
		t.Error("SkipTagging should be false")
	}
	if opts.MaxConcurrency != 10 {
		t.Errorf("MaxConcurrency = %d, want 10", opts.MaxConcurrency)
	}
	if opts.Timeout != 5*time.Minute {
		t.Errorf("Timeout = %v, want 5m", opts.Timeout)
	}
	if len(opts.CustomOptions) != 2 {
		t.Errorf("len(CustomOptions) = %d, want 2", len(opts.CustomOptions))
	}
}

// ============================================================
// INSTALL RESULT TESTS
// ============================================================

// TestInstallResult_String tests string representation.
//
// 🎓 CONCEPT: Stringer interface testing
// - Verify human-readable output format
// - Test both success and failure cases
func TestInstallResult_String(t *testing.T) {
	tests := []struct {
		name           string
		result         *InstallResult
		expectedSubstr string
	}{
		{
			name: "successful installation",
			result: &InstallResult{
				Instance: &cloud.Instance{ID: "i-123abc"},
				Success:  true,
				Duration: 30 * time.Second,
			},
			expectedSubstr: "SUCCESS",
		},
		{
			name: "failed installation",
			result: &InstallResult{
				Instance: &cloud.Instance{ID: "i-456def"},
				Success:  false,
				Error:    errors.New("connection timeout"),
			},
			expectedSubstr: "FAILED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ACT
			str := tt.result.String()

			// ASSERT
			if !contains(str, tt.expectedSubstr) {
				t.Errorf("String() = %q, want to contain %q", str, tt.expectedSubstr)
			}

			// Should contain instance ID
			if !contains(str, tt.result.Instance.ID) {
				t.Errorf("String() = %q, want to contain instance ID %q", str, tt.result.Instance.ID)
			}

			// Success case should contain duration
			if tt.result.Success && !contains(str, "s") {
				t.Errorf("String() = %q, want to contain duration", str)
			}

			// Failure case should contain error message
			if !tt.result.Success && tt.result.Error != nil {
				if !contains(str, tt.result.Error.Error()) {
					t.Errorf("String() = %q, want to contain error %q", str, tt.result.Error.Error())
				}
			}
		})
	}
}

// TestInstallResult_CompleteStructure tests complete result structure.
func TestInstallResult_CompleteStructure(t *testing.T) {
	// ARRANGE
	now := time.Now()
	instance := &cloud.Instance{
		ID:      "i-complete",
		Account: "123456789012",
		Region:  "us-east-1",
	}

	// ACT - Create complete result
	result := &InstallResult{
		Instance:  instance,
		Success:   true,
		Error:     nil,
		Duration:  45 * time.Second,
		Output:    "Installation completed successfully",
		StartTime: now,
		EndTime:   now.Add(45 * time.Second),
		Tagged:    true,
	}

	// ASSERT - Verify all fields
	if result.Instance.ID != "i-complete" {
		t.Errorf("Instance.ID = %q, want %q", result.Instance.ID, "i-complete")
	}

	if !result.Success {
		t.Error("Success should be true")
	}

	if result.Error != nil {
		t.Errorf("Error should be nil, got %v", result.Error)
	}

	if result.Duration != 45*time.Second {
		t.Errorf("Duration = %v, want 45s", result.Duration)
	}

	if result.Output == "" {
		t.Error("Output should not be empty")
	}

	if result.StartTime.IsZero() {
		t.Error("StartTime should not be zero")
	}

	if result.EndTime.Before(result.StartTime) {
		t.Error("EndTime should be after StartTime")
	}

	if !result.Tagged {
		t.Error("Tagged should be true")
	}
}

// TestInstallResult_FailureCase tests failure scenario.
func TestInstallResult_FailureCase(t *testing.T) {
	// ARRANGE
	installError := errors.New("package repository unreachable")
	instance := &cloud.Instance{ID: "i-failed"}

	// ACT
	result := &InstallResult{
		Instance: instance,
		Success:  false,
		Error:    installError,
		Duration: 10 * time.Second,
		Output:   "Error: connection timeout",
		Tagged:   false,
	}

	// ASSERT
	if result.Success {
		t.Error("Success should be false for failed installation")
	}

	if result.Error == nil {
		t.Fatal("Error should not be nil for failed installation")
	}

	if result.Error.Error() != "package repository unreachable" {
		t.Errorf("Error = %q, want %q", result.Error.Error(), "package repository unreachable")
	}

	if result.Tagged {
		t.Error("Tagged should be false for failed installation")
	}

	str := result.String()
	if !contains(str, "FAILED") {
		t.Errorf("String() should indicate failure, got %q", str)
	}
}
