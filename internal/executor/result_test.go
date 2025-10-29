package executor

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/estudosdevops/opsmaster/internal/cloud"
)

// ============================================================
// HELPER FUNCTIONS - Test utilities
// ============================================================

// createTestInstance creates a standard test instance
func createTestInstance(id string) *cloud.Instance {
	return &cloud.Instance{
		ID:      id,
		Account: "123456789012",
		Region:  "us-east-1",
		Cloud:   "aws",
		Metadata: map[string]string{
			"environment": "production",
		},
	}
}

// createTestExecutionResult creates a test execution result
func createTestExecutionResult(status ExecutionStatus, instanceID string) *ExecutionResult {
	return &ExecutionResult{
		Instance:  createTestInstance(instanceID),
		Status:    status,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(5 * time.Second),
		Duration:  5 * time.Second,
	}
}

// ============================================================
// EXECUTION STATUS TESTS
// ============================================================

// TestExecutionStatus_String tests string representation of execution statuses.
//
// 🎓 CONCEPT: Enum testing
// - Validate that enum values map to correct strings
// - Test all possible enum values
// - Include unknown value handling
func TestExecutionStatus_String(t *testing.T) {
	tests := []struct {
		status   ExecutionStatus
		expected string
	}{
		{StatusPending, "PENDING"},
		{StatusRunning, "RUNNING"},
		{StatusSuccess, "SUCCESS"},
		{StatusFailed, "FAILED"},
		{StatusCancelled, "CANCELED"},
		{StatusSkipped, "SKIPPED"},
		{ExecutionStatus(999), "UNKNOWN"}, // Invalid status
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.status.String()
			if result != tt.expected {
				t.Errorf("String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// ============================================================
// EXECUTION RESULT TESTS
// ============================================================

// TestExecutionResult_Success tests the Success() method.
func TestExecutionResult_Success(t *testing.T) {
	tests := []struct {
		name     string
		status   ExecutionStatus
		expected bool
	}{
		{"success status", StatusSuccess, true},
		{"failed status", StatusFailed, false},
		{"pending status", StatusPending, false},
		{"running status", StatusRunning, false},
		{"canceled status", StatusCancelled, false},
		{"skipped status", StatusSkipped, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ExecutionResult{Status: tt.status}
			if result.Success() != tt.expected {
				t.Errorf("Success() = %v, want %v", result.Success(), tt.expected)
			}
		})
	}
}

// TestExecutionResult_Failed tests the Failed() method.
func TestExecutionResult_Failed(t *testing.T) {
	tests := []struct {
		name     string
		status   ExecutionStatus
		expected bool
	}{
		{"failed status", StatusFailed, true},
		{"success status", StatusSuccess, false},
		{"pending status", StatusPending, false},
		{"running status", StatusRunning, false},
		{"canceled status", StatusCancelled, false},
		{"skipped status", StatusSkipped, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ExecutionResult{Status: tt.status}
			if result.Failed() != tt.expected {
				t.Errorf("Failed() = %v, want %v", result.Failed(), tt.expected)
			}
		})
	}
}

// TestExecutionResult_GetError tests error priority logic.
//
// 🎓 CONCEPT: Error priority testing
// - Validate that errors are returned in correct order
// - ValidationErr > InstallationErr > TaggingErr
func TestExecutionResult_GetError(t *testing.T) {
	tests := []struct {
		name           string
		validationErr  error
		installErr     error
		taggingErr     error
		expectedErr    error
		expectedErrMsg string
	}{
		{
			name:           "no errors",
			validationErr:  nil,
			installErr:     nil,
			taggingErr:     nil,
			expectedErr:    nil,
			expectedErrMsg: "",
		},
		{
			name:           "only validation error",
			validationErr:  errors.New("validation failed"),
			installErr:     nil,
			taggingErr:     nil,
			expectedErrMsg: "validation failed",
		},
		{
			name:           "only installation error",
			validationErr:  nil,
			installErr:     errors.New("install failed"),
			taggingErr:     nil,
			expectedErrMsg: "install failed",
		},
		{
			name:           "only tagging error",
			validationErr:  nil,
			installErr:     nil,
			taggingErr:     errors.New("tagging failed"),
			expectedErrMsg: "tagging failed",
		},
		{
			name:           "validation error takes priority over installation",
			validationErr:  errors.New("validation failed"),
			installErr:     errors.New("install failed"),
			taggingErr:     nil,
			expectedErrMsg: "validation failed",
		},
		{
			name:           "validation error takes priority over all",
			validationErr:  errors.New("validation failed"),
			installErr:     errors.New("install failed"),
			taggingErr:     errors.New("tagging failed"),
			expectedErrMsg: "validation failed",
		},
		{
			name:           "installation error takes priority over tagging",
			validationErr:  nil,
			installErr:     errors.New("install failed"),
			taggingErr:     errors.New("tagging failed"),
			expectedErrMsg: "install failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ExecutionResult{
				ValidationErr:   tt.validationErr,
				InstallationErr: tt.installErr,
				TaggingErr:      tt.taggingErr,
			}

			err := result.GetError()

			if tt.expectedErrMsg == "" {
				if err != nil {
					t.Errorf("GetError() = %v, want nil", err)
				}
			} else {
				if err == nil {
					t.Errorf("GetError() = nil, want error containing %q", tt.expectedErrMsg)
				} else if err.Error() != tt.expectedErrMsg {
					t.Errorf("GetError() = %q, want %q", err.Error(), tt.expectedErrMsg)
				}
			}
		})
	}
}

