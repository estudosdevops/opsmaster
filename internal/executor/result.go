package executor

import (
	"fmt"
	"time"

	"github.com/estudosdevops/opsmaster/internal/cloud"
	"github.com/estudosdevops/opsmaster/internal/installer"
)

// percentageMultiplier is used to convert decimal rates to percentages (0.75 → 75.0%)
const percentageMultiplier = 100.0

// ExecutionStatus represents the state of an execution.
// Go doesn't have native enums, we use typed constants + iota.
type ExecutionStatus int

const (
	// StatusPending execution not yet started
	StatusPending ExecutionStatus = iota

	// StatusRunning execution in progress
	StatusRunning

	// StatusSuccess execution completed successfully
	StatusSuccess

	// StatusFailed execution failed
	StatusFailed

	// StatusCancelled execution canceled (Ctrl+C or timeout)
	StatusCancelled

	// StatusSkipped execution skipped (e.g., already has puppet=true tag)
	StatusSkipped
)

// String returns readable representation of the status.
// Implements Stringer interface.
func (s ExecutionStatus) String() string {
	switch s {
	case StatusPending:
		return "PENDING"
	case StatusRunning:
		return "RUNNING"
	case StatusSuccess:
		return "SUCCESS"
	case StatusFailed:
		return "FAILED"
	case StatusCancelled:
		return "CANCELED"
	case StatusSkipped:
		return "SKIPPED"
	default:
		return "UNKNOWN"
	}
}

// ExecutionResult represents installation result on an instance.
// Extends installer.InstallResult with execution information.
type ExecutionResult struct {
	Instance        *cloud.Instance          // Instance processed
	Status          ExecutionStatus          // Final execution status
	InstallResult   *installer.InstallResult // Detailed installation result
	ValidationErr   error                    // Validation error (if any)
	InstallationErr error                    // Installation error (if any)
	TaggingErr      error                    // Tagging error (if any)
	StartTime       time.Time                // When it started
	EndTime         time.Time                // When it finished
	Duration        time.Duration            // Total time
	Metadata        map[string]string        // Installation metadata (OS, certname, etc)
}

// Success returns true if execution was successful
func (er *ExecutionResult) Success() bool {
	return er.Status == StatusSuccess
}

// Failed returns true if execution failed
func (er *ExecutionResult) Failed() bool {
	return er.Status == StatusFailed
}

// String returns readable representation of the result
func (er *ExecutionResult) String() string {
	return fmt.Sprintf("[%s] %s - %s", er.Status, er.Instance.ID, er.Duration)
}

// GetError returns the first error found (if any).
// Order: ValidationErr > InstallationErr > TaggingErr
func (er *ExecutionResult) GetError() error {
	if er.ValidationErr != nil {
		return er.ValidationErr
	}
	if er.InstallationErr != nil {
		return er.InstallationErr
	}
	if er.TaggingErr != nil {
		return er.TaggingErr
	}
	return nil
}

// AggregatedResult aggregates results from multiple executions.
// Useful for final reports.
type AggregatedResult struct {
	Total     int                // Total instances processed
	Success   int                // Successful installations
	Failed    int                // Failed installations
	Skipped   int                // Skipped installations
	Canceled  int                // Canceled installations
	Results   []*ExecutionResult // Individual results
	TotalTime time.Duration      // Total execution time
	StartTime time.Time          // When it started
	EndTime   time.Time          // When it finished
}

// NewAggregatedResult creates empty aggregated result
func NewAggregatedResult() *AggregatedResult {
	return &AggregatedResult{
		Results:   make([]*ExecutionResult, 0),
		StartTime: time.Now(),
	}
}

// Add adds individual result to aggregate
func (ar *AggregatedResult) Add(result *ExecutionResult) {
	ar.Results = append(ar.Results, result)
	ar.Total++

	switch result.Status {
	case StatusSuccess:
		ar.Success++
	case StatusFailed:
		ar.Failed++
	case StatusSkipped:
		ar.Skipped++
	case StatusCancelled:
		ar.Canceled++
	}
}

// Finalize finalizes aggregation and calculates total time
func (ar *AggregatedResult) Finalize() {
	ar.EndTime = time.Now()
	ar.TotalTime = ar.EndTime.Sub(ar.StartTime)
}

// SuccessRate returns success rate in percentage
func (ar *AggregatedResult) SuccessRate() float64 {
	if ar.Total == 0 {
		return 0.0
	}
	return float64(ar.Success) / float64(ar.Total) * percentageMultiplier
}

// FailureRate returns failure rate in percentage
func (ar *AggregatedResult) FailureRate() float64 {
	if ar.Total == 0 {
		return 0.0
	}
	return float64(ar.Failed) / float64(ar.Total) * percentageMultiplier
}

// GetFailedInstances returns list of instances that failed
func (ar *AggregatedResult) GetFailedInstances() []*ExecutionResult {
	var failed []*ExecutionResult
	for _, result := range ar.Results {
		if result.Failed() {
			failed = append(failed, result)
		}
	}
	return failed
}

// String returns readable representation of aggregated result
func (ar *AggregatedResult) String() string {
	return fmt.Sprintf("Total: %d | Success: %d | Failed: %d | Skipped: %d | Time: %s",
		ar.Total, ar.Success, ar.Failed, ar.Skipped, ar.TotalTime)
}
