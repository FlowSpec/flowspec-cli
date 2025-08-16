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

package models

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractMeaningfulValues_ComprehensiveCoverage provides comprehensive test coverage
// This implements task 4.2: 补齐 extractMeaningfulValues 数据提取测试
func TestExtractMeaningfulValues_ComprehensiveCoverage(t *testing.T) {
	testCases := []struct {
		name        string
		span        *Span
		options     *ExtractionOptions
		expectError bool
		validate    func(*testing.T, *ExtractedValues)
		description string
	}{
		{
			name: "normal_data_extraction",
			span: createTestSpan("span-1", "test-operation", map[string]interface{}{
				"http.method":      "GET",
				"http.status_code": 200,
				"user.id":          "user_12345", // Make it clearly a string
				"request.size":     1024,
				"is_authenticated": true,
			}),
			options: DefaultExtractionOptions(),
			validate: func(t *testing.T, ev *ExtractedValues) {
				assert.Equal(t, "span-1", ev.SpanID)
				assert.Equal(t, "test-operation", ev.Name)
				assert.Equal(t, "GET", ev.StringValues["httpMethod"])
				assert.Equal(t, float64(200), ev.NumericValues["httpStatusCode"])
				assert.Equal(t, "user_12345", ev.StringValues["userId"])
				assert.Equal(t, float64(1024), ev.NumericValues["requestSize"])
				assert.Equal(t, true, ev.BooleanValues["isAuthenticated"])
			},
			description: "Tests normal data extraction with various data types",
		},
		{
			name: "complex_nested_data",
			span: createTestSpan("span-2", "complex-operation", map[string]interface{}{
				"nested.object": map[string]interface{}{
					"key1": "value1",
					"key2": 42,
				},
				"array_data": []interface{}{"item1", "item2", "item3"},
				"deep.nested.value": "deep_value",
			}),
			options: DefaultExtractionOptions(),
			validate: func(t *testing.T, ev *ExtractedValues) {
				assert.Contains(t, ev.ObjectValues, "nestedObject")
				assert.Contains(t, ev.ArrayValues, "arrayData")
				assert.Equal(t, "deep_value", ev.StringValues["deepNestedValue"])
				assert.Len(t, ev.ArrayValues["arrayData"], 3)
			},
			description: "Tests extraction of complex nested data structures",
		},
		{
			name: "type_coercion_enabled",
			span: createTestSpan("span-3", "coercion-test", map[string]interface{}{
				"string_number":  "123",
				"string_float":   "45.67",
				"string_boolean": "true",
				"string_false":   "false",
				"timestamp":      "1640995200", // Unix timestamp
			}),
			options: &ExtractionOptions{
				IncludeAttributes: true,
				TypeCoercion:      true,
				NormalizeKeys:     true,
				MaxDepth:          5,
				MaxArraySize:      100,
			},
			validate: func(t *testing.T, ev *ExtractedValues) {
				assert.Equal(t, float64(123), ev.NumericValues["stringNumber"])
				assert.Equal(t, 45.67, ev.NumericValues["stringFloat"])
				assert.Equal(t, true, ev.BooleanValues["stringBoolean"])
				assert.Equal(t, false, ev.BooleanValues["stringFalse"])
			},
			description: "Tests type coercion from string values",
		},
		{
			name: "attribute_filtering",
			span: createTestSpan("span-4", "filtered-operation", map[string]interface{}{
				"include_this":  "should_be_included",
				"exclude_this":  "should_be_excluded",
				"another_field": "normal_field",
			}),
			options: &ExtractionOptions{
				IncludeAttributes: true,
				AttributeFilter:   []string{"include"},
				ExcludeFilter:     []string{"exclude"},
				NormalizeKeys:     true,
				MaxDepth:          5,
				MaxArraySize:      100,
			},
			validate: func(t *testing.T, ev *ExtractedValues) {
				assert.Contains(t, ev.StringValues, "includeThis")
				assert.NotContains(t, ev.StringValues, "excludeThis")
				assert.NotContains(t, ev.StringValues, "anotherField") // Not in include filter
			},
			description: "Tests attribute filtering with include/exclude lists",
		},
		{
			name: "large_array_truncation",
			span: createTestSpan("span-5", "large-array-test", map[string]interface{}{
				"large_array": createLargeArray(150), // Exceeds default MaxArraySize of 100
			}),
			options: &ExtractionOptions{
				IncludeAttributes: true,
				MaxArraySize:      50, // Smaller limit
				MaxDepth:          5,
			},
			validate: func(t *testing.T, ev *ExtractedValues) {
				assert.Contains(t, ev.ArrayValues, "large_array")
				assert.Len(t, ev.ArrayValues["large_array"], 50) // Truncated to MaxArraySize
			},
			description: "Tests truncation of large arrays",
		},
		{
			name: "deep_nesting_limit",
			span: createTestSpan("span-6", "deep-nesting-test", map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"level3": map[string]interface{}{
							"level4": map[string]interface{}{
								"level5": map[string]interface{}{
									"level6": "too_deep",
								},
							},
						},
					},
				},
			}),
			options: &ExtractionOptions{
				IncludeAttributes: true,
				MaxDepth:          3, // Limit depth
				MaxArraySize:      100,
			},
			validate: func(t *testing.T, ev *ExtractedValues) {
				assert.Contains(t, ev.ObjectValues, "level1")
				// Should not extract beyond MaxDepth
			},
			description: "Tests deep nesting depth limits",
		},
		{
			name: "event_extraction",
			span: createTestSpanWithEvents("span-7", "event-test", 
				map[string]interface{}{"base_attr": "value"},
				[]SpanEvent{
					{
						Name:      "event1",
						Timestamp: time.Now().UnixNano(),
						Attributes: map[string]interface{}{
							"event_attr": "event_value",
						},
					},
					{
						Name:      "event2", 
						Timestamp: time.Now().UnixNano() + 1000000,
						Attributes: map[string]interface{}{
							"another_attr": 42,
						},
					},
				}),
			options: DefaultExtractionOptions(),
			validate: func(t *testing.T, ev *ExtractedValues) {
				assert.Len(t, ev.Events, 2)
				assert.Equal(t, "event1", ev.Events[0].Name)
				assert.Equal(t, "event2", ev.Events[1].Name)
				assert.Contains(t, ev.Events[0].Attributes, "eventAttr")
				assert.Contains(t, ev.Events[1].Attributes, "anotherAttr")
			},
			description: "Tests extraction of span events",
		},
		{
			name: "empty_span_attributes",
			span: createTestSpan("span-8", "empty-test", map[string]interface{}{}),
			options: DefaultExtractionOptions(),
			validate: func(t *testing.T, ev *ExtractedValues) {
				assert.Equal(t, "span-8", ev.SpanID)
				assert.Equal(t, "empty-test", ev.Name)
				assert.Empty(t, ev.StringValues)
				assert.Empty(t, ev.NumericValues)
				assert.Empty(t, ev.BooleanValues)
			},
			description: "Tests handling of spans with no attributes",
		},
		{
			name: "nil_and_empty_values",
			span: createTestSpan("span-9", "nil-test", map[string]interface{}{
				"nil_value":   nil,
				"empty_string": "",
				"zero_number":  0,
				"false_bool":   false,
			}),
			options: DefaultExtractionOptions(),
			validate: func(t *testing.T, ev *ExtractedValues) {
				assert.Equal(t, "", ev.StringValues["nilValue"])
				assert.Equal(t, "", ev.StringValues["emptyString"])
				assert.Equal(t, float64(0), ev.NumericValues["zeroNumber"])
				assert.Equal(t, false, ev.BooleanValues["falseBool"])
			},
			description: "Tests handling of nil and empty values",
		},
		{
			name: "special_characters_in_keys",
			span: createTestSpan("span-10", "special-chars-test", map[string]interface{}{
				"key-with-dashes":     "value1",
				"key_with_underscores": "value2",
				"key.with.dots":       "value3",
				"key with spaces":     "value4",
				"MixedCaseKey":        "value5",
			}),
			options: DefaultExtractionOptions(),
			validate: func(t *testing.T, ev *ExtractedValues) {
				assert.Contains(t, ev.StringValues, "keyWithDashes")
				assert.Contains(t, ev.StringValues, "keyWithUnderscores")
				assert.Contains(t, ev.StringValues, "keyWithDots")
				assert.Contains(t, ev.StringValues, "keyWithSpaces")
				assert.Contains(t, ev.StringValues, "mixedcasekey")
			},
			description: "Tests key normalization with special characters",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Execute extraction
			result, err := ExtractMeaningfulValues(tc.span, tc.options)

			// Verify error expectation
			if tc.expectError {
				assert.Error(t, err, tc.description)
				return
			}

			require.NoError(t, err, tc.description)
			require.NotNil(t, result, tc.description)

			// Validate basic properties
			assert.NotEmpty(t, result.SpanID, tc.description)
			assert.NotEmpty(t, result.Name, tc.description)
			assert.NotZero(t, result.ExtractedAt, tc.description)

			// Run custom validation
			if tc.validate != nil {
				tc.validate(t, result)
			}

			// Validate the result
			err = result.Validate()
			assert.NoError(t, err, tc.description)
		})
	}
}

