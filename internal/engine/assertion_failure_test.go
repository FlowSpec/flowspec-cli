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

	"github.com/flowspec/flowspec-cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDetailedValidationDetail_Success(t *testing.T) {
	engine := NewAlignmentEngine()

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
		},
	}

	context := NewEvaluationContext(span, nil)
	engine.populateEvaluationContext(context, span)

	assertion := map[string]interface{}{
		"==": []interface{}{
			map[string]interface{}{"var": "http_method"},
			"POST",
		},
	}

	assertionResult := &AssertionResult{
		Passed:     true,
		Expected:   true,
		Actual:     true,
		Expression: `{"==":[{"var":"http_method"},"POST"]}`,
		Message:    "Assertion passed",
	}

	detail := engine.createDetailedValidationDetail(
		"precondition", assertion, assertionResult, span, context)

	assert.NotNil(t, detail)
	assert.Equal(t, "precondition", detail.Type)
	assert.Equal(t, assertionResult.Expression, detail.Expression)
	assert.Equal(t, assertionResult.Expected, detail.Expected)
	assert.Equal(t, assertionResult.Actual, detail.Actual)
	assert.Equal(t, span, detail.SpanContext)
	assert.Contains(t, detail.Message, "Precondition assertion passed")

	// Success case should not have failure details
	assert.Empty(t, detail.FailureReason)
	assert.Nil(t, detail.ContextInfo)
	assert.Nil(t, detail.Suggestions)
}

func TestCreateDetailedValidationDetail_Failure(t *testing.T) {
	engine := NewAlignmentEngine()

	span := &models.Span{
		SpanID:  "test-span",
		TraceID: "test-trace",
		Name:    "test-operation",
		Status: models.SpanStatus{
			Code:    "ERROR",
			Message: "Internal server error",
		},
		Attributes: map[string]interface{}{
			"http.method":      "POST",
			"http.status_code": 500,
		},
	}

	context := NewEvaluationContext(span, nil)
	engine.populateEvaluationContext(context, span)

	assertion := map[string]interface{}{
		"==": []interface{}{
			map[string]interface{}{"var": "http_status_code"},
			200,
		},
	}

	assertionResult := &AssertionResult{
		Passed:     false,
		Expected:   200,
		Actual:     500,
		Expression: `{"==":[{"var":"http_status_code"},200]}`,
		Message:    "Assertion failed: expected 200, got 500",
	}

	detail := engine.createDetailedValidationDetail(
		"postcondition", assertion, assertionResult, span, context)

	assert.NotNil(t, detail)
	assert.Equal(t, "postcondition", detail.Type)
	assert.Equal(t, assertionResult.Expression, detail.Expression)
	assert.Equal(t, assertionResult.Expected, detail.Expected)
	assert.Equal(t, assertionResult.Actual, detail.Actual)
	assert.Equal(t, span, detail.SpanContext)

	// Failure case should have detailed information
	assert.Contains(t, detail.Message, "Postcondition assertion failed")
	assert.Contains(t, detail.Message, "test-span")
	assert.Contains(t, detail.Message, "Expected: 200")
	assert.Contains(t, detail.Message, "Actual: 500")
	assert.Contains(t, detail.Message, "Span Status: ERROR")

	assert.NotEmpty(t, detail.FailureReason)
	assert.NotNil(t, detail.ContextInfo)
	assert.NotEmpty(t, detail.Suggestions)
}

func TestGenerateActionableErrorMessage_Success(t *testing.T) {
	engine := NewAlignmentEngine()

	span := &models.Span{
		SpanID:  "test-span",
		TraceID: "test-trace",
		Name:    "test-operation",
		Status:  models.SpanStatus{Code: "OK"},
	}

	context := NewEvaluationContext(span, nil)
	assertion := map[string]interface{}{"==": []interface{}{"a", "a"}}
	result := &AssertionResult{
		Passed:  true,
		Message: "Test passed successfully",
	}

	message := engine.generateActionableErrorMessage("precondition", assertion, result, span, context)

	assert.Contains(t, message, "Precondition assertion passed")
	assert.Contains(t, message, "Test passed successfully")
}

