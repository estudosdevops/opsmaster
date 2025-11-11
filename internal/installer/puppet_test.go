package installer

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/estudosdevops/opsmaster/internal/cloud"
)

// ============================================================
// MOCKS - External dependency simulation
// ============================================================

// mockCloudProvider simulates a cloud provider (AWS/Azure/GCP) for testing.
// This allows testing the installer logic without requiring real infrastructure.
//
// 🎓 CONCEPT: A Mock is a "fake object" that simulates the behavior of a real object.
// We use mocks to:
// - Test without external dependencies (AWS, Azure, etc)
// - Control exactly what the dependency returns
// - Test error scenarios (e.g., SSM command fails)
type mockCloudProvider struct {
	// executeCommandFunc is a function we can configure for each test.
	// Allows simulating different responses (success, error, different OS, etc)
	executeCommandFunc func(ctx context.Context, instance *cloud.Instance, commands []string, timeout time.Duration) (*cloud.CommandResult, error)
}

// Name implements the cloud.CloudProvider interface
func (*mockCloudProvider) Name() string {
	return "mock"
}

// ExecuteCommand implements the cloud.CloudProvider interface
// Calls the configured function in the mock or returns error if not configured
//
// 🎓 CONCEPT: Interfaces in Go
// For a type to implement an interface, it must have EXACTLY
// the same methods with SAME signature (parameters and returns)
func (m *mockCloudProvider) ExecuteCommand(ctx context.Context, instance *cloud.Instance, commands []string, timeout time.Duration) (*cloud.CommandResult, error) {
	if m.executeCommandFunc != nil {
		return m.executeCommandFunc(ctx, instance, commands, timeout)
	}
	return nil, fmt.Errorf("executeCommandFunc not configured in mock")
}

// ValidateInstance implements the cloud.CloudProvider interface
// Simple mock that always returns success
func (*mockCloudProvider) ValidateInstance(context.Context, *cloud.Instance) error {
	return nil
}

// TestConnectivity implements the cloud.CloudProvider interface
// Simple mock that always returns success
func (*mockCloudProvider) TestConnectivity(context.Context, *cloud.Instance, string, int) error {
	return nil
}

// TagInstance implements the cloud.CloudProvider interface
// Simple mock that always returns success
func (*mockCloudProvider) TagInstance(context.Context, *cloud.Instance, map[string]string) error {
	return nil
}

// HasTag implements the cloud.CloudProvider interface
// Simple mock that always returns false
func (*mockCloudProvider) HasTag(context.Context, *cloud.Instance, string, string) (bool, error) {
	return false, nil
}

// ============================================================
// HELPER FUNCTIONS - Test utility functions
// ============================================================

// createTestInstance creates a standard test instance.
// Avoids code duplication in tests.
//
// 🎓 CONCEPT: Helper functions reduce duplicate code and make tests more readable.
func createTestInstance() *cloud.Instance {
	return &cloud.Instance{
		ID:      "i-1234567890abcdef0",
		Account: "123456789012",
		Region:  "us-east-1",
		Cloud:   "aws",
		Metadata: map[string]string{
			"environment": "production",
		},
	}
}

// createMockProviderWithOSResponse creates a mock that returns a specific OS.
// Helper to simplify mock creation in tests.
//
// 🎓 IMPORTANT CONCEPT: Smart Mock
// The mock needs to differentiate between DIFFERENT commands:
// - OS detection command: returns OS name
// - Certname reading command: returns error (file doesn't exist)
func createMockProviderWithOSResponse(osName string) *mockCloudProvider {
	return &mockCloudProvider{
		executeCommandFunc: func(_ context.Context, _ *cloud.Instance, commands []string, _ time.Duration) (*cloud.CommandResult, error) {
			// If it's an OS detection command (contains "os-release")
			if len(commands) > 0 && strings.Contains(commands[0], "os-release") {
				return &cloud.CommandResult{
					Stdout:   osName,
					ExitCode: 0,
				}, nil
			}

			// If it's a certname reading command (contains "puppet.conf")
			// Returns error because it's a first installation (file doesn't exist)
			if len(commands) > 0 && strings.Contains(commands[0], "puppet.conf") {
				return nil, fmt.Errorf("file does not exist")
			}

			// Other commands: return empty
			return &cloud.CommandResult{
				Stdout:   "",
				ExitCode: 0,
			}, nil
		},
	}
}

