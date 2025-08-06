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
	"testing"

	"github.com/flowspec/flowspec-cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockFileParser implements FileParser for testing
type MockFileParser struct {
	canParseFunc  func(filename string) bool
	parseFileFunc func(filepath string) ([]models.ServiceSpec, []models.ParseError)
}

func (m *MockFileParser) CanParse(filename string) bool {
	if m.canParseFunc != nil {
		return m.canParseFunc(filename)
	}
	return false
}

func (m *MockFileParser) ParseFile(filepath string) ([]models.ServiceSpec, []models.ParseError) {
	if m.parseFileFunc != nil {
		return m.parseFileFunc(filepath)
	}
	return []models.ServiceSpec{}, []models.ParseError{}
}

func TestNewSpecParser(t *testing.T) {
	parser := NewSpecParser()

	assert.NotNil(t, parser)
	assert.Equal(t, 4, parser.maxWorkers)
	assert.NotNil(t, parser.fileParsers)
	assert.NotNil(t, parser.supportedTypes)
	assert.Equal(t, 3, parser.GetFileCount()) // Java, TypeScript, and Go parsers registered by default
}

func TestNewSpecParserWithConfig(t *testing.T) {
	config := &ParserConfig{
		MaxWorkers: 8,
		SkipHidden: true,
		SkipVendor: true,
	}

	parser := NewSpecParserWithConfig(config)

	assert.NotNil(t, parser)
	assert.Equal(t, 8, parser.maxWorkers)
}

func TestDefaultParserConfig(t *testing.T) {
	config := DefaultParserConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 4, config.MaxWorkers)
	assert.True(t, config.SkipHidden)
	assert.True(t, config.SkipVendor)
	assert.Equal(t, int64(10*1024*1024), config.MaxFileSize)
	assert.Contains(t, config.ExcludedDirs, "node_modules")
	assert.Contains(t, config.ExcludedDirs, "vendor")
	assert.Contains(t, config.ExcludedDirs, ".git")
}

func TestRegisterFileParser(t *testing.T) {
	parser := NewSpecParser()
	mockParser := &MockFileParser{}

	parser.RegisterFileParser(LanguageJava, mockParser)

	retrievedParser, exists := parser.GetFileParser(LanguageJava)
	assert.True(t, exists)
	assert.Equal(t, mockParser, retrievedParser)
	assert.Equal(t, 3, parser.GetFileCount()) // Java, TypeScript, and Go are already registered
}

func TestGetFileParser_NotExists(t *testing.T) {
	parser := NewSpecParser()

	retrievedParser, exists := parser.GetFileParser(SupportedLanguage("python")) // Use Python which is not supported
	assert.False(t, exists)
	assert.Nil(t, retrievedParser)
}

func TestGetSupportedLanguages(t *testing.T) {
	parser := NewSpecParser()
	languages := parser.GetSupportedLanguages()

	assert.Contains(t, languages, LanguageJava)
	assert.Contains(t, languages, LanguageTypeScript)
	assert.Contains(t, languages, LanguageGo)
	assert.Len(t, languages, 3)
}

func TestGetSupportedExtensions(t *testing.T) {
	parser := NewSpecParser()
	extensions := parser.GetSupportedExtensions()

	assert.Contains(t, extensions, ".java")
	assert.Contains(t, extensions, ".ts")
	assert.Contains(t, extensions, ".tsx")
	assert.Contains(t, extensions, ".go")
	assert.Len(t, extensions, 4)
}

func TestIsLanguageSupported(t *testing.T) {
	parser := NewSpecParser()

	assert.True(t, parser.IsLanguageSupported(LanguageJava))
	assert.True(t, parser.IsLanguageSupported(LanguageTypeScript))
	assert.True(t, parser.IsLanguageSupported(LanguageGo))
	assert.False(t, parser.IsLanguageSupported(SupportedLanguage("python")))
}

func TestIsSupportedFile(t *testing.T) {
	parser := NewSpecParser()

	testCases := []struct {
		filename string
		expected bool
	}{
		{"test.java", true},
		{"test.ts", true},
		{"test.tsx", true},
		{"test.go", true},
		{"test.py", false},
		{"test.js", false},
		{"test.txt", false},
		{"test", false},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			result := parser.isSupportedFile(tc.filename)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetLanguageFromFile(t *testing.T) {
	parser := NewSpecParser()

	testCases := []struct {
		filename     string
		expectedLang SupportedLanguage
		expectError  bool
	}{
		{"test.java", LanguageJava, false},
		{"test.ts", LanguageTypeScript, false},
		{"test.tsx", LanguageTypeScript, false},
		{"test.go", LanguageGo, false},
		{"test.py", "", true},
		{"test.js", "", true},
		{"test", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			lang, err := parser.getLanguageFromFile(tc.filename)
			if tc.expectError {
				assert.Error(t, err)
				assert.Empty(t, lang)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedLang, lang)
			}
		})
	}
}

