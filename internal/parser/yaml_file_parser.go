// Copyright 2024-2025 FlowSpec
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flowspec/flowspec-cli/internal/models"
	"gopkg.in/yaml.v3"
)

// YAMLFileParser implements the FileParser interface for YAML files
type YAMLFileParser struct{}

// NewYAMLFileParser creates a new YAML file parser
func NewYAMLFileParser() *YAMLFileParser {
	return &YAMLFileParser{}
}

// CanParse returns true if the file has a .yaml or .yml extension
func (y *YAMLFileParser) CanParse(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".yaml" || ext == ".yml"
}

// ParseFile parses a YAML file and returns ServiceSpecs and any parse errors
func (y *YAMLFileParser) ParseFile(filepath string) ([]models.ServiceSpec, []models.ParseError) {
	var specs []models.ServiceSpec
	var errors []models.ParseError

	// Read the file
	data, err := os.ReadFile(filepath)
	if err != nil {
		errors = append(errors, models.ParseError{
			File:    filepath,
			Line:    0,
			Message: fmt.Sprintf("failed to read file: %s", err.Error()),
		})
		return specs, errors
	}

	// Parse YAML
	var spec models.ServiceSpec
	err = yaml.Unmarshal(data, &spec)
	if err != nil {
		// Try to extract line and column information from YAML error
		lineNum, colNum := extractLineColumnFromYAMLError(err)

		errors = append(errors, models.ParseError{
			File:    filepath,
			Line:    lineNum,
			Column:  colNum,
			Message: fmt.Sprintf("failed to parse YAML: %s", err.Error()),
		})
		return specs, errors
	}

	// Create schema validator
	validator, err := NewSchemaValidator()
	if err != nil {
		errors = append(errors, models.ParseError{
			File:    filepath,
			Line:    0,
			Message: fmt.Sprintf("failed to create schema validator: %s", err.Error()),
		})
		return specs, errors
	}

	// Validate using JSON Schema
	schemaErrors := validator.ValidateServiceSpec(&spec)
	for _, schemaError := range schemaErrors {
		schemaError.File = filepath
		errors = append(errors, schemaError)
	}

	// If there are validation errors, don't return the spec
	if len(errors) > 0 {
		return specs, errors
	}

	// Set source file information
	spec.SourceFile = filepath
	spec.LineNumber = 1 // YAML files start at line 1

	specs = append(specs, spec)
	return specs, errors
}

// extractLineColumnFromYAMLError attempts to extract line and column information from YAML error
func extractLineColumnFromYAMLError(err error) (int, int) {
	if err == nil {
		return 0, 0
	}

	errMsg := err.Error()
	
	// Handle yaml.TypeError which contains multiple errors
	if yamlErr, ok := err.(*yaml.TypeError); ok {
		return extractLineColumnFromYAMLErrorMessages(yamlErr.Errors)
	}

	// Handle other YAML errors that might contain line information
	return extractLineColumnFromErrorMessage(errMsg)
}

// extractLineColumnFromYAMLErrorMessages extracts line/column from YAML error messages
func extractLineColumnFromYAMLErrorMessages(errors []string) (int, int) {
	for _, errMsg := range errors {
		if line, col := extractLineColumnFromErrorMessage(errMsg); line > 0 {
			return line, col
		}
	}
	return 0, 0
}

// extractLineColumnFromErrorMessage extracts line and column from a single error message
func extractLineColumnFromErrorMessage(errMsg string) (int, int) {
	// YAML v3 error messages often contain line information like "line 5:" or "line 5, column 10:"
	if strings.Contains(errMsg, "line ") {
		// Try to extract line number - this is a best effort
		parts := strings.Split(errMsg, "line ")
		if len(parts) > 1 {
			linePart := parts[1]
			
			// Check if there's also column information
			if strings.Contains(linePart, "column ") {
				// Format: "line 5, column 10:"
				lineColParts := strings.Split(linePart, ",")
				if len(lineColParts) >= 2 {
					// Extract line
					lineStr := strings.TrimSpace(strings.Split(lineColParts[0], ":")[0])
					var lineNum int
					if _, err := fmt.Sscanf(lineStr, "%d", &lineNum); err == nil {
						// Extract column
						colPart := strings.TrimSpace(lineColParts[1])
						if strings.HasPrefix(colPart, "column ") {
							colStr := strings.TrimSpace(strings.Split(strings.TrimPrefix(colPart, "column "), ":")[0])
							var colNum int
							if _, err := fmt.Sscanf(colStr, "%d", &colNum); err == nil {
								return lineNum, colNum
							}
						}
						return lineNum, 0
					}
				}
			} else {
				// Format: "line 5:"
				lineStr := strings.TrimSpace(strings.Split(linePart, ":")[0])
				var lineNum int
				if _, err := fmt.Sscanf(lineStr, "%d", &lineNum); err == nil {
					return lineNum, 0
				}
			}
		}
	}
	return 0, 0
}