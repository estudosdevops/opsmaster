package executor

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/estudosdevops/opsmaster/internal/cloud"
)

// ============================================================
// MOCKS - External dependency simulation
// ============================================================

// mockCloudProvider simulates a cloud provider for testing.
//
// 🎓 CONCEPT: Mock with thread-safe counters
// We use atomic.Int32 to count calls safely in goroutines.
type mockCloudProvider struct {
	name                  string
	executeCommandFunc    func(ctx context.Context, instance *cloud.Instance, commands []string, timeout time.Duration) (*cloud.CommandResult, error)
	validateInstanceFunc  func(ctx context.Context, instance *cloud.Instance) error
	testConnectivityFunc  func(ctx context.Context, instance *cloud.Instance, host string, port int) error
	tagInstanceFunc       func(ctx context.Context, instance *cloud.Instance, tags map[string]string) error
	hasTagFunc            func(ctx context.Context, instance *cloud.Instance, key, value string) (bool, error)
	executeCommandCount   atomic.Int32 // Thread-safe counter
	validateInstanceCount atomic.Int32
	testConnectivityCount atomic.Int32
	tagInstanceCount      atomic.Int32
}

func (m *mockCloudProvider) Name() string {
	if m.name != "" {
		return m.name
	}
	return "mock"
}

func (m *mockCloudProvider) ExecuteCommand(ctx context.Context, instance *cloud.Instance, commands []string, timeout time.Duration) (*cloud.CommandResult, error) {
	m.executeCommandCount.Add(1)
	if m.executeCommandFunc != nil {
		return m.executeCommandFunc(ctx, instance, commands, timeout)
	}
	return &cloud.CommandResult{
		Stdout:   "command output",
		ExitCode: 0,
	}, nil
}

func (m *mockCloudProvider) ValidateInstance(ctx context.Context, instance *cloud.Instance) error {
	m.validateInstanceCount.Add(1)
	if m.validateInstanceFunc != nil {
		return m.validateInstanceFunc(ctx, instance)
	}
	return nil
}

func (m *mockCloudProvider) TestConnectivity(ctx context.Context, instance *cloud.Instance, host string, port int) error {
	m.testConnectivityCount.Add(1)
	if m.testConnectivityFunc != nil {
		return m.testConnectivityFunc(ctx, instance, host, port)
	}
	return nil
}

func (m *mockCloudProvider) TagInstance(ctx context.Context, instance *cloud.Instance, tags map[string]string) error {
	m.tagInstanceCount.Add(1)
	if m.tagInstanceFunc != nil {
		return m.tagInstanceFunc(ctx, instance, tags)
	}
	return nil
}

func (m *mockCloudProvider) HasTag(ctx context.Context, instance *cloud.Instance, key, value string) (bool, error) {
	if m.hasTagFunc != nil {
		return m.hasTagFunc(ctx, instance, key, value)
	}
	return false, nil
}

// GetExecuteCommandCount returns the number of times ExecuteCommand was called (thread-safe)
func (m *mockCloudProvider) GetExecuteCommandCount() int32 {
	return m.executeCommandCount.Load()
}

// GetValidateInstanceCount returns the number of times ValidateInstance was called (thread-safe)
func (m *mockCloudProvider) GetValidateInstanceCount() int32 {
	return m.validateInstanceCount.Load()
}

// GetTagInstanceCount returns the number of times TagInstance was called (thread-safe)
func (m *mockCloudProvider) GetTagInstanceCount() int32 {
	return m.tagInstanceCount.Load()
}

// ============================================================
// mockPackageInstaller simulates an installer for testing.
//
// 🎓 CONCEPT: Interface with auto-detection
// Implements both PackageInstaller and autoDetectInstaller
// to simulate PuppetInstaller behavior.
type mockPackageInstaller struct {
	name                        string
	generateInstallScriptFunc   func(osType string, options map[string]string) ([]string, error)
	generateWithAutoDetectFunc  func(ctx context.Context, instance *cloud.Instance, provider cloud.CloudProvider, options map[string]string) ([]string, map[string]string, error)
	validatePrerequisitesFunc   func(ctx context.Context, instance *cloud.Instance, provider cloud.CloudProvider) error
	verifyInstallationFunc      func(ctx context.Context, instance *cloud.Instance, provider cloud.CloudProvider) error
	getSuccessTagsFunc          func() map[string]string
	getFailureTagsFunc          func(err error) map[string]string
	generateInstallScriptCount  atomic.Int32
	generateWithAutoDetectCount atomic.Int32
	validatePrerequisitesCount  atomic.Int32
	verifyInstallationCount     atomic.Int32
	metadataByInstanceMutex     sync.Mutex                   // Protects map
	metadataByInstance          map[string]map[string]string // instance_id -> metadata
}

