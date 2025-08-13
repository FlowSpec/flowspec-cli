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

func TestNewYAMLFileParser(t *testing.T) {
	parser := NewYAMLFileParser()
	assert.NotNil(t, parser)
}

func TestYAMLFileParser_CanParse(t *testing.T) {
	parser := NewYAMLFileParser()

	testCases := []struct {
		filename string
		expected bool
	}{
		// Supported extensions
		{"service-spec.yaml", true},
		{"service-spec.yml", true},
		{"contract.yaml", true},
		{"contract.yml", true},
		
		// Case insensitive
		{"service-spec.YAML", true},
		{"service-spec.YML", true},
		{"service-spec.Yaml", true},
		{"service-spec.Yml", true},
		
		// Not supported
		{"service-spec.json", false},
		{"service-spec.txt", false},
		{"service-spec", false},
		{"", false},
		{"file.yaml.backup", false},
		{"yaml", false},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			result := parser.CanParse(tc.filename)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestYAMLFileParser_ParseFile_ValidYAML(t *testing.T) {
	parser := NewYAMLFileParser()

	// Create temporary YAML file
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "service-spec.yaml")

	validYAML := `apiVersion: flowspec/v1alpha1
kind: ServiceSpec
metadata:
  name: test-service
  version: v1.0.0
spec:
  endpoints:
    - path: /api/users/{id}
      operations:
        - method: GET
          responses:
            statusCodes: [200, 404]
            aggregation: exact
          required:
            headers: [authorization]
            query: []
          optional:
            headers: [accept-language]
            query: [include]
        - method: PUT
          responses:
            statusRanges: ["2xx", "4xx"]
            aggregation: range
          required:
            headers: [authorization, content-type]
            query: []
    - path: /api/posts
      operations:
        - method: GET
          responses:
            statusCodes: [200]
          required:
            headers: []
            query: []
`

	err := os.WriteFile(yamlFile, []byte(validYAML), 0644)
	require.NoError(t, err)

	// Parse the file
	specs, errors := parser.ParseFile(yamlFile)

	// Should have no errors
	assert.Empty(t, errors)
	require.Len(t, specs, 1)

	spec := specs[0]
	
	// Verify basic structure
	assert.Equal(t, "flowspec/v1alpha1", spec.APIVersion)
	assert.Equal(t, "ServiceSpec", spec.Kind)
	assert.Equal(t, "test-service", spec.Metadata.Name)
	assert.Equal(t, "v1.0.0", spec.Metadata.Version)
	assert.Equal(t, yamlFile, spec.SourceFile)
	assert.Equal(t, 1, spec.LineNumber)

	// Verify endpoints
	require.Len(t, spec.Spec.Endpoints, 2)

	// First endpoint
	endpoint1 := spec.Spec.Endpoints[0]
	assert.Equal(t, "/api/users/{id}", endpoint1.Path)
	require.Len(t, endpoint1.Operations, 2)

	// GET operation
	getOp := endpoint1.Operations[0]
	assert.Equal(t, "GET", getOp.Method)
	assert.Equal(t, []int{200, 404}, getOp.Responses.StatusCodes)
	assert.Equal(t, "exact", getOp.Responses.Aggregation)
	assert.Equal(t, []string{"authorization"}, getOp.Required.Headers)
	assert.Empty(t, getOp.Required.Query)
	assert.Equal(t, []string{"accept-language"}, getOp.Optional.Headers)
	assert.Equal(t, []string{"include"}, getOp.Optional.Query)

	// PUT operation
	putOp := endpoint1.Operations[1]
	assert.Equal(t, "PUT", putOp.Method)
	assert.Equal(t, []string{"2xx", "4xx"}, putOp.Responses.StatusRanges)
	assert.Equal(t, "range", putOp.Responses.Aggregation)
	assert.Equal(t, []string{"authorization", "content-type"}, putOp.Required.Headers)
	assert.Empty(t, putOp.Required.Query)

	// Second endpoint
	endpoint2 := spec.Spec.Endpoints[1]
	assert.Equal(t, "/api/posts", endpoint2.Path)
	require.Len(t, endpoint2.Operations, 1)

	getPostsOp := endpoint2.Operations[0]
	assert.Equal(t, "GET", getPostsOp.Method)
	assert.Equal(t, []int{200}, getPostsOp.Responses.StatusCodes)
	assert.Empty(t, getPostsOp.Required.Headers)
	assert.Empty(t, getPostsOp.Required.Query)
}

func TestYAMLFileParser_ParseFile_InvalidYAML(t *testing.T) {
	parser := NewYAMLFileParser()

	// Create temporary file with invalid YAML
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "invalid.yaml")

	invalidYAML := `apiVersion: flowspec/v1alpha1
kind: ServiceSpec
metadata:
  name: test-service
  version: v1.0.0
spec:
  endpoints:
    - path: /api/users
      operations:
        - method: GET
          responses:
            statusCodes: [200
          # Missing closing bracket - invalid YAML
`

	err := os.WriteFile(yamlFile, []byte(invalidYAML), 0644)
	require.NoError(t, err)

	// Parse the file
	specs, errors := parser.ParseFile(yamlFile)

	// Should have errors
	assert.NotEmpty(t, errors)
	assert.Empty(t, specs)

	// Check error details
	parseError := errors[0]
	assert.Equal(t, yamlFile, parseError.File)
	assert.Contains(t, parseError.Message, "failed to parse YAML")
}

