package validator

import (
	"context"
	"fmt"
	"time"

	"github.com/estudosdevops/opsmaster/internal/cloud"
)

const (
	// defaultValidationTimeout is the default timeout for validation operations.
	// Used when no specific timeout is provided.
	defaultValidationTimeout = 10 * time.Second
)

// ValidationResult represents the result of a validation check.
// Contains success status and any error encountered.
type ValidationResult struct {
	Name    string // Name of validation (e.g., "ssm_connectivity", "puppet_server_reachable")
	Success bool   // Whether validation passed
	Error   error  // Error if validation failed
	Message string // Human-readable message
}

// Validator interface for reusable validation logic.
// Different validators can be composed together.
type Validator interface {
	// Validate runs the validation check
	Validate(ctx context.Context, instance *cloud.Instance, provider cloud.CloudProvider) *ValidationResult
}

// ConnectivityValidator validates network connectivity to a specific host:port.
// This is useful for checking if instance can reach external services
// (e.g., Puppet Server, Docker Registry, etc).
type ConnectivityValidator struct {
	Host    string        // Target hostname or IP
	Port    int           // Target port
	Timeout time.Duration // Connection timeout
	Name    string        // Validation name for reporting
}

// NewConnectivityValidator creates a new connectivity validator.
func NewConnectivityValidator(name, host string, port int, timeout time.Duration) *ConnectivityValidator {
	if timeout == 0 {
		timeout = 10 * time.Second // Default 10s timeout
	}

	return &ConnectivityValidator{
		Host:    host,
		Port:    port,
		Timeout: timeout,
		Name:    name,
	}
}

// Validate checks if instance can reach the target host:port.
func (cv *ConnectivityValidator) Validate(ctx context.Context, instance *cloud.Instance, provider cloud.CloudProvider) *ValidationResult {
	result := &ValidationResult{
		Name: cv.Name,
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, cv.Timeout)
	defer cancel()

	// Test connectivity using cloud provider
	err := provider.TestConnectivity(timeoutCtx, instance, cv.Host, cv.Port)
	if err != nil {
		result.Success = false
		result.Error = err
		result.Message = fmt.Sprintf("Cannot reach %s:%d - %v", cv.Host, cv.Port, err)
		return result
	}

	result.Success = true
	result.Message = fmt.Sprintf("Successfully connected to %s:%d", cv.Host, cv.Port)
	return result
}

// SSMValidator validates that instance is accessible via Systems Manager (SSM).
// This is AWS-specific but could be extended for Azure Run Command, etc.
type SSMValidator struct {
	Name    string        // Validation name
	Timeout time.Duration // Validation timeout
}

// NewSSMValidator creates a new SSM validator.
func NewSSMValidator(timeout time.Duration) *SSMValidator {
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &SSMValidator{
		Name:    "ssm_connectivity",
		Timeout: timeout,
	}
}

// Validate checks if instance is online and accessible via SSM.
func (sv *SSMValidator) Validate(ctx context.Context, instance *cloud.Instance, provider cloud.CloudProvider) *ValidationResult {
	result := &ValidationResult{
		Name: sv.Name,
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, sv.Timeout)
	defer cancel()

	// Validate instance using cloud provider
	err := provider.ValidateInstance(timeoutCtx, instance)
	if err != nil {
		result.Success = false
		result.Error = err
		result.Message = fmt.Sprintf("Instance not accessible via SSM: %v", err)
		return result
	}

	result.Success = true
	result.Message = "Instance is online and accessible via SSM"
	return result
}

// CompositeValidator runs multiple validators in sequence.
// Useful for running all prerequisite checks before installation.
type CompositeValidator struct {
	Validators []Validator
	StopOnFail bool // Stop on first failure if true
}

// NewCompositeValidator creates validator that runs multiple checks.
func NewCompositeValidator(validators []Validator, stopOnFail bool) *CompositeValidator {
	return &CompositeValidator{
		Validators: validators,
		StopOnFail: stopOnFail,
	}
}

// Validate runs all validators in sequence.
// Returns aggregated results from all validators.
func (cv *CompositeValidator) Validate(ctx context.Context, instance *cloud.Instance, provider cloud.CloudProvider) []*ValidationResult {
	var results []*ValidationResult

	for _, validator := range cv.Validators {
		// Check if context was canceled
		select {
		case <-ctx.Done():
			// Context canceled, stop validation
			results = append(results, &ValidationResult{
				Name:    "validation_canceled",
				Success: false,
				Error:   ctx.Err(),
				Message: "Validation canceled",
			})
			return results
		default:
		}

		// Run validator
		result := validator.Validate(ctx, instance, provider)
		results = append(results, result)

		// Stop on first failure if configured
		if cv.StopOnFail && !result.Success {
			break
		}
	}

	return results
}

// AllPassed checks if all validation results passed.
func AllPassed(results []*ValidationResult) bool {
	for _, result := range results {
		if !result.Success {
			return false
		}
	}
	return true
}

// GetFailedValidations returns only the failed validation results.
func GetFailedValidations(results []*ValidationResult) []*ValidationResult {
	var failed []*ValidationResult
	for _, result := range results {
		if !result.Success {
			failed = append(failed, result)
		}
	}
	return failed
}

// FormatValidationResults returns human-readable string with all results.
func FormatValidationResults(results []*ValidationResult) string {
	if len(results) == 0 {
		return "No validations run"
	}

	var output string
	for i, result := range results {
		status := "✓"
		if !result.Success {
			status = "✗"
		}
		output += fmt.Sprintf("%s %s: %s", status, result.Name, result.Message)
		if i < len(results)-1 {
			output += "\n"
		}
	}
	return output
}

// ValidatePuppetPrerequisites is a convenience function for Puppet installation.
// Validates SSM connectivity and Puppet Server reachability.
func ValidatePuppetPrerequisites(ctx context.Context, instance *cloud.Instance, provider cloud.CloudProvider, puppetServer string, puppetPort int) ([]*ValidationResult, error) {
	// Create validators
	validators := []Validator{
		NewSSMValidator(defaultValidationTimeout),
		NewConnectivityValidator("puppet_server_reachable", puppetServer, puppetPort, defaultValidationTimeout),
	}

	// Run all validations
	composite := NewCompositeValidator(validators, false) // Run all, don't stop on first failure
	results := composite.Validate(ctx, instance, provider)

	// Check if all passed
	if !AllPassed(results) {
		failed := GetFailedValidations(results)
		return results, fmt.Errorf("%d validation(s) failed", len(failed))
	}

	return results, nil
}
