package csv

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/estudosdevops/opsmaster/internal/cloud"
)

// Test constants
const (
	testInstanceID = "i-001"
	testCloudAWS   = "aws"
)

// ============================================================
// HELPER FUNCTIONS - Test utilities
// ============================================================

// createTempCSVFile creates a temporary CSV file for testing.
// Returns file path and cleanup function.
func createTempCSVFile(t *testing.T, content string) (filePath string, cleanup func()) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "csv-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Write CSV file
	csvPath := filepath.Join(tmpDir, "instances.csv")
	if err := os.WriteFile(csvPath, []byte(content), 0600); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to write CSV file: %v", err)
	}

	// Return path and cleanup function
	cleanup = func() {
		os.RemoveAll(tmpDir)
	}

	return csvPath, cleanup
}

// ============================================================
// NEW PARSER TESTS
// ============================================================

// TestNewParser tests parser creation with different configurations.
//
// 🎓 CONCEPT: Constructor testing with default values
// - Validate default values are applied when config is empty
// - Test custom configuration values are preserved
func TestNewParser(t *testing.T) {
	tests := []struct {
		name           string
		config         CSVConfig
		expectedDelim  rune
		expectedCloud  string
		expectedFields int
	}{
		{
			name:           "empty config uses defaults",
			config:         CSVConfig{},
			expectedDelim:  ',',
			expectedCloud:  "aws",
			expectedFields: 3,
		},
		{
			name: "custom configuration",
			config: CSVConfig{
				HasHeader:      true,
				Delimiter:      ';',
				CloudDefault:   "azure",
				RequiredFields: []string{"instance_id", "account"},
			},
			expectedDelim:  ';',
			expectedCloud:  "azure",
			expectedFields: 2,
		},
		{
			name: "partial config uses defaults for missing",
			config: CSVConfig{
				HasHeader: true,
			},
			expectedDelim:  ',',
			expectedCloud:  "aws",
			expectedFields: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ACT
			parser := NewParser(tt.config)

			// ASSERT
			if parser == nil {
				t.Fatal("NewParser() returned nil")
			}

			if parser.config.Delimiter != tt.expectedDelim {
				t.Errorf("Delimiter = %q, want %q", parser.config.Delimiter, tt.expectedDelim)
			}

			if parser.config.CloudDefault != tt.expectedCloud {
				t.Errorf("CloudDefault = %q, want %q", parser.config.CloudDefault, tt.expectedCloud)
			}

			if len(parser.config.RequiredFields) != tt.expectedFields {
				t.Errorf("len(RequiredFields) = %d, want %d", len(parser.config.RequiredFields), tt.expectedFields)
			}
		})
	}
}

// ============================================================
// PARSE FILE TESTS - SUCCESS CASES
// ============================================================

