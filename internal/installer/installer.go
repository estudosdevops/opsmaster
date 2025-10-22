package installer

import (
	"context"
	"time"

	"github.com/estudosdevops/opsmaster/internal/cloud"
)

// PackageInstaller abstracts the installation of any package/software.
// Each package (Puppet, Docker, K8s agent, etc) implements this interface.
// This is the STRATEGY PATTERN in Go - different installation strategies.
type PackageInstaller interface {
	// Name returns the package name (puppet, docker, kubernetes-agent, etc)
	Name() string

	// GenerateInstallScript generates installation script based on the operating system.
	// os: detected operating system (ubuntu, debian, rhel, amzn)
	// options: custom options (version, specific configurations)
	// Returns: slice of shell commands to execute
	GenerateInstallScript(os string, options map[string]string) ([]string, error)

	// ValidatePrerequisites validates prerequisites BEFORE installing.
	// E.g., Puppet needs to validate connectivity with Puppet Server
	// E.g., Docker can validate if there are sufficient resources
	// Returns error if any prerequisite is not met.
	ValidatePrerequisites(ctx context.Context, instance *cloud.Instance, provider cloud.CloudProvider) error

	// VerifyInstallation verifies if installation was successful.
	// E.g., execute 'puppet --version' and verify exit code
	// E.g., execute 'docker ps' and verify it works
	// Returns error if verification fails.
	VerifyInstallation(ctx context.Context, instance *cloud.Instance, provider cloud.CloudProvider) error

	// GetSuccessTags returns tags that should be applied after successful installation.
	// E.g., puppet=true, puppet_server=puppet.example.com, puppet_installed_at=2025-01-15
	// This marks the instance and enables idempotency (don't reinstall).
	GetSuccessTags() map[string]string

	// GetFailureTags returns tags to apply when installation fails (optional).
	// E.g., puppet=failed, puppet_error=connection_timeout
	// Useful for troubleshooting and later retry.
	GetFailureTags(err error) map[string]string

	// GetInstallMetadata returns metadata from the last installation attempt.
	// E.g., os=rhel, certname=abc123.puppet, certname_preserved=true
	// Used for reporting and auditing purposes.
	// Returns empty map if no installation attempt was made yet.
	GetInstallMetadata() map[string]string
}

// InstallOptions contains generic installation options.
// Used to pass common configurations between all installers.
type InstallOptions struct {
	// DryRun simulates installation without executing real commands
	DryRun bool

	// SkipValidation skips prerequisite validations (use with caution!)
	SkipValidation bool

	// SkipTagging doesn't apply tags after installation (useful for testing)
	SkipTagging bool

	// MaxConcurrency defines maximum number of simultaneous installations
	MaxConcurrency int

	// Timeout global timeout per installation
	Timeout time.Duration

	// CustomOptions package-specific options
	// E.g., for Puppet: {"server": "puppet.example.com", "environment": "production"}
	CustomOptions map[string]string
}

// InstallResult represents the result of an installation
type InstallResult struct {
	Instance  *cloud.Instance // Instance where it was installed
	Success   bool            // true if installation was successful
	Error     error           // Error if it failed
	Duration  time.Duration   // Time it took
	Output    string          // Output of executed commands
	StartTime time.Time       // When it started
	EndTime   time.Time       // When it finished
	Tagged    bool            // true if tags were applied successfully
}

// String returns readable representation of the result
func (ir *InstallResult) String() string {
	if ir.Success {
		return "SUCCESS: " + ir.Instance.ID + " (" + ir.Duration.String() + ")"
	}
	return "FAILED: " + ir.Instance.ID + " - " + ir.Error.Error()
}