// TestExtractMeaningfulValues_ErrorCases tests various error conditions
func TestExtractMeaningfulValues_ErrorCases(t *testing.T) {
	testCases := []struct {
		name        string
		span        *Span
		options     *ExtractionOptions
		expectedErr string
		description string
	}{
		{
			name:        "nil_span",
			span:        nil,
			options:     DefaultExtractionOptions(),
			expectedErr: "span cannot be nil",
			description: "Tests error handling for nil span",
		},
		{
			name: "invalid_max_depth",
			span: createTestSpan("span-1", "test", map[string]interface{}{
				"deep": map[string]interface{}{
					"nested": map[string]interface{}{
						"value": "too_deep",
					},
				},
			}),
			options: &ExtractionOptions{
				IncludeAttributes: true,
				MaxDepth:          1, // Should fail at depth 2 (deep.nested)
				MaxArraySize:      100,
			},
			expectedErr: "maximum depth exceeded",
			description: "Tests error handling for invalid max depth",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ExtractMeaningfulValues(tc.span, tc.options)

			require.Error(t, err, tc.description)
			assert.Contains(t, err.Error(), tc.expectedErr, tc.description)
			assert.Nil(t, result, tc.description)
		})
	}
}

// TestExtractMeaningfulValuesFromMultipleSpans tests batch extraction
func TestExtractMeaningfulValuesFromMultipleSpans(t *testing.T) {
	spans := []*Span{
		createTestSpan("span-1", "operation-1", map[string]interface{}{"attr1": "value1"}),
		createTestSpan("span-2", "operation-2", map[string]interface{}{"attr2": "value2"}),
		createTestSpan("span-3", "operation-3", map[string]interface{}{"attr3": "value3"}),
	}

	results, err := ExtractMeaningfulValuesFromMultipleSpans(spans, DefaultExtractionOptions())

	require.NoError(t, err)
	assert.Len(t, results, 3)

	for i, result := range results {
		assert.Equal(t, spans[i].SpanID, result.SpanID)
		assert.Equal(t, spans[i].Name, result.Name)
	}
}