// TestParseFile_WithHeader tests parsing CSV with header row.
//
// 🎓 CONCEPT: File I/O testing with headers
// - Test standard CSV format with column names
// - Validate metadata extraction from extra columns
func TestParseFile_WithHeader(t *testing.T) {
	tests := []struct {
		name             string
		csvContent       string
		expectedCount    int
		validateInstance func(t *testing.T, instances []*cloud.Instance)
	}{
		{
			name: "standard format with all fields",
			csvContent: `instance_id,account,region,cloud
i-001,111111111111,us-east-1,aws
i-002,222222222222,us-west-2,aws`,
			expectedCount: 2,
			validateInstance: func(t *testing.T, instances []*cloud.Instance) {
				if instances[0].ID != testInstanceID {
					t.Errorf("instances[0].ID = %q, want %q", instances[0].ID, testInstanceID)
				}
				if instances[0].Account != "111111111111" {
					t.Errorf("instances[0].Account = %q, want %q", instances[0].Account, "111111111111")
				}
				if instances[0].Region != "us-east-1" {
					t.Errorf("instances[0].Region = %q, want %q", instances[0].Region, "us-east-1")
				}
				if instances[0].Cloud != testCloudAWS {
					t.Errorf("instances[0].Cloud = %q, want %q", instances[0].Cloud, testCloudAWS)
				}
			},
		},
		{
			name: "with metadata columns",
			csvContent: `instance_id,account,region,environment,team
i-001,111111111111,us-east-1,production,platform
i-002,222222222222,us-west-2,staging,devops`,
			expectedCount: 2,
			validateInstance: func(t *testing.T, instances []*cloud.Instance) {
				if instances[0].Metadata["environment"] != "production" {
					t.Errorf("Metadata[environment] = %q, want %q", instances[0].Metadata["environment"], "production")
				}
				if instances[0].Metadata["team"] != "platform" {
					t.Errorf("Metadata[team] = %q, want %q", instances[0].Metadata["team"], "platform")
				}
				if instances[1].Metadata["environment"] != "staging" {
					t.Errorf("Metadata[environment] = %q, want %q", instances[1].Metadata["environment"], "staging")
				}
			},
		},
		{
			name: "case insensitive headers",
			csvContent: `Instance_ID,ACCOUNT,Region
i-001,111111111111,us-east-1`,
			expectedCount: 1,
			validateInstance: func(t *testing.T, instances []*cloud.Instance) {
				if instances[0].ID != testInstanceID {
					t.Errorf("Failed to parse with case insensitive headers")
				}
			},
		},
		{
			name: "with spaces in headers",
			csvContent: `instance_id , account , region
i-001,111111111111,us-east-1`,
			expectedCount: 1,
			validateInstance: func(t *testing.T, instances []*cloud.Instance) {
				if instances[0].ID != testInstanceID {
					t.Errorf("Failed to parse with spaces in headers")
				}
			},
		},
		{
			name: "with empty cloud defaults to config",
			csvContent: `instance_id,account,region,cloud
i-001,111111111111,us-east-1,`,
			expectedCount: 1,
			validateInstance: func(t *testing.T, instances []*cloud.Instance) {
				if instances[0].Cloud != testCloudAWS {
					t.Errorf("Cloud = %q, want %q (default)", instances[0].Cloud, testCloudAWS)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			csvPath, cleanup := createTempCSVFile(t, tt.csvContent)
			defer cleanup()

			config := CSVConfig{HasHeader: true}
			parser := NewParser(config)

			// ACT
			instances, err := parser.ParseFile(csvPath)

			// ASSERT
			if err != nil {
				t.Fatalf("ParseFile() error = %v, want nil", err)
			}

			if len(instances) != tt.expectedCount {
				t.Fatalf("len(instances) = %d, want %d", len(instances), tt.expectedCount)
			}

			if tt.validateInstance != nil {
				tt.validateInstance(t, instances)
			}
		})
	}
}

// TestParseFile_WithoutHeader tests parsing CSV without header row.
//
// 🎓 CONCEPT: Default column order parsing
// - Test fixed format: instance_id, account, region, cloud
func TestParseFile_WithoutHeader(t *testing.T) {
	tests := []struct {
		name          string
		csvContent    string
		expectedCount int
		expectedID    string
	}{
		{
			name: "standard format without header",
			csvContent: `i-001,111111111111,us-east-1,aws
i-002,222222222222,us-west-2,azure`,
			expectedCount: 2,
			expectedID:    "i-001",
		},
		{
			name: "without cloud column (uses default)",
			csvContent: `i-001,111111111111,us-east-1
i-002,222222222222,us-west-2`,
			expectedCount: 2,
			expectedID:    "i-001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			csvPath, cleanup := createTempCSVFile(t, tt.csvContent)
			defer cleanup()

			config := CSVConfig{HasHeader: false}
			parser := NewParser(config)

			// ACT
			instances, err := parser.ParseFile(csvPath)

			// ASSERT
			if err != nil {
				t.Fatalf("ParseFile() error = %v, want nil", err)
			}

			if len(instances) != tt.expectedCount {
				t.Fatalf("len(instances) = %d, want %d", len(instances), tt.expectedCount)
			}

			if instances[0].ID != tt.expectedID {
				t.Errorf("instances[0].ID = %q, want %q", instances[0].ID, tt.expectedID)
			}
		})
	}
}

// TestParseFile_SkipEmptyLines tests that parser skips empty lines.
func TestParseFile_SkipEmptyLines(t *testing.T) {
	csvContent := `instance_id,account,region

i-001,111111111111,us-east-1

i-002,222222222222,us-west-2
`
	// ARRANGE
	csvPath, cleanup := createTempCSVFile(t, csvContent)
	defer cleanup()

	config := CSVConfig{HasHeader: true}
	parser := NewParser(config)

	// ACT
	instances, err := parser.ParseFile(csvPath)

	// ASSERT
	if err != nil {
		t.Fatalf("ParseFile() error = %v, want nil", err)
	}

	if len(instances) != 2 {
		t.Errorf("len(instances) = %d, want 2 (empty lines should be skipped)", len(instances))
	}
}

// ============================================================
// PARSE FILE TESTS - ERROR CASES
// ============================================================