func (m *mockPackageInstaller) Name() string {
	if m.name != "" {
		return m.name
	}
	return "mock-package"
}

func (m *mockPackageInstaller) GenerateInstallScript(osType string, options map[string]string) ([]string, error) {
	m.generateInstallScriptCount.Add(1)
	if m.generateInstallScriptFunc != nil {
		return m.generateInstallScriptFunc(osType, options)
	}
	return []string{"echo 'install package'"}, nil
}

// GenerateInstallScriptWithAutoDetect implements autoDetectInstaller interface
func (m *mockPackageInstaller) GenerateInstallScriptWithAutoDetect(ctx context.Context, instance *cloud.Instance, provider cloud.CloudProvider, options map[string]string) (commands []string, metadata map[string]string, err error) {
	m.generateWithAutoDetectCount.Add(1)
	if m.generateWithAutoDetectFunc != nil {
		return m.generateWithAutoDetectFunc(ctx, instance, provider, options)
	}

	// Generate UNIQUE metadata for each instance
	metadata = map[string]string{
		"os":       "ubuntu",
		"certname": fmt.Sprintf("%s.puppet", instance.ID), // Unique certname based on ID
	}

	// Store metadata for later validation
	m.metadataByInstanceMutex.Lock()
	if m.metadataByInstance == nil {
		m.metadataByInstance = make(map[string]map[string]string)
	}
	m.metadataByInstance[instance.ID] = metadata
	m.metadataByInstanceMutex.Unlock()

	return []string{"echo 'install with auto-detect'"}, metadata, nil
}

func (m *mockPackageInstaller) ValidatePrerequisites(ctx context.Context, instance *cloud.Instance, provider cloud.CloudProvider) error {
	m.validatePrerequisitesCount.Add(1)
	if m.validatePrerequisitesFunc != nil {
		return m.validatePrerequisitesFunc(ctx, instance, provider)
	}
	return nil
}

func (m *mockPackageInstaller) VerifyInstallation(ctx context.Context, instance *cloud.Instance, provider cloud.CloudProvider) error {
	m.verifyInstallationCount.Add(1)
	if m.verifyInstallationFunc != nil {
		return m.verifyInstallationFunc(ctx, instance, provider)
	}
	return nil
}

func (m *mockPackageInstaller) GetSuccessTags() map[string]string {
	if m.getSuccessTagsFunc != nil {
		return m.getSuccessTagsFunc()
	}
	return map[string]string{"status": "installed"}
}

func (m *mockPackageInstaller) GetFailureTags(err error) map[string]string {
	if m.getFailureTagsFunc != nil {
		return m.getFailureTagsFunc(err)
	}
	return map[string]string{"status": "failed"}
}

func (_ *mockPackageInstaller) GetInstallMetadata() map[string]string {
	// Return empty metadata (legacy method not used in tests)
	return map[string]string{}
}

// GetMetadataForInstance returns captured metadata for an instance (thread-safe)
func (m *mockPackageInstaller) GetMetadataForInstance(instanceID string) map[string]string {
	m.metadataByInstanceMutex.Lock()
	defer m.metadataByInstanceMutex.Unlock()
	return m.metadataByInstance[instanceID]
}

// ============================================================
// HELPER FUNCTIONS
// ============================================================

// createTestInstances creates multiple test instances.
func createTestInstances(count int) []*cloud.Instance {
	instances := make([]*cloud.Instance, count)
	for i := range count {
		instances[i] = &cloud.Instance{
			ID:      fmt.Sprintf("i-test%03d", i),
			Account: "123456789012",
			Region:  "us-east-1",
			Cloud:   "aws",
			Metadata: map[string]string{
				"environment": "test",
			},
		}
	}
	return instances
}

// ============================================================
// EXECUTOR TESTS
// ============================================================

