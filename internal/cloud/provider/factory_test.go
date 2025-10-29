package provider

import (
	"testing"

	"github.com/estudosdevops/opsmaster/internal/cloud"
)

// ============================================================
// CONCEPT: Factory Pattern Testing
// 🎓 Factory pattern centralizes object creation logic.
// We test that the factory creates the correct provider type
// based on input, handles errors gracefully, and validates inputs.
// ============================================================

// TestNewProvider tests the factory function for creating cloud providers
func TestNewProvider(t *testing.T) {
	tests := []struct {
		name          string
		cloudType     string
		expectError   bool
		expectedName  string
		errorContains string
	}{
		{
			name:         "AWS provider lowercase",
			cloudType:    "aws",
			expectError:  false,
			expectedName: "aws",
		},
		{
			name:         "AWS provider uppercase",
			cloudType:    "AWS",
			expectError:  false,
			expectedName: "aws",
		},
		{
			name:         "AWS provider with spaces",
			cloudType:    "  aws  ",
			expectError:  false,
			expectedName: "aws",
		},
		{
			name:          "GCP provider not implemented",
			cloudType:     "gcp",
			expectError:   true,
			errorContains: "not yet implemented",
		},
		{
			name:          "Azure provider not implemented",
			cloudType:     "azure",
			expectError:   true,
			errorContains: "not yet implemented",
		},
		{
			name:          "unsupported provider",
			cloudType:     "digitalocean",
			expectError:   true,
			errorContains: "unsupported cloud provider",
		},
		{
			name:          "empty cloud type",
			cloudType:     "",
			expectError:   true,
			errorContains: "unsupported cloud provider",
		},
		{
			name:          "only spaces",
			cloudType:     "   ",
			expectError:   true,
			errorContains: "unsupported cloud provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewProvider(tt.cloudType)

			if tt.expectError {
				if err == nil {
					t.Errorf("NewProvider(%q) expected error but got nil", tt.cloudType)
				}
				if err != nil && tt.errorContains != "" {
					if !contains(err.Error(), tt.errorContains) {
						t.Errorf("NewProvider(%q) error = %q, should contain %q", tt.cloudType, err.Error(), tt.errorContains)
					}
				}
				if provider != nil {
					t.Errorf("NewProvider(%q) expected nil provider on error, but got %T", tt.cloudType, provider)
				}
			} else {
				if err != nil {
					t.Errorf("NewProvider(%q) unexpected error: %v", tt.cloudType, err)
				}
				if provider == nil {
					t.Errorf("NewProvider(%q) returned nil provider", tt.cloudType)
				}
				if provider != nil && provider.Name() != tt.expectedName {
					t.Errorf("NewProvider(%q).Name() = %q, want %q", tt.cloudType, provider.Name(), tt.expectedName)
				}
			}
		})
	}
}

// TestNewProvider_WithOptions tests functional options pattern
func TestNewProvider_WithOptions(t *testing.T) {
	t.Run("with profile option", func(t *testing.T) {
		provider, err := NewProvider("aws", WithProfile("production"))
		if err != nil {
			t.Fatalf("NewProvider() with profile returned error: %v", err)
		}
		if provider == nil {
			t.Fatal("NewProvider() returned nil provider")
		}
		// Note: Current AWS provider doesn't use config yet
		// This test validates that options don't break creation
	})

	t.Run("with region option", func(t *testing.T) {
		provider, err := NewProvider("aws", WithRegion("us-west-2"))
		if err != nil {
			t.Fatalf("NewProvider() with region returned error: %v", err)
		}
		if provider == nil {
			t.Fatal("NewProvider() returned nil provider")
		}
	})

	t.Run("with multiple options", func(t *testing.T) {
		provider, err := NewProvider("aws",
			WithProfile("staging"),
			WithRegion("sa-east-1"),
		)
		if err != nil {
			t.Fatalf("NewProvider() with multiple options returned error: %v", err)
		}
		if provider == nil {
			t.Fatal("NewProvider() returned nil provider")
		}
	})

	t.Run("options don't affect error cases", func(t *testing.T) {
		_, err := NewProvider("unsupported", WithProfile("test"))
		if err == nil {
			t.Error("NewProvider() with invalid type should return error even with valid options")
		}
	})
}