// TestExecutionResult_String tests string representation.
func TestExecutionResult_String(t *testing.T) {
	result := &ExecutionResult{
		Instance: createTestInstance("i-123abc"),
		Status:   StatusSuccess,
		Duration: 5 * time.Second,
	}

	str := result.String()

	// Should contain status, instance ID, and duration
	if !contains(str, "SUCCESS") {
		t.Errorf("String() = %q, want to contain 'SUCCESS'", str)
	}
	if !contains(str, "i-123abc") {
		t.Errorf("String() = %q, want to contain 'i-123abc'", str)
	}
	if !contains(str, "5s") {
		t.Errorf("String() = %q, want to contain '5s'", str)
	}
}

// ============================================================
// AGGREGATED RESULT TESTS
// ============================================================

// TestNewAggregatedResult tests creation of aggregated result.
func TestNewAggregatedResult(t *testing.T) {
	// ACT
	ar := NewAggregatedResult()

	// ASSERT
	if ar == nil {
		t.Fatal("NewAggregatedResult() returned nil")
	}

	if ar.Total != 0 {
		t.Errorf("Total = %d, want 0", ar.Total)
	}

	if ar.Success != 0 {
		t.Errorf("Success = %d, want 0", ar.Success)
	}

	if ar.Results == nil {
		t.Error("Results slice is nil, want empty slice")
	}

	if len(ar.Results) != 0 {
		t.Errorf("len(Results) = %d, want 0", len(ar.Results))
	}

	if ar.StartTime.IsZero() {
		t.Error("StartTime is zero, want current time")
	}
}

// TestAggregatedResult_Add tests adding individual results.
//
// 🎓 CONCEPT: Aggregation testing
// - Test accumulation of different statuses
// - Verify counters are correctly incremented
func TestAggregatedResult_Add(t *testing.T) {
	// ARRANGE
	ar := NewAggregatedResult()

	// ACT - Add results with different statuses
	ar.Add(createTestExecutionResult(StatusSuccess, "i-001"))
	ar.Add(createTestExecutionResult(StatusSuccess, "i-002"))
	ar.Add(createTestExecutionResult(StatusFailed, "i-003"))
	ar.Add(createTestExecutionResult(StatusSkipped, "i-004"))
	ar.Add(createTestExecutionResult(StatusCancelled, "i-005"))

	// ASSERT
	if ar.Total != 5 {
		t.Errorf("Total = %d, want 5", ar.Total)
	}

	if ar.Success != 2 {
		t.Errorf("Success = %d, want 2", ar.Success)
	}

	if ar.Failed != 1 {
		t.Errorf("Failed = %d, want 1", ar.Failed)
	}

	if ar.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", ar.Skipped)
	}

	if ar.Canceled != 1 {
		t.Errorf("Canceled = %d, want 1", ar.Canceled)
	}

	if len(ar.Results) != 5 {
		t.Errorf("len(Results) = %d, want 5", len(ar.Results))
	}
}