func TestGenerateActionableErrorMessage_Failure(t *testing.T) {
	engine := NewAlignmentEngine()

	span := &models.Span{
		SpanID:   "test-span",
		TraceID:  "test-trace",
		Name:     "test-operation",
		ParentID: "parent-span",
		Status: models.SpanStatus{
			Code:    "ERROR",
			Message: "Internal error",
		},
		Attributes: map[string]interface{}{
			"http.method":  "POST",
			"service.name": "user-service",
		},
	}

	context := NewEvaluationContext(span, nil)
	assertion := map[string]interface{}{"==": []interface{}{"a", "b"}}
	result := &AssertionResult{
		Passed:     false,
		Expected:   "a",
		Actual:     "b",
		Expression: `{"==":[{"var":"test"},"expected"]}`,
		Message:    "Assertion failed",
	}

	message := engine.generateActionableErrorMessage("postcondition", assertion, result, span, context)

	// Check all required components are present
	assert.Contains(t, message, "Postcondition assertion failed")
	assert.Contains(t, message, "test-operation")
	assert.Contains(t, message, "test-span")
	assert.Contains(t, message, "Expected: a (type: string)")
	assert.Contains(t, message, "Actual: b (type: string)")
	assert.Contains(t, message, "JSONLogic Result:")
	assert.Contains(t, message, "Span Status: ERROR - Internal error")
	assert.Contains(t, message, "Relevant Span Attributes:")
	assert.Contains(t, message, "http.method: POST")
	assert.Contains(t, message, "service.name: user-service")
	assert.Contains(t, message, "Trace ID: test-trace")
	assert.Contains(t, message, "Parent Span ID: parent-span")
}

func TestAnalyzeFailureReason_TypeMismatch(t *testing.T) {
	engine := NewAlignmentEngine()
	context := NewEvaluationContext(nil, nil)

	assertion := map[string]interface{}{"==": []interface{}{"200", 200}}
	result := &AssertionResult{
		Passed:   false,
		Expected: "200",
		Actual:   200,
	}

	reason := engine.analyzeFailureReason(assertion, result, context)

	assert.Contains(t, reason, "Type mismatch")
	assert.Contains(t, reason, "expected string")
	assert.Contains(t, reason, "got int")
}

func TestAnalyzeFailureReason_NilValues(t *testing.T) {
	engine := NewAlignmentEngine()
	context := NewEvaluationContext(nil, nil)

	testCases := []struct {
		name     string
		expected interface{}
		actual   interface{}
		contains string
	}{
		{
			name:     "Expected non-nil, got nil",
			expected: "value",
			actual:   nil,
			contains: "Expected non-nil value but got nil",
		},
		{
			name:     "Expected nil, got non-nil",
			expected: nil,
			actual:   "value",
			contains: "Expected nil value but got non-nil",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertion := map[string]interface{}{"==": []interface{}{tc.expected, tc.actual}}
			result := &AssertionResult{
				Passed:   false,
				Expected: tc.expected,
				Actual:   tc.actual,
			}

			reason := engine.analyzeFailureReason(assertion, result, context)
			assert.Contains(t, reason, tc.contains)
		})
	}
}

func TestAnalyzeFailureReason_NumericDifference(t *testing.T) {
	engine := NewAlignmentEngine()
	context := NewEvaluationContext(nil, nil)

	testCases := []struct {
		name     string
		expected interface{}
		actual   interface{}
		contains string
	}{
		{
			name:     "Actual greater than expected",
			expected: 100,
			actual:   150,
			contains: "50.00 greater than expected",
		},
		{
			name:     "Actual less than expected",
			expected: 200,
			actual:   150,
			contains: "50.00 less than expected",
		},
		{
			name:     "Float comparison",
			expected: 3.14,
			actual:   2.71,
			contains: "0.43 less than expected",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertion := map[string]interface{}{"==": []interface{}{tc.expected, tc.actual}}
			result := &AssertionResult{
				Passed:   false,
				Expected: tc.expected,
				Actual:   tc.actual,
			}

			reason := engine.analyzeFailureReason(assertion, result, context)
			assert.Contains(t, reason, tc.contains)
		})
	}
}

func TestAnalyzeFailureReason_StringDifference(t *testing.T) {
	engine := NewAlignmentEngine()
	context := NewEvaluationContext(nil, nil)

	testCases := []struct {
		name     string
		expected string
		actual   string
		contains []string
	}{
		{
			name:     "Length difference",
			expected: "hello",
			actual:   "hello world",
			contains: []string{"String length mismatch", "expected 5 characters, got 11"},
		},
		{
			name:     "Character difference",
			expected: "hello",
			actual:   "hallo",
			contains: []string{"First difference at position 1", "expected 'e', got 'a'"},
		},
		{
			name:     "Case difference",
			expected: "Hello",
			actual:   "hello",
			contains: []string{"First difference at position 0", "expected 'H', got 'h'"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertion := map[string]interface{}{"==": []interface{}{tc.expected, tc.actual}}
			result := &AssertionResult{
				Passed:   false,
				Expected: tc.expected,
				Actual:   tc.actual,
			}

			reason := engine.analyzeFailureReason(assertion, result, context)
			for _, expectedContent := range tc.contains {
				assert.Contains(t, reason, expectedContent)
			}
		})
	}
}

