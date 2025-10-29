package installer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/estudosdevops/opsmaster/internal/cloud"
)

// ============================================================
// HELPER FUNCTIONS - Test utilities
// ============================================================

// createTempYAMLFile creates a temporary YAML file for testing.
// Returns file path and cleanup function.
func createTempYAMLFile(t *testing.T, content string) (filePath string, cleanup func()) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "yaml-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Write YAML file
	yamlPath := filepath.Join(tmpDir, "facts.yaml")
	if err := os.WriteFile(yamlPath, []byte(content), 0600); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to write YAML file: %v", err)
	}

	// Return path and cleanup function
	cleanup = func() {
		os.RemoveAll(tmpDir)
	}

	return yamlPath, cleanup
}

// ============================================================
// GET DEFAULT CUSTOM FACTS TESTS
// ============================================================

// TestGetDefaultCustomFacts tests the default custom facts configuration.
//
// 🎓 CONCEPT: Default configuration testing
// - Validate that defaults are correctly structured
// - Ensure expected facts are present
// - Verify field mappings are correct
func TestGetDefaultCustomFacts(t *testing.T) {
	// ACT
	facts := GetDefaultCustomFacts()

	// ASSERT - Should have location fact
	if len(facts) == 0 {
		t.Fatal("GetDefaultCustomFacts() returned empty map")
	}

	locationFact, exists := facts["location"]
	if !exists {
		t.Fatal("Expected 'location' fact to exist in default facts")
	}

	// Verify location fact structure
	if locationFact.FilePath != "location.yaml" {
		t.Errorf("FilePath = %q, want %q", locationFact.FilePath, "location.yaml")
	}

	if locationFact.FactName != "location" {
		t.Errorf("FactName = %q, want %q", locationFact.FactName, "location")
	}

	// Verify field mappings
	expectedFields := map[string]string{
		"account":     "account",
		"environment": "environment",
		"region":      "region",
	}

	if len(locationFact.Fields) != len(expectedFields) {
		t.Errorf("len(Fields) = %d, want %d", len(locationFact.Fields), len(expectedFields))
	}

	for csvCol, factField := range expectedFields {
		actualField, exists := locationFact.Fields[csvCol]
		if !exists {
			t.Errorf("Expected field mapping for CSV column %q", csvCol)
			continue
		}
		if actualField != factField {
			t.Errorf("Fields[%q] = %q, want %q", csvCol, actualField, factField)
		}
	}
}

// ============================================================
// LOAD CUSTOM FACTS FROM YAML TESTS
// ============================================================

