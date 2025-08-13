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

func TestDefaultSpecParser_ParseFromSource_SingleYAMLFile(t *testing.T) {
	parser := NewSpecParser()

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
`

	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "service-spec.yaml")
	err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Parse the single YAML file
	result, err := parser.ParseFromSource(yamlFile)

	// Verify results
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Specs, 1)
	assert.Empty(t, result.Errors)

	spec := result.Specs[0]
	assert.Equal(t, "flowspec/v1alpha1", spec.APIVersion)
	assert.Equal(t, "ServiceSpec", spec.Kind)
	assert.Equal(t, "user-service", spec.Metadata.Name)
	assert.Equal(t, yamlFile, spec.SourceFile)
}

func TestDefaultSpecParser_ParseFromSource_DirectoryWithServiceSpecYAML(t *testing.T) {
	parser := NewSpecParser()

	tmpDir := t.TempDir()

	// Create service-spec.yaml (preferred)
	serviceSpecContent := `apiVersion: flowspec/v1alpha1
kind: ServiceSpec
metadata:
  name: preferred-service
  version: v1.0.0
spec:
  endpoints:
    - path: /api/preferred
      operations:
        - method: GET
          responses:
            statusRanges: ["2xx"]
          required:
            headers: []
            query: []
`

	serviceSpecFile := filepath.Join(tmpDir, "service-spec.yaml")
	err := os.WriteFile(serviceSpecFile, []byte(serviceSpecContent), 0644)
	require.NoError(t, err)

	// Create another YAML file (should be ignored)
	otherYamlContent := `apiVersion: flowspec/v1alpha1
kind: ServiceSpec
metadata:
  name: other-service
  version: v1.0.0
spec:
  endpoints:
    - path: /api/other
      operations:
        - method: POST
          responses:
            statusRanges: ["2xx"]
          required:
            headers: []
            query: []
`

	otherYamlFile := filepath.Join(tmpDir, "other.yaml")
	err = os.WriteFile(otherYamlFile, []byte(otherYamlContent), 0644)
	require.NoError(t, err)

	// Create a Java source file (should be ignored when YAML is present)
	javaContent := `
@ServiceSpec(operationId = "getUserById", description = "Get user by ID")
public class UserService {
    public User getUserById(String id) {
        return userRepository.findById(id);
    }
}
`

	javaFile := filepath.Join(tmpDir, "UserService.java")
	err = os.WriteFile(javaFile, []byte(javaContent), 0644)
	require.NoError(t, err)

	// Parse the directory
	result, err := parser.ParseFromSource(tmpDir)

	// Verify results - should only parse service-spec.yaml
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Specs, 1)
	assert.Empty(t, result.Errors)

	spec := result.Specs[0]
	assert.Equal(t, "preferred-service", spec.Metadata.Name)
	assert.Equal(t, serviceSpecFile, spec.SourceFile)
}

func TestDefaultSpecParser_ParseFromSource_DirectoryWithoutYAML(t *testing.T) {
	parser := NewSpecParser()

	tmpDir := t.TempDir()

	// Create only Java source files (no YAML)
	javaContent := `
package com.example;

import com.flowspec.ServiceSpec;

@ServiceSpec(operationId = "getUserById", description = "Get user by ID")
public class UserService {
    public User getUserById(String id) {
        return userRepository.findById(id);
    }
}
`

	javaFile := filepath.Join(tmpDir, "UserService.java")
	err := os.WriteFile(javaFile, []byte(javaContent), 0644)
	require.NoError(t, err)

	// Parse the directory
	result, err := parser.ParseFromSource(tmpDir)

	// Verify results - should fallback to source code parsing
	require.NoError(t, err)
	assert.NotNil(t, result)
	// Note: The actual parsing of Java annotations would depend on the Java parser implementation
	// For this test, we're just verifying that the directory scanning works correctly
}

func TestDefaultSpecParser_ParseFromSource_DirectoryWithMultipleYAMLFiles(t *testing.T) {
	parser := NewSpecParser()

	tmpDir := t.TempDir()

	// Create multiple YAML files (no service-spec.yaml)
	yaml1Content := `apiVersion: flowspec/v1alpha1
kind: ServiceSpec
metadata:
  name: service1
  version: v1.0.0
spec:
  endpoints:
    - path: /api/service1
      operations:
        - method: GET
          responses:
            statusRanges: ["2xx"]
          required:
            headers: []
            query: []
`

	yaml1File := filepath.Join(tmpDir, "service1.yaml")
	err := os.WriteFile(yaml1File, []byte(yaml1Content), 0644)
	require.NoError(t, err)

	yaml2Content := `apiVersion: flowspec/v1alpha1
kind: ServiceSpec
metadata:
  name: service2
  version: v1.0.0
spec:
  endpoints:
    - path: /api/service2
      operations:
        - method: POST
          responses:
            statusRanges: ["2xx"]
          required:
            headers: []
            query: []
`

	yaml2File := filepath.Join(tmpDir, "service2.yaml")
	err = os.WriteFile(yaml2File, []byte(yaml2Content), 0644)
	require.NoError(t, err)

	// Parse the directory
	result, err := parser.ParseFromSource(tmpDir)

	// Verify results - should return conflict error when multiple YAML files exist without service-spec.yaml
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple YAML files found but no service-spec.yaml")
	assert.Contains(t, err.Error(), "service1.yaml")
	assert.Contains(t, err.Error(), "service2.yaml")
	assert.Contains(t, err.Error(), "Please use --path to specify the exact file")
	assert.Nil(t, result)
}

func TestDefaultSpecParser_ParseFromSource_UnsupportedSingleFile(t *testing.T) {
	parser := NewSpecParser()

	tmpDir := t.TempDir()

	// Create an unsupported file type
	txtFile := filepath.Join(tmpDir, "readme.txt")
	err := os.WriteFile(txtFile, []byte("This is a text file"), 0644)
	require.NoError(t, err)

	// Try to parse the unsupported file
	result, err := parser.ParseFromSource(txtFile)

	// Verify error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported file type")
	assert.Nil(t, result)
}

func TestDefaultSpecParser_isYAMLFile(t *testing.T) {
	parser := NewSpecParser()

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
		{"service-spec", false},
	}

	for _, test := range tests {
		t.Run(test.filename, func(t *testing.T) {
			result := parser.isYAMLFile(test.filename)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestDefaultSpecParser_prioritizeYAMLFiles(t *testing.T) {
	parser := NewSpecParser()

	tests := []struct {
		name      string
		yamlFiles []string
		expected  []string
	}{
		{
			name:      "service-spec.yaml present",
			yamlFiles: []string{"/path/other.yaml", "/path/service-spec.yaml", "/path/another.yml"},
			expected:  []string{"/path/service-spec.yaml"},
		},
		{
			name:      "no service-spec.yaml",
			yamlFiles: []string{"/path/other.yaml", "/path/another.yml"},
			expected:  []string{"/path/other.yaml", "/path/another.yml"},
		},
		{
			name:      "single file",
			yamlFiles: []string{"/path/single.yaml"},
			expected:  []string{"/path/single.yaml"},
		},
		{
			name:      "empty list",
			yamlFiles: []string{},
			expected:  []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := parser.prioritizeYAMLFiles(test.yamlFiles)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestDefaultSpecParser_ParseFromSource_MultipleYAMLConflict(t *testing.T) {
	parser := NewSpecParser()

	tmpDir := t.TempDir()

	// Create multiple YAML files without service-spec.yaml
	yaml1Content := `apiVersion: flowspec/v1alpha1
kind: ServiceSpec
metadata:
  name: service1
  version: v1.0.0
spec:
  endpoints:
    - path: /api/service1
      operations:
        - method: GET
          responses:
            statusRanges: ["2xx"]
          required:
            headers: []
            query: []
`

	yaml1File := filepath.Join(tmpDir, "service1.yaml")
	err := os.WriteFile(yaml1File, []byte(yaml1Content), 0644)
	require.NoError(t, err)

	yaml2Content := `apiVersion: flowspec/v1alpha1
kind: ServiceSpec
metadata:
  name: service2
  version: v1.0.0
spec:
  endpoints:
    - path: /api/service2
      operations:
        - method: POST
          responses:
            statusRanges: ["2xx"]
          required:
            headers: []
            query: []
`

	yaml2File := filepath.Join(tmpDir, "service2.yaml")
	err = os.WriteFile(yaml2File, []byte(yaml2Content), 0644)
	require.NoError(t, err)

	yaml3File := filepath.Join(tmpDir, "service3.yml")
	err = os.WriteFile(yaml3File, []byte(yaml1Content), 0644) // Reuse content
	require.NoError(t, err)

	// Parse the directory - should return conflict error
	result, err := parser.ParseFromSource(tmpDir)

	// Verify error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple YAML files found but no service-spec.yaml")
	assert.Contains(t, err.Error(), "service1.yaml")
	assert.Contains(t, err.Error(), "service2.yaml")
	assert.Contains(t, err.Error(), "service3.yml")
	assert.Contains(t, err.Error(), "Please use --path to specify the exact file")
	assert.Nil(t, result)
}

func TestDefaultSpecParser_hasServiceSpecYAML(t *testing.T) {
	parser := NewSpecParser()

	tests := []struct {
		name      string
		yamlFiles []string
		expected  bool
	}{
		{
			name:      "has service-spec.yaml",
			yamlFiles: []string{"/path/other.yaml", "/path/service-spec.yaml", "/path/another.yml"},
			expected:  true,
		},
		{
			name:      "has SERVICE-SPEC.YAML (case insensitive)",
			yamlFiles: []string{"/path/other.yaml", "/path/SERVICE-SPEC.YAML", "/path/another.yml"},
			expected:  true,
		},
		{
			name:      "no service-spec.yaml",
			yamlFiles: []string{"/path/other.yaml", "/path/another.yml"},
			expected:  false,
		},
		{
			name:      "empty list",
			yamlFiles: []string{},
			expected:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := parser.hasServiceSpecYAML(test.yamlFiles)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestDefaultSpecParser_getFileNames(t *testing.T) {
	parser := NewSpecParser()

	tests := []struct {
		name     string
		files    []string
		expected []string
	}{
		{
			name:     "full paths",
			files:    []string{"/path/to/service1.yaml", "/another/path/service2.yml", "/service3.yaml"},
			expected: []string{"service1.yaml", "service2.yml", "service3.yaml"},
		},
		{
			name:     "relative paths",
			files:    []string{"./service1.yaml", "../service2.yml"},
			expected: []string{"service1.yaml", "service2.yml"},
		},
		{
			name:     "just filenames",
			files:    []string{"service1.yaml", "service2.yml"},
			expected: []string{"service1.yaml", "service2.yml"},
		},
		{
			name:     "empty list",
			files:    []string{},
			expected: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := parser.getFileNames(test.files)
			assert.Equal(t, test.expected, result)
		})
	}
}