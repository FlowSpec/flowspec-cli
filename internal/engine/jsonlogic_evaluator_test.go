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

package engine

import (
	"strings"
	"testing"
	"time"

	"github.com/flowspec/flowspec-cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJSONLogicEvaluator(t *testing.T) {
	evaluator := NewJSONLogicEvaluator()

	assert.NotNil(t, evaluator)
	assert.NotNil(t, evaluator.config)
	assert.Equal(t, 10, evaluator.config.MaxDepth)
	assert.Equal(t, 5*time.Second, evaluator.config.Timeout)
	assert.False(t, evaluator.config.StrictMode)
	assert.Empty(t, evaluator.config.AllowedOperators)
	assert.True(t, evaluator.config.SandboxMode)
}

func TestNewJSONLogicEvaluatorWithConfig(t *testing.T) {
	config := &JSONLogicConfig{
		MaxDepth:         5,
		Timeout:          10 * time.Second,
		StrictMode:       true,
		AllowedOperators: []string{"==", "!=", ">", "<"},
		SandboxMode:      false,
	}

	evaluator := NewJSONLogicEvaluatorWithConfig(config)

	assert.NotNil(t, evaluator)
	assert.Equal(t, config, evaluator.config)
	assert.Equal(t, 5, evaluator.config.MaxDepth)
	assert.Equal(t, 10*time.Second, evaluator.config.Timeout)
	assert.True(t, evaluator.config.StrictMode)
	assert.Equal(t, []string{"==", "!=", ">", "<"}, evaluator.config.AllowedOperators)
	assert.False(t, evaluator.config.SandboxMode)
}

func TestDefaultJSONLogicConfig(t *testing.T) {
	config := DefaultJSONLogicConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 10, config.MaxDepth)
	assert.Equal(t, 5*time.Second, config.Timeout)
	assert.False(t, config.StrictMode)
	assert.Empty(t, config.AllowedOperators)
	assert.True(t, config.SandboxMode)
}

func TestValidateJSONLogicConfig(t *testing.T) {
	testCases := []struct {
		name        string
		config      *JSONLogicConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid config",
			config:      DefaultJSONLogicConfig(),
			expectError: false,
		},
		{
			name: "Invalid MaxDepth",
			config: &JSONLogicConfig{
				MaxDepth: 0,
				Timeout:  5 * time.Second,
			},
			expectError: true,
			errorMsg:    "MaxDepth must be positive",
		},
		{
			name: "Invalid Timeout",
			config: &JSONLogicConfig{
				MaxDepth: 10,
				Timeout:  -1 * time.Second,
			},
			expectError: true,
			errorMsg:    "Timeout cannot be negative",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateJSONLogicConfig(tc.config)
			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestJSONLogicEvaluator_EvaluateAssertion_EmptyAssertion(t *testing.T) {
	evaluator := NewJSONLogicEvaluator()
	context := NewEvaluationContext(nil, nil)

	// Test nil assertion
	result, err := evaluator.EvaluateAssertion(nil, context)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Passed)
	assert.Equal(t, "empty_assertion", result.Expression)

	// Test empty assertion
	result, err = evaluator.EvaluateAssertion(map[string]interface{}{}, context)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Passed)
	assert.Equal(t, "empty_assertion", result.Expression)
}

func TestJSONLogicEvaluator_EvaluateAssertion_SimpleComparisons(t *testing.T) {
	evaluator := NewJSONLogicEvaluator()

	// Create test span
	span := &models.Span{
		SpanID:  "test-span",
		TraceID: "test-trace",
		Name:    "test-operation",
		Status: models.SpanStatus{
			Code:    "OK",
			Message: "Success",
		},
		Attributes: map[string]interface{}{
			"http.method":      "POST",
			"http.status_code": 200,
			"service.name":     "test-service",
		},
	}

	context := NewEvaluationContext(span, nil)

	testCases := []struct {
		name      string
		assertion map[string]interface{}
		expected  bool
	}{
		{
			name: "Simple equality - true",
			assertion: map[string]interface{}{
				"==": []interface{}{
					map[string]interface{}{"var": "http_method"},
					"POST",
				},
			},
			expected: true,
		},
		{
			name: "Simple equality - false",
			assertion: map[string]interface{}{
				"==": []interface{}{
					map[string]interface{}{"var": "http_method"},
					"GET",
				},
			},
			expected: false,
		},
		{
			name: "Numeric comparison - greater than",
			assertion: map[string]interface{}{
				">": []interface{}{
					map[string]interface{}{"var": "http_status_code"},
					199,
				},
			},
			expected: true,
		},
		{
			name: "Numeric comparison - less than",
			assertion: map[string]interface{}{
				"<": []interface{}{
					map[string]interface{}{"var": "http_status_code"},
					300,
				},
			},
			expected: true,
		},
		{
			name: "Not equal",
			assertion: map[string]interface{}{
				"!=": []interface{}{
					map[string]interface{}{"var": "service_name"},
					"other-service",
				},
			},
			expected: true,
		},
		{
			name: "Span status check",
			assertion: map[string]interface{}{
				"==": []interface{}{
					map[string]interface{}{"var": "span.status.code"},
					"OK",
				},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := evaluator.EvaluateAssertion(tc.assertion, context)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tc.expected, result.Passed)
			assert.NotEmpty(t, result.Expression)
			assert.NotEmpty(t, result.Message)
		})
	}
}