// ============================================================
// CONCEPT: Cloud Detection from Instances
// 🎓 When reading instances from CSV, we need to detect which cloud
// they belong to. This validates all instances use the same cloud.
// ============================================================

// TestDetectCloudFromInstances tests cloud type detection from instance list
func TestDetectCloudFromInstances(t *testing.T) {
	tests := []struct {
		name          string
		instances     []*cloud.Instance
		expectedCloud string
		expectError   bool
		errorContains string
	}{
		{
			name: "single AWS instance",
			instances: []*cloud.Instance{
				{ID: "i-123", Cloud: "aws", Account: "111111111111", Region: "us-east-1"},
			},
			expectedCloud: "aws",
			expectError:   false,
		},
		{
			name: "multiple AWS instances",
			instances: []*cloud.Instance{
				{ID: "i-123", Cloud: "aws", Account: "111111111111", Region: "us-east-1"},
				{ID: "i-456", Cloud: "aws", Account: "222222222222", Region: "us-west-2"},
				{ID: "i-789", Cloud: "aws", Account: "111111111111", Region: "sa-east-1"},
			},
			expectedCloud: "aws",
			expectError:   false,
		},
		{
			name: "single GCP instance",
			instances: []*cloud.Instance{
				{ID: "instance-1", Cloud: "gcp", Account: "my-project", Region: "us-central1"},
			},
			expectedCloud: "gcp",
			expectError:   false,
		},
		{
			name: "single Azure instance",
			instances: []*cloud.Instance{
				{ID: "vm-001", Cloud: "azure", Account: "subscription-id", Region: "eastus"},
			},
			expectedCloud: "azure",
			expectError:   false,
		},
		{
			name:          "empty instance list",
			instances:     []*cloud.Instance{},
			expectError:   true,
			errorContains: "no instances provided",
		},
		{
			name:          "nil instance list",
			instances:     nil,
			expectError:   true,
			errorContains: "no instances provided",
		},
		{
			name: "mixed clouds (AWS and GCP)",
			instances: []*cloud.Instance{
				{ID: "i-123", Cloud: "aws", Account: "111111111111", Region: "us-east-1"},
				{ID: "instance-1", Cloud: "gcp", Account: "my-project", Region: "us-central1"},
			},
			expectError:   true,
			errorContains: "mixed cloud providers detected",
		},
		{
			name: "mixed clouds detected at different positions",
			instances: []*cloud.Instance{
				{ID: "i-001", Cloud: "aws", Account: "111111111111", Region: "us-east-1"},
				{ID: "i-002", Cloud: "aws", Account: "111111111111", Region: "us-east-1"},
				{ID: "i-003", Cloud: "aws", Account: "111111111111", Region: "us-east-1"},
				{ID: "vm-1", Cloud: "azure", Account: "sub-123", Region: "eastus"}, // Different at index 3
			},
			expectError:   true,
			errorContains: "instance[3]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cloudType, err := DetectCloudFromInstances(tt.instances)

			if tt.expectError {
				if err == nil {
					t.Error("DetectCloudFromInstances() expected error but got nil")
				}
				if err != nil && tt.errorContains != "" {
					if !contains(err.Error(), tt.errorContains) {
						t.Errorf("Error = %q, should contain %q", err.Error(), tt.errorContains)
					}
				}
				if cloudType != "" {
					t.Errorf("DetectCloudFromInstances() returned cloud type %q on error, expected empty string", cloudType)
				}
			} else {
				if err != nil {
					t.Errorf("DetectCloudFromInstances() unexpected error: %v", err)
				}
				if cloudType != tt.expectedCloud {
					t.Errorf("DetectCloudFromInstances() = %q, want %q", cloudType, tt.expectedCloud)
				}
			}
		})
	}
}

// ============================================================
// CONCEPT: Convenience Wrapper Testing
// 🎓 NewProviderFromInstances combines detection + creation.
// This tests the happy path and error propagation.
// ============================================================