func TestYAMLFileParser_ParseFile_MissingRequiredFields(t *testing.T) {
	parser := NewYAMLFileParser()

	// Create temporary file with missing required fields
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "missing-fields.yaml")

	missingFieldsYAML := `# Missing apiVersion and kind
metadata:
  name: test-service
  version: v1.0.0
spec:
  endpoints: []
`

	err := os.WriteFile(yamlFile, []byte(missingFieldsYAML), 0644)
	require.NoError(t, err)

	// Parse the file
	specs, errors := parser.ParseFile(yamlFile)

	// Should have validation errors
	assert.NotEmpty(t, errors)
	assert.Empty(t, specs)

	// Should have multiple validation errors
	assert.Greater(t, len(errors), 0)
	
	// Check that errors mention missing fields
	errorMessages := make([]string, len(errors))
	for i, err := range errors {
		errorMessages[i] = err.Message
	}
	
	// Should have errors about missing apiVersion and kind
	hasAPIVersionError := false
	hasKindError := false
	for _, msg := range errorMessages {
		if contains(msg, "apiVersion") {
			hasAPIVersionError = true
		}
		if contains(msg, "kind") {
			hasKindError = true
		}
	}
	
	assert.True(t, hasAPIVersionError, "Should have error about missing apiVersion")
	assert.True(t, hasKindError, "Should have error about missing kind")
}

func TestYAMLFileParser_ParseFile_FileNotFound(t *testing.T) {
	parser := NewYAMLFileParser()

	// Try to parse non-existent file
	specs, errors := parser.ParseFile("/non/existent/file.yaml")

	// Should have file read error
	assert.NotEmpty(t, errors)
	assert.Empty(t, specs)

	parseError := errors[0]
	assert.Equal(t, "/non/existent/file.yaml", parseError.File)
	assert.Equal(t, 0, parseError.Line)
	assert.Contains(t, parseError.Message, "failed to read file")
}

func TestYAMLFileParser_ParseFile_EmptyFile(t *testing.T) {
	parser := NewYAMLFileParser()

	// Create empty file
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "empty.yaml")

	err := os.WriteFile(yamlFile, []byte(""), 0644)
	require.NoError(t, err)

	// Parse the file
	specs, errors := parser.ParseFile(yamlFile)

	// Should have validation errors (missing required fields)
	assert.NotEmpty(t, errors)
	assert.Empty(t, specs)
}

