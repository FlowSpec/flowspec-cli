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
		// Try to extract line information from YAML error
		line := 0
		if yamlErr, ok := err.(*yaml.TypeError); ok {
			// yaml.TypeError contains line information
			errors = append(errors, models.ParseError{
				File:    filepath,
				Line:    line,
				Message: fmt.Sprintf("YAML parsing error: %s", yamlErr.Error()),
			})
		} else {
			errors = append(errors, models.ParseError{
				File:    filepath,
				Line:    line,
				Message: fmt.Sprintf("YAML parsing error: %s", err.Error()),
			})
		}
		return specs, errors
	}

	// Ensure this is a YAML format spec (has apiVersion and kind)
	if !spec.IsYAMLFormat() {
		errors = append(errors, models.ParseError{
			File:    filepath,
			Line:    0,
			Message: "YAML file must contain apiVersion and kind fields",
		})
		return specs, errors
	}

	// Validate the parsed ServiceSpec
	if err := spec.Validate(); err != nil {
		errors = append(errors, models.ParseError{
			File:    filepath,
			Line:    0,
			Message: fmt.Sprintf("validation error: %s", err.Error()),
		})
		return specs, errors
	}

	specs = append(specs, spec)
	return specs, errors
}