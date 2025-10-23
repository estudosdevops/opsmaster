package installer

import (
	"fmt"
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/estudosdevops/opsmaster/internal/cloud"
)

// GetDefaultCustomFacts returns default custom facts configuration.
// Creates a location.yaml fact with standard fields from CSV.
//
// Default fact mapping:
//   - account (CSV) → account (fact field)
//   - environment (CSV metadata) → environment (fact field)
//   - region (CSV) → region (fact field)
//
// This function is public (capitalized) so it can be used by CLI commands,
// API handlers, or any other code that needs Puppet's default fact configuration.
//
// Example usage:
//
//	facts := installer.GetDefaultCustomFacts()
//	puppetInstaller := installer.NewPuppetInstaller(installer.PuppetOptions{
//	    CustomFacts: facts,
//	})
func GetDefaultCustomFacts() map[string]FactDefinition {
	return map[string]FactDefinition{
		"location": {
			FilePath: "location.yaml",
			FactName: "location",
			Fields: map[string]string{
				"account":     "account",
				"environment": "environment",
				"region":      "region",
			},
		},
	}
}

// LoadCustomFactsFromYAML loads custom fact definitions from YAML file.
// Returns map of fact definitions or error if file cannot be read/parsed.
//
// Expected YAML format:
//
//	location:
//	  file_path: "location.yaml"
//	  fact_name: "location"
//	  fields:
//	    account: "account"
//	    environment: "environment"
//	    region: "region"
//	compliance:
//	  file_path: "compliance.yaml"
//	  fact_name: "compliance"
//	  fields:
//	    compliance_level: "level"
//	    data_classification: "classification"
//
// Example usage:
//
//	facts, err := installer.LoadCustomFactsFromYAML("custom-facts.yaml")
//	if err != nil {
//	    return fmt.Errorf("failed to load facts: %w", err)
//	}
func LoadCustomFactsFromYAML(filepath string) (map[string]FactDefinition, error) {
	// Read YAML file
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read custom facts file: %w", err)
	}

	// Parse YAML into intermediate structure
	var rawFacts map[string]struct {
		FilePath string            `yaml:"file_path"`
		FactName string            `yaml:"fact_name"`
		Fields   map[string]string `yaml:"fields"`
	}

	if err := yaml.Unmarshal(data, &rawFacts); err != nil {
		return nil, fmt.Errorf("failed to parse custom facts YAML: %w", err)
	}

	// Validate and convert to FactDefinition map
	facts := make(map[string]FactDefinition)
	for key, raw := range rawFacts {
		// Validate required fields
		if raw.FilePath == "" {
			return nil, fmt.Errorf("fact '%s': file_path is required", key)
		}
		if raw.FactName == "" {
			return nil, fmt.Errorf("fact '%s': fact_name is required", key)
		}
		if len(raw.Fields) == 0 {
			return nil, fmt.Errorf("fact '%s': at least one field mapping is required", key)
		}

		facts[key] = FactDefinition{
			FilePath: raw.FilePath,
			FactName: raw.FactName,
			Fields:   raw.Fields,
		}
	}

	// Ensure at least one fact was defined
	if len(facts) == 0 {
		return nil, fmt.Errorf("no custom facts defined in file")
	}

	return facts, nil
}

// ValidateFactColumns checks if CSV has all columns referenced in custom facts.
// Returns list of missing columns that will result in empty fact fields.
//
// This function validates that the instance data (from CSV) contains all columns
// referenced in the fact definitions. Missing columns will result in empty or
// omitted fields in the generated Facter facts.
//
// Standard columns (account, region) are always available and not validated.
//
// Returns:
//   - Empty slice if all columns are present
//   - Slice of missing column names if any are missing
//
// Example usage:
//
//	missingCols := installer.ValidateFactColumns(facts, instances[0])
//	if len(missingCols) > 0 {
//	    log.Warn("Missing CSV columns", "columns", missingCols)
//	}
func ValidateFactColumns(facts map[string]FactDefinition, instance *cloud.Instance) []string {
	// Collect all CSV columns referenced in facts
	requiredColumns := make(map[string]bool)
	for _, factDef := range facts {
		for csvColumn := range factDef.Fields {
			requiredColumns[csvColumn] = true
		}
	}

	// Check each required column
	missingColumns := []string{}
	for column := range requiredColumns {
		// Standard columns (always available)
		if column == "account" || column == "region" {
			continue
		}

		// Check if column exists in instance metadata
		if instance.Metadata == nil || instance.Metadata[column] == "" {
			missingColumns = append(missingColumns, column)
		}
	}

	return missingColumns
}

// LogMissingFactColumns logs a warning if custom fact columns are missing from CSV.
// This is a convenience wrapper around ValidateFactColumns for CLI usage.
//
// Example usage:
//
//	installer.LogMissingFactColumns(log, facts, instances[0])
func LogMissingFactColumns(log *slog.Logger, facts map[string]FactDefinition, instance *cloud.Instance) {
	missingColumns := ValidateFactColumns(facts, instance)

	if len(missingColumns) > 0 {
		log.Warn("⚠️  Some custom fact columns are missing or empty in CSV",
			"missing_columns", missingColumns,
		)
		log.Warn("   → These fact fields will be empty or omitted in generated facts")
	}
}
