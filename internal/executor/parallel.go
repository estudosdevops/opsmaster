package executor

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/estudosdevops/opsmaster/internal/cloud"
	"github.com/estudosdevops/opsmaster/internal/installer"
	"github.com/estudosdevops/opsmaster/internal/logger"
)

// ParallelExecutor executes package installations across multiple instances concurrently.
// Uses goroutines with semaphore pattern to limit concurrency and avoid overwhelming
// cloud APIs or network resources.
type ParallelExecutor struct {
	provider       cloud.CloudProvider
	installer      installer.PackageInstaller
	maxConcurrency int
	skipValidation bool
	skipTagging    bool
	dryRun         bool
	log            *slog.Logger
}

// ExecutorConfig contains configuration for the parallel executor.
type ExecutorConfig struct {
	Provider       cloud.CloudProvider        // Cloud provider (AWS, Azure, GCP)
	Installer      installer.PackageInstaller // Package installer (Puppet, Docker, etc)
	MaxConcurrency int                        // Max simultaneous installations (default: 10)
	SkipValidation bool                       // Skip prerequisite validations
	SkipTagging    bool                       // Skip tagging after installation
	DryRun         bool                       // Simulate without executing
}

// NewParallelExecutor creates a new parallel executor with given configuration.
func NewParallelExecutor(config ExecutorConfig) *ParallelExecutor {
	// Set default concurrency if not specified
	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = 10
	}

	return &ParallelExecutor{
		provider:       config.Provider,
		installer:      config.Installer,
		maxConcurrency: config.MaxConcurrency,
		skipValidation: config.SkipValidation,
		skipTagging:    config.SkipTagging,
		dryRun:         config.DryRun,
		log:            logger.Get(),
	}
}