func TestShouldSkipDirectory(t *testing.T) {
	parser := NewSpecParser()

	testCases := []struct {
		path     string
		name     string
		expected bool
	}{
		{"/project/src", "src", false},
		{"/project/.git", ".git", true},
		{"/project/node_modules", "node_modules", true},
		{"/project/vendor", "vendor", true},
		{"/project/build", "build", true},
		{"/project/dist", "dist", true},
		{"/project/target", "target", true},
		{"/project/.idea", ".idea", true},
		{"/project/.vscode", ".vscode", true},
		{"/project/normal", "normal", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parser.shouldSkipDirectory(tc.path, tc.name)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestShouldProcessFile(t *testing.T) {
	parser := NewSpecParser()

	// Create temporary files for testing
	tmpDir := t.TempDir()

	// Create a small supported file
	smallFile := filepath.Join(tmpDir, "small.java")
	err := os.WriteFile(smallFile, []byte("small content"), 0644)
	require.NoError(t, err)

	// Create a large file (simulate by checking file info)
	largeFile := filepath.Join(tmpDir, "large.java")
	err = os.WriteFile(largeFile, make([]byte, 11*1024*1024), 0644) // 11MB
	require.NoError(t, err)

	// Create an unsupported file
	unsupportedFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(unsupportedFile, []byte("content"), 0644)
	require.NoError(t, err)

	testCases := []struct {
		filename string
		expected bool
	}{
		{smallFile, true},
		{largeFile, false},       // Too large
		{unsupportedFile, false}, // Unsupported extension
	}

	for _, tc := range testCases {
		t.Run(filepath.Base(tc.filename), func(t *testing.T) {
			info, err := os.Stat(tc.filename)
			require.NoError(t, err)

			result := parser.shouldProcessFile(tc.filename, info)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseFromSource_InvalidPath(t *testing.T) {
	parser := NewSpecParser()

	// Test empty path
	result, err := parser.ParseFromSource("")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "source path cannot be empty")

	// Test non-existent path
	result, err = parser.ParseFromSource("/non/existent/path")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to access source path")

	// Test file instead of directory
	tmpFile := filepath.Join(t.TempDir(), "test.java")
	err = os.WriteFile(tmpFile, []byte("content"), 0644)
	require.NoError(t, err)

	result, err = parser.ParseFromSource(tmpFile)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "is not a directory")
}

func TestParseFromSource_EmptyDirectory(t *testing.T) {
	parser := NewSpecParser()
	tmpDir := t.TempDir()

	result, err := parser.ParseFromSource(tmpDir)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Specs)
	assert.Empty(t, result.Errors)
}

func TestParseFromSource_WithMockParser(t *testing.T) {
	parser := NewSpecParser()

	// Create test directory structure
	tmpDir := t.TempDir()
	javaFile := filepath.Join(tmpDir, "Test.java")
	err := os.WriteFile(javaFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// Register mock parser
	mockParser := &MockFileParser{
		canParseFunc: func(filename string) bool {
			return filepath.Ext(filename) == ".java"
		},
		parseFileFunc: func(filepath string) ([]models.ServiceSpec, []models.ParseError) {
			return []models.ServiceSpec{
				{
					OperationID: "testOp",
					Description: "Test operation",
					SourceFile:  filepath,
					LineNumber:  1,
				},
			}, []models.ParseError{}
		},
	}

	parser.RegisterFileParser(LanguageJava, mockParser)

	result, err := parser.ParseFromSource(tmpDir)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Specs, 1)
	assert.Equal(t, "testOp", result.Specs[0].OperationID)
	assert.Empty(t, result.Errors)
}

func TestParseFromSource_WithErrors(t *testing.T) {
	parser := NewSpecParser()

	// Create test directory structure
	tmpDir := t.TempDir()
	javaFile := filepath.Join(tmpDir, "Test.java")
	err := os.WriteFile(javaFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// Register mock parser that returns errors
	mockParser := &MockFileParser{
		canParseFunc: func(filename string) bool {
			return filepath.Ext(filename) == ".java"
		},
		parseFileFunc: func(filepath string) ([]models.ServiceSpec, []models.ParseError) {
			return []models.ServiceSpec{}, []models.ParseError{
				{
					File:    filepath,
					Line:    10,
					Message: "Parse error occurred",
				},
			}
		},
	}

	parser.RegisterFileParser(LanguageJava, mockParser)

	result, err := parser.ParseFromSource(tmpDir)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Specs)
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, "Parse error occurred", result.Errors[0].Message)
}

func TestParseFromSource_NoParserRegistered(t *testing.T) {
	parser := NewSpecParser()

	// Create test directory structure
	tmpDir := t.TempDir()
	javaFile := filepath.Join(tmpDir, "Test.java")
	err := os.WriteFile(javaFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// Create a Python file instead (no parser registered for Python)
	pyFile := filepath.Join(tmpDir, "test.py")
	err = os.WriteFile(pyFile, []byte("test content"), 0644)
	require.NoError(t, err)

	result, err := parser.ParseFromSource(tmpDir)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Specs)
	assert.Empty(t, result.Errors) // Python files are skipped, not errors
}

func TestErrorCollector(t *testing.T) {
	collector := NewErrorCollector()

	assert.NotNil(t, collector)
	assert.False(t, collector.HasErrors())
	assert.Equal(t, 0, collector.Count())
	assert.Empty(t, collector.GetErrors())

	// Add an error
	collector.AddError("test.java", 10, "Test error")

	assert.True(t, collector.HasErrors())
	assert.Equal(t, 1, collector.Count())

	errors := collector.GetErrors()
	assert.Len(t, errors, 1)
	assert.Equal(t, "test.java", errors[0].File)
	assert.Equal(t, 10, errors[0].Line)
	assert.Equal(t, "Test error", errors[0].Message)

	// Add formatted error
	collector.AddErrorf("test2.java", 20, "Error with %s: %d", "value", 42)

	assert.Equal(t, 2, collector.Count())
	errors = collector.GetErrors()
	assert.Len(t, errors, 2)
	assert.Equal(t, "Error with value: 42", errors[1].Message)

	// Clear errors
	collector.Clear()
	assert.False(t, collector.HasErrors())
	assert.Equal(t, 0, collector.Count())
	assert.Empty(t, collector.GetErrors())
}

func TestScanFiles_WithDirectoryStructure(t *testing.T) {
	parser := NewSpecParser()

	// Create test directory structure
	tmpDir := t.TempDir()

	// Create valid files
	javaFile := filepath.Join(tmpDir, "Test.java")
	err := os.WriteFile(javaFile, []byte("java content"), 0644)
	require.NoError(t, err)

	tsFile := filepath.Join(tmpDir, "test.ts")
	err = os.WriteFile(tsFile, []byte("ts content"), 0644)
	require.NoError(t, err)

	// Create subdirectory with files
	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	goFile := filepath.Join(subDir, "test.go")
	err = os.WriteFile(goFile, []byte("go content"), 0644)
	require.NoError(t, err)

	// Create excluded directory
	nodeModules := filepath.Join(tmpDir, "node_modules")
	err = os.Mkdir(nodeModules, 0755)
	require.NoError(t, err)

	excludedFile := filepath.Join(nodeModules, "excluded.java")
	err = os.WriteFile(excludedFile, []byte("excluded content"), 0644)
	require.NoError(t, err)

	// Create unsupported file
	txtFile := filepath.Join(tmpDir, "readme.txt")
	err = os.WriteFile(txtFile, []byte("readme content"), 0644)
	require.NoError(t, err)

	files, err := parser.scanFiles(tmpDir)
	assert.NoError(t, err)

	// Should find 3 supported files, excluding the one in node_modules and the .txt file
	assert.Len(t, files, 3)

	// Convert to relative paths for easier testing
	var relativeFiles []string
	for _, file := range files {
		rel, _ := filepath.Rel(tmpDir, file)
		relativeFiles = append(relativeFiles, rel)
	}

	assert.Contains(t, relativeFiles, "Test.java")
	assert.Contains(t, relativeFiles, "test.ts")
	assert.Contains(t, relativeFiles, filepath.Join("subdir", "test.go"))
	assert.NotContains(t, relativeFiles, filepath.Join("node_modules", "excluded.java"))
	assert.NotContains(t, relativeFiles, "readme.txt")
}

func TestConcurrentParsing(t *testing.T) {
	parser := NewSpecParser()

	// Create test directory with multiple files
	tmpDir := t.TempDir()

	// Create multiple Java files
	for i := 0; i < 10; i++ {
		filename := filepath.Join(tmpDir, fmt.Sprintf("Test%d.java", i))
		err := os.WriteFile(filename, []byte("java content"), 0644)
		require.NoError(t, err)
	}

	// Register mock parser
	mockParser := &MockFileParser{
		canParseFunc: func(filename string) bool {
			return filepath.Ext(filename) == ".java"
		},
		parseFileFunc: func(filepath string) ([]models.ServiceSpec, []models.ParseError) {
			return []models.ServiceSpec{
				{
					OperationID: "testOp",
					Description: "Test operation",
					SourceFile:  filepath,
					LineNumber:  1,
				},
			}, []models.ParseError{}
		},
	}

	parser.RegisterFileParser(LanguageJava, mockParser)

	result, err := parser.ParseFromSource(tmpDir)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Specs, 10) // Should parse all 10 files
	assert.Empty(t, result.Errors)
}