// TestNewProviderFromInstances tests the convenience wrapper
func TestNewProviderFromInstances(t *testing.T) {
	t.Run("create AWS provider from instances", func(t *testing.T) {
		instances := []*cloud.Instance{
			{ID: "i-123", Cloud: "aws", Account: "111111111111", Region: "us-east-1"},
			{ID: "i-456", Cloud: "aws", Account: "111111111111", Region: "us-east-1"},
		}

		provider, err := NewProviderFromInstances(instances)
		if err != nil {
			t.Fatalf("NewProviderFromInstances() returned error: %v", err)
		}
		if provider == nil {
			t.Fatal("NewProviderFromInstances() returned nil provider")
		}
		if provider.Name() != "aws" {
			t.Errorf("Provider.Name() = %q, want %q", provider.Name(), "aws")
		}
	})

	t.Run("error when instances have mixed clouds", func(t *testing.T) {
		instances := []*cloud.Instance{
			{ID: "i-123", Cloud: "aws", Account: "111111111111", Region: "us-east-1"},
			{ID: "vm-1", Cloud: "azure", Account: "sub-123", Region: "eastus"},
		}

		_, err := NewProviderFromInstances(instances)
		if err == nil {
			t.Error("NewProviderFromInstances() expected error for mixed clouds")
		}
		if !contains(err.Error(), "detect cloud provider") {
			t.Errorf("Error should mention detection failure, got: %v", err)
		}
	})

	t.Run("error when no instances", func(t *testing.T) {
		instances := []*cloud.Instance{}

		_, err := NewProviderFromInstances(instances)
		if err == nil {
			t.Error("NewProviderFromInstances() expected error for empty instances")
		}
	})

	t.Run("error when cloud not implemented", func(t *testing.T) {
		instances := []*cloud.Instance{
			{ID: "instance-1", Cloud: "gcp", Account: "project-123", Region: "us-central1"},
		}

		_, err := NewProviderFromInstances(instances)
		if err == nil {
			t.Error("NewProviderFromInstances() expected error for GCP (not implemented)")
		}
		if !contains(err.Error(), "not yet implemented") {
			t.Errorf("Error should mention provider not implemented, got: %v", err)
		}
	})

	t.Run("with functional options", func(t *testing.T) {
		instances := []*cloud.Instance{
			{ID: "i-123", Cloud: "aws", Account: "111111111111", Region: "us-east-1"},
		}

		provider, err := NewProviderFromInstances(instances,
			WithProfile("production"),
			WithRegion("us-east-1"),
		)
		if err != nil {
			t.Fatalf("NewProviderFromInstances() with options returned error: %v", err)
		}
		if provider == nil {
			t.Fatal("NewProviderFromInstances() returned nil provider")
		}
	})
}

// ============================================================
// CONCEPT: Supported Providers List
// 🎓 These functions help with validation and help text.
// ============================================================

// TestGetSupportedProviders tests the list of supported cloud providers
func TestGetSupportedProviders(t *testing.T) {
	providers := GetSupportedProviders()

	if len(providers) == 0 {
		t.Fatal("GetSupportedProviders() returned empty list")
	}

	expectedProviders := map[string]bool{
		"aws":   true,
		"gcp":   true,
		"azure": true,
	}

	if len(providers) != len(expectedProviders) {
		t.Errorf("GetSupportedProviders() returned %d providers, want %d", len(providers), len(expectedProviders))
	}

	for _, provider := range providers {
		if !expectedProviders[provider] {
			t.Errorf("Unexpected provider in list: %s", provider)
		}
	}

	// Verify specific providers are present
	hasAWS := false
	hasGCP := false
	hasAzure := false

	for _, p := range providers {
		switch p {
		case "aws":
			hasAWS = true
		case "gcp":
			hasGCP = true
		case "azure":
			hasAzure = true
		}
	}

	if !hasAWS {
		t.Error("GetSupportedProviders() missing 'aws'")
	}
	if !hasGCP {
		t.Error("GetSupportedProviders() missing 'gcp'")
	}
	if !hasAzure {
		t.Error("GetSupportedProviders() missing 'azure'")
	}
}