func TestAnalyzeVariableResolution(t *testing.T) {
	engine := NewAlignmentEngine()

	// Create context with some variables
	context := NewEvaluationContext(nil, nil)
	context.SetVariable("existing_var", "value")
	context.SetVariable("nil_var", nil)

	testCases := []struct {
		name      string
		assertion map[string]interface{}
		contains  string
	}{
		{
			name: "Missing variable",
			assertion: map[string]interface{}{
				"==": []interface{}{
					map[string]interface{}{"var": "missing_var"},
					"expected",
				},
			},
			contains: "Variable 'missing_var' not found in context",
		},
		{
			name: "Nil variable",
			assertion: map[string]interface{}{
				"==": []interface{}{
					map[string]interface{}{"var": "nil_var"},
					"expected",
				},
			},
			contains: "Variable 'nil_var' is nil",
		},
		{
			name: "Multiple variable issues",
			assertion: map[string]interface{}{
				"and": []interface{}{
					map[string]interface{}{
						"==": []interface{}{
							map[string]interface{}{"var": "missing_var"},
							"expected",
						},
					},
					map[string]interface{}{
						"==": []interface{}{
							map[string]interface{}{"var": "nil_var"},
							"expected",
						},
					},
				},
			},
			contains: "Variable resolution issues",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := engine.analyzeVariableResolution(tc.assertion, context)
			assert.Contains(t, result, tc.contains)
		})
	}
}

func TestExtractVariablesFromAssertion(t *testing.T) {
	engine := NewAlignmentEngine()

	testCases := []struct {
		name      string
		assertion map[string]interface{}
		expected  []string
	}{
		{
			name: "Simple variable",
			assertion: map[string]interface{}{
				"==": []interface{}{
					map[string]interface{}{"var": "test_var"},
					"value",
				},
			},
			expected: []string{"test_var"},
		},
		{
			name: "Multiple variables",
			assertion: map[string]interface{}{
				"and": []interface{}{
					map[string]interface{}{
						"==": []interface{}{
							map[string]interface{}{"var": "var1"},
							"value1",
						},
					},
					map[string]interface{}{
						">": []interface{}{
							map[string]interface{}{"var": "var2"},
							100,
						},
					},
				},
			},
			expected: []string{"var1", "var2"},
		},
		{
			name: "Nested variables",
			assertion: map[string]interface{}{
				"if": []interface{}{
					map[string]interface{}{
						"==": []interface{}{
							map[string]interface{}{"var": "condition_var"},
							true,
						},
					},
					map[string]interface{}{"var": "true_var"},
					map[string]interface{}{"var": "false_var"},
				},
			},
			expected: []string{"condition_var", "true_var", "false_var"},
		},
		{
			name:      "No variables",
			assertion: map[string]interface{}{"==": []interface{}{1, 2}},
			expected:  []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			variables := engine.extractVariablesFromAssertion(tc.assertion)
			assert.ElementsMatch(t, tc.expected, variables)
		})
	}
}