// TestExtractMeaningfulValues_Performance benchmarks extraction performance
// This implements the performance testing requirement from task 4.2
func BenchmarkExtractMeaningfulValues_Performance(b *testing.B) {
	benchmarks := []struct {
		name        string
		span        *Span
		options     *ExtractionOptions
		description string
	}{
		{
			name:        "simple_attributes",
			span:        createTestSpan("bench-1", "simple", createSimpleAttributes(10)),
			options:     DefaultExtractionOptions(),
			description: "Benchmark simple attribute extraction",
		},
		{
			name:        "complex_attributes",
			span:        createTestSpan("bench-2", "complex", createComplexAttributes(50)),
			options:     DefaultExtractionOptions(),
			description: "Benchmark complex attribute extraction",
		},
		{
			name:        "large_attributes",
			span:        createTestSpan("bench-3", "large", createSimpleAttributes(1000)),
			options:     DefaultExtractionOptions(),
			description: "Benchmark large number of attributes",
		},
		{
			name: "with_type_coercion",
			span: createTestSpan("bench-4", "coercion", createStringAttributes(100)),
			options: &ExtractionOptions{
				IncludeAttributes: true,
				TypeCoercion:      true,
				NormalizeKeys:     true,
				MaxDepth:          5,
				MaxArraySize:      100,
			},
			description: "Benchmark with type coercion enabled",
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				result, err := ExtractMeaningfulValues(bm.span, bm.options)
				if err != nil {
					b.Fatalf("Extraction failed: %v", err)
				}
				if result == nil {
					b.Fatal("Result is nil")
				}
			}
		})
	}
}

// TestExtractMeaningfulValues_MemoryUsage tests memory usage patterns
// This implements the memory usage testing requirement from task 4.2
func TestExtractMeaningfulValues_MemoryUsage(t *testing.T) {
	testCases := []struct {
		name         string
		span         *Span
		options      *ExtractionOptions
		maxMemoryMB  float64
		description  string
	}{
		{
			name:         "small_span_memory",
			span:         createTestSpan("mem-1", "small", createSimpleAttributes(10)),
			options:      DefaultExtractionOptions(),
			maxMemoryMB:  1.0,
			description:  "Tests memory usage for small span",
		},
		{
			name:         "large_span_memory",
			span:         createTestSpan("mem-2", "large", createComplexAttributes(100)),
			options:      DefaultExtractionOptions(),
			maxMemoryMB:  10.0,
			description:  "Tests memory usage for large span",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Force garbage collection
			runtime.GC()
			runtime.GC()

			var initialMem runtime.MemStats
			runtime.ReadMemStats(&initialMem)

			// Perform extraction
			result, err := ExtractMeaningfulValues(tc.span, tc.options)
			require.NoError(t, err)
			require.NotNil(t, result)

			runtime.GC()
			runtime.GC()

			var finalMem runtime.MemStats
			runtime.ReadMemStats(&finalMem)

			memoryUsedMB := float64(finalMem.Alloc-initialMem.Alloc) / (1024 * 1024)
			t.Logf("Memory usage for %s: %.2f MB", tc.name, memoryUsedMB)

			assert.LessOrEqual(t, memoryUsedMB, tc.maxMemoryMB, tc.description)
		})
	}
}

