package csv

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/estudosdevops/opsmaster/internal/cloud"
)

// Parser parses CSV files containing instance information.
// Supports flexible formats with/without headers and extra columns.
type Parser struct {
	config CSVConfig
}

// NewParser creates a new CSV parser with given configuration.
// If config is empty, uses DefaultCSVConfig().
func NewParser(config CSVConfig) *Parser {
	// Set defaults if not provided
	if config.Delimiter == 0 {
		config.Delimiter = ','
	}
	if config.CloudDefault == "" {
		config.CloudDefault = "aws"
	}
	if len(config.RequiredFields) == 0 {
		config.RequiredFields = []string{"instance_id", "account", "region"}
	}

	return &Parser{config: config}
}

// ParseFile reads CSV file and returns list of instances.
// Validates required fields and converts each row to cloud.Instance.
//
// CSV Format Examples:
//
// With header:
//
//	instance_id,account,region,environment
//	i-123,111111111111,us-east-1,prod
//
// Without header (fixed order):
//
//	i-123,111111111111,us-east-1
//
// Returns ParseError with line/column information if validation fails.
func (p *Parser) ParseFile(filePath string) ([]*cloud.Instance, error) {
	// Open CSV file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	// Create CSV reader
	reader := csv.NewReader(file)
	reader.Comma = p.config.Delimiter
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	// Read all records at once
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	// Check if CSV is empty
	if len(records) == 0 {
		return nil, &ParseError{
			Line:    0,
			Message: "CSV file is empty",
		}
	}

	// Process header and determine column mapping
	var headerMap map[string]int
	startIndex := 0

	if p.config.HasHeader {
		// First row is header - build column mapping
		headerMap = p.buildHeaderMap(records[0])
		startIndex = 1

		// Validate that all required fields exist in header
		if err := p.validateHeaders(headerMap); err != nil {
			return nil, err
		}
	} else {
		// No header - use default column order
		headerMap = p.buildDefaultHeaderMap()
	}

	// Parse data rows
	var instances []*cloud.Instance
	for i := startIndex; i < len(records); i++ {
		lineNumber := i + 1 // Line numbers are 1-based for users

		// Skip empty lines
		if len(records[i]) == 0 || (len(records[i]) == 1 && strings.TrimSpace(records[i][0]) == "") {
			continue
		}

		instance, err := p.parseRecord(records[i], headerMap, lineNumber)
		if err != nil {
			return nil, err
		}

		instances = append(instances, instance)
	}

	// Check if we got any instances
	if len(instances) == 0 {
		return nil, &ParseError{
			Line:    0,
			Message: "no valid instances found in CSV",
		}
	}

	return instances, nil
}

// buildHeaderMap creates mapping from column names to their indices.
// Column names are normalized (lowercase, trimmed) for case-insensitive matching.
func (*Parser) buildHeaderMap(header []string) map[string]int {
	headerMap := make(map[string]int)
	for i, col := range header {
		// Normalize: lowercase and trim spaces
		normalized := strings.ToLower(strings.TrimSpace(col))
		headerMap[normalized] = i
	}
	return headerMap
}

// buildDefaultHeaderMap creates default column mapping when no header exists.
// Default order: instance_id, account, region, cloud (optional)
func (*Parser) buildDefaultHeaderMap() map[string]int {
	return map[string]int{
		"instance_id": 0,
		"account":     1,
		"region":      2,
		"cloud":       3, // Optional
	}
}

// validateHeaders checks if all required fields exist in CSV header.
// Returns ParseError if any required field is missing.
func (*Parser) validateHeaders(headerMap map[string]int) error {
	requiredFields := []string{"instance_id", "account", "region"}

	var missingFields []string
	for _, field := range requiredFields {
		if _, ok := headerMap[field]; !ok {
			missingFields = append(missingFields, field)
		}
	}

	if len(missingFields) > 0 {
		return &ParseError{
			Line:    1,
			Column:  strings.Join(missingFields, ", "),
			Message: fmt.Sprintf("missing required columns: %s", strings.Join(missingFields, ", ")),
		}
	}

	return nil
}

// extractRequiredField extracts and validates a required field from CSV record.
// Returns the field value or a ParseError if missing or empty.
func extractRequiredField(record []string, headerMap map[string]int, fieldName string, lineNumber int) (string, error) {
	idx, ok := headerMap[fieldName]
	if !ok || idx >= len(record) {
		return "", &ParseError{
			Line:    lineNumber,
			Column:  fieldName,
			Message: fmt.Sprintf("%s column not found or record too short", fieldName),
		}
	}

	value := strings.TrimSpace(record[idx])
	if value == "" {
		return "", &ParseError{
			Line:    lineNumber,
			Column:  fieldName,
			Message: fmt.Sprintf("%s is required and cannot be empty", fieldName),
		}
	}

	return value, nil
}

// extractOptionalCloud extracts cloud provider with fallback to default.
// Returns normalized cloud value (lowercase) or default from config.
func extractOptionalCloud(record []string, headerMap map[string]int, defaultCloud string) string {
	idx, ok := headerMap["cloud"]
	if !ok || idx >= len(record) {
		return defaultCloud
	}

	cloudProvider := strings.TrimSpace(record[idx])
	if cloudProvider == "" {
		return defaultCloud
	}

	return strings.ToLower(cloudProvider)
}

// extractMetadata captures extra CSV columns as instance metadata.
// Skips known columns (instance_id, account, region, cloud).
func extractMetadata(record []string, headerMap map[string]int, instance *cloud.Instance) {
	knownColumns := map[string]bool{
		"instance_id": true,
		"account":     true,
		"region":      true,
		"cloud":       true,
	}

	for colName, idx := range headerMap {
		if knownColumns[colName] {
			continue
		}

		if idx < len(record) {
			value := strings.TrimSpace(record[idx])
			if value != "" {
				instance.Metadata[colName] = value
			}
		}
	}
}

// parseRecord converts a CSV row into cloud.Instance.
// Validates required fields and captures extra columns as metadata.
func (p *Parser) parseRecord(record []string, headerMap map[string]int, lineNumber int) (*cloud.Instance, error) {
	instance := &cloud.Instance{
		Metadata: make(map[string]string),
	}

	// Extract required fields
	var err error
	instance.ID, err = extractRequiredField(record, headerMap, "instance_id", lineNumber)
	if err != nil {
		return nil, err
	}

	instance.Account, err = extractRequiredField(record, headerMap, "account", lineNumber)
	if err != nil {
		return nil, err
	}

	instance.Region, err = extractRequiredField(record, headerMap, "region", lineNumber)
	if err != nil {
		return nil, err
	}

	// Extract optional cloud with default
	instance.Cloud = extractOptionalCloud(record, headerMap, p.config.CloudDefault)

	// Capture extra columns as metadata
	extractMetadata(record, headerMap, instance)

	return instance, nil
}

// ParseString parses CSV content from a string instead of file.
// Useful for testing or when CSV content comes from other sources.
func (p *Parser) ParseString(content string) ([]*cloud.Instance, error) {
	// Create temporary file with content
	tmpFile, err := os.CreateTemp("", "opsmaster-csv-*.csv")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write content to temp file
	if _, err := tmpFile.WriteString(content); err != nil {
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}

	// Parse the temp file
	return p.ParseFile(tmpFile.Name())
}
