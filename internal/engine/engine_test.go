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
	"testing"
	"time"

	"github.com/flowspec/flowspec-cli/internal/models"
	"github.com/stretchr/testify/assert"
)

// MockAssertionEvaluator for testing
type MockAssertionEvaluator struct {
	evaluateFunc func(assertion map[string]interface{}, context *EvaluationContext) (*AssertionResult, error)
	validateFunc func(assertion map[string]interface{}) error
}

func (m *MockAssertionEvaluator) EvaluateAssertion(assertion map[string]interface{}, context *EvaluationContext) (*AssertionResult, error) {
	if m.evaluateFunc != nil {
		return m.evaluateFunc(assertion, context)
	}
	return &AssertionResult{
		Passed:     true,
		Expected:   true,
		Actual:     true,
		Expression: "mock_assertion",
		Message:    "Mock evaluation passed",
	}, nil
}

func (m *MockAssertionEvaluator) ValidateAssertion(assertion map[string]interface{}) error {
	if m.validateFunc != nil {
		return m.validateFunc(assertion)
	}
	return nil
}

func TestNewAlignmentEngine(t *testing.T) {
	engine := NewAlignmentEngine()

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.config)
	assert.Equal(t, 4, engine.config.MaxConcurrency)
	assert.Equal(t, 30*time.Second, engine.config.Timeout)
	assert.True(t, engine.config.EnableMetrics)
	assert.False(t, engine.config.StrictMode)
	assert.True(t, engine.config.SkipMissingSpans)
}

func TestNewAlignmentEngineWithConfig(t *testing.T) {
	config := &EngineConfig{
		MaxConcurrency:   8,
		Timeout:          60 * time.Second,
		EnableMetrics:    false,
		StrictMode:       true,
		SkipMissingSpans: false,
	}

	engine := NewAlignmentEngineWithConfig(config)

	assert.NotNil(t, engine)
	assert.Equal(t, config, engine.config)
	assert.Equal(t, 8, engine.config.MaxConcurrency)
	assert.Equal(t, 60*time.Second, engine.config.Timeout)
	assert.False(t, engine.config.EnableMetrics)
	assert.True(t, engine.config.StrictMode)
	assert.False(t, engine.config.SkipMissingSpans)
}

func TestDefaultEngineConfig(t *testing.T) {
	config := DefaultEngineConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 4, config.MaxConcurrency)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.True(t, config.EnableMetrics)
	assert.False(t, config.StrictMode)
	assert.True(t, config.SkipMissingSpans)
}

