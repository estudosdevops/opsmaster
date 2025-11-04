package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// AuthConfig holds AWS authentication configuration options.
// This struct follows the "options pattern" commonly used in Go.
//
// 🎓 CONCEPT: Struct in Go
// Structs group related data together. Think of it as a "class" but simpler.
type AuthConfig struct {
	Profile string // AWS profile name (supports SSO profiles)
	Region  string // AWS region (optional, can come from profile)
}

// NewAWSConfig creates a new AWS config using the specified profile.
// This function handles both traditional and SSO profiles automatically.
// It uses AWS SDK v2 for better performance and modern Go patterns.
//
// 🎓 CONCEPT: Context-First Design (AWS SDK v2)
// SDK v2 uses context as the first parameter for cancellation and timeouts.
// This is a modern Go pattern for handling request lifecycle.
//
// 🎓 CONCEPT: Functional Options Pattern (AWS SDK v2)
// Instead of a struct with many fields, v2 uses functions as options.
// This makes the API more flexible and easier to extend.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - authConfig: Authentication configuration
//
// Returns:
//   - aws.Config: AWS configuration ready to use with any AWS service
//   - error: nil if successful, error if authentication fails
func NewAWSConfig(ctx context.Context, authConfig AuthConfig) (aws.Config, error) {
	// Build configuration options using functional options pattern
	var opts []func(*config.LoadOptions) error

	// If profile is specified, use it. Otherwise, AWS SDK uses default profile
	if authConfig.Profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(authConfig.Profile))
	}

	// If region is specified, set it in options
	if authConfig.Region != "" {
		opts = append(opts, config.WithRegion(authConfig.Region))
	}

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		// Error wrapping: provides context while preserving original error
		return aws.Config{}, fmt.Errorf("failed to load AWS config with profile '%s': %w",
			authConfig.Profile, err)
	}

	return cfg, nil
}
