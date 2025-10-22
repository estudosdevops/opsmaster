package csv

import (
	"github.com/estudosdevops/opsmaster/internal/cloud"
)

// CSVRecord represents a row from the instances CSV file.
// Each CSV line becomes a CSVRecord.
type CSVRecord struct {
	InstanceID string            // Instance ID (required)
	Cloud      string            // Provider: aws, azure, gcp (default: aws)
	Account    string            // Account/Subscription/Project (required)
	Region     string            // Instance region (required)
	Extra      map[string]string // Optional extra columns (environment, team, etc)
}

// ToInstance converts CSVRecord to cloud.Instance.
// This decouples CSV format from internal structure.
func (r *CSVRecord) ToInstance() *cloud.Instance {
	return &cloud.Instance{
		ID:       r.InstanceID,
		Cloud:    r.Cloud,
		Account:  r.Account,
		Region:   r.Region,
		Metadata: r.Extra,
	}
}

// CSVConfig defines CSV parser configurations.
// Allows customizing parser behavior.
type CSVConfig struct {
	// HasHeader indicates if first line is header
	// true: first line has column names
	// false: fixed format (instance_id, cloud, account, region)
	HasHeader bool

	// Delimiter column separator character (default: comma)
	Delimiter rune

	// RequiredFields list of required fields in CSV
	// Parser will validate these fields exist and are not empty
	RequiredFields []string

	// CloudDefault default value for cloud if column doesn't exist
	// Useful for legacy CSVs that didn't have cloud column
	CloudDefault string
}

// DefaultCSVConfig returns default configuration.
// Format: instance_id,account,region (with header, comma, aws default)
func DefaultCSVConfig() CSVConfig {
	return CSVConfig{
		HasHeader:      true,
		Delimiter:      ',',
		RequiredFields: []string{"instance_id", "account", "region"},
		CloudDefault:   "aws",
	}
}

// ParseError represents error during CSV parsing
type ParseError struct {
	Line    int    // Line number where error occurred
	Column  string // Column name with problem (if applicable)
	Message string // Error message
	Err     error  // Original error (if any)
}

// Error implements error interface
func (pe *ParseError) Error() string {
	if pe.Column != "" {
		return "line " + string(rune(pe.Line)) + ", column '" + pe.Column + "': " + pe.Message
	}
	return "line " + string(rune(pe.Line)) + ": " + pe.Message
}

// Unwrap allows using errors.Is() and errors.As()
func (pe *ParseError) Unwrap() error {
	return pe.Err
}
