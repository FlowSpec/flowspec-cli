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

func TestNewBaseFileParser(t *testing.T) {
	parser := NewBaseFileParser(LanguageJava)

	assert.NotNil(t, parser)
	assert.Equal(t, LanguageJava, parser.language)
	assert.NotNil(t, parser.commentPatterns)
	assert.Len(t, parser.commentPatterns, 2) // Java has /** */ and /* */ patterns
}

func TestGetCommentPatterns(t *testing.T) {
	testCases := []struct {
		language      SupportedLanguage
		expectedCount int
	}{
		{LanguageJava, 2},       // /** */ and /* */
		{LanguageTypeScript, 3}, // /** */, /* */, and //
		{LanguageGo, 2},         // /* */ and //
	}

	for _, tc := range testCases {
		t.Run(string(tc.language), func(t *testing.T) {
			patterns := getCommentPatterns(tc.language)
			assert.Len(t, patterns, tc.expectedCount)
		})
	}
}

func TestCanParse(t *testing.T) {
	testCases := []struct {
		language SupportedLanguage
		filename string
		expected bool
	}{
		{LanguageJava, "Test.java", true},
		{LanguageJava, "test.py", false},
		{LanguageTypeScript, "test.ts", true},
		{LanguageTypeScript, "test.tsx", true},
		{LanguageTypeScript, "test.js", false},
		{LanguageGo, "test.go", true},
		{LanguageGo, "test.java", false},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			parser := NewBaseFileParser(tc.language)
			result := parser.CanParse(tc.filename)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsServiceSpecStart(t *testing.T) {
	parser := NewBaseFileParser(LanguageJava)

	testCases := []struct {
		line     string
		expected bool
	}{
		{"@ServiceSpec", true},
		{" * @ServiceSpec", true},
		{"// @ServiceSpec", true},
		{"  @ServiceSpec  ", true},
		{"regular comment", false},
		{"@OtherAnnotation", false},
		{"", false}, // Empty line case
	}

	for _, tc := range testCases {
		t.Run(tc.line, func(t *testing.T) {
			result := parser.isServiceSpecStart(tc.line)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDetectCommentPattern(t *testing.T) {
	parser := NewBaseFileParser(LanguageJava)

	testCases := []struct {
		line            string
		expectedPattern string
	}{
		{"/**", "/**"},
		{"/*", "/*"},
		{"// comment", ""},   // Java parser doesn't have // pattern
		{"regular line", ""}, // No pattern
	}

	for _, tc := range testCases {
		t.Run(tc.line, func(t *testing.T) {
			pattern := parser.detectCommentPattern(tc.line)
			if tc.expectedPattern == "" {
				assert.Nil(t, pattern)
			} else {
				assert.NotNil(t, pattern)
				assert.Equal(t, tc.expectedPattern, pattern.StartPattern)
			}
		})
	}
}

func TestCleanCommentLine(t *testing.T) {
	parser := NewBaseFileParser(LanguageJava)
	pattern := &CommentPattern{
		StartPattern: "/**",
		EndPattern:   "*/",
		LinePrefix:   " * ",
		IsMultiLine:  true,
	}

	testCases := []struct {
		line     string
		expected string
	}{
		{"/** @ServiceSpec", " "},
		{" * operationId: \"test\"", " operationId: \"test\""},
		{" * description: \"Test operation\"", " description: \"Test operation\""},
		{" */", " "},
		{"/**", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.line, func(t *testing.T) {
			result := parser.cleanCommentLine(tc.line, pattern)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseAnnotationContent_YAML(t *testing.T) {
	parser := NewBaseFileParser(LanguageJava)
	annotation := &ServiceSpecAnnotation{}

	content := `operationId: "createUser"
description: "Create a new user"
preconditions:
  "request.body.email": {"!=": null}
postconditions:
  "response.status": {"==": 201}`

	err := parser.parseAnnotationContent(content, annotation, "test.java")

	assert.NoError(t, err)
	assert.Equal(t, "createUser", annotation.OperationID)
	assert.Equal(t, "Create a new user", annotation.Description)
	assert.NotNil(t, annotation.Preconditions)
	assert.NotNil(t, annotation.Postconditions)
}

func TestParseAnnotationContent_JSON(t *testing.T) {
	parser := NewBaseFileParser(LanguageJava)
	annotation := &ServiceSpecAnnotation{}

	content := `{
  "operationId": "createUser",
  "description": "Create a new user",
  "preconditions": {
    "request.body.email": {"!=": null}
  },
  "postconditions": {
    "response.status": {"==": 201}
  }
}`

	err := parser.parseAnnotationContent(content, annotation, "test.java")

	assert.NoError(t, err)
	assert.Equal(t, "createUser", annotation.OperationID)
	assert.Equal(t, "Create a new user", annotation.Description)
	assert.NotNil(t, annotation.Preconditions)
	assert.NotNil(t, annotation.Postconditions)
}

func TestParseAnnotationContent_MissingFields(t *testing.T) {
	parser := NewBaseFileParser(LanguageJava)
	annotation := &ServiceSpecAnnotation{}

	// Missing operationId
	content := `description: "Create a new user"`
	err := parser.parseAnnotationContent(content, annotation, "test.java")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing or invalid operationId")

	// Missing description
	content = `operationId: "createUser"`
	err = parser.parseAnnotationContent(content, annotation, "test.java")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing or invalid description")
}

func TestParseAnnotationContent_InvalidFormat(t *testing.T) {
	parser := NewBaseFileParser(LanguageJava)
	annotation := &ServiceSpecAnnotation{}

	content := `invalid yaml content: [unclosed bracket`
	err := parser.parseAnnotationContent(content, annotation, "test.java")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse annotation content")
}

func TestConvertAnnotationToSpec(t *testing.T) {
	parser := NewBaseFileParser(LanguageJava)

	annotation := ServiceSpecAnnotation{
		OperationID: "createUser",
		Description: "Create a new user",
		StartLine:   10,
		Preconditions: map[string]interface{}{
			"request.body.email": map[string]interface{}{"!=": nil},
		},
		Postconditions: map[string]interface{}{
			"response.status": map[string]interface{}{"==": 201},
		},
	}

	spec, err := parser.convertAnnotationToSpec(annotation, "test.java")

	assert.NoError(t, err)
	assert.Equal(t, "createUser", spec.OperationID)
	assert.Equal(t, "Create a new user", spec.Description)
	assert.Equal(t, "test.java", spec.SourceFile)
	assert.Equal(t, 10, spec.LineNumber)
	assert.NotNil(t, spec.Preconditions)
	assert.NotNil(t, spec.Postconditions)
}

func TestConvertAnnotationToSpec_InvalidPreconditions(t *testing.T) {
	parser := NewBaseFileParser(LanguageJava)

	annotation := ServiceSpecAnnotation{
		OperationID:    "createUser",
		Description:    "Create a new user",
		StartLine:      10,
		Preconditions:  "invalid_type", // Should be map[string]interface{}
		Postconditions: map[string]interface{}{},
	}

	_, err := parser.convertAnnotationToSpec(annotation, "test.java")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "preconditions must be a map/object")
}

func TestValidateJSONLogic(t *testing.T) {
	parser := NewBaseFileParser(LanguageJava)

	testCases := []struct {
		name        string
		data        interface{}
		expectError bool
	}{
		{
			name:        "nil data",
			data:        nil,
			expectError: false,
		},
		{
			name: "valid map",
			data: map[string]interface{}{
				"==": []interface{}{1, 1},
			},
			expectError: false,
		},
		{
			name: "valid nested structure",
			data: map[string]interface{}{
				"and": []interface{}{
					map[string]interface{}{"==": []interface{}{1, 1}},
					map[string]interface{}{"!=": []interface{}{2, 3}},
				},
			},
			expectError: false,
		},
		{
			name:        "invalid data (function)",
			data:        func() {},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := parser.ValidateJSONLogic(tc.data)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseFile_JavaMultiLineComment(t *testing.T) {
	parser := NewBaseFileParser(LanguageJava)

	// Create temporary file with Java ServiceSpec annotation
	tmpDir := t.TempDir()
	javaFile := filepath.Join(tmpDir, "Test.java")

	content := `package com.example;

/**
 * @ServiceSpec
 * operationId: "createUser"
 * description: "Create a new user account"
 * preconditions:
 *   "request.body.email": {"!=": null}
 *   "request.body.password": {">=": 8}
 * postconditions:
 *   "response.status": {"==": 201}
 *   "response.body.userId": {"!=": null}
 */
public User createUser(CreateUserRequest request) {
    // Implementation
    return new User();
}`

	err := os.WriteFile(javaFile, []byte(content), 0644)
	require.NoError(t, err)

	specs, errors := parser.ParseFile(javaFile)

	assert.Empty(t, errors)
	assert.Len(t, specs, 1)

	spec := specs[0]
	assert.Equal(t, "createUser", spec.OperationID)
	assert.Equal(t, "Create a new user account", spec.Description)
	assert.Equal(t, javaFile, spec.SourceFile)
	assert.Equal(t, 4, spec.LineNumber) // Line where @ServiceSpec appears
	assert.NotEmpty(t, spec.Preconditions)
	assert.NotEmpty(t, spec.Postconditions)
}

func TestParseFile_GoSingleLineComment(t *testing.T) {
	parser := NewBaseFileParser(LanguageGo)

	// Create temporary file with Go ServiceSpec annotation
	tmpDir := t.TempDir()
	goFile := filepath.Join(tmpDir, "test.go")

	content := `package main

// @ServiceSpec
// operationId: "createUser"
// description: "Create a new user account"
// preconditions:
//   "request.body.email": {"!=": null}
// postconditions:
//   "response.status": {"==": 201}
func CreateUser(request CreateUserRequest) (*User, error) {
    // Implementation
    return &User{}, nil
}`

	err := os.WriteFile(goFile, []byte(content), 0644)
	require.NoError(t, err)

	specs, errors := parser.ParseFile(goFile)

	assert.Empty(t, errors)
	assert.Len(t, specs, 1)

	spec := specs[0]
	assert.Equal(t, "createUser", spec.OperationID)
	assert.Equal(t, "Create a new user account", spec.Description)
	assert.Equal(t, goFile, spec.SourceFile)
	assert.Equal(t, 3, spec.LineNumber) // Line where @ServiceSpec appears
}

func TestParseFile_MultipleAnnotations(t *testing.T) {
	parser := NewBaseFileParser(LanguageJava)

	// Create temporary file with multiple ServiceSpec annotations
	tmpDir := t.TempDir()
	javaFile := filepath.Join(tmpDir, "Test.java")

	content := `package com.example;

/**
 * @ServiceSpec
 * operationId: "createUser"
 * description: "Create a new user"
 */
public User createUser() { return null; }

/**
 * @ServiceSpec
 * operationId: "updateUser"
 * description: "Update existing user"
 */
public User updateUser() { return null; }`

	err := os.WriteFile(javaFile, []byte(content), 0644)
	require.NoError(t, err)

	specs, errors := parser.ParseFile(javaFile)

	assert.Empty(t, errors)
	assert.Len(t, specs, 2)

	assert.Equal(t, "createUser", specs[0].OperationID)
	assert.Equal(t, "updateUser", specs[1].OperationID)
}

func TestParseFile_InvalidAnnotation(t *testing.T) {
	parser := NewBaseFileParser(LanguageJava)

	// Create temporary file with invalid ServiceSpec annotation
	tmpDir := t.TempDir()
	javaFile := filepath.Join(tmpDir, "Test.java")

	content := `package com.example;

/**
 * @ServiceSpec
 * operationId: "createUser"
 * // Missing description field - this will cause YAML parse error
 */
public User createUser() { return null; }`

	err := os.WriteFile(javaFile, []byte(content), 0644)
	require.NoError(t, err)

	specs, errors := parser.ParseFile(javaFile)

	assert.Empty(t, specs)
	assert.Len(t, errors, 1)
	assert.Contains(t, errors[0].Message, "failed to parse annotation content as YAML or JSON")
}

func TestParseFile_FileNotFound(t *testing.T) {
	parser := NewBaseFileParser(LanguageJava)

	specs, errors := parser.ParseFile("/non/existent/file.java")

	assert.Empty(t, specs)
	assert.Len(t, errors, 1)
	assert.Contains(t, errors[0].Message, "failed to open file")
}

func TestParseFile_EmptyFile(t *testing.T) {
	parser := NewBaseFileParser(LanguageJava)

	// Create empty file
	tmpDir := t.TempDir()
	javaFile := filepath.Join(tmpDir, "Empty.java")
	err := os.WriteFile(javaFile, []byte(""), 0644)
	require.NoError(t, err)

	specs, errors := parser.ParseFile(javaFile)

	assert.Empty(t, specs)
	assert.Empty(t, errors)
}

func TestGetLanguage(t *testing.T) {
	parser := NewBaseFileParser(LanguageJava)
	assert.Equal(t, LanguageJava, parser.GetLanguage())

	parser = NewBaseFileParser(LanguageGo)
	assert.Equal(t, LanguageGo, parser.GetLanguage())
}