// createMockProviderWithCertnameResponse creates a mock that returns an existing certname.
// Simulates scenario where Puppet is already installed.
func createMockProviderWithCertnameResponse(certname string, hasError bool) *mockCloudProvider {
	return &mockCloudProvider{
		executeCommandFunc: func(_ context.Context, _ *cloud.Instance, commands []string, _ time.Duration) (*cloud.CommandResult, error) {
			// If it's OS detection command, return "ubuntu"
			if len(commands) > 0 && strings.Contains(commands[0], "os-release") {
				return &cloud.CommandResult{
					Stdout:   "ubuntu",
					ExitCode: 0,
				}, nil
			}

			// If it's certname reading command
			if len(commands) > 0 && strings.Contains(commands[0], "puppet.conf") {
				if hasError {
					return nil, fmt.Errorf("file not found")
				}
				return &cloud.CommandResult{
					Stdout:   certname,
					ExitCode: 0,
				}, nil
			}

			return &cloud.CommandResult{
				Stdout:   "",
				ExitCode: 0,
			}, nil
		},
	}
}

// ============================================================
// UTILITY FUNCTION TESTS
// ============================================================

// TestNormalizeOS tests the normalizeOS function that converts OS aliases to normalized types.
//
// 🎓 CONCEPT: Table-driven test
// - List of test cases in a struct (table)
// - Loop through cases with t.Run() to create subtests
// - Each case has: name, input, expected output, error expectation
func TestNormalizeOS(t *testing.T) {
	// Define test cases (TABLE)
	tests := []struct {
		name        string // Test name (appears in output)
		input       string // Input OS
		expected    string // Expected normalized type
		expectError bool   // Do we expect an error?
	}{
		// Debian family
		{
			name:        "debian lowercase",
			input:       "debian",
			expected:    OSTypeDebian,
			expectError: false,
		},
		{
			name:        "ubuntu lowercase",
			input:       "ubuntu",
			expected:    OSTypeDebian,
			expectError: false,
		},
		{
			name:        "Ubuntu capitalized",
			input:       "Ubuntu",
			expected:    OSTypeDebian,
			expectError: false,
		},
		{
			name:        "debian with spaces",
			input:       "  debian  ",
			expected:    OSTypeDebian,
			expectError: false,
		},

		// RHEL family
		{
			name:        "rhel lowercase",
			input:       "rhel",
			expected:    OSTypeRHEL,
			expectError: false,
		},
		{
			name:        "centos lowercase",
			input:       "centos",
			expected:    OSTypeRHEL,
			expectError: false,
		},
		{
			name:        "amzn (Amazon Linux)",
			input:       "amzn",
			expected:    OSTypeRHEL,
			expectError: false,
		},
		{
			name:        "amazon lowercase",
			input:       "amazon",
			expected:    OSTypeRHEL,
			expectError: false,
		},
		{
			name:        "amazonlinux lowercase",
			input:       "amazonlinux",
			expected:    OSTypeRHEL,
			expectError: false,
		},
		{
			name:        "rocky lowercase",
			input:       "rocky",
			expected:    OSTypeRHEL,
			expectError: false,
		},
		{
			name:        "almalinux lowercase",
			input:       "almalinux",
			expected:    OSTypeRHEL,
			expectError: false,
		},

		// Error cases
		{
			name:        "unsupported OS",
			input:       "windows",
			expected:    "",
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expected:    "",
			expectError: true,
		},
	}

	// Loop through all test cases (DRIVEN)
	for _, tt := range tests {
		// t.Run creates an isolated subtest for each case
		// Benefit: if one case fails, others continue executing
		t.Run(tt.name, func(t *testing.T) {
			// ACT: Execute function being tested
			got, err := normalizeOS(tt.input)

			// ASSERT: Verify error
			if tt.expectError {
				if err == nil {
					t.Errorf("normalizeOS(%q) expected error, got nil", tt.input)
				}
				return // Don't validate result if we expect error
			}

			// ASSERT: Verify that there was no error when unexpected
			if err != nil {
				t.Errorf("normalizeOS(%q) unexpected error: %v", tt.input, err)
				return
			}

			// ASSERT: Verificar resultado
			if got != tt.expected {
				t.Errorf("normalizeOS(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestGeneratePuppetCertname tests the generation of unique certnames.
//
// 🎓 CONCEPT: Uniqueness and format testing
// - Validate that each call generates a different value
// - Validate output format (regex, suffix, etc)
func TestGeneratePuppetCertname(t *testing.T) {
	t.Run("generates unique certnames", func(t *testing.T) {
		// Generate multiple certnames
		certnames := make(map[string]bool)
		iterations := 100

		for range iterations {
			certname := generatePuppetCertname()

			// Verify it's not empty
			if certname == "" {
				t.Fatal("generatePuppetCertname() returned empty string")
			}

			// Verify format: must end with .puppet
			if !strings.HasSuffix(certname, ".puppet") {
				t.Errorf("generatePuppetCertname() = %q, want suffix '.puppet'", certname)
			}

			// Verify uniqueness: must not have been generated before
			if certnames[certname] {
				t.Errorf("generatePuppetCertname() generated duplicate: %q", certname)
			}

			certnames[certname] = true
		}

		// Verify we generated exactly 'iterations' unique certnames
		if len(certnames) != iterations {
			t.Errorf("generated %d unique certnames, want %d", len(certnames), iterations)
		}
	})

	t.Run("format is correct", func(t *testing.T) {
		certname := generatePuppetCertname()

		// Must have format: <32 chars hex>.puppet = 39 chars total
		// UUID without dashes: 32 chars + ".puppet": 7 chars = 39 chars
		expectedLength := 32 + len(".puppet")
		if len(certname) != expectedLength {
			t.Errorf("generatePuppetCertname() length = %d, want %d", len(certname), expectedLength)
		}

		// Verify UUID part has no dashes
		uuidPart := strings.TrimSuffix(certname, ".puppet")
		if strings.Contains(uuidPart, "-") {
			t.Errorf("generatePuppetCertname() UUID part contains dashes: %q", uuidPart)
		}
	})
}

// ============================================================
// MAIN FUNCTION TESTS
// ============================================================

// TestGenerateInstallScriptWithAutoDetect_ReturnsMetadata tests that the function
// returns correct metadata (os, certname, certname_preserved).
//
// 🎓 CONCEPT: Testing with mocks
// - Create cloud provider mock
// - Configure expected response
// - Validate that function processes correctly
func TestGenerateInstallScriptWithAutoDetect_ReturnsMetadata(t *testing.T) {
	tests := []struct {
		name                      string
		osResponse                string
		expectedOS                string
		expectedCertnamePreserved string
	}{
		{
			name:                      "debian OS",
			osResponse:                "ubuntu",
			expectedOS:                "ubuntu",
			expectedCertnamePreserved: "false",
		},
		{
			name:                      "rhel OS",
			osResponse:                "amzn",
			expectedOS:                "amzn",
			expectedCertnamePreserved: "false",
		},
		{
			name:                      "centos OS",
			osResponse:                "centos",
			expectedOS:                "centos",
			expectedCertnamePreserved: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE: Prepare test
			ctx := context.Background()
			instance := createTestInstance()
			mockProvider := createMockProviderWithOSResponse(tt.osResponse)

			installer := NewPuppetInstaller(PuppetOptions{
				Server:      "puppet.example.com",
				Port:        8140,
				Version:     "7",
				Environment: "production",
			})

			// ACT: Execute function
			commands, metadata, err := installer.GenerateInstallScriptWithAutoDetect(
				ctx,
				instance,
				mockProvider,
				nil,
			)

			// ASSERT: Verify result
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify that commands were returned
			if len(commands) == 0 {
				t.Error("expected commands, got empty slice")
			}

			// Verify metadata - OS
			if metadata["os"] != tt.expectedOS {
				t.Errorf("metadata[os] = %q, want %q", metadata["os"], tt.expectedOS)
			}

			// Verify metadata - certname exists and is not empty
			certname := metadata["certname"]
			if certname == "" {
				t.Error("metadata[certname] is empty")
			}

			// Verify metadata - certname has correct format
			if !strings.HasSuffix(certname, ".puppet") {
				t.Errorf("metadata[certname] = %q, want suffix '.puppet'", certname)
			}

			// Verify metadata - certname_preserved
			if metadata["certname_preserved"] != tt.expectedCertnamePreserved {
				t.Errorf("metadata[certname_preserved] = %q, want %q",
					metadata["certname_preserved"], tt.expectedCertnamePreserved)
			}
		})
	}
}

// TestGenerateInstallScriptWithAutoDetect_PreservesCertname tests that existing
// certname is preserved on re-installations.
//
// 🎓 CONCEPT: Testing conditional behavior
// - Simulate initial installation (no certname)
// - Simulate re-installation (with existing certname)
// - Validate that behavior changes correctly
func TestGenerateInstallScriptWithAutoDetect_PreservesCertname(t *testing.T) {
	t.Run("preserves existing certname", func(t *testing.T) {
		// ARRANGE
		ctx := context.Background()
		instance := createTestInstance()
		existingCertname := "existing123.puppet"

		// Mock that returns existing certname
		mockProvider := createMockProviderWithCertnameResponse(existingCertname, false)

		installer := NewPuppetInstaller(PuppetOptions{
			Server: "puppet.example.com",
		})

		// ACT
		_, metadata, err := installer.GenerateInstallScriptWithAutoDetect(
			ctx,
			instance,
			mockProvider,
			nil,
		)

		// ASSERT
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify that certname was preserved
		if metadata["certname"] != existingCertname {
			t.Errorf("metadata[certname] = %q, want %q", metadata["certname"], existingCertname)
		}

		// Verify that preservation flag is true
		if metadata["certname_preserved"] != "true" {
			t.Errorf("metadata[certname_preserved] = %q, want %q", metadata["certname_preserved"], "true")
		}
	})

	t.Run("generates new certname when none exists", func(t *testing.T) {
		// ARRANGE
		ctx := context.Background()
		instance := createTestInstance()

		// Mock that returns error (certname doesn't exist)
		mockProvider := createMockProviderWithCertnameResponse("", true)

		installer := NewPuppetInstaller(PuppetOptions{
			Server: "puppet.example.com",
		})

		// ACT
		_, metadata, err := installer.GenerateInstallScriptWithAutoDetect(
			ctx,
			instance,
			mockProvider,
			nil,
		)

		// ASSERT
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify that certname was generated (not empty)
		if metadata["certname"] == "" {
			t.Error("metadata[certname] is empty, expected generated certname")
		}

		// Verify that preservation flag is false
		if metadata["certname_preserved"] != "false" {
			t.Errorf("metadata[certname_preserved] = %q, want %q", metadata["certname_preserved"], "false")
		}
	})
}

// TestGenerateInstallScriptWithAutoDetect_ErrorHandling tests error handling.
//
// 🎓 CONCEPT: Error testing
// - Validate that errors are propagated correctly
// - Validate useful error messages
func TestGenerateInstallScriptWithAutoDetect_ErrorHandling(t *testing.T) {
	t.Run("returns error when OS detection fails", func(t *testing.T) {
		// ARRANGE
		ctx := context.Background()
		instance := createTestInstance()

		// Mock que retorna erro
		mockProvider := &mockCloudProvider{
			executeCommandFunc: func(_ context.Context, _ *cloud.Instance, _ []string, _ time.Duration) (*cloud.CommandResult, error) {
				return nil, fmt.Errorf("SSM command failed")
			},
		}

		installer := NewPuppetInstaller(PuppetOptions{
			Server: "puppet.example.com",
		})

		// ACT
		_, _, err := installer.GenerateInstallScriptWithAutoDetect(
			ctx,
			instance,
			mockProvider,
			nil,
		)

		// ASSERT
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		// Verificar que mensagem de erro contém contexto útil
		if !strings.Contains(err.Error(), "failed to detect OS") {
			t.Errorf("error message = %q, want substring 'failed to detect OS'", err.Error())
		}
	})

	t.Run("returns error for unsupported OS", func(t *testing.T) {
		// ARRANGE
		ctx := context.Background()
		instance := createTestInstance()
		mockProvider := createMockProviderWithOSResponse("windows")

		installer := NewPuppetInstaller(PuppetOptions{
			Server: "puppet.example.com",
		})

		// ACT
		_, _, err := installer.GenerateInstallScriptWithAutoDetect(
			ctx,
			instance,
			mockProvider,
			nil,
		)

		// ASSERT
		if err == nil {
			t.Fatal("expected error for unsupported OS, got nil")
		}

		// Verificar que mensagem menciona OS não suportado
		if !strings.Contains(err.Error(), "unsupported OS") {
			t.Errorf("error message = %q, want substring 'unsupported OS'", err.Error())
		}
	})
}

// ============================================================
// CONCURRENCY TESTS (RACE CONDITION)
// ============================================================

// concurrentTestResult stores result from a single goroutine execution
type concurrentTestResult struct {
	commands []string
	metadata map[string]string
	err      error
}

// executeConcurrentInstalls runs GenerateInstallScriptWithAutoDetect in parallel
// 🎓 CONCEPT: WaitGroup coordinates goroutines
func executeConcurrentInstalls(ctx context.Context, t *testing.T, installer *PuppetInstaller, numGoroutines int) []concurrentTestResult {
	t.Helper()
	results := make([]concurrentTestResult, numGoroutines)
	var wg sync.WaitGroup

	for i := range numGoroutines {
		wg.Add(1)
		idx := i

		go func(index int) {
			defer wg.Done()

			instance := &cloud.Instance{
				ID:      fmt.Sprintf("i-instance%d", index),
				Account: "123456789012",
				Region:  "us-east-1",
				Cloud:   "aws",
				Metadata: map[string]string{
					"environment": "production",
				},
			}

			mockProvider := createMockProviderWithOSResponse("ubuntu")
			commands, metadata, err := installer.GenerateInstallScriptWithAutoDetect(ctx, instance, mockProvider, nil)

			results[index] = concurrentTestResult{
				commands: commands,
				metadata: metadata,
				err:      err,
			}
		}(idx)
	}

	wg.Wait()
	return results
}

// validateNoErrors checks that no goroutine encountered an error
func validateNoErrors(t *testing.T, results []concurrentTestResult) {
	t.Helper()
	for i, r := range results {
		if r.err != nil {
			t.Errorf("goroutine %d: unexpected error: %v", i, r.err)
		}
	}
}

// validateUniqueCertnames ensures each result has a unique certname
// 🎓 THIS IS THE MAIN VALIDATION - detects race conditions!
func validateUniqueCertnames(t *testing.T, results []concurrentTestResult, expectedCount int) {
	t.Helper()
	certnames := make(map[string]int)

	for i, r := range results {
		certname := r.metadata["certname"]

		if certname == "" {
			t.Errorf("goroutine %d: metadata[certname] is empty", i)
			continue
		}

		if previousIndex, exists := certnames[certname]; exists {
			t.Errorf("RACE CONDITION DETECTED: goroutine %d and %d have same certname: %q",
				i, previousIndex, certname)
		}

		certnames[certname] = i
	}

	if len(certnames) != expectedCount {
		t.Errorf("generated %d unique certnames, want %d (possible race condition)",
			len(certnames), expectedCount)
	}
}

// validateCertnameFormat checks that all certnames have correct .puppet suffix
func validateCertnameFormat(t *testing.T, results []concurrentTestResult) {
	t.Helper()
	for _, r := range results {
		certname := r.metadata["certname"]
		if certname != "" && !strings.HasSuffix(certname, ".puppet") {
			t.Errorf("certname %q does not have .puppet suffix", certname)
		}
	}
}

// TestGenerateInstallScriptWithAutoDetect_Concurrent_NoRaceCondition validates that
// multiple concurrent calls DO NOT have race condition.
//
// 🎓 CONCEPT: Race Condition
// A race condition occurs when:
// - Multiple goroutines access the same variable
// - At least one goroutine WRITES to the variable
// - There's no synchronization (mutex, channel, etc)
//
// PROBLEM WE HAD:
// - `pi.lastMetadata` was a shared variable
// - Multiple goroutines wrote to it simultaneously
// - Result: metadata from one instance overwrote another
//
// SOLUTION:
// - Return metadata as return value (local copy)
// - Each goroutine has its own copy
// - No sharing = no race condition
//
// THIS TEST VALIDATES THAT THE SOLUTION WORKS!
func TestGenerateInstallScriptWithAutoDetect_Concurrent_NoRaceCondition(t *testing.T) {
	t.Run("concurrent calls return unique metadata", func(t *testing.T) {
		// ARRANGE
		ctx := context.Background()
		installer := NewPuppetInstaller(PuppetOptions{
			Server: "puppet.example.com",
		})

		numGoroutines := 50
		results := executeConcurrentInstalls(ctx, t, installer, numGoroutines)

		// ASSERT
		validateNoErrors(t, results)
		validateUniqueCertnames(t, results, numGoroutines)
		validateCertnameFormat(t, results)
	})

	t.Run("concurrent calls with different OS return correct metadata", func(t *testing.T) {
		// ARRANGE
		ctx := context.Background()
		installer := NewPuppetInstaller(PuppetOptions{
			Server: "puppet.example.com",
		})

		// Test with different operating systems in parallel
		osTypes := []string{"ubuntu", "debian", "centos", "rhel", "amzn", "rocky"}
		numIterations := 10 // Each OS will be tested 10 times
		totalGoroutines := len(osTypes) * numIterations

		type result struct {
			osRequested string
			osReturned  string
			certname    string
			err         error
		}
		results := make([]result, 0, totalGoroutines)
		var resultsMutex sync.Mutex // Protects results slice (append is not thread-safe)

		var wg sync.WaitGroup

		// ACT: Execute with different OS in parallel
		for _, osType := range osTypes {
			for i := range numIterations {
				wg.Add(1)

				// Capture variables in local scope
				os := osType
				iteration := i

				go func() {
					defer wg.Done()

					instance := &cloud.Instance{
						ID:      fmt.Sprintf("i-%s-%d", os, iteration),
						Account: "123456789012",
						Region:  "us-east-1",
						Cloud:   "aws",
					}

					mockProvider := createMockProviderWithOSResponse(os)

					_, metadata, err := installer.GenerateInstallScriptWithAutoDetect(
						ctx,
						instance,
						mockProvider,
						nil,
					)

					// Store result in thread-safe manner
					resultsMutex.Lock()
					results = append(results, result{
						osRequested: os,
						osReturned:  metadata["os"],
						certname:    metadata["certname"],
						err:         err,
					})
					resultsMutex.Unlock()
				}()
			}
		}

		wg.Wait()

		// ASSERT

		// 1. Verify that all succeeded
		for i, r := range results {
			if r.err != nil {
				t.Errorf("result %d (OS=%s): unexpected error: %v", i, r.osRequested, r.err)
			}
		}

		// 2. Verify that returned OS matches requested
		for i, r := range results {
			if r.osReturned != r.osRequested {
				t.Errorf("result %d: metadata[os] = %q, want %q",
					i, r.osReturned, r.osRequested)
			}
		}

		// 3. Verify uniqueness of certnames (CRITICAL)
		certnames := make(map[string]bool)
		for i, r := range results {
			if certnames[r.certname] {
				t.Errorf("result %d: duplicate certname detected: %q", i, r.certname)
			}
			certnames[r.certname] = true
		}

		// 4. Verify that we have exactly totalGoroutines unique certnames
		if len(certnames) != totalGoroutines {
			t.Errorf("generated %d unique certnames, want %d",
				len(certnames), totalGoroutines)
		}
	})
}

// TestGenerateInstallScriptWithAutoDetect_RaceDetector validates using Go race detector.
//
// 🎓 CONCEPT: Race Detector
// Go has a BUILT-IN race condition detector!
// Run: go test -race
//
// How it works:
// - Instruments code at runtime
// - Detects concurrent accesses to same variable
// - Reports complete stack trace of conflict
//
// THIS TEST forces race conditions to validate that they DON'T exist.
func TestGenerateInstallScriptWithAutoDetect_RaceDetector(t *testing.T) {
	// ARRANGE
	ctx := context.Background()
	installer := NewPuppetInstaller(PuppetOptions{
		Server: "puppet.example.com",
	})

	// Use high number of goroutines to increase race probability
	numGoroutines := 100
	var wg sync.WaitGroup

	// ACT: Stress test with many goroutines
	for i := range numGoroutines {
		wg.Add(1)

		go func(idx int) {
			defer wg.Done()

			instance := &cloud.Instance{
				ID:      fmt.Sprintf("i-stress%d", idx),
				Account: "123456789012",
				Region:  "us-east-1",
				Cloud:   "aws",
			}

			mockProvider := createMockProviderWithOSResponse("ubuntu")

			// Execute multiple times in the same goroutine
			for j := range 5 {
				_, metadata, err := installer.GenerateInstallScriptWithAutoDetect(
					ctx,
					instance,
					mockProvider,
					nil,
				)

				// Basic verifications
				if err != nil {
					t.Errorf("goroutine %d, iteration %d: unexpected error: %v", idx, j, err)
				}
				if metadata["certname"] == "" {
					t.Errorf("goroutine %d, iteration %d: empty certname", idx, j)
				}
			}
		}(i)
	}

	wg.Wait()

	// ASSERT
	// If there's a race condition, `go test -race` will detect and report!
	// No need for explicit assertions here - the race detector does the work.
}

// ============================================================
// ELASTIC AGENT FIX TESTS
// ============================================================

// TestGenerateElasticPreventionScript tests the Elastic Agent prevention script generation.
//
// 🎓 CONCEPT: Testing bash script generation
// When functions generate shell scripts, we need to validate:
// - Script contains required logic components
// - Conditional statements are correct
// - Error messages are informative
// - Scripts follow bash best practices
func TestGenerateElasticPreventionScript(t *testing.T) {
	installer := NewPuppetInstaller(PuppetOptions{
		Server: "puppet.example.com",
		Port:   8140,
	})

	t.Run("script contains all required components", func(t *testing.T) {
		script := installer.generateElasticPreventionScript()

		// 🎓 CONCEPT: Component validation
		// For bash scripts, we validate that key components exist
		requiredComponents := []string{
			"Elastic Agent Prevention Handler", // Title/purpose (updated)
			"systemctl list-unit-files",        // Service detection
			"elastic-agent.service",            // Service name
			"/opt/elastic_flagfile",            // Flag file path
			"touch /opt/elastic_flagfile",      // File creation
			"chmod 644 /opt/elastic_flagfile",  // File permissions
			"creating preventively",            // Prevention messaging (new)
		}

		for _, component := range requiredComponents {
			if !strings.Contains(script, component) {
				t.Errorf("Script missing required component: %q", component)
			}
		}
	})

	t.Run("script has proper conditional logic", func(t *testing.T) {
		script := installer.generateElasticPreventionScript()

		// 🎓 CONCEPT: Logic validation
		// Validate that conditional statements follow expected patterns
		conditionals := []string{
			"if systemctl list-unit-files",      // Service check
			"if [ ! -f /opt/elastic_flagfile ]", // File missing check (prevention)
			"if touch /opt/elastic_flagfile",    // File creation check (prevention)
		}

		for _, conditional := range conditionals {
			if !strings.Contains(script, conditional) {
				t.Errorf("Script missing conditional logic: %q", conditional)
			}
		}
	})

	t.Run("script has informative messages", func(t *testing.T) {
		script := installer.generateElasticPreventionScript()

		// 🎓 CONCEPT: UX validation
		// Scripts should provide clear feedback to users about what's happening
		messages := []string{
			"Checking for Elastic Agent enrollment prevention",
			"Elastic Agent service detected",
			"Missing Elastic Agent flag file - creating preventively",
			"Elastic flag file created successfully",
			"Elastic Agent Prevention completed",
			"Elastic Agent service not found",
			"no prevention needed",
		}

		for _, message := range messages {
			if !strings.Contains(script, message) {
				t.Errorf("Script missing informative message: %q", message)
			}
		}
	})

	t.Run("script follows bash best practices", func(t *testing.T) {
		script := installer.generateElasticPreventionScript()

		// 🎓 CONCEPT: Bash best practices validation
		// Good bash scripts should follow certain patterns for reliability
		bestPractices := []string{
			"2>/dev/null", // Error redirection
			"chmod 644",   // Explicit permissions
			"else",        // Proper else branches
			"fi",          // Proper if closures
		}

		for _, practice := range bestPractices {
			if !strings.Contains(script, practice) {
				t.Errorf("Script missing bash best practice: %q", practice)
			}
		}
	})

	t.Run("script handles idempotent execution", func(t *testing.T) {
		script := installer.generateElasticPreventionScript()

		// 🎓 CONCEPT: Idempotency validation
		// Scripts should handle being run multiple times safely
		idempotencyChecks := []string{
			"if [ ! -f /opt/elastic_flagfile ]", // Check if file is missing (prevention)
			"flag file already exists",          // Message for existing file
			"no prevention needed",              // Skip when not needed
		}

		for _, check := range idempotencyChecks {
			if !strings.Contains(script, check) {
				t.Errorf("Script missing idempotency check: %q", check)
			}
		}
	})
}

// TestGenerateElasticPreventionScript_Integration tests that the elastic prevention is properly
// integrated into the main installation scripts.
//
// 🎓 CONCEPT: Integration testing
// Unlike unit tests that test isolated functions, integration tests verify
// that multiple components work together correctly.
func TestGenerateElasticPreventionScript_Integration(t *testing.T) {
	installer := NewPuppetInstaller(PuppetOptions{
		Server: "puppet.example.com",
		Port:   8140,
	})

	t.Run("debian script includes elastic prevention", func(t *testing.T) {
		// Generate Debian installation script
		scripts, err := installer.GenerateInstallScript("debian", map[string]string{})
		if err != nil {
			t.Fatalf("Failed to generate Debian script: %v", err)
		}

		if len(scripts) == 0 {
			t.Fatal("No scripts generated")
		}

		script := scripts[0]

		// 🎓 CONCEPT: Integration verification
		// Verify that the elastic prevention is included in the main script
		elasticPreventionMarkers := []string{
			"Elastic Agent Prevention Handler",
			"Checking for Elastic Agent enrollment prevention",
			"elastic-agent.service",
			"/opt/elastic_flagfile",
		}

		for _, marker := range elasticPreventionMarkers {
			if !strings.Contains(script, marker) {
				t.Errorf("Debian script missing elastic prevention marker: %q", marker)
			}
		}
	})

	t.Run("rhel script includes elastic prevention", func(t *testing.T) {
		// Generate RHEL installation script
		scripts, err := installer.GenerateInstallScript("rhel", map[string]string{})
		if err != nil {
			t.Fatalf("Failed to generate RHEL script: %v", err)
		}

		if len(scripts) == 0 {
			t.Fatal("No scripts generated")
		}

		script := scripts[0]

		// Verify that the elastic prevention is included
		elasticPreventionMarkers := []string{
			"Elastic Agent Prevention Handler",
			"Checking for Elastic Agent enrollment prevention",
			"elastic-agent.service",
			"/opt/elastic_flagfile",
		}

		for _, marker := range elasticPreventionMarkers {
			if !strings.Contains(script, marker) {
				t.Errorf("RHEL script missing elastic prevention marker: %q", marker)
			}
		}
	})

	t.Run("dry principle compliance", func(t *testing.T) {
		// 🎓 CONCEPT: DRY compliance testing
		// Verify that both scripts use the same elastic fix implementation
		debianScripts, err := installer.GenerateInstallScript("debian", map[string]string{})
		if err != nil {
			t.Fatalf("Failed to generate Debian script: %v", err)
		}

		rhelScripts, err := installer.GenerateInstallScript("rhel", map[string]string{})
		if err != nil {
			t.Fatalf("Failed to generate RHEL script: %v", err)
		}

		debianScript := debianScripts[0]
		rhelScript := rhelScripts[0]

		// Extract the elastic prevention section from both scripts
		debianPreventionStart := strings.Index(debianScript, "Elastic Agent Prevention Handler")
		rhelPreventionStart := strings.Index(rhelScript, "Elastic Agent Prevention Handler")

		if debianPreventionStart == -1 {
			t.Fatal("Debian script missing elastic prevention section")
		}
		if rhelPreventionStart == -1 {
			t.Fatal("RHEL script missing elastic prevention section")
		}

		debianPreventionEnd := strings.Index(debianScript[debianPreventionStart:], "============================================================")
		rhelPreventionEnd := strings.Index(rhelScript[rhelPreventionStart:], "============================================================")

		// Add offset to get absolute position
		debianPreventionEnd += debianPreventionStart
		rhelPreventionEnd += rhelPreventionStart

		debianPreventionSection := debianScript[debianPreventionStart:debianPreventionEnd]
		rhelPreventionSection := rhelScript[rhelPreventionStart:rhelPreventionEnd]

		// 🎓 CONCEPT: Exact match verification
		// The elastic prevention logic should be identical in both scripts (DRY principle)
		if debianPreventionSection != rhelPreventionSection {
			t.Error("Elastic prevention sections differ between Debian and RHEL scripts - DRY principle violated")
			t.Logf("Debian prevention section length: %d", len(debianPreventionSection))
			t.Logf("RHEL prevention section length: %d", len(rhelPreventionSection))
		}
	})
}