func TestValidateEngineConfig(t *testing.T) {
	testCases := []struct {
		name        string
		config      *EngineConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid config",
			config:      DefaultEngineConfig(),
			expectError: false,
		},
		{
			name: "Invalid MaxConcurrency",
			config: &EngineConfig{
				MaxConcurrency: 0,
				Timeout:        30 * time.Second,
			},
			expectError: true,
			errorMsg:    "MaxConcurrency must be positive",
		},
		{
			name: "Invalid Timeout",
			config: &EngineConfig{
				MaxConcurrency: 4,
				Timeout:        0,
			},
			expectError: true,
			errorMsg:    "Timeout must be positive",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateEngineConfig(tc.config)
			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSetAndGetEvaluator(t *testing.T) {
	engine := NewAlignmentEngine()
	mockEvaluator := &MockAssertionEvaluator{}

	// Initially has default JSONLogic evaluator
	assert.NotNil(t, engine.GetEvaluator())
	_, ok := engine.GetEvaluator().(*JSONLogicEvaluator)
	assert.True(t, ok, "Default evaluator should be JSONLogicEvaluator")

	// Set custom evaluator
	engine.SetEvaluator(mockEvaluator)
	assert.Equal(t, mockEvaluator, engine.GetEvaluator())
}

func TestNewEvaluationContext(t *testing.T) {
	span := &models.Span{
		SpanID:  "test-span",
		TraceID: "test-trace",
		Name:    "test-operation",
	}

	traceData := &models.TraceData{
		TraceID: "test-trace",
		Spans: map[string]*models.Span{
			"test-span": span,
		},
	}

	context := NewEvaluationContext(span, traceData)

	assert.NotNil(t, context)
	assert.Equal(t, span, context.Span)
	assert.Equal(t, traceData, context.TraceData)
	assert.NotNil(t, context.Variables)
	assert.False(t, context.Timestamp.IsZero())
}

func TestEvaluationContext_Variables(t *testing.T) {
	context := NewEvaluationContext(nil, nil)

	// Test setting and getting variables
	context.SetVariable("test_key", "test_value")
	value, exists := context.GetVariable("test_key")
	assert.True(t, exists)
	assert.Equal(t, "test_value", value)

	// Test non-existent variable
	value, exists = context.GetVariable("non_existent")
	assert.False(t, exists)
	assert.Nil(t, value)

	// Test getting all variables
	context.SetVariable("key1", "value1")
	context.SetVariable("key2", "value2")

	allVars := context.GetAllVariables()
	assert.Len(t, allVars, 3) // test_key, key1, key2
	assert.Equal(t, "test_value", allVars["test_key"])
	assert.Equal(t, "value1", allVars["key1"])
	assert.Equal(t, "value2", allVars["key2"])
}

func TestNewSpecMatcher(t *testing.T) {
	matcher := NewSpecMatcher()

	assert.NotNil(t, matcher)
	assert.Len(t, matcher.matchStrategies, 3) // OperationID, SpanName, Attribute matchers
}

func TestSpecMatcher_AddStrategy(t *testing.T) {
	matcher := NewSpecMatcher()
	initialCount := len(matcher.matchStrategies)

	customMatcher := &AttributeMatcher{attributeKey: "custom.key"}
	matcher.AddStrategy(customMatcher)

	assert.Len(t, matcher.matchStrategies, initialCount+1)
}

func TestOperationIDMatcher(t *testing.T) {
	matcher := &OperationIDMatcher{}

	// Test matcher properties
	assert.Equal(t, "operation_id", matcher.GetName())
	assert.Equal(t, 100, matcher.GetPriority())

	// Create test data
	spec := models.ServiceSpec{
		OperationID: "testOperation",
		Description: "Test operation",
	}

	span1 := &models.Span{
		SpanID:  "span1",
		TraceID: "trace1",
		Name:    "span1-name",
		Attributes: map[string]interface{}{
			"operation.id": "testOperation",
		},
	}

	span2 := &models.Span{
		SpanID:  "span2",
		TraceID: "trace1",
		Name:    "span2-name",
		Attributes: map[string]interface{}{
			"operation.id": "otherOperation",
		},
	}

	traceData := &models.TraceData{
		TraceID: "trace1",
		Spans: map[string]*models.Span{
			"span1": span1,
			"span2": span2,
		},
	}

	// Test matching
	matches, err := matcher.Match(spec, traceData)
	assert.NoError(t, err)
	assert.Len(t, matches, 1)
	assert.Equal(t, "span1", matches[0].SpanID)
}

func TestSpanNameMatcher(t *testing.T) {
	matcher := &SpanNameMatcher{}

	// Test matcher properties
	assert.Equal(t, "span_name", matcher.GetName())
	assert.Equal(t, 80, matcher.GetPriority())

	// Create test data
	spec := models.ServiceSpec{
		OperationID: "testOperation",
		Description: "Test operation",
	}

	span1 := &models.Span{
		SpanID:  "span1",
		TraceID: "trace1",
		Name:    "testOperation", // Matches operation ID
	}

	span2 := &models.Span{
		SpanID:  "span2",
		TraceID: "trace1",
		Name:    "otherOperation",
	}

	traceData := &models.TraceData{
		TraceID: "trace1",
		Spans: map[string]*models.Span{
			"span1": span1,
			"span2": span2,
		},
	}

	// Test matching
	matches, err := matcher.Match(spec, traceData)
	assert.NoError(t, err)
	assert.Len(t, matches, 1)
	assert.Equal(t, "span1", matches[0].SpanID)
}

func TestAttributeMatcher(t *testing.T) {
	matcher := &AttributeMatcher{attributeKey: "operation.name"}

	// Test matcher properties
	assert.Equal(t, "attribute_operation.name", matcher.GetName())
	assert.Equal(t, 60, matcher.GetPriority())

	// Create test data
	spec := models.ServiceSpec{
		OperationID: "testOperation",
		Description: "Test operation",
	}

	span1 := &models.Span{
		SpanID:  "span1",
		TraceID: "trace1",
		Name:    "span1-name",
		Attributes: map[string]interface{}{
			"operation.name": "testOperation",
		},
	}

	span2 := &models.Span{
		SpanID:  "span2",
		TraceID: "trace1",
		Name:    "span2-name",
		Attributes: map[string]interface{}{
			"operation.name": "otherOperation",
		},
	}

	traceData := &models.TraceData{
		TraceID: "trace1",
		Spans: map[string]*models.Span{
			"span1": span1,
			"span2": span2,
		},
	}

	// Test matching
	matches, err := matcher.Match(spec, traceData)
	assert.NoError(t, err)
	assert.Len(t, matches, 1)
	assert.Equal(t, "span1", matches[0].SpanID)
}

func TestSpecMatcher_FindMatchingSpans(t *testing.T) {
	matcher := NewSpecMatcher()

	// Create test data
	spec := models.ServiceSpec{
		OperationID: "testOperation",
		Description: "Test operation",
	}

	span1 := &models.Span{
		SpanID:  "span1",
		TraceID: "trace1",
		Name:    "span1-name",
		Attributes: map[string]interface{}{
			"operation.id": "testOperation", // Should match with OperationIDMatcher
		},
	}

	span2 := &models.Span{
		SpanID:  "span2",
		TraceID: "trace1",
		Name:    "testOperation", // Should match with SpanNameMatcher
	}

	traceData := &models.TraceData{
		TraceID: "trace1",
		Spans: map[string]*models.Span{
			"span1": span1,
			"span2": span2,
		},
	}

	// Test finding matching spans
	matches, err := matcher.FindMatchingSpans(spec, traceData)
	assert.NoError(t, err)
	assert.Len(t, matches, 1) // Should find span1 first (higher priority matcher)
	assert.Equal(t, "span1", matches[0].SpanID)
}

func TestSpecMatcher_FindMatchingSpans_NoMatches(t *testing.T) {
	matcher := NewSpecMatcher()

	// Create test data with no matches
	spec := models.ServiceSpec{
		OperationID: "testOperation",
		Description: "Test operation",
	}

	span1 := &models.Span{
		SpanID:  "span1",
		TraceID: "trace1",
		Name:    "different-name",
		Attributes: map[string]interface{}{
			"operation.id": "differentOperation",
		},
	}

	traceData := &models.TraceData{
		TraceID: "trace1",
		Spans: map[string]*models.Span{
			"span1": span1,
		},
	}

	// Test finding matching spans
	matches, err := matcher.FindMatchingSpans(spec, traceData)
	assert.NoError(t, err)
	assert.Empty(t, matches)
}

func TestAlignmentEngine_AlignSpecsWithTrace_EmptySpecs(t *testing.T) {
	engine := NewAlignmentEngine()
	traceData := &models.TraceData{
		TraceID: "test-trace",
		Spans:   map[string]*models.Span{},
	}

	report, err := engine.AlignSpecsWithTrace([]models.ServiceSpec{}, traceData)

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Empty(t, report.Results)
	assert.Equal(t, 0, report.Summary.Total)
}

func TestAlignmentEngine_AlignSpecsWithTrace_NilTraceData(t *testing.T) {
	engine := NewAlignmentEngine()
	specs := []models.ServiceSpec{
		{
			OperationID: "testOp",
			Description: "Test operation",
		},
	}

	report, err := engine.AlignSpecsWithTrace(specs, nil)

	assert.Error(t, err)
	assert.Nil(t, report)
	assert.Contains(t, err.Error(), "trace data is empty or nil")
}

func TestAlignmentEngine_AlignSingleSpec_NoEvaluator(t *testing.T) {
	engine := NewAlignmentEngine()
	// Clear the default evaluator
	engine.SetEvaluator(nil)

	spec := models.ServiceSpec{
		OperationID: "testOp",
		Description: "Test operation",
	}

	traceData := &models.TraceData{
		TraceID: "test-trace",
		Spans:   map[string]*models.Span{},
	}

	result, err := engine.AlignSingleSpec(spec, traceData)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no assertion evaluator configured")
}

func TestAlignmentEngine_AlignSingleSpec_NoMatchingSpans(t *testing.T) {
	engine := NewAlignmentEngine()
	engine.SetEvaluator(&MockAssertionEvaluator{})

	spec := models.ServiceSpec{
		OperationID: "testOp",
		Description: "Test operation",
	}

	// Create trace data with no matching spans
	span := &models.Span{
		SpanID:  "span1",
		TraceID: "trace1",
		Name:    "different-operation",
		Attributes: map[string]interface{}{
			"operation.id": "differentOp",
		},
	}

	traceData := &models.TraceData{
		TraceID: "trace1",
		Spans: map[string]*models.Span{
			"span1": span,
		},
	}

	result, err := engine.AlignSingleSpec(spec, traceData)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.StatusSkipped, result.Status) // Default config skips missing spans
	assert.Len(t, result.Details, 1)
	assert.Equal(t, "matching", result.Details[0].Type)
}

func TestAlignmentEngine_AlignSingleSpec_WithMatchingSpan(t *testing.T) {
	engine := NewAlignmentEngine()

	// Set up mock evaluator
	mockEvaluator := &MockAssertionEvaluator{
		evaluateFunc: func(assertion map[string]interface{}, context *EvaluationContext) (*AssertionResult, error) {
			return &AssertionResult{
				Passed:     true,
				Expected:   true,
				Actual:     true,
				Expression: "test_assertion",
				Message:    "Test passed",
			}, nil
		},
	}
	engine.SetEvaluator(mockEvaluator)

	spec := models.ServiceSpec{
		OperationID: "testOp",
		Description: "Test operation",
		Preconditions: map[string]interface{}{
			"test": true,
		},
		Postconditions: map[string]interface{}{
			"result": true,
		},
	}

	// Create trace data with matching span
	span := &models.Span{
		SpanID:  "span1",
		TraceID: "trace1",
		Name:    "test-operation",
		Attributes: map[string]interface{}{
			"operation.id": "testOp",
		},
	}

	traceData := &models.TraceData{
		TraceID: "trace1",
		Spans: map[string]*models.Span{
			"span1": span,
		},
	}

	result, err := engine.AlignSingleSpec(spec, traceData)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "testOp", result.SpecOperationID)
	assert.Len(t, result.Details, 2) // Precondition + Postcondition
	assert.Greater(t, result.ExecutionTime, int64(0))

	// Check validation details
	preconditionDetail := result.GetPreconditionDetails()
	assert.Len(t, preconditionDetail, 1)
	assert.Equal(t, "precondition", preconditionDetail[0].Type)

	postconditionDetail := result.GetPostconditionDetails()
	assert.Len(t, postconditionDetail, 1)
	assert.Equal(t, "postcondition", postconditionDetail[0].Type)
}