// TestParseFile_ValidationErrors tests various validation error scenarios.
//
// 🎓 CONCEPT: Error handling and validation
// - Test missing required fields
// - Test empty required fields
// - Validate error messages and line numbers
func TestParseFile_ValidationErrors(t *testing.T) {
	tests := []struct {
		name           string
		csvContent     string
		hasHeader      bool
		expectedErrMsg string
	}{
		{
			name:           "empty CSV file",
			csvContent:     "",
			hasHeader:      true,
			expectedErrMsg: "CSV file is empty",
		},
		{
			name: "missing required header columns",
			csvContent: `instance_id,account
i-001,111111111111`,
			hasHeader:      true,
			expectedErrMsg: "missing required columns",
		},
		{
			name: "empty instance_id",
			csvContent: `instance_id,account,region
,111111111111,us-east-1`,
			hasHeader:      true,
			expectedErrMsg: "instance_id is required",
		},
		{
			name: "empty account",
			csvContent: `instance_id,account,region
i-001,,us-east-1`,
			hasHeader:      true,
			expectedErrMsg: "account is required",
		},
		{
			name: "empty region",
			csvContent: `instance_id,account,region
i-001,111111111111,`,
			hasHeader:      true,
			expectedErrMsg: "region is required",
		},
		{
			name: "only header no data",
			csvContent: `instance_id,account,region
`,
			hasHeader:      true,
			expectedErrMsg: "no valid instances found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			csvPath, cleanup := createTempCSVFile(t, tt.csvContent)
			defer cleanup()

			config := CSVConfig{HasHeader: tt.hasHeader}
			parser := NewParser(config)

			// ACT
			instances, err := parser.ParseFile(csvPath)

			// ASSERT
			if err == nil {
				t.Fatalf("ParseFile() error = nil, want error containing %q", tt.expectedErrMsg)
			}

			if !contains(err.Error(), tt.expectedErrMsg) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.expectedErrMsg)
			}

			if instances != nil {
				t.Errorf("instances = %v, want nil on error", instances)
			}
		})
	}
}

// TestParseFile_FileErrors tests file-related errors.
func TestParseFile_FileErrors(t *testing.T) {
	tests := []struct {
		name           string
		filePath       string
		expectedErrMsg string
	}{
		{
			name:           "file does not exist",
			filePath:       "/nonexistent/path/instances.csv",
			expectedErrMsg: "failed to open CSV file",
		},
		{
			name:           "empty file path",
			filePath:       "",
			expectedErrMsg: "failed to open CSV file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			parser := NewParser(CSVConfig{HasHeader: true})

			// ACT
			instances, err := parser.ParseFile(tt.filePath)

			// ASSERT
			if err == nil {
				t.Fatal("ParseFile() error = nil, want error")
			}

			if !contains(err.Error(), tt.expectedErrMsg) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.expectedErrMsg)
			}

			if instances != nil {
				t.Error("Expected nil instances on error")
			}
		})
	}
}

// ============================================================
// PARSE STRING TESTS
// ============================================================

// TestParseString tests parsing CSV from string instead of file.
//
// 🎓 CONCEPT: Alternative input sources
// - Test in-memory CSV parsing
// - Useful for testing and API integrations
func TestParseString(t *testing.T) {
	tests := []struct {
		name          string
		csvContent    string
		hasHeader     bool
		expectedCount int
	}{
		{
			name: "simple CSV string",
			csvContent: `instance_id,account,region
i-001,111111111111,us-east-1
i-002,222222222222,us-west-2`,
			hasHeader:     true,
			expectedCount: 2,
		},
		{
			name: "CSV string without header",
			csvContent: `i-001,111111111111,us-east-1
i-002,222222222222,us-west-2`,
			hasHeader:     false,
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			config := CSVConfig{HasHeader: tt.hasHeader}
			parser := NewParser(config)

			// ACT
			instances, err := parser.ParseString(tt.csvContent)

			// ASSERT
			if err != nil {
				t.Fatalf("ParseString() error = %v, want nil", err)
			}

			if len(instances) != tt.expectedCount {
				t.Errorf("len(instances) = %d, want %d", len(instances), tt.expectedCount)
			}
		})
	}
}

// ============================================================
// CUSTOM DELIMITER TESTS
// ============================================================

// TestParseFile_CustomDelimiter tests parsing with different delimiters.
func TestParseFile_CustomDelimiter(t *testing.T) {
	tests := []struct {
		name          string
		csvContent    string
		delimiter     rune
		expectedCount int
	}{
		{
			name: "semicolon delimiter",
			csvContent: `instance_id;account;region
i-001;111111111111;us-east-1
i-002;222222222222;us-west-2`,
			delimiter:     ';',
			expectedCount: 2,
		},
		{
			name: "pipe delimiter",
			csvContent: `instance_id|account|region
i-001|111111111111|us-east-1`,
			delimiter:     '|',
			expectedCount: 1,
		},
		{
			name:          "tab delimiter",
			csvContent:    "instance_id\taccount\tregion\ni-001\t111111111111\tus-east-1",
			delimiter:     '\t',
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			csvPath, cleanup := createTempCSVFile(t, tt.csvContent)
			defer cleanup()

			config := CSVConfig{
				HasHeader: true,
				Delimiter: tt.delimiter,
			}
			parser := NewParser(config)

			// ACT
			instances, err := parser.ParseFile(csvPath)

			// ASSERT
			if err != nil {
				t.Fatalf("ParseFile() error = %v, want nil", err)
			}

			if len(instances) != tt.expectedCount {
				t.Errorf("len(instances) = %d, want %d", len(instances), tt.expectedCount)
			}
		})
	}
}

