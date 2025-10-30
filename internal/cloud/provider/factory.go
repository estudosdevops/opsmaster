package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/estudosdevops/opsmaster/internal/cloud"
	"github.com/estudosdevops/opsmaster/internal/cloud/aws"
)

// ProviderType represents supported cloud provider types
type ProviderType string

const (
	// ProviderAWS represents Amazon Web Services
	ProviderAWS ProviderType = "aws"

	// ProviderGCP represents Google Cloud Platform
	// Currently not implemented, reserved for future use
	ProviderGCP ProviderType = "gcp"

	// ProviderAzure represents Microsoft Azure
	// Currently not implemented, reserved for future use
	ProviderAzure ProviderType = "azure"
)

// Config holds configuration for cloud provider initialization.
// Used with functional options pattern for flexible provider creation.
type Config struct {
	// Profile is the cloud provider profile/credential to use
	// For AWS: ~/.aws/credentials profile name
	// For GCP: service account key file path
	// For Azure: subscription ID
	Profile string

	// Region is the default region for cloud operations
	// Optional: can be overridden per-instance from CSV
	Region string

	// Additional provider-specific options can be added here
	// Examples: Timeout, RetryConfig, CustomEndpoint, etc.
}

// Option is a functional option for configuring Config
type Option func(*Config)

// WithProfile sets the cloud provider profile/credential
func WithProfile(profile string) Option {
	return func(c *Config) {
		c.Profile = profile
	}
}

// WithRegion sets the default region for cloud operations
func WithRegion(region string) Option {
	return func(c *Config) {
		c.Region = region
	}
}

// NewProvider creates a new cloud provider based on the provider type.
// Uses Factory Pattern to abstract provider creation logic from CLI layer.
//
// Supported providers:
//   - "aws": Amazon Web Services (implemented)
//   - "gcp": Google Cloud Platform (not yet implemented)
//   - "azure": Microsoft Azure (not yet implemented)
//
// Parameters:
//   - cloudType: Provider type ("aws", "gcp", "azure")
//   - options: Functional options for provider configuration
//
// Returns:
//   - cloud.CloudProvider: Initialized provider instance
//   - error: Error if provider type is unsupported or initialization fails
//
// Example usage:
//
//	// Create AWS provider with default config
//	provider, err := provider.NewProvider("aws")
//
//	// Create AWS provider with custom profile
//	provider, err := provider.NewProvider("aws", provider.WithProfile("production"))
//
//	// Create AWS provider with profile and region
//	provider, err := provider.NewProvider("aws",
//	    provider.WithProfile("production"),
//	    provider.WithRegion("us-east-1"),
//	)
//
// Multi-cloud support:
//
//	// Detect cloud from CSV and create appropriate provider
//	cloudType := instances[0].Cloud  // "aws", "gcp", "azure"
//	provider, err := provider.NewProvider(cloudType)
func NewProvider(cloudType string, options ...Option) (cloud.CloudProvider, error) {
	// Apply functional options to config
	config := &Config{}
	for _, opt := range options {
		opt(config)
	}

	// Normalize cloud type (lowercase, trim)
	normalizedType := strings.ToLower(strings.TrimSpace(cloudType))

	// Create provider based on type
	switch ProviderType(normalizedType) {
	case ProviderAWS:
		// Create AWS provider with profile support
		if config.Profile != "" {
			// Use profile-based authentication (supports SSO)
			return aws.NewAWSProviderWithProfile(context.Background(), config.Profile)
		}
		// Fallback to default provider (uses default credentials)
		return aws.NewAWSProvider(), nil

	case ProviderGCP:
		// GCP provider not yet implemented
		return nil, fmt.Errorf("GCP provider not yet implemented (coming soon)")

	case ProviderAzure:
		// Azure provider not yet implemented
		return nil, fmt.Errorf("azure provider not yet implemented (coming soon)")

	default:
		return nil, fmt.Errorf("unsupported cloud provider: %s (supported: aws, gcp, azure)", cloudType)
	}
}

// DetectCloudFromInstances detects the cloud provider from a list of instances.
// Returns the most common cloud provider or error if instances use different clouds.
//
// This is useful when CSV contains instances from multiple clouds - validates
// that all instances are from the same cloud provider.
//
// Parameters:
//   - instances: List of cloud instances
//
// Returns:
//   - string: Detected cloud type ("aws", "gcp", "azure")
//   - error: Error if no instances, or multiple different clouds detected
//
// Example usage:
//
//	cloudType, err := provider.DetectCloudFromInstances(instances)
//	if err != nil {
//	    return fmt.Errorf("cloud detection failed: %w", err)
//	}
//	p, err := provider.NewProvider(cloudType)
func DetectCloudFromInstances(instances []*cloud.Instance) (string, error) {
	if len(instances) == 0 {
		return "", fmt.Errorf("cannot detect cloud: no instances provided")
	}

	// Get cloud from first instance
	detectedCloud := instances[0].Cloud

	// Validate all instances use same cloud
	for i, instance := range instances {
		if instance.Cloud != detectedCloud {
			return "", fmt.Errorf(
				"mixed cloud providers detected: instance[0] uses '%s' but instance[%d] uses '%s' (multi-cloud not supported in single execution)",
				detectedCloud,
				i,
				instance.Cloud,
			)
		}
	}

	return detectedCloud, nil
}

// NewProviderFromInstances detects cloud type from instances and creates provider.
// Convenience wrapper around DetectCloudFromInstances + NewProvider.
//
// Parameters:
//   - instances: List of cloud instances
//   - options: Functional options for provider configuration
//
// Returns:
//   - cloud.CloudProvider: Initialized provider instance
//   - error: Error if detection fails or provider creation fails
//
// Example usage:
//
//	// Detect cloud and create provider automatically
//	p, err := provider.NewProviderFromInstances(instances,
//	    provider.WithProfile("production"),
//	)
func NewProviderFromInstances(instances []*cloud.Instance, options ...Option) (cloud.CloudProvider, error) {
	// Detect cloud type
	cloudType, err := DetectCloudFromInstances(instances)
	if err != nil {
		return nil, fmt.Errorf("failed to detect cloud provider: %w", err)
	}

	// Create provider
	p, err := NewProvider(cloudType, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s provider: %w", cloudType, err)
	}

	return p, nil
}

// GetSupportedProviders returns list of supported cloud provider types.
// Useful for CLI help text and validation.
//
// Example usage:
//
//	supportedProviders := provider.GetSupportedProviders()
//	fmt.Printf("Supported clouds: %v\n", supportedProviders)
//	// Output: Supported clouds: [aws gcp azure]
func GetSupportedProviders() []string {
	return []string{
		string(ProviderAWS),
		string(ProviderGCP),
		string(ProviderAzure),
	}
}

// IsProviderSupported checks if a cloud provider type is supported.
// Case-insensitive comparison.
//
// Example usage:
//
//	if !provider.IsProviderSupported(userInput) {
//	    return fmt.Errorf("unsupported cloud: %s", userInput)
//	}
func IsProviderSupported(cloudType string) bool {
	normalized := strings.ToLower(strings.TrimSpace(cloudType))

	switch ProviderType(normalized) {
	case ProviderAWS, ProviderGCP, ProviderAzure:
		return true
	default:
		return false
	}
}