// TestAggregatedResult_Finalize tests finalization logic.
func TestAggregatedResult_Finalize(t *testing.T) {
	// ARRANGE
	ar := NewAggregatedResult()
	startTime := ar.StartTime

	// Add small delay to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// ACT
	ar.Finalize()

	// ASSERT
	if ar.EndTime.IsZero() {
		t.Error("EndTime is zero after Finalize()")
	}

	if ar.EndTime.Before(startTime) {
		t.Error("EndTime is before StartTime")
	}

	if ar.TotalTime == 0 {
		t.Error("TotalTime is zero after Finalize()")
	}

	if ar.TotalTime < 10*time.Millisecond {
		t.Errorf("TotalTime = %v, want >= 10ms", ar.TotalTime)
	}
}

// TestAggregatedResult_SuccessRate tests success rate calculation.
//
// 🎓 CONCEPT: Percentage calculation testing
// - Test with different success/failure ratios
// - Test edge case (zero total)
func TestAggregatedResult_SuccessRate(t *testing.T) {
	tests := []struct {
		name         string
		success      int
		failed       int
		expectedRate float64
	}{
		{
			name:         "100% success",
			success:      10,
			failed:       0,
			expectedRate: 100.0,
		},
		{
			name:         "0% success",
			success:      0,
			failed:       10,
			expectedRate: 0.0,
		},
		{
			name:         "50% success",
			success:      5,
			failed:       5,
			expectedRate: 50.0,
		},
		{
			name:         "75% success",
			success:      3,
			failed:       1,
			expectedRate: 75.0,
		},
		{
			name:         "no executions",
			success:      0,
			failed:       0,
			expectedRate: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			ar := NewAggregatedResult()
			ar.Total = tt.success + tt.failed
			ar.Success = tt.success
			ar.Failed = tt.failed

			// ACT
			rate := ar.SuccessRate()

			// ASSERT
			if rate != tt.expectedRate {
				t.Errorf("SuccessRate() = %.1f, want %.1f", rate, tt.expectedRate)
			}
		})
	}
}

// TestAggregatedResult_FailureRate tests failure rate calculation.
func TestAggregatedResult_FailureRate(t *testing.T) {
	tests := []struct {
		name         string
		success      int
		failed       int
		expectedRate float64
	}{
		{
			name:         "100% failure",
			success:      0,
			failed:       10,
			expectedRate: 100.0,
		},
		{
			name:         "0% failure",
			success:      10,
			failed:       0,
			expectedRate: 0.0,
		},
		{
			name:         "50% failure",
			success:      5,
			failed:       5,
			expectedRate: 50.0,
		},
		{
			name:         "25% failure",
			success:      3,
			failed:       1,
			expectedRate: 25.0,
		},
		{
			name:         "no executions",
			success:      0,
			failed:       0,
			expectedRate: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			ar := NewAggregatedResult()
			ar.Total = tt.success + tt.failed
			ar.Success = tt.success
			ar.Failed = tt.failed

			// ACT
			rate := ar.FailureRate()

			// ASSERT
			if rate != tt.expectedRate {
				t.Errorf("FailureRate() = %.1f, want %.1f", rate, tt.expectedRate)
			}
		})
	}
}

// TestAggregatedResult_GetFailedInstances tests retrieval of failed instances.
func TestAggregatedResult_GetFailedInstances(t *testing.T) {
	// ARRANGE
	ar := NewAggregatedResult()
	ar.Add(createTestExecutionResult(StatusSuccess, "i-001"))
	ar.Add(createTestExecutionResult(StatusFailed, "i-002"))
	ar.Add(createTestExecutionResult(StatusSuccess, "i-003"))
	ar.Add(createTestExecutionResult(StatusFailed, "i-004"))
	ar.Add(createTestExecutionResult(StatusSkipped, "i-005"))

	// ACT
	failed := ar.GetFailedInstances()

	// ASSERT
	if len(failed) != 2 {
		t.Fatalf("len(failed) = %d, want 2", len(failed))
	}

	// Verify failed instances are correct
	failedIDs := make(map[string]bool)
	for _, result := range failed {
		failedIDs[result.Instance.ID] = true
	}

	if !failedIDs["i-002"] {
		t.Error("Expected i-002 in failed instances")
	}

	if !failedIDs["i-004"] {
		t.Error("Expected i-004 in failed instances")
	}
}

