package aws

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// SessionManager manages AWS client connections using connection pooling.
// This implements the Pool Pattern - reusing connections instead of creating new ones
// for each request, which significantly improves performance.
//
// Thread-safe for concurrent access using RWMutex.
type SessionManager struct {
	sessions   map[string]*awsSession // Key: "profile-region"
	ec2Clients map[string]*ec2.Client // Pool of EC2 clients for tagging
	mu         sync.RWMutex           // Read-write lock for thread safety
}

// awsSession represents a cached AWS session with its clients
type awsSession struct {
	ssmClient *ssm.Client
	profile   string
	region    string
}

// NewSessionManager creates a new session manager with empty pools
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions:   make(map[string]*awsSession),
		ec2Clients: make(map[string]*ec2.Client),
	}
}

// NewSessionManagerWithProfile creates a session manager with a specific AWS profile.
// This constructor validates the profile configuration and ensures SSO compatibility.
//
// Parameters:
//   - ctx: Context for timeout and cancellation
//   - profile: AWS profile name from ~/.aws/config
//
// Returns:
//   - *SessionManager: Configured session manager
//   - error: Error if profile is invalid or AWS config loading fails
//
// Authentication Flow:
//  1. Loads AWS config using our auth.go module
//  2. Validates profile exists and is accessible
//  3. Creates session manager ready to use with that profile
//
// Note: This doesn't create actual AWS clients yet - that happens lazily
// when GetSSMClient() or GetEC2Client() are called.
func NewSessionManagerWithProfile(ctx context.Context, profile string) (*SessionManager, error) {
	// Use our auth.go to validate the profile configuration
	// This will fail early if profile doesn't exist or SSO isn't configured
	authConfig := AuthConfig{
		Profile: profile,
		Region:  "", // Region will be determined per-instance
	}

	_, err := NewAWSConfig(ctx, authConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to validate AWS profile '%s': %w", profile, err)
	}

	// Profile is valid, create session manager
	// The actual clients will be created with the profile when needed
	return &SessionManager{
		sessions:   make(map[string]*awsSession),
		ec2Clients: make(map[string]*ec2.Client),
	}, nil
}

// GetSSMClient returns a cached SSM client or creates a new one if needed.
// Uses connection pooling for performance - reuses existing clients when possible.
//
// Parameters:
//   - ctx: context for timeout/cancellation
//   - profile: AWS profile name from ~/.aws/credentials
//   - region: AWS region (e.g., us-east-1)
//
// Returns cached client if exists, creates new one otherwise.
// Thread-safe using RWMutex with double-check locking pattern.
func (sm *SessionManager) GetSSMClient(ctx context.Context, profile, region string) (*ssm.Client, error) {
	key := sm.makeKey(profile, region)

	// First check: read lock (allows multiple concurrent reads)
	sm.mu.RLock()
	if session, exists := sm.sessions[key]; exists {
		sm.mu.RUnlock()
		return session.ssmClient, nil
	}
	sm.mu.RUnlock()

	// Client not found, need to create - acquire write lock
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Double-check: another goroutine might have created it while we waited for write lock
	if session, exists := sm.sessions[key]; exists {
		return session.ssmClient, nil
	}

	// Create new AWS session
	cfg, err := sm.loadAWSConfig(ctx, profile, region)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for profile '%s' in region '%s': %w", profile, region, err)
	}

	// Create SSM client
	ssmClient := ssm.NewFromConfig(cfg)

	// Cache session for future reuse
	sm.sessions[key] = &awsSession{
		ssmClient: ssmClient,
		profile:   profile,
		region:    region,
	}

	return ssmClient, nil
}

// GetEC2Client returns a cached EC2 client or creates a new one if needed.
// Used for tagging instances after successful installation.
//
// Parameters:
//   - ctx: context for timeout/cancellation
//   - profile: AWS profile name from ~/.aws/credentials
//   - region: AWS region (e.g., us-east-1)
//
// Returns cached client if exists, creates new one otherwise.
// Thread-safe using RWMutex with double-check locking pattern.
func (sm *SessionManager) GetEC2Client(ctx context.Context, profile, region string) (*ec2.Client, error) {
	key := sm.makeKey(profile, region)

	// First check: read lock (allows multiple concurrent reads)
	sm.mu.RLock()
	if client, exists := sm.ec2Clients[key]; exists {
		sm.mu.RUnlock()
		return client, nil
	}
	sm.mu.RUnlock()

	// Client not found, need to create - acquire write lock
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Double-check: another goroutine might have created it while we waited for write lock
	if client, exists := sm.ec2Clients[key]; exists {
		return client, nil
	}

	// Create new AWS session
	cfg, err := sm.loadAWSConfig(ctx, profile, region)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for profile '%s' in region '%s': %w", profile, region, err)
	}

	// Create EC2 client
	ec2Client := ec2.NewFromConfig(cfg)

	// Cache client for future reuse
	sm.ec2Clients[key] = ec2Client

	return ec2Client, nil
}

// loadAWSConfig loads AWS configuration using our centralized auth.go module.
// This eliminates code duplication and ensures consistent authentication behavior.
//
// Uses AWS SDK v2 default credential chain:
// 1. Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
// 2. Shared credentials file (~/.aws/credentials)
// 3. IAM role (if running on EC2)
func (*SessionManager) loadAWSConfig(ctx context.Context, profile, region string) (aws.Config, error) {
	// Use our centralized auth.go function instead of duplicating logic
	authConfig := AuthConfig{
		Profile: profile,
		Region:  region,
	}

	return NewAWSConfig(ctx, authConfig)
}

// makeKey creates a unique key for caching clients
// Format: "profile-region" (e.g., "default-us-east-1")
func (*SessionManager) makeKey(profile, region string) string {
	return profile + "-" + region
}

// Close closes all cached clients and clears pools
// Should be called when shutting down the application
func (sm *SessionManager) Close() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Clear all cached sessions
	sm.sessions = make(map[string]*awsSession)
	sm.ec2Clients = make(map[string]*ec2.Client)
}

// GetStats returns statistics about cached clients
// Useful for monitoring and debugging
func (sm *SessionManager) GetStats() map[string]int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return map[string]int{
		"ssm_clients": len(sm.sessions),
		"ec2_clients": len(sm.ec2Clients),
		"total":       len(sm.sessions) + len(sm.ec2Clients),
	}
}