// TestLoadCustomFactsFromYAML_Success tests successful loading of custom facts.
//
// 🎓 CONCEPT: File I/O testing with temporary files
// - Create temporary test files
// - Test parsing and validation
// - Clean up after test
func TestLoadCustomFactsFromYAML_Success(t *testing.T) {
	tests := []struct {
		name          string
		yamlContent   string
		expectedFacts int
		validateFunc  func(t *testing.T, facts map[string]FactDefinition)
	}{
		{
			name: "single fact definition",
			yamlContent: `
location:
  file_path: "location.yaml"
  fact_name: "location"
  fields:
    account: "account"
    environment: "environment"
    region: "region"
`,
			expectedFacts: 1,
			validateFunc: func(t *testing.T, facts map[string]FactDefinition) {
				fact, exists := facts["location"]
				if !exists {
					t.Fatal("Expected 'location' fact")
				}
				if fact.FilePath != "location.yaml" {
					t.Errorf("FilePath = %q, want %q", fact.FilePath, "location.yaml")
				}
				if len(fact.Fields) != 3 {
					t.Errorf("len(Fields) = %d, want 3", len(fact.Fields))
				}
			},
		},
		{
			name: "multiple fact definitions",
			yamlContent: `
location:
  file_path: "location.yaml"
  fact_name: "location"
  fields:
    account: "account"
    region: "region"
compliance:
  file_path: "compliance.yaml"
  fact_name: "compliance"
  fields:
    level: "compliance_level"
    classification: "data_classification"
`,
			expectedFacts: 2,
			validateFunc: func(t *testing.T, facts map[string]FactDefinition) {
				if _, exists := facts["location"]; !exists {
					t.Error("Expected 'location' fact")
				}
				if _, exists := facts["compliance"]; !exists {
					t.Error("Expected 'compliance' fact")
				}

				complianceFact := facts["compliance"]
				if complianceFact.FactName != "compliance" {
					t.Errorf("FactName = %q, want %q", complianceFact.FactName, "compliance")
				}
				if len(complianceFact.Fields) != 2 {
					t.Errorf("len(compliance.Fields) = %d, want 2", len(complianceFact.Fields))
				}
			},
		},
		{
			name: "fact with single field",
			yamlContent: `
simple:
  file_path: "simple.yaml"
  fact_name: "simple"
  fields:
    key: "value"
`,
			expectedFacts: 1,
			validateFunc: func(t *testing.T, facts map[string]FactDefinition) {
				fact, exists := facts["simple"]
				if !exists {
					t.Fatal("Expected 'simple' fact")
				}
				if len(fact.Fields) != 1 {
					t.Errorf("len(Fields) = %d, want 1", len(fact.Fields))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			yamlPath, cleanup := createTempYAMLFile(t, tt.yamlContent)
			defer cleanup()

			// ACT
			facts, err := LoadCustomFactsFromYAML(yamlPath)

			// ASSERT
			if err != nil {
				t.Fatalf("LoadCustomFactsFromYAML() error = %v, want nil", err)
			}

			if len(facts) != tt.expectedFacts {
				t.Errorf("len(facts) = %d, want %d", len(facts), tt.expectedFacts)
			}

			// Run custom validation
			if tt.validateFunc != nil {
				tt.validateFunc(t, facts)
			}
		})
	}
}

// TestLoadCustomFactsFromYAML_ValidationErrors tests validation error scenarios.
//
// 🎓 CONCEPT: Error testing
// - Test various invalid configurations
// - Ensure proper error messages
// - Validate error handling
func TestLoadCustomFactsFromYAML_ValidationErrors(t *testing.T) {
	tests := []struct {
		name           string
		yamlContent    string
		expectedErrMsg string
	}{
		{
			name: "missing file_path",
			yamlContent: `
location:
  fact_name: "location"
  fields:
    account: "account"
`,
			expectedErrMsg: "file_path is required",
		},
		{
			name: "missing fact_name",
			yamlContent: `
location:
  file_path: "location.yaml"
  fields:
    account: "account"
`,
			expectedErrMsg: "fact_name is required",
		},
		{
			name: "missing fields",
			yamlContent: `
location:
  file_path: "location.yaml"
  fact_name: "location"
  fields: {}
`,
			expectedErrMsg: "at least one field mapping is required",
		},
		{
			name: "empty fields",
			yamlContent: `
location:
  file_path: "location.yaml"
  fact_name: "location"
`,
			expectedErrMsg: "at least one field mapping is required",
		},
		{
			name:           "empty YAML file",
			yamlContent:    "",
			expectedErrMsg: "no custom facts defined",
		},
		{
			name:           "empty YAML object",
			yamlContent:    "{}",
			expectedErrMsg: "no custom facts defined",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			yamlPath, cleanup := createTempYAMLFile(t, tt.yamlContent)
			defer cleanup()

			// ACT
			facts, err := LoadCustomFactsFromYAML(yamlPath)

			// ASSERT
			if err == nil {
				t.Fatalf("LoadCustomFactsFromYAML() error = nil, want error containing %q", tt.expectedErrMsg)
			}

			if !contains(err.Error(), tt.expectedErrMsg) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.expectedErrMsg)
			}

			if facts != nil {
				t.Errorf("facts = %v, want nil on error", facts)
			}
		})
	}
}

// TestLoadCustomFactsFromYAML_FileErrors tests file-related error scenarios.
func TestLoadCustomFactsFromYAML_FileErrors(t *testing.T) {
	tests := []struct {
		name           string
		filepath       string
		expectedErrMsg string
	}{
		{
			name:           "file does not exist",
			filepath:       "/nonexistent/path/facts.yaml",
			expectedErrMsg: "failed to read custom facts file",
		},
		{
			name:           "empty filepath",
			filepath:       "",
			expectedErrMsg: "failed to read custom facts file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ACT
			facts, err := LoadCustomFactsFromYAML(tt.filepath)

			// ASSERT
			if err == nil {
				t.Fatalf("LoadCustomFactsFromYAML() error = nil, want error")
			}

			if !contains(err.Error(), tt.expectedErrMsg) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.expectedErrMsg)
			}

			if facts != nil {
				t.Error("Expected nil facts on error")
			}
		})
	}
}