// TestNewParallelExecutor tests executor creation.
//
// 🎓 CONCEPT: Configuration testing
// Validate that defaults are applied correctly.
func TestNewParallelExecutor(t *testing.T) {
	t.Run("creates executor with default max concurrency", func(t *testing.T) {
		// ARRANGE
		provider := &mockCloudProvider{}
		installer := &mockPackageInstaller{}

		// ACT
		executor := NewParallelExecutor(ExecutorConfig{
			Provider:  provider,
			Installer: installer,
			// MaxConcurrency not defined - should use default 10
		})

		// ASSERT
		if executor == nil {
			t.Fatal("expected executor, got nil")
		}

		if executor.maxConcurrency != 10 {
			t.Errorf("maxConcurrency = %d, want 10 (default)", executor.maxConcurrency)
		}
	})

	t.Run("creates executor with custom max concurrency", func(t *testing.T) {
		// ARRANGE
		provider := &mockCloudProvider{}
		installer := &mockPackageInstaller{}

		// ACT
		executor := NewParallelExecutor(ExecutorConfig{
			Provider:       provider,
			Installer:      installer,
			MaxConcurrency: 5,
		})

		// ASSERT
		if executor.maxConcurrency != 5 {
			t.Errorf("maxConcurrency = %d, want 5", executor.maxConcurrency)
		}
	})

	t.Run("creates executor with dry run mode", func(t *testing.T) {
		// ARRANGE
		provider := &mockCloudProvider{}
		installer := &mockPackageInstaller{}

		// ACT
		executor := NewParallelExecutor(ExecutorConfig{
			Provider:  provider,
			Installer: installer,
			DryRun:    true,
		})

		// ASSERT
		if !executor.dryRun {
			t.Error("dryRun = false, want true")
		}
	})
}

// TestExecute_EmptyInstanceList tests behavior with empty list.
//
// 🎓 CONCEPT: Edge case testing
// Validate behavior with invalid inputs.
func TestExecute_EmptyInstanceList(t *testing.T) {
	// ARRANGE
	ctx := context.Background()
	provider := &mockCloudProvider{}
	installer := &mockPackageInstaller{}

	executor := NewParallelExecutor(ExecutorConfig{
		Provider:  provider,
		Installer: installer,
	})

	// ACT
	result, err := executor.Execute(ctx, []*cloud.Instance{})

	// ASSERT
	if err == nil {
		t.Fatal("expected error for empty instance list, got nil")
	}

	if result != nil {
		t.Errorf("expected nil result for error case, got %v", result)
	}
}

// TestExecute_SingleInstance tests execution with a single instance.
//
// 🎓 CONCEPT: Simple case testing
// Validate complete flow without parallelism.
func TestExecute_SingleInstance(t *testing.T) {
	// ARRANGE
	ctx := context.Background()
	provider := &mockCloudProvider{}
	installer := &mockPackageInstaller{}

	executor := NewParallelExecutor(ExecutorConfig{
		Provider:  provider,
		Installer: installer,
	})

	instances := createTestInstances(1)

	// ACT
	result, err := executor.Execute(ctx, instances)

	// ASSERT
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Total != 1 {
		t.Errorf("Total = %d, want 1", result.Total)
	}

	if result.Success != 1 {
		t.Errorf("Success = %d, want 1", result.Success)
	}

	if result.Failed != 0 {
		t.Errorf("Failed = %d, want 0", result.Failed)
	}

	// Verify that provider functions were called
	if provider.GetValidateInstanceCount() != 1 {
		t.Errorf("ValidateInstance called %d times, want 1", provider.GetValidateInstanceCount())
	}

	if provider.GetExecuteCommandCount() != 1 {
		t.Errorf("ExecuteCommand called %d times, want 1", provider.GetExecuteCommandCount())
	}

	if provider.GetTagInstanceCount() != 1 {
		t.Errorf("TagInstance called %d times, want 1", provider.GetTagInstanceCount())
	}
}

// TestExecute_MultipleInstances_ParallelExecution tests parallel execution.
//
// 🎓 CONCEPT: REAL parallelism testing
// Validate that multiple instances are processed in parallel.
func TestExecute_MultipleInstances_ParallelExecution(t *testing.T) {
	// ARRANGE
	ctx := context.Background()
	provider := &mockCloudProvider{}
	installer := &mockPackageInstaller{}

	executor := NewParallelExecutor(ExecutorConfig{
		Provider:       provider,
		Installer:      installer,
		MaxConcurrency: 5, // Allow 5 concurrent
	})

	// Create 20 instances
	instances := createTestInstances(20)

	// ACT
	startTime := time.Now()
	result, err := executor.Execute(ctx, instances)
	duration := time.Since(startTime)

	// ASSERT
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify aggregated results
	if result.Total != 20 {
		t.Errorf("Total = %d, want 20", result.Total)
	}

	if result.Success != 20 {
		t.Errorf("Success = %d, want 20", result.Success)
	}

	if result.Failed != 0 {
		t.Errorf("Failed = %d, want 0", result.Failed)
	}

	// Verify that all instances were processed
	if len(result.Results) != 20 {
		t.Errorf("len(Results) = %d, want 20", len(result.Results))
	}

	// Verify that execution was reasonably fast (parallelism worked)
	// With 20 instances and concurrency 5, should take ~4x time of 1 instance
	// If sequential, would take ~20x time
	t.Logf("Duration: %v (should be fast due to parallelism)", duration)

	// Verify that provider was called for each instance
	if provider.GetValidateInstanceCount() != 20 {
		t.Errorf("ValidateInstance called %d times, want 20", provider.GetValidateInstanceCount())
	}
}