// TestAggregatedResult_GetFailedInstances_Empty tests when no failures.
func TestAggregatedResult_GetFailedInstances_Empty(t *testing.T) {
	// ARRANGE
	ar := NewAggregatedResult()
	ar.Add(createTestExecutionResult(StatusSuccess, "i-001"))
	ar.Add(createTestExecutionResult(StatusSuccess, "i-002"))

	// ACT
	failed := ar.GetFailedInstances()

	// ASSERT
	if len(failed) != 0 {
		t.Errorf("len(failed) = %d, want 0", len(failed))
	}
}

// TestAggregatedResult_String tests string representation.
func TestAggregatedResult_String(t *testing.T) {
	// ARRANGE
	ar := NewAggregatedResult()
	ar.Add(createTestExecutionResult(StatusSuccess, "i-001"))
	ar.Add(createTestExecutionResult(StatusFailed, "i-002"))
	ar.Add(createTestExecutionResult(StatusSkipped, "i-003"))
	ar.Finalize()

	// ACT
	str := ar.String()

	// ASSERT - Should contain all key metrics
	if !contains(str, "Total: 3") {
		t.Errorf("String() = %q, want to contain 'Total: 3'", str)
	}

	if !contains(str, "Success: 1") {
		t.Errorf("String() = %q, want to contain 'Success: 1'", str)
	}

	if !contains(str, "Failed: 1") {
		t.Errorf("String() = %q, want to contain 'Failed: 1'", str)
	}

	if !contains(str, "Skipped: 1") {
		t.Errorf("String() = %q, want to contain 'Skipped: 1'", str)
	}

	if !contains(str, "Time:") {
		t.Errorf("String() = %q, want to contain 'Time:'", str)
	}
}

// TestAggregatedResult_CompleteWorkflow tests complete aggregation workflow.
//
// 🎓 CONCEPT: Integration testing
// - Test complete lifecycle: create → add → finalize
// - Verify all metrics are correctly calculated
func TestAggregatedResult_CompleteWorkflow(t *testing.T) {
	// ARRANGE
	ar := NewAggregatedResult()

	// ACT - Simulate complete execution workflow
	startTime := time.Now()

	// Add 10 results
	for i := 1; i <= 10; i++ {
		var status ExecutionStatus
		switch {
		case i <= 7:
			status = StatusSuccess
		case i <= 9:
			status = StatusFailed
		default:
			status = StatusSkipped
		}

		ar.Add(createTestExecutionResult(status, fmt.Sprintf("i-%03d", i)))
	}

	time.Sleep(5 * time.Millisecond) // Ensure time passes
	ar.Finalize()

	// ASSERT - Verify all metrics
	if ar.Total != 10 {
		t.Errorf("Total = %d, want 10", ar.Total)
	}

	if ar.Success != 7 {
		t.Errorf("Success = %d, want 7", ar.Success)
	}

	if ar.Failed != 2 {
		t.Errorf("Failed = %d, want 2", ar.Failed)
	}

	if ar.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", ar.Skipped)
	}

	expectedSuccessRate := 70.0
	if ar.SuccessRate() != expectedSuccessRate {
		t.Errorf("SuccessRate() = %.1f, want %.1f", ar.SuccessRate(), expectedSuccessRate)
	}

	expectedFailureRate := 20.0
	if ar.FailureRate() != expectedFailureRate {
		t.Errorf("FailureRate() = %.1f, want %.1f", ar.FailureRate(), expectedFailureRate)
	}

	if ar.EndTime.Before(startTime) {
		t.Error("EndTime is before execution start")
	}

	if ar.TotalTime == 0 {
		t.Error("TotalTime is zero")
	}

	failed := ar.GetFailedInstances()
	if len(failed) != 2 {
		t.Errorf("len(GetFailedInstances()) = %d, want 2", len(failed))
	}
}

// ============================================================
// UTILITY FUNCTIONS
// ============================================================

// contains checks if a string contains a substring
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