func TestJSONLogicEvaluator_EvaluateAssertion_ComplexLogic(t *testing.T) {
	evaluator := NewJSONLogicEvaluator()

	// Create test span
	span := &models.Span{
		SpanID:  "test-span",
		TraceID: "test-trace",
		Name:    "test-operation",
		Status: models.SpanStatus{
			Code:    "OK",
			Message: "Success",
		},
		Attributes: map[string]interface{}{
			"http.method":      "POST",
			"http.status_code": 201,
			"service.name":     "user-service",
			"operation.type":   "create",
		},
	}

	context := NewEvaluationContext(span, nil)

	testCases := []struct {
		name      string
		assertion map[string]interface{}
		expected  bool
	}{
		{
			name: "AND logic - both true",
			assertion: map[string]interface{}{
				"and": []interface{}{
					map[string]interface{}{
						"==": []interface{}{
							map[string]interface{}{"var": "http_method"},
							"POST",
						},
					},
					map[string]interface{}{
						"==": []interface{}{
							map[string]interface{}{"var": "span.status.code"},
							"OK",
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "AND logic - one false",
			assertion: map[string]interface{}{
				"and": []interface{}{
					map[string]interface{}{
						"==": []interface{}{
							map[string]interface{}{"var": "http_method"},
							"GET",
						},
					},
					map[string]interface{}{
						"==": []interface{}{
							map[string]interface{}{"var": "span.status.code"},
							"OK",
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "OR logic - one true",
			assertion: map[string]interface{}{
				"or": []interface{}{
					map[string]interface{}{
						"==": []interface{}{
							map[string]interface{}{"var": "http_method"},
							"GET",
						},
					},
					map[string]interface{}{
						"==": []interface{}{
							map[string]interface{}{"var": "http_method"},
							"POST",
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "Range check - status code 2xx",
			assertion: map[string]interface{}{
				"and": []interface{}{
					map[string]interface{}{
						">=": []interface{}{
							map[string]interface{}{"var": "http_status_code"},
							200,
						},
					},
					map[string]interface{}{
						"<": []interface{}{
							map[string]interface{}{"var": "http_status_code"},
							300,
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "String contains check",
			assertion: map[string]interface{}{
				"in": []interface{}{
					"user",
					map[string]interface{}{"var": "service_name"},
				},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := evaluator.EvaluateAssertion(tc.assertion, context)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tc.expected, result.Passed, "Assertion: %v", tc.assertion)
			assert.NotEmpty(t, result.Expression)
			assert.NotEmpty(t, result.Message)
		})
	}
}

func TestJSONLogicEvaluator_EvaluateAssertion_WithTraceData(t *testing.T) {
	evaluator := NewJSONLogicEvaluator()

	// Create test span
	span := &models.Span{
		SpanID:  "test-span",
		TraceID: "test-trace",
		Name:    "test-operation",
		Status: models.SpanStatus{
			Code:    "OK",
			Message: "Success",
		},
		Attributes: map[string]interface{}{
			"service.name": "test-service",
		},
	}

	// Create test trace data
	traceData := &models.TraceData{
		TraceID: "test-trace",
		Spans: map[string]*models.Span{
			"test-span": span,
			"other-span": {
				SpanID:  "other-span",
				TraceID: "test-trace",
				Name:    "other-operation",
			},
		},
		RootSpan: span,
	}

	context := NewEvaluationContext(span, traceData)

	testCases := []struct {
		name      string
		assertion map[string]interface{}
		expected  bool
	}{
		{
			name: "Check trace span count",
			assertion: map[string]interface{}{
				"==": []interface{}{
					map[string]interface{}{"var": "trace.span_count"},
					2,
				},
			},
			expected: true,
		},
		{
			name: "Check trace ID",
			assertion: map[string]interface{}{
				"==": []interface{}{
					map[string]interface{}{"var": "trace.id"},
					"test-trace",
				},
			},
			expected: true,
		},
		{
			name: "Check root span ID",
			assertion: map[string]interface{}{
				"==": []interface{}{
					map[string]interface{}{"var": "trace.root_span.id"},
					"test-span",
				},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := evaluator.EvaluateAssertion(tc.assertion, context)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tc.expected, result.Passed)
		})
	}
}

func TestJSONLogicEvaluator_ValidateAssertion(t *testing.T) {
	evaluator := NewJSONLogicEvaluator()

	testCases := []struct {
		name        string
		assertion   map[string]interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Nil assertion",
			assertion:   nil,
			expectError: true,
			errorMsg:    "assertion cannot be nil",
		},
		{
			name: "Valid simple assertion",
			assertion: map[string]interface{}{
				"==": []interface{}{"a", "b"},
			},
			expectError: false,
		},
		{
			name: "Valid complex assertion",
			assertion: map[string]interface{}{
				"and": []interface{}{
					map[string]interface{}{
						"==": []interface{}{
							map[string]interface{}{"var": "status"},
							"OK",
						},
					},
					map[string]interface{}{
						">": []interface{}{
							map[string]interface{}{"var": "count"},
							0,
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := evaluator.ValidateAssertion(tc.assertion)
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestJSONLogicEvaluator_ValidateAssertion_StrictMode(t *testing.T) {
	config := DefaultJSONLogicConfig()
	config.StrictMode = true
	config.AllowedOperators = []string{"==", "!=", ">", "<", ">=", "<=", "and", "or"}

	evaluator := NewJSONLogicEvaluatorWithConfig(config)

	testCases := []struct {
		name        string
		assertion   map[string]interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name: "Allowed operator",
			assertion: map[string]interface{}{
				"==": []interface{}{"a", "b"},
			},
			expectError: false,
		},
		{
			name: "Disallowed operator",
			assertion: map[string]interface{}{
				"in": []interface{}{"a", "abc"},
			},
			expectError: true,
			errorMsg:    "operator 'in' is not allowed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := evaluator.ValidateAssertion(tc.assertion)
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestJSONLogicEvaluator_ValidateAssertion_MaxDepth(t *testing.T) {
	config := DefaultJSONLogicConfig()
	config.MaxDepth = 2

	evaluator := NewJSONLogicEvaluatorWithConfig(config)

	// Create deeply nested assertion (depth > 2)
	deepAssertion := map[string]interface{}{
		"and": []interface{}{
			map[string]interface{}{
				"or": []interface{}{
					map[string]interface{}{
						"==": []interface{}{
							map[string]interface{}{"var": "a"},
							"b",
						},
					},
				},
			},
		},
	}

	err := evaluator.ValidateAssertion(deepAssertion)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum depth")
}

func TestJSONLogicEvaluator_ConvertToBool(t *testing.T) {
	evaluator := NewJSONLogicEvaluator()

	testCases := []struct {
		name     string
		input    interface{}
		expected bool
	}{
		{"nil", nil, false},
		{"true", true, true},
		{"false", false, false},
		{"int zero", 0, false},
		{"int non-zero", 42, true},
		{"int64 zero", int64(0), false},
		{"int64 non-zero", int64(42), true},
		{"float zero", 0.0, false},
		{"float non-zero", 3.14, true},
		{"empty string", "", false},
		{"non-empty string", "hello", true},
		{"empty slice", []interface{}{}, false},
		{"non-empty slice", []interface{}{1, 2, 3}, true},
		{"empty map", map[string]interface{}{}, false},
		{"non-empty map", map[string]interface{}{"key": "value"}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := evaluator.convertToBool(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestJSONLogicEvaluator_BuildEvaluationData(t *testing.T) {
	evaluator := NewJSONLogicEvaluator()

	// Create test span with events
	span := &models.Span{
		SpanID:    "test-span",
		TraceID:   "test-trace",
		ParentID:  "parent-span",
		Name:      "test-operation",
		StartTime: 1640995200000000000,
		EndTime:   1640995201000000000,
		Status: models.SpanStatus{
			Code:    "OK",
			Message: "Success",
		},
		Attributes: map[string]interface{}{
			"service.name": "test-service",
			"http.method":  "POST",
		},
		Events: []models.SpanEvent{
			{
				Name:      "test-event",
				Timestamp: 1640995200500000000,
				Attributes: map[string]interface{}{
					"event.type": "info",
				},
			},
		},
	}

	// Create test trace data
	traceData := &models.TraceData{
		TraceID: "test-trace",
		Spans: map[string]*models.Span{
			"test-span": span,
		},
		RootSpan: span,
	}

	context := NewEvaluationContext(span, traceData)
	context.SetVariable("custom.var", "custom-value")

	data, err := evaluator.buildEvaluationData(context)
	require.NoError(t, err)
	require.NotNil(t, data)

	// Check span data
	spanData, ok := data["span"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test-span", spanData["id"])
	assert.Equal(t, "test-operation", spanData["name"])
	assert.Equal(t, "test-trace", spanData["trace_id"])
	assert.Equal(t, "parent-span", spanData["parent_id"])
	assert.Equal(t, int64(1000000000), spanData["duration"]) // 1 second

	// Check span status
	statusData, ok := spanData["status"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "OK", statusData["code"])
	assert.Equal(t, "Success", statusData["message"])

	// Check span attributes are available at root level
	assert.Equal(t, "test-service", data["service.name"])
	assert.Equal(t, "POST", data["http.method"])

	// Check attributes namespace
	attributes, ok := data["attributes"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test-service", attributes["service.name"])
	assert.Equal(t, "POST", attributes["http.method"])

	// Check events
	events, ok := data["events"].([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, events, 1)
	assert.Equal(t, "test-event", events[0]["name"])

	// Check trace data
	traceDataMap, ok := data["trace"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test-trace", traceDataMap["id"])
	assert.Equal(t, 1, traceDataMap["span_count"])

	rootSpanData, ok := traceDataMap["root_span"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test-span", rootSpanData["id"])
	assert.Equal(t, "test-operation", rootSpanData["name"])

	// Check custom variables
	vars, ok := data["vars"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "custom-value", vars["custom.var"])

	// Check metadata
	meta, ok := data["_meta"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "jsonlogic", meta["evaluator"])
	assert.Equal(t, "1.0", meta["version"])
}

func TestJSONLogicEvaluator_BuildEvaluationData_NilContext(t *testing.T) {
	evaluator := NewJSONLogicEvaluator()

	data, err := evaluator.buildEvaluationData(nil)
	assert.NoError(t, err)
	assert.NotNil(t, data)
	assert.Empty(t, data)
}

func TestJSONLogicEvaluator_ApplyWithTimeout(t *testing.T) {
	config := DefaultJSONLogicConfig()
	config.Timeout = 100 * time.Millisecond
	evaluator := NewJSONLogicEvaluatorWithConfig(config)

	// Test normal operation
	rule := map[string]interface{}{
		"==": []interface{}{1, 1},
	}
	data := map[string]interface{}{}

	result, err := evaluator.applyWithTimeout(rule, data)
	assert.NoError(t, err)
	assert.Equal(t, true, result)

	// Test with no timeout (timeout = 0)
	config.Timeout = 0
	evaluator.SetConfig(config)

	result, err = evaluator.applyWithTimeout(rule, data)
	assert.NoError(t, err)
	assert.Equal(t, true, result)
}

func TestJSONLogicEvaluator_GetSetConfig(t *testing.T) {
	evaluator := NewJSONLogicEvaluator()
	originalConfig := evaluator.GetConfig()

	newConfig := &JSONLogicConfig{
		MaxDepth:    5,
		Timeout:     10 * time.Second,
		StrictMode:  true,
		SandboxMode: false,
	}

	evaluator.SetConfig(newConfig)
	assert.Equal(t, newConfig, evaluator.GetConfig())
	assert.NotEqual(t, originalConfig, evaluator.GetConfig())
}

func TestJSONLogicEvaluator_Integration_WithEngine(t *testing.T) {
	// Test integration with the alignment engine
	engine := NewAlignmentEngine()

	// Verify that the engine has a JSONLogic evaluator by default
	evaluator := engine.GetEvaluator()
	assert.NotNil(t, evaluator)

	// Verify it's a JSONLogic evaluator by checking if it can handle JSONLogic expressions
	jsonLogicEvaluator, ok := evaluator.(*JSONLogicEvaluator)
	assert.True(t, ok)
	assert.NotNil(t, jsonLogicEvaluator)
	assert.NotNil(t, jsonLogicEvaluator.GetConfig())
}

func TestJSONLogicEvaluator_ErrorHandling(t *testing.T) {
	evaluator := NewJSONLogicEvaluator()

	// Test with truly invalid JSONLogic structure that will cause an error
	invalidAssertion := map[string]interface{}{
		"==": "not_an_array", // Invalid structure - should be an array
	}

	context := NewEvaluationContext(nil, nil)
	result, err := evaluator.EvaluateAssertion(invalidAssertion, context)

	// Should return a result with error information, not panic
	assert.NoError(t, err) // The function handles JSONLogic errors gracefully
	assert.NotNil(t, result)
	assert.False(t, result.Passed)
	// The error might be nil if JSONLogic handles it gracefully, so we check the result
	// The message should contain either "Assertion failed" or "JSONLogic evaluation failed"
	assert.True(t, strings.Contains(result.Message, "Assertion failed") || strings.Contains(result.Message, "JSONLogic evaluation failed"))
}