// TestLoadCustomFactsFromYAML_InvalidYAML tests YAML parsing errors.
func TestLoadCustomFactsFromYAML_InvalidYAML(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
	}{
		{
			name: "invalid YAML syntax",
			yamlContent: `
location:
  file_path: "location.yaml"
  fact_name: "location
  fields:
    account: "account"
`,
		},
		{
			name: "invalid indentation",
			yamlContent: `
location:
file_path: "location.yaml"
fact_name: "location"
fields:
  account: "account"
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			yamlPath, cleanup := createTempYAMLFile(t, tt.yamlContent)
			defer cleanup()

			// ACT
			facts, err := LoadCustomFactsFromYAML(yamlPath)

			// ASSERT
			if err == nil {
				t.Fatal("LoadCustomFactsFromYAML() error = nil, want YAML parse error")
			}

			if !contains(err.Error(), "failed to parse") {
				t.Errorf("error = %q, want to contain 'failed to parse'", err.Error())
			}

			if facts != nil {
				t.Error("Expected nil facts on parse error")
			}
		})
	}
}

// ============================================================
// VALIDATE FACT COLUMNS TESTS
// ============================================================

// TestValidateFactColumns tests column validation logic.
//
// 🎓 CONCEPT: Data validation testing
// - Test with complete data
// - Test with missing columns
// - Test edge cases (nil metadata, empty strings)
func TestValidateFactColumns(t *testing.T) {
	tests := []struct {
		name            string
		facts           map[string]FactDefinition
		instance        *cloud.Instance
		expectedMissing []string
	}{
		{
			name: "all columns present",
			facts: map[string]FactDefinition{
				"location": {
					FilePath: "location.yaml",
					FactName: "location",
					Fields: map[string]string{
						"account":     "account",
						"environment": "environment",
						"region":      "region",
					},
				},
			},
			instance: &cloud.Instance{
				ID:      "i-123",
				Account: "123456789012",
				Region:  "us-east-1",
				Metadata: map[string]string{
					"environment": "production",
				},
			},
			expectedMissing: []string{},
		},
		{
			name: "missing metadata column",
			facts: map[string]FactDefinition{
				"location": {
					FilePath: "location.yaml",
					FactName: "location",
					Fields: map[string]string{
						"account":     "account",
						"environment": "environment",
						"team":        "team",
					},
				},
			},
			instance: &cloud.Instance{
				ID:      "i-123",
				Account: "123456789012",
				Region:  "us-east-1",
				Metadata: map[string]string{
					"environment": "production",
					// "team" is missing
				},
			},
			expectedMissing: []string{"team"},
		},
		{
			name: "empty metadata value",
			facts: map[string]FactDefinition{
				"location": {
					FilePath: "location.yaml",
					FactName: "location",
					Fields: map[string]string{
						"environment": "environment",
						"owner":       "owner",
					},
				},
			},
			instance: &cloud.Instance{
				ID:      "i-123",
				Account: "123456789012",
				Region:  "us-east-1",
				Metadata: map[string]string{
					"environment": "production",
					"owner":       "", // Empty string
				},
			},
			expectedMissing: []string{"owner"},
		},
		{
			name: "nil metadata",
			facts: map[string]FactDefinition{
				"location": {
					FilePath: "location.yaml",
					FactName: "location",
					Fields: map[string]string{
						"environment": "environment",
					},
				},
			},
			instance: &cloud.Instance{
				ID:       "i-123",
				Account:  "123456789012",
				Region:   "us-east-1",
				Metadata: nil,
			},
			expectedMissing: []string{"environment"},
		},
		{
			name: "multiple missing columns",
			facts: map[string]FactDefinition{
				"location": {
					FilePath: "location.yaml",
					FactName: "location",
					Fields: map[string]string{
						"environment": "environment",
						"team":        "team",
						"owner":       "owner",
					},
				},
			},
			instance: &cloud.Instance{
				ID:      "i-123",
				Account: "123456789012",
				Region:  "us-east-1",
				Metadata: map[string]string{
					"environment": "production",
					// "team" and "owner" missing
				},
			},
			expectedMissing: []string{"team", "owner"},
		},
		{
			name: "standard columns (account, region) not reported as missing",
			facts: map[string]FactDefinition{
				"location": {
					FilePath: "location.yaml",
					FactName: "location",
					Fields: map[string]string{
						"account": "account",
						"region":  "region",
					},
				},
			},
			instance: &cloud.Instance{
				ID:       "i-123",
				Account:  "123456789012",
				Region:   "us-east-1",
				Metadata: nil,
			},
			expectedMissing: []string{},
		},
		{
			name: "multiple facts with mixed columns",
			facts: map[string]FactDefinition{
				"location": {
					FilePath: "location.yaml",
					FactName: "location",
					Fields: map[string]string{
						"environment": "environment",
					},
				},
				"compliance": {
					FilePath: "compliance.yaml",
					FactName: "compliance",
					Fields: map[string]string{
						"compliance_level": "level", // CSV column → fact field
					},
				},
			},
			instance: &cloud.Instance{
				ID:      "i-123",
				Account: "123456789012",
				Region:  "us-east-1",
				Metadata: map[string]string{
					"environment": "production",
					// "compliance_level" CSV column is missing
				},
			},
			expectedMissing: []string{"compliance_level"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ACT
			missing := ValidateFactColumns(tt.facts, tt.instance)

			// ASSERT
			if len(missing) != len(tt.expectedMissing) {
				t.Errorf("len(missing) = %d, want %d. Got: %v, Want: %v",
					len(missing), len(tt.expectedMissing), missing, tt.expectedMissing)
				return
			}

			// Check that all expected missing columns are present
			missingMap := make(map[string]bool)
			for _, col := range missing {
				missingMap[col] = true
			}

			for _, expectedCol := range tt.expectedMissing {
				if !missingMap[expectedCol] {
					t.Errorf("Expected missing column %q not found in result", expectedCol)
				}
			}
		})
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
