package cloud

import (
	"context"
	"time"
)

// CloudProvider abstracts operations for any cloud provider (AWS, Azure, GCP).
// This interface defines the "contract" that any cloud implementation must follow.
// This allows us to write generic code that works with any cloud!
type CloudProvider interface {
	// Name returns the provider name (aws, azure, gcp)
	Name() string

	// ValidateInstance checks if instance is accessible via remote management
	// (SSM for AWS, Run Command for Azure, OS Login for GCP).
	// Returns error if instance is not online or not manageable.
	ValidateInstance(ctx context.Context, instance *Instance) error

	// ExecuteCommand executes shell commands remotely on the instance.
	// commands: slice of commands to execute
	// timeout: maximum execution time
	// Returns execution result with stdout, stderr and exit code.
	ExecuteCommand(ctx context.Context, instance *Instance, commands []string, timeout time.Duration) (*CommandResult, error)

	// TestConnectivity tests network connectivity from instance to a host:port.
	// Useful for validating if instance can reach external services
	// (e.g., Puppet Server on port 8140).
	TestConnectivity(ctx context.Context, instance *Instance, host string, port int) error

	// TagInstance adds tags/labels to the instance.
	// tags: map of key-value to apply
	// Used to mark instances after successful installation.
	TagInstance(ctx context.Context, instance *Instance, tags map[string]string) error

	// HasTag checks if instance already has a specific tag.
	// Useful for idempotency - don't reprocess already configured instances.
	HasTag(ctx context.Context, instance *Instance, key, value string) (bool, error)
}

// Instance represents a generic VM instance in any cloud.
// This struct is cloud-agnostic - works for AWS EC2, Azure VM, GCP Compute.
type Instance struct {
	ID       string            // Unique instance ID (e.g., i-1234567890abcdef0)
	Cloud    string            // Provider: "aws", "azure", "gcp"
	Account  string            // Account/Subscription/Project ID
	Region   string            // Instance region (e.g., us-east-1)
	Metadata map[string]string // Optional extra data (existing tags, hostname, etc)
}

// String returns readable representation of the instance
func (i *Instance) String() string {
	return i.Cloud + ":" + i.Account + ":" + i.Region + ":" + i.ID
}

// CommandResult encapsulates the result of a remote command execution
type CommandResult struct {
	InstanceID string        // ID of instance where command was executed
	ExitCode   int           // Command exit code (0 = success)
	Stdout     string        // Standard output of the command
	Stderr     string        // Error output of the command
	Duration   time.Duration // Time it took to execute
	Error      error         // Go error (if any) during execution
}

// Success returns true if command executed successfully (exit code 0)
func (cr *CommandResult) Success() bool {
	return cr.ExitCode == 0 && cr.Error == nil
}

// Failed returns true if command failed
func (cr *CommandResult) Failed() bool {
	return !cr.Success()
}