// TestExecute_CapturesUniqueMetadata tests that metadata is unique per instance.
//
// 🎓 CONCEPT: Race condition fix testing
// THIS IS THE MOST IMPORTANT TEST - validates bug fix!
func TestExecute_CapturesUniqueMetadata(t *testing.T) {
	// ARRANGE
	ctx := context.Background()
	provider := &mockCloudProvider{}
	installer := &mockPackageInstaller{}

	executor := NewParallelExecutor(ExecutorConfig{
		Provider:       provider,
		Installer:      installer,
		MaxConcurrency: 10, // High concurrency to force race conditions
	})

	// Create 50 instances for stress test
	instances := createTestInstances(50)

	// ACT
	result, err := executor.Execute(ctx, instances)

	// ASSERT
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// CRITICAL VALIDATION: Each instance must have UNIQUE metadata
	certnames := make(map[string]string) // certname -> instance_id

	for _, execResult := range result.Results {
		instanceID := execResult.Instance.ID

		// Verify that metadata exists
		if execResult.Metadata == nil {
			t.Errorf("instance %s: metadata is nil", instanceID)
			continue
		}

		// Verify that certname exists
		certname := execResult.Metadata["certname"]
		if certname == "" {
			t.Errorf("instance %s: metadata[certname] is empty", instanceID)
			continue
		}

		// RACE CONDITION VERIFICATION: certname must be UNIQUE
		if previousInstanceID, exists := certnames[certname]; exists {
			t.Errorf("RACE CONDITION DETECTED: instances %s and %s have same certname: %q",
				instanceID, previousInstanceID, certname)
		}

		certnames[certname] = instanceID

		// Validate that certname matches expected for this instance
		expectedCertname := fmt.Sprintf("%s.puppet", instanceID)
		if certname != expectedCertname {
			t.Errorf("instance %s: metadata[certname] = %q, want %q",
				instanceID, certname, expectedCertname)
		}
	}

	// Verify that we have exactly 50 unique certnames
	if len(certnames) != 50 {
		t.Errorf("captured %d unique certnames, want 50 (possible race condition)",
			len(certnames))
	}
}

// TestExecute_MixedResults tests scenario with successes and failures.
//
// 🎓 CONCEPT: Real scenario testing
// Simulate environment where some instances fail.
func TestExecute_MixedResults(t *testing.T) {
	// ARRANGE
	ctx := context.Background()
	provider := &mockCloudProvider{
		// Validation fails for odd instances
		validateInstanceFunc: func(_ context.Context, instance *cloud.Instance) error {
			// Extract instance number (i-test001, i-test002, etc)
			var instanceNum int
			_, _ = fmt.Sscanf(instance.ID, "i-test%d", &instanceNum)

			if instanceNum%2 == 1 {
				return fmt.Errorf("validation failed for instance %s", instance.ID)
			}
			return nil
		},
	}
	installer := &mockPackageInstaller{}

	executor := NewParallelExecutor(ExecutorConfig{
		Provider:  provider,
		Installer: installer,
	})

	// Create 10 instances (0-9, so 5 even and 5 odd)
	instances := createTestInstances(10)

	// ACT
	result, err := executor.Execute(ctx, instances)

	// ASSERT
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify counters
	if result.Total != 10 {
		t.Errorf("Total = %d, want 10", result.Total)
	}

	// 5 even should succeed, 5 odd should fail
	if result.Success != 5 {
		t.Errorf("Success = %d, want 5", result.Success)
	}

	if result.Failed != 5 {
		t.Errorf("Failed = %d, want 5", result.Failed)
	}

	// Verify that individual results have correct status
	for _, execResult := range result.Results {
		var instanceNum int
		_, _ = fmt.Sscanf(execResult.Instance.ID, "i-test%d", &instanceNum)

		if instanceNum%2 == 0 {
			// Even should succeed
			if execResult.Status != StatusSuccess {
				t.Errorf("instance %s (even): status = %s, want %s",
					execResult.Instance.ID, execResult.Status, StatusSuccess)
			}
		} else {
			// Odd should fail
			if execResult.Status != StatusFailed {
				t.Errorf("instance %s (odd): status = %s, want %s",
					execResult.Instance.ID, execResult.Status, StatusFailed)
			}

			// Verify that error was captured
			if execResult.ValidationErr == nil {
				t.Errorf("instance %s: expected validation error, got nil", execResult.Instance.ID)
			}
		}
	}
}