// TestExtractMeaningfulValues_ConcurrencySafety tests concurrent extraction
// This implements the concurrent safety testing requirement from task 4.2
func TestExtractMeaningfulValues_ConcurrencySafety(t *testing.T) {
	const numGoroutines = 10
	const numIterations = 100

	span := createTestSpan("concurrent-test", "concurrent-operation", createSimpleAttributes(50))
	options := DefaultExtractionOptions()

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numIterations)

	// Launch multiple goroutines performing extraction concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numIterations; j++ {
				result, err := ExtractMeaningfulValues(span, options)
				if err != nil {
					errors <- err
					return
				}
				if result == nil {
					errors <- assert.AnError
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}

	assert.Empty(t, errorList, "Concurrent extraction should not produce errors")
	t.Logf("Successfully executed %d concurrent extractions", numGoroutines*numIterations)
}

// TestExtractedValues_UtilityMethods tests utility methods on ExtractedValues
func TestExtractedValues_UtilityMethods(t *testing.T) {
	span := createTestSpan("util-test", "utility-test", map[string]interface{}{
		"string_attr":  "test_value",
		"numeric_attr": 42,
		"boolean_attr": true,
		"array_attr":   []interface{}{1, 2, 3},
		"object_attr":  map[string]interface{}{"key": "value"},
	})

	result, err := ExtractMeaningfulValues(span, DefaultExtractionOptions())
	require.NoError(t, err)

	// Test GetValuesByType
	stringValues := result.GetValuesByType("string")
	assert.NotNil(t, stringValues)

	// Test GetSortedKeys
	stringKeys := result.GetSortedKeys("string")
	assert.NotEmpty(t, stringKeys)

	// Test GetTotalValueCount
	totalCount := result.GetTotalValueCount()
	assert.Greater(t, totalCount, 0)

	// Test HasValue
	assert.True(t, result.HasValue("stringAttr"))
	assert.False(t, result.HasValue("nonexistent"))

	// Test GetValue
	value, exists := result.GetValue("stringAttr")
	assert.True(t, exists)
	assert.Equal(t, "test_value", value)

	// Test Summary
	summary := result.Summary()
	assert.Contains(t, summary, "spanId")
	assert.Contains(t, summary, "totalValues")
}

// Helper functions for creating test data

func createTestSpan(spanID, name string, attributes map[string]interface{}) *Span {
	return &Span{
		SpanID:     spanID,
		Name:       name,
		StartTime:  time.Now().UnixNano(),
		EndTime:    time.Now().UnixNano() + 1000000, // 1ms duration
		Status:     SpanStatus{Code: "OK", Message: ""},
		Attributes: attributes,
		Events:     []SpanEvent{},
	}
}

func createTestSpanWithEvents(spanID, name string, attributes map[string]interface{}, events []SpanEvent) *Span {
	span := createTestSpan(spanID, name, attributes)
	span.Events = events
	return span
}

func createLargeArray(size int) []interface{} {
	arr := make([]interface{}, size)
	for i := 0; i < size; i++ {
		arr[i] = i
	}
	return arr
}

func createSimpleAttributes(count int) map[string]interface{} {
	attrs := make(map[string]interface{})
	for i := 0; i < count; i++ {
		attrs[fmt.Sprintf("attr_%d", i)] = fmt.Sprintf("value_%d", i)
	}
	return attrs
}

func createComplexAttributes(count int) map[string]interface{} {
	attrs := make(map[string]interface{})
	for i := 0; i < count; i++ {
		attrs[fmt.Sprintf("string_attr_%d", i)] = fmt.Sprintf("value_%d", i)
		attrs[fmt.Sprintf("numeric_attr_%d", i)] = i
		attrs[fmt.Sprintf("boolean_attr_%d", i)] = i%2 == 0
		attrs[fmt.Sprintf("array_attr_%d", i)] = []interface{}{i, i + 1, i + 2}
		attrs[fmt.Sprintf("object_attr_%d", i)] = map[string]interface{}{
			"nested_key": fmt.Sprintf("nested_value_%d", i),
		}
	}
	return attrs
}

func createStringAttributes(count int) map[string]interface{} {
	attrs := make(map[string]interface{})
	for i := 0; i < count; i++ {
		attrs[fmt.Sprintf("string_number_%d", i)] = fmt.Sprintf("%d", i)
		attrs[fmt.Sprintf("string_float_%d", i)] = fmt.Sprintf("%.2f", float64(i)+0.5)
		attrs[fmt.Sprintf("string_bool_%d", i)] = fmt.Sprintf("%t", i%2 == 0)
	}
	return attrs
}