// ============================================================
// METADATA EXTRACTION TESTS
// ============================================================

// TestExtractMetadata tests metadata extraction from extra columns.
func TestExtractMetadata(t *testing.T) {
	csvContent := `instance_id,account,region,environment,team,owner,cost_center
i-001,111111111111,us-east-1,production,platform,john.doe,CC-1234`

	// ARRANGE
	csvPath, cleanup := createTempCSVFile(t, csvContent)
	defer cleanup()

	config := CSVConfig{HasHeader: true}
	parser := NewParser(config)

	// ACT
	instances, err := parser.ParseFile(csvPath)

	// ASSERT
	if err != nil {
		t.Fatalf("ParseFile() error = %v, want nil", err)
	}

	if len(instances) != 1 {
		t.Fatalf("len(instances) = %d, want 1", len(instances))
	}

	instance := instances[0]

	// Verify metadata was captured
	expectedMetadata := map[string]string{
		"environment": "production",
		"team":        "platform",
		"owner":       "john.doe",
		"cost_center": "CC-1234",
	}

	for key, expectedValue := range expectedMetadata {
		actualValue, exists := instance.Metadata[key]
		if !exists {
			t.Errorf("Metadata[%q] not found", key)
		}
		if actualValue != expectedValue {
			t.Errorf("Metadata[%q] = %q, want %q", key, actualValue, expectedValue)
		}
	}

	// Verify standard fields are NOT in metadata
	standardFields := []string{"instance_id", "account", "region", "cloud"}
	for _, field := range standardFields {
		if _, exists := instance.Metadata[field]; exists {
			t.Errorf("Standard field %q should not be in Metadata", field)
		}
	}
}

// ============================================================
// COMPLEX SCENARIO TESTS
// ============================================================

// TestParseFile_RealWorldScenario tests realistic CSV with many columns.
//
// 🎓 CONCEPT: Integration testing
// - Test complete real-world CSV format
// - Validate all features working together
func TestParseFile_RealWorldScenario(t *testing.T) {
	csvContent := `instance_id,account,region,cloud,environment,team,application,version
i-001,111111111111,us-east-1,aws,production,platform,api-gateway,v2.1.0
i-002,222222222222,us-west-2,aws,staging,devops,web-frontend,v1.5.3
i-003,333333333333,eu-west-1,aws,production,data,analytics-engine,v3.0.0`

	// ARRANGE
	csvPath, cleanup := createTempCSVFile(t, csvContent)
	defer cleanup()

	config := CSVConfig{HasHeader: true}
	parser := NewParser(config)

	// ACT
	instances, err := parser.ParseFile(csvPath)

	// ASSERT
	if err != nil {
		t.Fatalf("ParseFile() error = %v, want nil", err)
	}

	if len(instances) != 3 {
		t.Fatalf("len(instances) = %d, want 3", len(instances))
	}

	// Verify first instance completely
	first := instances[0]
	if first.ID != "i-001" {
		t.Errorf("ID = %q, want %q", first.ID, "i-001")
	}
	if first.Account != "111111111111" {
		t.Errorf("Account = %q, want %q", first.Account, "111111111111")
	}
	if first.Region != "us-east-1" {
		t.Errorf("Region = %q, want %q", first.Region, "us-east-1")
	}
	if first.Cloud != testCloudAWS {
		t.Errorf("Cloud = %q, want %q", first.Cloud, testCloudAWS)
	}
	if first.Metadata["environment"] != "production" {
		t.Errorf("Metadata[environment] = %q, want %q", first.Metadata["environment"], "production")
	}
	if first.Metadata["application"] != "api-gateway" {
		t.Errorf("Metadata[application] = %q, want %q", first.Metadata["application"], "api-gateway")
	}

	// Verify second instance has different values
	second := instances[1]
	if second.Cloud != "aws" {
		t.Errorf("second.Cloud = %q, want %q", second.Cloud, "aws")
	}
	if second.Metadata["environment"] != "staging" {
		t.Errorf("second.Metadata[environment] = %q, want %q", second.Metadata["environment"], "staging")
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