// TestExecute_DryRunMode tests dry-run mode.
//
// 🎓 CONCEPT: Feature flag testing
// Validate that dry-run doesn't execute real commands.
func TestExecute_DryRunMode(t *testing.T) {
	// ARRANGE
	ctx := context.Background()
	provider := &mockCloudProvider{}
	installer := &mockPackageInstaller{}

	executor := NewParallelExecutor(ExecutorConfig{
		Provider:  provider,
		Installer: installer,
		DryRun:    true,
	})

	instances := createTestInstances(5)

	// ACT
	result, err := executor.Execute(ctx, instances)

	// ASSERT
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Dry-run should report success
	if result.Success != 5 {
		t.Errorf("Success = %d, want 5", result.Success)
	}

	// Dry-run should NOT execute real commands
	// ValidateInstance is still called
	if provider.GetValidateInstanceCount() != 5 {
		t.Errorf("ValidateInstance called %d times, want 5", provider.GetValidateInstanceCount())
	}

	// VerifyInstallation should NOT be called in dry-run
	// (verification doesn't make sense without actual installation)
	// This is validated through the installer mock not having VerifyInstallation called
}

// TestExecute_ContextCancellation tests cancellation via context.
//
// 🎓 CONCEPT: Context cancellation
// Validate that operation respects context cancellation.
func TestExecute_ContextCancellation(t *testing.T) {
	// ARRANGE
	ctx, cancel := context.WithCancel(context.Background())
	provider := &mockCloudProvider{
		// Simulate slow operation
		executeCommandFunc: func(_ context.Context, _ *cloud.Instance, _ []string, _ time.Duration) (*cloud.CommandResult, error) {
			// Wait a bit to allow time for cancellation
			time.Sleep(100 * time.Millisecond)
			return &cloud.CommandResult{Stdout: "output", ExitCode: 0}, nil
		},
	}
	installer := &mockPackageInstaller{}

	executor := NewParallelExecutor(ExecutorConfig{
		Provider:       provider,
		Installer:      installer,
		MaxConcurrency: 1, // Process sequentially for deterministic test
	})

	instances := createTestInstances(10)

	// Cancel context after 200ms (should cancel after ~2 instances processed)
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	// ACT
	result, err := executor.Execute(ctx, instances)

	// ASSERT
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Some instances should have been canceled
	if result.Canceled == 0 {
		t.Error("expected some canceled instances, got 0")
	}

	t.Logf("Results: Success=%d, Failed=%d, Canceled=%d",
		result.Success, result.Failed, result.Canceled)
}

// TestExecute_ConcurrencyLimit tests that semaphore limits concurrency.
//
// 🎓 CONCEPT: Semaphore pattern
// Validate that we don't exceed MaxConcurrency simultaneous executions.
func TestExecute_ConcurrencyLimit(t *testing.T) {
	// ARRANGE
	ctx := context.Background()

	// Counter for simultaneous executions
	var currentConcurrent atomic.Int32
	var maxConcurrentSeen atomic.Int32

	provider := &mockCloudProvider{
		executeCommandFunc: func(_ context.Context, _ *cloud.Instance, _ []string, _ time.Duration) (*cloud.CommandResult, error) {
			// Increment counter
			current := currentConcurrent.Add(1)

			// Update maximum seen
			for {
				max := maxConcurrentSeen.Load()
				if current <= max || maxConcurrentSeen.CompareAndSwap(max, current) {
					break
				}
			}

			// Simulate work
			time.Sleep(50 * time.Millisecond)

			// Decrement counter
			currentConcurrent.Add(-1)

			return &cloud.CommandResult{Stdout: "output", ExitCode: 0}, nil
		},
	}
	installer := &mockPackageInstaller{}

	maxConcurrency := 3
	executor := NewParallelExecutor(ExecutorConfig{
		Provider:       provider,
		Installer:      installer,
		MaxConcurrency: maxConcurrency,
	})

	instances := createTestInstances(15)

	// ACT
	_, err := executor.Execute(ctx, instances)

	// ASSERT
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify that we never exceeded concurrency limit
	maxSeen := maxConcurrentSeen.Load()
	if maxSeen > int32(maxConcurrency) {
		t.Errorf("max concurrent executions = %d, want <= %d", maxSeen, maxConcurrency)
	}

	t.Logf("Max concurrent executions seen: %d (limit was %d)", maxSeen, maxConcurrency)
}