func TestPopulateEvaluationContext(t *testing.T) {
	engine := NewAlignmentEngine()

	span := &models.Span{
		SpanID:    "test-span",
		TraceID:   "test-trace",
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
	}

	traceData := &models.TraceData{
		TraceID: "test-trace",
		Spans: map[string]*models.Span{
			"test-span": span,
		},
		RootSpan: span,
	}

	context := NewEvaluationContext(span, traceData)
	engine.populateEvaluationContext(context, span)

	// Check span attributes
	serviceName, exists := context.GetVariable("service.name")
	assert.True(t, exists)
	assert.Equal(t, "test-service", serviceName)

	httpMethod, exists := context.GetVariable("http.method")
	assert.True(t, exists)
	assert.Equal(t, "POST", httpMethod)

	// Check span metadata
	spanID, exists := context.GetVariable("span.id")
	assert.True(t, exists)
	assert.Equal(t, "test-span", spanID)

	spanName, exists := context.GetVariable("span.name")
	assert.True(t, exists)
	assert.Equal(t, "test-operation", spanName)

	duration, exists := context.GetVariable("span.duration")
	assert.True(t, exists)
	assert.Equal(t, int64(1000000000), duration) // 1 second in nanoseconds

	statusCode, exists := context.GetVariable("span.status.code")
	assert.True(t, exists)
	assert.Equal(t, "OK", statusCode)

	// Check trace metadata
	traceID, exists := context.GetVariable("trace.id")
	assert.True(t, exists)
	assert.Equal(t, "test-trace", traceID)

	spanCount, exists := context.GetVariable("trace.span_count")
	assert.True(t, exists)
	assert.Equal(t, 1, spanCount)

	rootSpanID, exists := context.GetVariable("trace.root_span.id")
	assert.True(t, exists)
	assert.Equal(t, "test-span", rootSpanID)
}

func TestNewValidationContext(t *testing.T) {
	spec := models.ServiceSpec{
		OperationID: "testOp",
		Description: "Test operation",
	}

	span := &models.Span{
		SpanID:  "test-span",
		TraceID: "test-trace",
		Name:    "test-operation",
	}

	traceData := &models.TraceData{
		TraceID: "test-trace",
		Spans: map[string]*models.Span{
			"test-span": span,
		},
	}

	context := NewValidationContext(spec, span, traceData)

	assert.NotNil(t, context)
	assert.Equal(t, spec, context.GetSpec())
	assert.Equal(t, span, context.GetSpan())
	assert.Equal(t, traceData, context.GetTraceData())
	assert.Greater(t, context.GetElapsedTime(), time.Duration(0))

	// Test variables
	context.SetVariable("test_key", "test_value")
	value, exists := context.GetVariable("test_key")
	assert.True(t, exists)
	assert.Equal(t, "test_value", value)
}