func TestExtractContextInfo(t *testing.T) {
	engine := NewAlignmentEngine()

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

	traceData := &models.TraceData{
		TraceID: "test-trace",
		Spans: map[string]*models.Span{
			"test-span": span,
		},
		RootSpan: span,
	}

	context := NewEvaluationContext(span, traceData)
	context.SetVariable("custom_var", "custom_value")

	info := engine.extractContextInfo(span, context)

	require.NotNil(t, info)

	// Check span information
	spanInfo, ok := info["span"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test-span", spanInfo["id"])
	assert.Equal(t, "test-operation", spanInfo["name"])
	assert.Equal(t, "test-trace", spanInfo["trace_id"])
	assert.Equal(t, "parent-span", spanInfo["parent_id"])
	assert.Equal(t, int64(1000000000), spanInfo["duration"])
	assert.Equal(t, false, spanInfo["has_error"])
	assert.Equal(t, false, spanInfo["is_root"])

	// Check attributes
	attributes, ok := info["attributes"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test-service", attributes["service.name"])
	assert.Equal(t, "POST", attributes["http.method"])

	// Check events
	events, ok := info["events"].([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, events, 1)
	assert.Equal(t, "test-event", events[0]["name"])

	// Check variables
	variables, ok := info["variables"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "custom_value", variables["custom_var"])

	// Check trace information
	traceInfo, ok := info["trace"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test-trace", traceInfo["id"])
	assert.Equal(t, 1, traceInfo["span_count"])

	rootSpanInfo, ok := traceInfo["root_span"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test-span", rootSpanInfo["id"])
	assert.Equal(t, "test-operation", rootSpanInfo["name"])
}

func TestGenerateSuggestions(t *testing.T) {
	engine := NewAlignmentEngine()

	testCases := []struct {
		name       string
		detailType string
		span       *models.Span
		result     *AssertionResult
		contains   []string
	}{
		{
			name:       "Success case - no suggestions",
			detailType: "precondition",
			span:       &models.Span{Status: models.SpanStatus{Code: "OK"}},
			result:     &AssertionResult{Passed: true},
			contains:   nil,
		},
		{
			name:       "Type mismatch",
			detailType: "postcondition",
			span:       &models.Span{Status: models.SpanStatus{Code: "OK"}},
			result: &AssertionResult{
				Passed:   false,
				Expected: "200",
				Actual:   200,
			},
			contains: []string{
				"Consider converting the actual value to string",
				"updating the assertion to expect int",
			},
		},
		{
			name:       "Nil value",
			detailType: "precondition",
			span:       &models.Span{Status: models.SpanStatus{Code: "OK"}},
			result: &AssertionResult{
				Passed:   false,
				Expected: "value",
				Actual:   nil,
			},
			contains: []string{
				"Check if the span attribute or variable exists",
			},
		},
		{
			name:       "Numeric comparison",
			detailType: "postcondition",
			span:       &models.Span{Status: models.SpanStatus{Code: "OK"}},
			result: &AssertionResult{
				Passed:   false,
				Expected: 200,
				Actual:   500,
			},
			contains: []string{
				"Verify the expected numeric value",
			},
		},
		{
			name:       "String comparison",
			detailType: "precondition",
			span:       &models.Span{Status: models.SpanStatus{Code: "OK"}},
			result: &AssertionResult{
				Passed:   false,
				Expected: "Hello",
				Actual:   "hello",
			},
			contains: []string{
				"Check for case sensitivity, whitespace, or encoding differences",
			},
		},
		{
			name:       "Span with error",
			detailType: "postcondition",
			span:       &models.Span{Status: models.SpanStatus{Code: "ERROR"}},
			result: &AssertionResult{
				Passed:   false,
				Expected: true,
				Actual:   false,
			},
			contains: []string{
				"The span has an error status",
			},
		},
		{
			name:       "Precondition specific",
			detailType: "precondition",
			span:       &models.Span{Status: models.SpanStatus{Code: "OK"}},
			result: &AssertionResult{
				Passed:   false,
				Expected: true,
				Actual:   false,
			},
			contains: []string{
				"Precondition failures may indicate that the service was called with unexpected input parameters",
			},
		},
		{
			name:       "Postcondition specific",
			detailType: "postcondition",
			span:       &models.Span{Status: models.SpanStatus{Code: "OK"}},
			result: &AssertionResult{
				Passed:   false,
				Expected: true,
				Actual:   false,
			},
			contains: []string{
				"Postcondition failures may indicate that the service behavior has changed",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertion := map[string]interface{}{"==": []interface{}{tc.result.Expected, tc.result.Actual}}
			suggestions := engine.generateSuggestions(tc.detailType, assertion, tc.result, tc.span)

			if tc.contains == nil {
				assert.Nil(t, suggestions)
			} else {
				require.NotNil(t, suggestions)
				suggestionText := strings.Join(suggestions, " ")
				for _, expectedContent := range tc.contains {
					assert.Contains(t, suggestionText, expectedContent)
				}
			}
		})
	}
}

func TestUtilityFunctions(t *testing.T) {
	t.Run("isNumeric", func(t *testing.T) {
		testCases := []struct {
			value    interface{}
			expected bool
		}{
			{42, true},
			{int8(42), true},
			{int16(42), true},
			{int32(42), true},
			{int64(42), true},
			{uint(42), true},
			{uint8(42), true},
			{uint16(42), true},
			{uint32(42), true},
			{uint64(42), true},
			{float32(3.14), true},
			{float64(3.14), true},
			{"42", false},
			{true, false},
			{nil, false},
		}

		for _, tc := range testCases {
			assert.Equal(t, tc.expected, isNumeric(tc.value), "Value: %v", tc.value)
		}
	})

	t.Run("toFloat64", func(t *testing.T) {
		testCases := []struct {
			value    interface{}
			expected float64
		}{
			{42, 42.0},
			{int8(42), 42.0},
			{int16(42), 42.0},
			{int32(42), 42.0},
			{int64(42), 42.0},
			{uint(42), 42.0},
			{uint8(42), 42.0},
			{uint16(42), 42.0},
			{uint32(42), 42.0},
			{uint64(42), 42.0},
			{float32(3.14), 3.140000104904175}, // float32 precision
			{float64(3.14), 3.14},
			{"42", 0.0}, // Non-numeric returns 0
		}

		for _, tc := range testCases {
			assert.Equal(t, tc.expected, toFloat64(tc.value), "Value: %v", tc.value)
		}
	})

	t.Run("abs", func(t *testing.T) {
		assert.Equal(t, 5.0, abs(5.0))
		assert.Equal(t, 5.0, abs(-5.0))
		assert.Equal(t, 0.0, abs(0.0))
		assert.Equal(t, 3.14, abs(-3.14))
	})

	t.Run("min", func(t *testing.T) {
		assert.Equal(t, 3, min(3, 5))
		assert.Equal(t, 3, min(5, 3))
		assert.Equal(t, 3, min(3, 3))
		assert.Equal(t, -5, min(-3, -5))
	})
}

func TestIntegration_DetailedValidationWithEngine(t *testing.T) {
	// Integration test to verify the enhanced validation details work with the full engine
	engine := NewAlignmentEngine()

	// Create a spec with failing assertions
	spec := models.ServiceSpec{
		OperationID: "testOp",
		Description: "Test operation",
		Preconditions: map[string]interface{}{
			"==": []interface{}{
				map[string]interface{}{"var": "http_method"},
				"GET", // This will fail
			},
		},
		Postconditions: map[string]interface{}{
			">=": []interface{}{
				map[string]interface{}{"var": "http_status_code"},
				200, // This will pass
			},
		},
	}

	// Create trace data with a span that will cause precondition failure
	span := &models.Span{
		SpanID:  "test-span",
		TraceID: "test-trace",
		Name:    "test-operation",
		Status: models.SpanStatus{
			Code:    "OK",
			Message: "Success",
		},
		Attributes: map[string]interface{}{
			"operation.id":     "testOp",
			"http.method":      "POST", // Different from expected GET
			"http.status_code": 201,    // Satisfies postcondition
		},
	}

	traceData := &models.TraceData{
		TraceID: "test-trace",
		Spans: map[string]*models.Span{
			"test-span": span,
		},
	}

	// Execute alignment
	result, err := engine.AlignSingleSpec(spec, traceData)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, models.StatusFailed, result.Status) // Should fail due to precondition
	assert.Len(t, result.Details, 2)                    // Precondition + Postcondition

	// Check precondition detail (should fail)
	preconditionDetails := result.GetPreconditionDetails()
	require.Len(t, preconditionDetails, 1)

	preconditionDetail := preconditionDetails[0]
	assert.Equal(t, "precondition", preconditionDetail.Type)
	assert.NotEqual(t, preconditionDetail.Expected, preconditionDetail.Actual)
	assert.Contains(t, preconditionDetail.Message, "Precondition assertion failed")
	assert.Contains(t, preconditionDetail.Message, "Expected: GET")
	assert.Contains(t, preconditionDetail.Message, "Actual: POST")
	assert.Contains(t, preconditionDetail.Message, "JSONLogic Result:")
	assert.NotEmpty(t, preconditionDetail.FailureReason)
	assert.NotNil(t, preconditionDetail.ContextInfo)
	assert.NotEmpty(t, preconditionDetail.Suggestions)
	assert.Equal(t, span, preconditionDetail.SpanContext)

	// Check postcondition detail (should pass)
	postconditionDetails := result.GetPostconditionDetails()
	require.Len(t, postconditionDetails, 1)

	postconditionDetail := postconditionDetails[0]
	assert.Equal(t, "postcondition", postconditionDetail.Type)
	assert.Contains(t, postconditionDetail.Message, "Postcondition assertion passed")
	assert.Empty(t, postconditionDetail.FailureReason) // Success case
	assert.Nil(t, postconditionDetail.ContextInfo)     // Success case
	assert.Nil(t, postconditionDetail.Suggestions)     // Success case
}