// Execute processes multiple instances in parallel.
// Returns aggregated results with success/failure counts.
//
// Workflow:
// 1. Create semaphore channel to limit concurrency
// 2. Launch goroutine for each instance
// 3. Each goroutine: validate -> install -> verify -> tag
// 4. Collect all results
// 5. Return aggregated result
func (pe *ParallelExecutor) Execute(ctx context.Context, instances []*cloud.Instance) (*AggregatedResult, error) {
	if len(instances) == 0 {
		return nil, fmt.Errorf("no instances to process")
	}

	pe.log.Info("Starting parallel execution",
		"total_instances", len(instances),
		"max_concurrency", pe.maxConcurrency,
		"package", pe.installer.Name(),
		"cloud", pe.provider.Name())

	// Create aggregated result tracker
	aggResult := NewAggregatedResult()

	// Create semaphore channel to limit concurrency
	// Buffer size = max concurrent goroutines
	semaphore := make(chan struct{}, pe.maxConcurrency)

	// Create channel to collect results
	results := make(chan *ExecutionResult, len(instances))

	// WaitGroup to wait for all goroutines to complete
	var wg sync.WaitGroup

	// Launch goroutine for each instance
	for _, instance := range instances {
		wg.Add(1)

		go func(inst *cloud.Instance) {
			defer wg.Done()

			// Acquire semaphore (blocks if max concurrency reached)
			select {
			case semaphore <- struct{}{}:
				// Acquired semaphore, proceed
				defer func() { <-semaphore }() // Release semaphore when done
			case <-ctx.Done():
				// Context canceled while waiting for semaphore
				results <- &ExecutionResult{
					Instance:  inst,
					Status:    StatusCancelled,
					StartTime: time.Now(),
					EndTime:   time.Now(),
				}
				return
			}

			// Process instance
			result := pe.processInstance(ctx, inst)
			results <- result
		}(instance)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for result := range results {
		aggResult.Add(result)

		// Log progress
		pe.log.Info("Instance processed",
			"instance_id", result.Instance.ID,
			"status", result.Status,
			"duration", result.Duration,
			"progress", fmt.Sprintf("%d/%d", aggResult.Total, len(instances)))
	}

	// Finalize aggregated result
	aggResult.Finalize()

	pe.log.Info("Parallel execution completed",
		"total", aggResult.Total,
		"success", aggResult.Success,
		"failed", aggResult.Failed,
		"skipped", aggResult.Skipped,
		"total_time", aggResult.TotalTime)

	return aggResult, nil
}

// validateInstanceAndPrereqs validates instance accessibility and prerequisites.
// Returns error if validation fails, nil on success.
func (pe *ParallelExecutor) validateInstanceAndPrereqs(ctx context.Context, instance *cloud.Instance) error {
	// Validate instance accessibility
	pe.log.Debug("Validating instance", "instance_id", instance.ID)
	if err := pe.provider.ValidateInstance(ctx, instance); err != nil {
		pe.log.Error("Instance validation failed",
			"instance_id", instance.ID,
			"error", err)
		return fmt.Errorf("instance validation failed: %w", err)
	}

	// Validate prerequisites (unless skipped)
	if !pe.skipValidation {
		pe.log.Debug("Validating prerequisites", "instance_id", instance.ID)
		if err := pe.installer.ValidatePrerequisites(ctx, instance, pe.provider); err != nil {
			pe.log.Error("Prerequisite validation failed",
				"instance_id", instance.ID,
				"error", err)
			return fmt.Errorf("prerequisite validation failed: %w", err)
		}
	}

	return nil
}

// executeInstallation performs package installation or dry-run simulation.
// Returns error if installation fails, nil on success.
func (pe *ParallelExecutor) executeInstallation(ctx context.Context, instance *cloud.Instance) error {
	// Dry run mode - simulate installation
	if pe.dryRun {
		pe.log.Info("DRY RUN: Would install package",
			"instance_id", instance.ID,
			"package", pe.installer.Name())
		return nil
	}

	// Actual installation
	pe.log.Info("Installing package",
		"instance_id", instance.ID,
		"package", pe.installer.Name())

	if err := pe.installPackage(ctx, instance); err != nil {
		pe.log.Error("Installation failed",
			"instance_id", instance.ID,
			"error", err)
		return fmt.Errorf("installation failed: %w", err)
	}

	return nil
}

// verifyAndTag verifies installation and tags instance with success.
// Returns verification error if any, tagging errors are logged but not returned.
func (pe *ParallelExecutor) verifyAndTag(ctx context.Context, instance *cloud.Instance, result *ExecutionResult) error {
	// Verify installation
	pe.log.Debug("Verifying installation", "instance_id", instance.ID)
	if err := pe.installer.VerifyInstallation(ctx, instance, pe.provider); err != nil {
		pe.log.Error("Installation verification failed",
			"instance_id", instance.ID,
			"error", err)
		return fmt.Errorf("installation verification failed: %w", err)
	}

	// Tag instance with success (unless skipped)
	if !pe.skipTagging {
		pe.log.Debug("Tagging instance", "instance_id", instance.ID)
		tags := pe.installer.GetSuccessTags()
		if err := pe.provider.TagInstance(ctx, instance, tags); err != nil {
			// Log warning but don't fail the installation
			result.TaggingErr = err
			pe.log.Warn("Failed to tag instance, but installation succeeded",
				"instance_id", instance.ID,
				"error", err)
		}
	}

	return nil
}

// finalizeResult updates execution result with final status, timing and error.
// Automatically classifies error type based on current result state.
func (*ParallelExecutor) finalizeResult(result *ExecutionResult, status ExecutionStatus, err error) {
	result.Status = status
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Classify error type based on when it occurred
	if err != nil {
		// If we already have installation metadata, error happened during/after install
		if result.Metadata != nil || status == StatusSuccess {
			result.InstallationErr = err
		} else {
			// Error happened during validation
			result.ValidationErr = err
		}
	}
}

// processInstance processes a single instance through the complete workflow.
// Workflow: validate -> install -> verify -> tag
func (pe *ParallelExecutor) processInstance(ctx context.Context, instance *cloud.Instance) *ExecutionResult {
	result := &ExecutionResult{
		Instance:  instance,
		Status:    StatusRunning,
		StartTime: time.Now(),
	}

	pe.log.Info("Processing instance",
		"instance_id", instance.ID,
		"cloud", instance.Cloud,
		"account", instance.Account,
		"region", instance.Region)

	// Check if context already canceled
	select {
	case <-ctx.Done():
		pe.finalizeResult(result, StatusCancelled, ctx.Err())
		return result
	default:
	}

	// STEP 1-2: Validate instance and prerequisites
	if err := pe.validateInstanceAndPrereqs(ctx, instance); err != nil {
		pe.finalizeResult(result, StatusFailed, err)
		if !pe.skipTagging && !pe.dryRun {
			pe.tagFailure(ctx, instance, err)
		}
		return result
	}

	// STEP 3: Install package (or dry-run)
	if err := pe.executeInstallation(ctx, instance); err != nil {
		pe.finalizeResult(result, StatusFailed, err)
		result.Metadata = pe.installer.GetInstallMetadata()
		if !pe.skipTagging {
			pe.tagFailure(ctx, instance, err)
		}
		return result
	}

	// Handle dry-run success early
	if pe.dryRun {
		pe.finalizeResult(result, StatusSuccess, nil)
		return result
	}

	// STEP 4-5: Verify installation and tag success
	if err := pe.verifyAndTag(ctx, instance, result); err != nil {
		pe.finalizeResult(result, StatusFailed, err)
		if !pe.skipTagging {
			pe.tagFailure(ctx, instance, err)
		}
		return result
	}

	// STEP 6: Capture installation metadata and finalize
	result.Metadata = pe.installer.GetInstallMetadata()
	pe.finalizeResult(result, StatusSuccess, nil)

	pe.log.Info("Instance processed successfully",
		"instance_id", instance.ID,
		"duration", result.Duration)

	return result
}

// installPackage performs the actual package installation.
func (pe *ParallelExecutor) installPackage(ctx context.Context, instance *cloud.Instance) error {
	var commands []string
	var err error

	// Check if installer supports auto-detection (e.g., PuppetInstaller)
	// Use type assertion to access GenerateInstallScriptWithAutoDetect if available
	type autoDetectInstaller interface {
		GenerateInstallScriptWithAutoDetect(ctx context.Context, instance *cloud.Instance, provider cloud.CloudProvider, options map[string]string) ([]string, error)
	}

	if autoDetect, ok := pe.installer.(autoDetectInstaller); ok {
		if pe.dryRun {
			// DRY-RUN MODE: Skip OS detection (remote command execution)
			// Use metadata from CSV or fallback to default
			osType := instance.Metadata["os"]
			if osType == "" {
				osType = "ubuntu" // Default fallback
				pe.log.Warn("Dry-run: OS not specified in CSV, assuming Ubuntu",
					"instance_id", instance.ID,
					"tip", "Add 'os' column to CSV for accurate dry-run preview")
			} else {
				pe.log.Info("Dry-run: Using OS from CSV metadata",
					"instance_id", instance.ID,
					"os", osType)
			}

			// Generate script WITHOUT remote detection
			commands, err = pe.installer.GenerateInstallScript(osType, map[string]string{})
			if err != nil {
				return fmt.Errorf("failed to generate install script: %w", err)
			}

			pe.log.Info("Dry-run: Installation script generated",
				"instance_id", instance.ID,
				"os", osType,
				"script_lines", len(commands))
		} else {
			// REAL EXECUTION: Use auto-detection
			pe.log.Info("Detecting OS for installation", "instance_id", instance.ID)
			commands, err = autoDetect.GenerateInstallScriptWithAutoDetect(ctx, instance, pe.provider, map[string]string{})
			if err != nil {
				return fmt.Errorf("failed to generate install script with auto-detect: %w", err)
			}
		}
	} else {
		// Fallback: installer doesn't support auto-detection
		osType := instance.Metadata["os"]
		if osType == "" {
			osType = "ubuntu" // Default
		}

		// Generate installation script
		commands, err = pe.installer.GenerateInstallScript(osType, map[string]string{})
		if err != nil {
			return fmt.Errorf("failed to generate install script: %w", err)
		}
	}

	// DRY-RUN MODE: Skip actual execution
	if pe.dryRun {
		pe.log.Info("Dry-run: Skipping installation execution",
			"instance_id", instance.ID,
			"package", pe.installer.Name())
		return nil
	}

	// REAL EXECUTION: Execute installation commands
	pe.log.Info("Installing package",
		"instance_id", instance.ID,
		"package", pe.installer.Name())

	// Set generous timeout for installation (30 minutes)
	installTimeout := 30 * time.Minute
	result, err := pe.provider.ExecuteCommand(ctx, instance, commands, installTimeout)
	if err != nil {
		return fmt.Errorf("failed to execute install commands: %w", err)
	}

	// Check if command succeeded
	if result.ExitCode != 0 {
		return fmt.Errorf("installation script failed with exit code %d:\nstdout: %s\nstderr: %s",
			result.ExitCode, result.Stdout, result.Stderr)
	}

	return nil
}

// tagFailure applies failure tags to instance.
// Doesn't fail the operation if tagging fails - just logs warning.
func (pe *ParallelExecutor) tagFailure(ctx context.Context, instance *cloud.Instance, err error) {
	tags := pe.installer.GetFailureTags(err)
	if tagErr := pe.provider.TagInstance(ctx, instance, tags); tagErr != nil {
		pe.log.Warn("Failed to tag instance with failure status",
			"instance_id", instance.ID,
			"error", tagErr)
	}
}