// TestIsProviderSupported tests provider validation
func TestIsProviderSupported(t *testing.T) {
	tests := []struct {
		name      string
		cloudType string
		expected  bool
	}{
		{
			name:      "AWS lowercase",
			cloudType: "aws",
			expected:  true,
		},
		{
			name:      "AWS uppercase",
			cloudType: "AWS",
			expected:  true,
		},
		{
			name:      "AWS mixed case",
			cloudType: "AwS",
			expected:  true,
		},
		{
			name:      "AWS with leading spaces",
			cloudType: "  aws",
			expected:  true,
		},
		{
			name:      "AWS with trailing spaces",
			cloudType: "aws  ",
			expected:  true,
		},
		{
			name:      "AWS with both spaces",
			cloudType: "  aws  ",
			expected:  true,
		},
		{
			name:      "GCP lowercase",
			cloudType: "gcp",
			expected:  true,
		},
		{
			name:      "GCP uppercase",
			cloudType: "GCP",
			expected:  true,
		},
		{
			name:      "Azure lowercase",
			cloudType: "azure",
			expected:  true,
		},
		{
			name:      "Azure uppercase",
			cloudType: "AZURE",
			expected:  true,
		},
		{
			name:      "unsupported provider",
			cloudType: "digitalocean",
			expected:  false,
		},
		{
			name:      "unsupported provider uppercase",
			cloudType: "DIGITALOCEAN",
			expected:  false,
		},
		{
			name:      "empty string",
			cloudType: "",
			expected:  false,
		},
		{
			name:      "only spaces",
			cloudType: "   ",
			expected:  false,
		},
		{
			name:      "invalid characters",
			cloudType: "aws@123",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsProviderSupported(tt.cloudType)
			if result != tt.expected {
				t.Errorf("IsProviderSupported(%q) = %v, want %v", tt.cloudType, result, tt.expected)
			}
		})
	}
}

// ============================================================
// CONCEPT: ProviderType Constants Testing
// 🎓 Validates that constants have expected values.
// ============================================================

// TestProviderType_Constants tests the ProviderType constants
func TestProviderType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		provider ProviderType
		expected string
	}{
		{
			name:     "ProviderAWS constant",
			provider: ProviderAWS,
			expected: "aws",
		},
		{
			name:     "ProviderGCP constant",
			provider: ProviderGCP,
			expected: "gcp",
		},
		{
			name:     "ProviderAzure constant",
			provider: ProviderAzure,
			expected: "azure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.provider) != tt.expected {
				t.Errorf("ProviderType constant = %q, want %q", string(tt.provider), tt.expected)
			}
		})
	}
}

// TestConfig_FunctionalOptions tests the Config struct and option functions
func TestConfig_FunctionalOptions(t *testing.T) {
	t.Run("WithProfile option", func(t *testing.T) {
		config := &Config{}
		opt := WithProfile("production")
		opt(config)

		if config.Profile != "production" {
			t.Errorf("Config.Profile = %q, want %q", config.Profile, "production")
		}
	})

	t.Run("WithRegion option", func(t *testing.T) {
		config := &Config{}
		opt := WithRegion("us-west-2")
		opt(config)

		if config.Region != "us-west-2" {
			t.Errorf("Config.Region = %q, want %q", config.Region, "us-west-2")
		}
	})

	t.Run("multiple options", func(t *testing.T) {
		config := &Config{}
		options := []Option{
			WithProfile("staging"),
			WithRegion("sa-east-1"),
		}

		for _, opt := range options {
			opt(config)
		}

		if config.Profile != "staging" {
			t.Errorf("Config.Profile = %q, want %q", config.Profile, "staging")
		}
		if config.Region != "sa-east-1" {
			t.Errorf("Config.Region = %q, want %q", config.Region, "sa-east-1")
		}
	})

	t.Run("empty config by default", func(t *testing.T) {
		config := &Config{}

		if config.Profile != "" {
			t.Errorf("Config.Profile should be empty by default, but is %q", config.Profile)
		}
		if config.Region != "" {
			t.Errorf("Config.Region should be empty by default, but is %q", config.Region)
		}
	})
}

// ============================================================
// HELPER FUNCTIONS
// ============================================================

// contains checks if a string contains a substring (simple helper)
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
