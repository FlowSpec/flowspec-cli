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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParserPerformance_LargeProject(t *testing.T) {
	// Create a temporary directory with many files
	tmpDir := t.TempDir()

	// Create 100 Java files with ServiceSpec annotations
	for i := 0; i < 100; i++ {
		javaFile := filepath.Join(tmpDir, fmt.Sprintf("Service%d.java", i))
		content := fmt.Sprintf(`package com.example.service%d;

/**
 * @ServiceSpec
 * operationId: "operation%d"
 * description: "Test operation %d"
 * preconditions:
 *   "request.valid": {"==": [true, true]}
 * postconditions:
 *   "response.status": {"==": [200, 200]}
 */
public class Service%d {
    public void operation%d() {}
}`, i, i, i, i, i)

		err := os.WriteFile(javaFile, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Create 50 TypeScript files
	for i := 0; i < 50; i++ {
		tsFile := filepath.Join(tmpDir, fmt.Sprintf("service%d.ts", i))
		content := fmt.Sprintf(`/**
 * @ServiceSpec
 * operationId: "tsOperation%d"
 * description: "TypeScript operation %d"
 * preconditions:
 *   "request.valid": {"==": [true, true]}
 * postconditions:
 *   "response.status": {"==": [200, 200]}
 */
export function tsOperation%d() {}`, i, i, i)

		err := os.WriteFile(tsFile, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Create 50 Go files
	for i := 0; i < 50; i++ {
		goFile := filepath.Join(tmpDir, fmt.Sprintf("service%d.go", i))
		content := fmt.Sprintf(`package service%d

// @ServiceSpec
// operationId: "goOperation%d"
// description: "Go operation %d"
// preconditions:
//   "request.valid": {"==": [true, true]}
// postconditions:
//   "response.status": {"==": [200, 200]}
func GoOperation%d() {}`, i, i, i, i)

		err := os.WriteFile(goFile, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Test parsing performance
	parser := NewSpecParser()

	startTime := time.Now()
	result, err := parser.ParseFromSource(tmpDir)
	duration := time.Since(startTime)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify results
	assert.Equal(t, 200, len(result.Specs)) // 100 Java + 50 TS + 50 Go
	assert.Empty(t, result.Errors)

	// Performance assertions
	assert.Less(t, duration, 30*time.Second, "Parsing should complete within 30 seconds")

	// Check metrics
	assert.NotNil(t, result.Metrics)
	assert.Equal(t, 200, result.Metrics["total_files"])
	assert.Equal(t, 200, result.Metrics["processed_files"])
	assert.Equal(t, 200, result.Metrics["total_specs"])
	assert.Equal(t, 0, result.Metrics["total_errors"])

	filesPerSecond := result.Metrics["files_per_second"].(float64)
	assert.Greater(t, filesPerSecond, 5.0, "Should process at least 5 files per second")

	t.Logf("Performance metrics:")
	t.Logf("  - Total files: %d", result.Metrics["total_files"])
	t.Logf("  - Parse duration: %s", result.Metrics["parse_duration"])
	t.Logf("  - Files per second: %.2f", filesPerSecond)
}

func TestParserCache_Performance(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a few test files
	for i := 0; i < 10; i++ {
		javaFile := filepath.Join(tmpDir, fmt.Sprintf("Service%d.java", i))
		content := fmt.Sprintf(`/**
 * @ServiceSpec
 * operationId: "operation%d"
 * description: "Test operation %d"
 */
public void operation%d() {}`, i, i, i)

		err := os.WriteFile(javaFile, []byte(content), 0644)
		require.NoError(t, err)
	}

	parser := NewSpecParser()

	// First parse (cache miss)
	start1 := time.Now()
	result1, err := parser.ParseFromSource(tmpDir)
	duration1 := time.Since(start1)

	require.NoError(t, err)
	assert.Len(t, result1.Specs, 10)

	// Second parse (should use cache)
	start2 := time.Now()
	result2, err := parser.ParseFromSource(tmpDir)
	duration2 := time.Since(start2)

	require.NoError(t, err)
	assert.Len(t, result2.Specs, 10)

	// Cache should make second parse faster
	assert.Less(t, duration2, duration1, "Cached parse should be faster")

	t.Logf("First parse: %s", duration1)
	t.Logf("Second parse (cached): %s", duration2)
	t.Logf("Cache size: %d", parser.GetCacheSize())
}

func TestParserConcurrency_Safety(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	for i := 0; i < 20; i++ {
		javaFile := filepath.Join(tmpDir, fmt.Sprintf("Service%d.java", i))
		content := fmt.Sprintf(`/**
 * @ServiceSpec
 * operationId: "operation%d"
 * description: "Test operation %d"
 */
public void operation%d() {}`, i, i, i)

		err := os.WriteFile(javaFile, []byte(content), 0644)
		require.NoError(t, err)
	}

	parser := NewSpecParser()

	// Run multiple concurrent parses
	const numGoroutines = 5
	res := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			result, err := parser.ParseFromSource(tmpDir)
			if err != nil {
				res <- err
				return
			}
			if len(result.Specs) != 20 {
				res <- fmt.Errorf("expected 20 specs, got %d", len(result.Specs))
				return
			}
			res <- nil
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		err := <-res
		assert.NoError(t, err)
	}
}

func TestParserMemoryUsage(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a large file (but within limits)
	largeFile := filepath.Join(tmpDir, "LargeService.java")
	content := `package com.example;

/**
 * @ServiceSpec
 * operationId: "largeOperation"
 * description: "Large file operation"
 * preconditions:
 *   "request.valid": {"==": [true, true]}
 * postconditions:
 *   "response.status": {"==": [200, 200]}
 */
public class LargeService {
    public void largeOperation() {`

	// Add many lines to make it large but not exceed 10MB limit
	for i := 0; i < 10000; i++ {
		content += fmt.Sprintf("\n        // Line %d", i)
	}
	content += "\n    }\n}"

	err := os.WriteFile(largeFile, []byte(content), 0644)
	require.NoError(t, err)

	// Verify file size is reasonable
	info, err := os.Stat(largeFile)
	require.NoError(t, err)
	assert.Less(t, info.Size(), int64(5*1024*1024), "File should be less than 5MB")

	parser := NewSpecParser()
	result, err := parser.ParseFromSource(tmpDir)

	require.NoError(t, err)
	assert.Len(t, result.Specs, 1)
	assert.Equal(t, "largeOperation", result.Specs[0].OperationID)
}

func TestParserResourceLimits(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file that exceeds size limit
	largeFile := filepath.Join(tmpDir, "TooLarge.java")
	content := make([]byte, 11*1024*1024) // 11MB - exceeds 10MB limit
	for i := range content {
		content[i] = 'a'
	}

	err := os.WriteFile(largeFile, content, 0644)
	require.NoError(t, err)

	parser := NewSpecParser()
	result, err := parser.ParseFromSource(tmpDir)

	require.NoError(t, err)
	// Large file should be skipped, not cause errors
	assert.Empty(t, result.Specs)
	assert.Empty(t, result.Errors)
}

func BenchmarkParserPerformance(b *testing.B) {
	tmpDir := b.TempDir()

	// Create test files
	for i := 0; i < 50; i++ {
		javaFile := filepath.Join(tmpDir, fmt.Sprintf("Service%d.java", i))
		content := fmt.Sprintf(`/**
 * @ServiceSpec
 * operationId: "operation%d"
 * description: "Test operation %d"
 */
public void operation%d() {}`, i, i, i)

		err := os.WriteFile(javaFile, []byte(content), 0644)
		require.NoError(b, err)
	}

	parser := NewSpecParser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := parser.ParseFromSource(tmpDir)
		require.NoError(b, err)
		require.Len(b, result.Specs, 50)
	}
}

func BenchmarkParserWithCache(b *testing.B) {
	tmpDir := b.TempDir()

	// Create test files
	for i := 0; i < 50; i++ {
		javaFile := filepath.Join(tmpDir, fmt.Sprintf("Service%d.java", i))
		content := fmt.Sprintf(`/**
 * @ServiceSpec
 * operationId: "operation%d"
 * description: "Test operation %d"
 */
public void operation%d() {}`, i, i, i)

		err := os.WriteFile(javaFile, []byte(content), 0644)
		require.NoError(b, err)
	}

	config := DefaultParserConfig()
	config.EnableCache = true
	parser := NewSpecParserWithConfig(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := parser.ParseFromSource(tmpDir)
		require.NoError(b, err)
		require.Len(b, result.Specs, 50)
	}
}