func TestYAMLFileParser_ParseFile_WithStats(t *testing.T) {
	parser := NewYAMLFileParser()

	// Create temporary YAML file with stats
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "with-stats.yaml")

	yamlWithStats := `apiVersion: flowspec/v1alpha1
kind: ServiceSpec
metadata:
  name: test-service
  version: v1.0.0
spec:
  endpoints:
    - path: /api/users/{id}
      stats:
        supportCount: 150
        firstSeen: "2025-08-01T12:00:00Z"
        lastSeen: "2025-08-10T12:00:00Z"
      operations:
        - method: GET
          responses:
            statusCodes: [200, 404]
          required:
            headers: [authorization]
            query: []
          stats:
            supportCount: 100
            firstSeen: "2025-08-01T12:00:00Z"
            lastSeen: "2025-08-10T12:00:00Z"
        - method: PUT
          responses:
            statusCodes: [200, 400]
          required:
            headers: [authorization]
            query: []
          stats:
            supportCount: 50
            firstSeen: "2025-08-02T12:00:00Z"
            lastSeen: "2025-08-09T12:00:00Z"
`

	err := os.WriteFile(yamlFile, []byte(yamlWithStats), 0644)
	require.NoError(t, err)

	// Parse the file
	specs, errors := parser.ParseFile(yamlFile)

	// Should have no errors
	assert.Empty(t, errors)
	require.Len(t, specs, 1)

	spec := specs[0]
	require.Len(t, spec.Spec.Endpoints, 1)

	endpoint := spec.Spec.Endpoints[0]
	
	// Verify endpoint stats
	require.NotNil(t, endpoint.Stats)
	assert.Equal(t, 150, endpoint.Stats.SupportCount)
	assert.Equal(t, "2025-08-01T12:00:00Z", endpoint.Stats.FirstSeen.Format("2006-01-02T15:04:05Z"))
	assert.Equal(t, "2025-08-10T12:00:00Z", endpoint.Stats.LastSeen.Format("2006-01-02T15:04:05Z"))

	// Verify operation stats
	require.Len(t, endpoint.Operations, 2)
	
	getOp := endpoint.Operations[0]
	require.NotNil(t, getOp.Stats)
	assert.Equal(t, 100, getOp.Stats.SupportCount)
	
	putOp := endpoint.Operations[1]
	require.NotNil(t, putOp.Stats)
	assert.Equal(t, 50, putOp.Stats.SupportCount)
}

func TestExtractLineColumnFromYAMLError(t *testing.T) {
	testCases := []struct {
		name         string
		errorMsg     string
		expectedLine int
		expectedCol  int
	}{
		{
			name:         "No line information",
			errorMsg:     "generic error message",
			expectedLine: 0,
			expectedCol:  0,
		},
		{
			name:         "Line only",
			errorMsg:     "error at line 5: invalid syntax",
			expectedLine: 5,
			expectedCol:  0,
		},
		{
			name:         "Line and column",
			errorMsg:     "error at line 10, column 15: missing value",
			expectedLine: 10,
			expectedCol:  15,
		},
		{
			name:         "Different format",
			errorMsg:     "yaml: line 3: found unexpected end of stream",
			expectedLine: 3,
			expectedCol:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			line, col := extractLineColumnFromErrorMessage(tc.errorMsg)
			assert.Equal(t, tc.expectedLine, line)
			assert.Equal(t, tc.expectedCol, col)
		})
	}
}

func TestYAMLFileParser_ParseFile_ComplexValidation(t *testing.T) {
	parser := NewYAMLFileParser()

	// Create temporary file with complex validation issues
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "complex-validation.yaml")

	complexYAML := `apiVersion: flowspec/v1alpha1
kind: ServiceSpec
metadata:
  name: test-service
  version: v1.0.0
spec:
  endpoints:
    - path: /api/users/{id}
      operations:
        - method: INVALID_METHOD  # Invalid HTTP method
          responses:
            statusCodes: [999]     # Invalid status code
            statusRanges: ["9xx"]  # Invalid status range
            aggregation: invalid   # Invalid aggregation
          required:
            headers: [authorization]
            query: []
        - method: GET
          responses:
            # Missing statusCodes or statusRanges
            aggregation: exact
          required:
            headers: []
            query: []
`

	err := os.WriteFile(yamlFile, []byte(complexYAML), 0644)
	require.NoError(t, err)

	// Parse the file
	specs, errors := parser.ParseFile(yamlFile)

	// Should have validation errors
	assert.NotEmpty(t, errors)
	assert.Empty(t, specs)

	// Should have multiple validation errors
	assert.Greater(t, len(errors), 1)

	// Check that we have specific validation errors
	errorMessages := make([]string, len(errors))
	for i, err := range errors {
		errorMessages[i] = err.Message
	}

	// Should have errors about various validation issues
	hasMethodError := false
	hasStatusError := false
	for _, msg := range errorMessages {
		if contains(msg, "method") || contains(msg, "INVALID_METHOD") {
			hasMethodError = true
		}
		if contains(msg, "status") || contains(msg, "999") || contains(msg, "9xx") {
			hasStatusError = true
		}
	}

	assert.True(t, hasMethodError || hasStatusError, "Should have validation errors about invalid fields")
}

// Helper function to check if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    len(s) > len(substr) && 
		    (s[:len(substr)] == substr || 
		     s[len(s)-len(substr):] == substr || 
		     containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}