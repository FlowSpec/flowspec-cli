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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYAMLFileParser_CanParse(t *testing.T) {
	parser := NewYAMLFileParser()

	tests := []struct {
		filename string
		expected bool
	}{
		{"service-spec.yaml", true},
		{"service-spec.yml", true},
		{"SERVICE-SPEC.YAML", true},
		{"SERVICE-SPEC.YML", true},
		{"service-spec.json", false},
		{"service-spec.go", false},
		{"service-spec.java", false},
		{"service-spec.ts", false},
		{"service-spec", false},
		{"", false},
	}

	for _, test := range tests {
		t.Run(test.filename, func(t *testing.T) {
			result := parser.CanParse(test.filename)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestYAMLFileParser_ParseFile_ValidYAML(t *testing.T) {
	parser := NewYAMLFileParser()

	// Create a temporary YAML file
	yamlContent := `apiVersion: flowspec/v1alpha1
kind: ServiceSpec
metadata:
  name: user-service
  version: v1.0.0
spec:
  endpoints:
    - path: /api/users/{id}
      operations:
        - method: GET
          responses:
            statusRanges: ["2xx", "4xx"]
            aggregation: "range"
          required:
            headers: ["authorization"]
            query: []
          optional:
            headers: ["accept-language"]
            query: ["include"]
        - method: PUT
          responses:
            statusCodes: [200, 400, 500]
            aggregation: "exact"
          required:
            headers: ["authorization"]
            query: []
`

	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "service-spec.yaml")
	err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Parse the file
	specs, errors := parser.ParseFile(yamlFile)

	// Verify results
	assert.Empty(t, errors, "Expected no parse errors")
	assert.Len(t, specs, 1, "Expected exactly one ServiceSpec")

	spec := specs[0]
	assert.Equal(t, "flowspec/v1alpha1", spec.APIVersion)
	assert.Equal(t, "ServiceSpec", spec.Kind)
	assert.Equal(t, "user-service", spec.Metadata.Name)
	assert.Equal(t, "v1.0.0", spec.Metadata.Version)
	assert.Len(t, spec.Spec.Endpoints, 1)

	endpoint := spec.Spec.Endpoints[0]
	assert.Equal(t, "/api/users/{id}", endpoint.Path)
	assert.Len(t, endpoint.Operations, 2)

	// Check GET operation
	getOp := endpoint.Operations[0]
	assert.Equal(t, "GET", getOp.Method)
	assert.Equal(t, []string{"2xx", "4xx"}, getOp.Responses.StatusRanges)
	assert.Equal(t, "range", getOp.Responses.Aggregation)
	assert.Equal(t, []string{"authorization"}, getOp.Required.Headers)
	assert.Equal(t, []string{"accept-language"}, getOp.Optional.Headers)

	// Check PUT operation
	putOp := endpoint.Operations[1]
	assert.Equal(t, "PUT", putOp.Method)
	assert.Equal(t, []int{200, 400, 500}, putOp.Responses.StatusCodes)
	assert.Equal(t, "exact", putOp.Responses.Aggregation)
}

func TestYAMLFileParser_ParseFile_InvalidYAML(t *testing.T) {
	parser := NewYAMLFileParser()

	// Create a temporary invalid YAML file
	invalidYamlContent := `apiVersion: flowspec/v1alpha1
kind: ServiceSpec
metadata:
  name: user-service
  version: v1.0.0
spec:
  endpoints:
    - path: /api/users/{id}
      operations:
        - method: GET
          responses:
            statusRanges: ["2xx", "4xx"
            # Missing closing bracket - invalid YAML
`

	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "invalid.yaml")
	err := os.WriteFile(yamlFile, []byte(invalidYamlContent), 0644)
	require.NoError(t, err)

	// Parse the file
	specs, errors := parser.ParseFile(yamlFile)

	// Verify results
	assert.Empty(t, specs, "Expected no specs for invalid YAML")
	assert.NotEmpty(t, errors, "Expected parse errors for invalid YAML")
	assert.Contains(t, errors[0].Message, "YAML parsing error")
}

func TestYAMLFileParser_ParseFile_MissingRequiredFields(t *testing.T) {
	parser := NewYAMLFileParser()

	// Create a YAML file missing required fields
	yamlContent := `apiVersion: flowspec/v1alpha1
# Missing kind field
metadata:
  name: user-service
  version: v1.0.0
spec:
  endpoints:
    - path: /api/users/{id}
      operations:
        - method: GET
          responses:
            statusRanges: ["2xx"]
          required:
            headers: []
            query: []
`

	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "missing-kind.yaml")
	err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Parse the file
	specs, errors := parser.ParseFile(yamlFile)

	// Verify results
	assert.Empty(t, specs, "Expected no specs for invalid ServiceSpec")
	assert.NotEmpty(t, errors, "Expected validation errors")
	assert.Contains(t, errors[0].Message, "YAML file must contain apiVersion and kind fields")
}

func TestYAMLFileParser_ParseFile_ValidationError(t *testing.T) {
	parser := NewYAMLFileParser()

	// Create a YAML file with validation errors (missing metadata.name)
	yamlContent := `apiVersion: flowspec/v1alpha1
kind: ServiceSpec
metadata:
  # Missing name field
  version: v1.0.0
spec:
  endpoints:
    - path: /api/users/{id}
      operations:
        - method: GET
          responses:
            statusRanges: ["2xx"]
          required:
            headers: []
            query: []
`

	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "validation-error.yaml")
	err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Parse the file
	specs, errors := parser.ParseFile(yamlFile)

	// Verify results
	assert.Empty(t, specs, "Expected no specs for invalid ServiceSpec")
	assert.NotEmpty(t, errors, "Expected validation errors")
	assert.Contains(t, errors[0].Message, "validation error")
}

func TestYAMLFileParser_ParseFile_LegacyFormat(t *testing.T) {
	parser := NewYAMLFileParser()

	// Create a YAML file with legacy format (no apiVersion/kind)
	yamlContent := `operationId: getUserById
description: Get user by ID
preconditions:
  user.exists: true
postconditions:
  response.status: 200
sourceFile: UserService.java
lineNumber: 42
`

	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "legacy.yaml")
	err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Parse the file
	specs, errors := parser.ParseFile(yamlFile)

	// Verify results
	assert.Empty(t, specs, "Expected no specs for legacy format in YAML file")
	assert.NotEmpty(t, errors, "Expected error for legacy format")
	assert.Contains(t, errors[0].Message, "YAML file must contain apiVersion and kind fields")
}

func TestYAMLFileParser_ParseFile_FileNotFound(t *testing.T) {
	parser := NewYAMLFileParser()

	// Try to parse a non-existent file
	specs, errors := parser.ParseFile("/non/existent/file.yaml")

	// Verify results
	assert.Empty(t, specs, "Expected no specs for non-existent file")
	assert.NotEmpty(t, errors, "Expected file read error")
	assert.Contains(t, errors[0].Message, "failed to read file")
}