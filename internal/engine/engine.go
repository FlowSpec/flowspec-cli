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
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/flowspec/flowspec-cli/internal/models"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// AlignmentEngine defines the interface for aligning ServiceSpecs with trace data
type AlignmentEngine interface {
	AlignSpecsWithTrace(specs []models.ServiceSpec, traceData *models.TraceData) (*models.AlignmentReport, error)
	AlignSingleSpec(spec models.ServiceSpec, traceData *models.TraceData) (*models.AlignmentResult, error)
	SetEvaluator(evaluator AssertionEvaluator)
	GetEvaluator() AssertionEvaluator
}

// AssertionEvaluator defines the interface for evaluating assertions
type AssertionEvaluator interface {
	EvaluateAssertion(assertion map[string]interface{}, context *EvaluationContext) (*AssertionResult, error)
	ValidateAssertion(assertion map[string]interface{}) error
}

// EvaluationContext provides context for assertion evaluation
type EvaluationContext struct {
	Span      *models.Span
	TraceData *models.TraceData
	Variables map[string]interface{}
	Timestamp time.Time
	mu        sync.RWMutex
}

// AssertionResult represents the result of evaluating an assertion
type AssertionResult struct {
	Passed     bool
	Expected   interface{}
	Actual     interface{}
	Expression string
	Message    string
	Error      error
}

// DefaultAlignmentEngine implements the AlignmentEngine interface
type DefaultAlignmentEngine struct {
	evaluator AssertionEvaluator
	config    *EngineConfig
	mu        sync.RWMutex
}

// EngineConfig holds configuration for the alignment engine
type EngineConfig struct {
	MaxConcurrency   int           // Maximum number of concurrent alignments
	Timeout          time.Duration // Timeout for individual spec alignment
	EnableMetrics    bool          // Enable performance metrics
	StrictMode       bool          // Strict mode for validation
	SkipMissingSpans bool          // Skip specs when corresponding spans are not found
}

// SpecMatcher handles matching ServiceSpecs to spans
type SpecMatcher struct {
	matchStrategies []MatchStrategy
	mu              sync.RWMutex
}

// MatchStrategy defines how to match specs to spans
type MatchStrategy interface {
	Match(spec models.ServiceSpec, traceData *models.TraceData) ([]*models.Span, error)
	GetName() string
	GetPriority() int
}

// OperationIDMatcher matches specs to spans by operation ID
type OperationIDMatcher struct{}

// SpanNameMatcher matches specs to spans by span name
type SpanNameMatcher struct{}

// AttributeMatcher matches specs to spans by attributes
type AttributeMatcher struct {
	attributeKey string
}

// ValidationContext manages the context during validation
type ValidationContext struct {
	spec      models.ServiceSpec
	span      *models.Span
	traceData *models.TraceData
	variables map[string]interface{}
	startTime time.Time
	mu        sync.RWMutex
}

// DefaultEngineConfig returns a default engine configuration
func DefaultEngineConfig() *EngineConfig {
	return &EngineConfig{
		MaxConcurrency:   4,
		Timeout:          30 * time.Second,
		EnableMetrics:    true,
		StrictMode:       false,
		SkipMissingSpans: true,
	}
}

// NewAlignmentEngine creates a new alignment engine with default configuration
func NewAlignmentEngine() *DefaultAlignmentEngine {
	return NewAlignmentEngineWithConfig(DefaultEngineConfig())
}

// NewAlignmentEngineWithConfig creates a new alignment engine with custom configuration
func NewAlignmentEngineWithConfig(config *EngineConfig) *DefaultAlignmentEngine {
	engine := &DefaultAlignmentEngine{
		config: config,
	}

	// Set default JSONLogic evaluator
	engine.evaluator = NewJSONLogicEvaluator()

	return engine
}

// NewEvaluationContext creates a new evaluation context
func NewEvaluationContext(span *models.Span, traceData *models.TraceData) *EvaluationContext {
	return &EvaluationContext{
		Span:      span,
		TraceData: traceData,
		Variables: make(map[string]interface{}),
		Timestamp: time.Now(),
	}
}

// NewSpecMatcher creates a new spec matcher with default strategies
func NewSpecMatcher() *SpecMatcher {
	matcher := &SpecMatcher{
		matchStrategies: make([]MatchStrategy, 0),
	}

	// Register default matching strategies in order of priority
	matcher.AddStrategy(&OperationIDMatcher{})
	matcher.AddStrategy(&SpanNameMatcher{})
	matcher.AddStrategy(&AttributeMatcher{attributeKey: "operation.name"})

	return matcher
}

// NewValidationContext creates a new validation context
func NewValidationContext(spec models.ServiceSpec, span *models.Span, traceData *models.TraceData) *ValidationContext {
	return &ValidationContext{
		spec:      spec,
		span:      span,
		traceData: traceData,
		variables: make(map[string]interface{}),
		startTime: time.Now(),
	}
}

// AlignmentEngine methods

// AlignSpecsWithTrace implements the AlignmentEngine interface
func (engine *DefaultAlignmentEngine) AlignSpecsWithTrace(
	specs []models.ServiceSpec,
	traceData *models.TraceData,
) (*models.AlignmentReport, error) {
	if len(specs) == 0 {
		return models.NewAlignmentReport(), nil
	}

	if traceData == nil || len(traceData.Spans) == 0 {
		return nil, fmt.Errorf("trace data is empty or nil")
	}

	// Initialize report with timing information
	startTime := time.Now()
	report := models.NewAlignmentReport()
	report.StartTime = startTime.UnixNano()

	// Initialize performance monitoring if enabled
	var performanceInfo models.PerformanceInfo
	if engine.config.EnableMetrics {
		performanceInfo = models.PerformanceInfo{
			SpecsProcessed:      0,
			SpansMatched:        0,
			AssertionsEvaluated: 0,
			ConcurrentWorkers:   0,
			MemoryUsageMB:       0.0,
			ProcessingRate:      0.0,
		}
	}

	// Create channels for concurrent processing
	specChan := make(chan models.ServiceSpec, len(specs))
	resultChan := make(chan *models.AlignmentResult, len(specs))
	errorChan := make(chan error, len(specs))

	// Determine number of workers
	numWorkers := engine.config.MaxConcurrency
	if numWorkers > len(specs) {
		numWorkers = len(specs)
	}

	// Update performance info with worker count
	if engine.config.EnableMetrics {
		performanceInfo.ConcurrentWorkers = numWorkers
	}

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			engine.alignmentWorker(specChan, resultChan, errorChan, traceData)
		}()
	}

	// Send specs to workers
	for _, spec := range specs {
		specChan <- spec
	}
	close(specChan)

	// Wait for workers to complete
	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	// Collect results and update performance metrics
	var errors []error
	spansMatched := 0
	assertionsEvaluated := 0

	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				resultChan = nil
			} else {
				report.AddResult(*result)

				// Update performance metrics
				if engine.config.EnableMetrics {
					performanceInfo.SpecsProcessed++
					spansMatched += len(result.MatchedSpans)
					assertionsEvaluated += result.AssertionsTotal
				}
			}
		case err, ok := <-errorChan:
			if !ok {
				errorChan = nil
			} else {
				errors = append(errors, err)
			}
		}

		if resultChan == nil && errorChan == nil {
			break
		}
	}

	// Finalize report timing and performance information
	endTime := time.Now()
	report.EndTime = endTime.UnixNano()
	report.ExecutionTime = endTime.Sub(startTime).Nanoseconds()

	// Complete performance information
	if engine.config.EnableMetrics {
		performanceInfo.SpansMatched = spansMatched
		performanceInfo.AssertionsEvaluated = assertionsEvaluated

		// Calculate processing rate (specs per second)
		executionSeconds := float64(report.ExecutionTime) / 1e9
		if executionSeconds > 0 {
			performanceInfo.ProcessingRate = float64(performanceInfo.SpecsProcessed) / executionSeconds
		}

		// Get memory usage (simplified - in a real implementation, you'd use runtime.MemStats)
		performanceInfo.MemoryUsageMB = engine.getMemoryUsageMB()

		report.PerformanceInfo = performanceInfo
	}

	// Return error if any critical errors occurred
	if len(errors) > 0 && len(report.Results) == 0 {
		return nil, fmt.Errorf("alignment failed with %d errors: %v", len(errors), errors[0])
	}

	return report, nil
}

// AlignSingleSpec implements the AlignmentEngine interface
func (engine *DefaultAlignmentEngine) AlignSingleSpec(
	spec models.ServiceSpec,
	traceData *models.TraceData,
) (*models.AlignmentResult, error) {
	if engine.evaluator == nil {
		return nil, fmt.Errorf("no assertion evaluator configured")
	}

	startTime := time.Now()
	result := models.NewAlignmentResult(spec.OperationID)
	result.StartTime = startTime.UnixNano()

	// Find matching spans
	matcher := NewSpecMatcher()
	matchingSpans, err := matcher.FindMatchingSpans(spec, traceData)
	if err != nil {
		return nil, fmt.Errorf("failed to find matching spans: %w", err)
	}

	if len(matchingSpans) == 0 {
		if engine.config.SkipMissingSpans {
			result.AddValidationDetail(*models.NewValidationDetail(
				"matching", "span_match", "found", "found",
				"No matching spans found for operation: "+spec.OperationID))
			result.Status = models.StatusSkipped // Set after adding detail
		} else {
			result.AddValidationDetail(*models.NewValidationDetail(
				"matching", "span_match", "found", "not_found",
				"Required spans not found for operation: "+spec.OperationID))
			// Status will be set to FAILED by updateStatus due to mismatch
		}

		// Finalize timing
		endTime := time.Now()
		result.EndTime = endTime.UnixNano()
		result.ExecutionTime = endTime.Sub(startTime).Nanoseconds()
		return result, nil
	}

	// Record matched span IDs
	result.MatchedSpans = make([]string, len(matchingSpans))
	for i, span := range matchingSpans {
		result.MatchedSpans[i] = span.SpanID
	}

	// Evaluate assertions for each matching span
	for _, span := range matchingSpans {
		if err := engine.evaluateSpecForSpan(spec, span, traceData, result); err != nil {
			return nil, fmt.Errorf("failed to evaluate spec for span %s: %w", span.SpanID, err)
		}
	}

	// Finalize timing
	endTime := time.Now()
	result.EndTime = endTime.UnixNano()
	result.ExecutionTime = endTime.Sub(startTime).Nanoseconds()
	return result, nil
}

// SetEvaluator implements the AlignmentEngine interface
func (engine *DefaultAlignmentEngine) SetEvaluator(evaluator AssertionEvaluator) {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	engine.evaluator = evaluator
}

// GetEvaluator implements the AlignmentEngine interface
func (engine *DefaultAlignmentEngine) GetEvaluator() AssertionEvaluator {
	engine.mu.RLock()
	defer engine.mu.RUnlock()
	return engine.evaluator
}

// alignmentWorker processes specs concurrently
func (engine *DefaultAlignmentEngine) alignmentWorker(
	specChan <-chan models.ServiceSpec,
	resultChan chan<- *models.AlignmentResult,
	errorChan chan<- error,
	traceData *models.TraceData,
) {
	for spec := range specChan {
		result, err := engine.AlignSingleSpec(spec, traceData)
		if err != nil {
			errorChan <- err
		} else {
			resultChan <- result
		}
	}
}

// evaluateSpecForSpan evaluates a spec against a specific span
func (engine *DefaultAlignmentEngine) evaluateSpecForSpan(
	spec models.ServiceSpec,
	span *models.Span,
	traceData *models.TraceData,
	result *models.AlignmentResult,
) error {
	context := NewEvaluationContext(span, traceData)

	// Populate context with span data
	engine.populateEvaluationContext(context, span)

	// Evaluate preconditions
	if len(spec.Preconditions) > 0 {
		preconditionResult, err := engine.evaluator.EvaluateAssertion(spec.Preconditions, context)
		if err != nil {
			return fmt.Errorf("failed to evaluate preconditions: %w", err)
		}

		detail := engine.createDetailedValidationDetail(
			"precondition",
			spec.Preconditions,
			preconditionResult,
			span,
			context,
		)
		result.AddValidationDetail(*detail)
	}

	// Evaluate postconditions
	if len(spec.Postconditions) > 0 {
		postconditionResult, err := engine.evaluator.EvaluateAssertion(spec.Postconditions, context)
		if err != nil {
			return fmt.Errorf("failed to evaluate postconditions: %w", err)
		}

		detail := engine.createDetailedValidationDetail(
			"postcondition",
			spec.Postconditions,
			postconditionResult,
			span,
			context,
		)
		result.AddValidationDetail(*detail)
	}

	return nil
}

// createDetailedValidationDetail creates a detailed validation detail with enhanced error information
func (engine *DefaultAlignmentEngine) createDetailedValidationDetail(
	detailType string,
	assertion map[string]interface{},
	assertionResult *AssertionResult,
	span *models.Span,
	context *EvaluationContext,
) *models.ValidationDetail {
	// Create enhanced validation detail
	detail := &models.ValidationDetail{
		Type:        detailType,
		Expression:  assertionResult.Expression,
		Expected:    assertionResult.Expected,
		Actual:      assertionResult.Actual,
		Message:     engine.generateActionableErrorMessage(detailType, assertion, assertionResult, span, context),
		SpanContext: span,
	}

	// Add failure analysis if assertion failed
	if !assertionResult.Passed {
		detail.FailureReason = engine.analyzeFailureReason(assertion, assertionResult, context)
		detail.ContextInfo = engine.extractContextInfo(span, context)
		detail.Suggestions = engine.generateSuggestions(detailType, assertion, assertionResult, span)
	}

	return detail
}

// generateActionableErrorMessage creates a detailed, actionable error message
func (engine *DefaultAlignmentEngine) generateActionableErrorMessage(
	detailType string,
	assertion map[string]interface{},
	result *AssertionResult,
	span *models.Span,
	context *EvaluationContext,
) string {
	if result.Passed {
		return fmt.Sprintf("%s assertion passed: %s",
			cases.Title(language.English).String(detailType), result.Message)
	}

	// Build detailed failure message
	var msgBuilder strings.Builder

	msgBuilder.WriteString(fmt.Sprintf("%s assertion failed in span '%s' (ID: %s)\n",
		cases.Title(language.English).String(detailType), span.Name, span.SpanID))

	// Add assertion details with enhanced context
	msgBuilder.WriteString(fmt.Sprintf("Assertion: %s\n", result.Expression))

	// Try to extract more meaningful expected/actual values from the assertion
	expectedVal, actualVal := engine.extractMeaningfulValues(assertion, result, context)
	msgBuilder.WriteString(fmt.Sprintf("Expected: %v (type: %T)\n", expectedVal, expectedVal))
	msgBuilder.WriteString(fmt.Sprintf("Actual: %v (type: %T)\n", actualVal, actualVal))

	// Add JSONLogic evaluation result for reference
	msgBuilder.WriteString(fmt.Sprintf("JSONLogic Result: Expected %v, Got %v\n", result.Expected, result.Actual))

	// Add span context
	msgBuilder.WriteString(fmt.Sprintf("Span Status: %s", span.Status.Code))
	if span.Status.Message != "" {
		msgBuilder.WriteString(fmt.Sprintf(" - %s", span.Status.Message))
	}
	msgBuilder.WriteString("\n")

	// Add relevant span attributes
	if len(span.Attributes) > 0 {
		msgBuilder.WriteString("Relevant Span Attributes:\n")
		for key, value := range span.Attributes {
			msgBuilder.WriteString(fmt.Sprintf("  %s: %v\n", key, value))
		}
	}

	// Add trace context
	msgBuilder.WriteString(fmt.Sprintf("Trace ID: %s\n", span.TraceID))
	if span.ParentID != "" {
		msgBuilder.WriteString(fmt.Sprintf("Parent Span ID: %s\n", span.ParentID))
	}

	return msgBuilder.String()
}

// extractMeaningfulValues attempts to extract the actual values being compared from the assertion
func (engine *DefaultAlignmentEngine) extractMeaningfulValues(
	assertion map[string]interface{},
	result *AssertionResult,
	context *EvaluationContext,
) (interface{}, interface{}) {
	// For simple equality comparisons, try to extract the actual values
	if eqAssertion, ok := assertion["=="]; ok {
		if eqSlice, ok := eqAssertion.([]interface{}); ok && len(eqSlice) == 2 {
			// First element might be a variable reference
			if varRef, ok := eqSlice[0].(map[string]interface{}); ok {
				if varName, ok := varRef["var"].(string); ok {
					if actualValue, exists := context.GetVariable(varName); exists {
						return eqSlice[1], actualValue // expected, actual
					}
					// Try with underscore conversion for JSONLogic compatibility
					safeVarName := strings.ReplaceAll(varName, ".", "_")
					if actualValue, exists := context.GetVariable(safeVarName); exists {
						return eqSlice[1], actualValue // expected, actual
					}
				}
			}
			// Second element might be a variable reference
			if varRef, ok := eqSlice[1].(map[string]interface{}); ok {
				if varName, ok := varRef["var"].(string); ok {
					if actualValue, exists := context.GetVariable(varName); exists {
						return eqSlice[0], actualValue // expected, actual
					}
					// Try with underscore conversion for JSONLogic compatibility
					safeVarName := strings.ReplaceAll(varName, ".", "_")
					if actualValue, exists := context.GetVariable(safeVarName); exists {
						return eqSlice[0], actualValue // expected, actual
					}
				}
			}
		}
	}

	// For other comparison operators, try similar extraction
	for op, opAssertion := range assertion {
		switch op {
		case "!=", ">", "<", ">=", "<=":
			if opSlice, ok := opAssertion.([]interface{}); ok && len(opSlice) == 2 {
				// Check if first element is a variable
				if varRef, ok := opSlice[0].(map[string]interface{}); ok {
					if varName, ok := varRef["var"].(string); ok {
						if actualValue, exists := context.GetVariable(varName); exists {
							return opSlice[1], actualValue // expected, actual
						}
						// Try with underscore conversion
						safeVarName := strings.ReplaceAll(varName, ".", "_")
						if actualValue, exists := context.GetVariable(safeVarName); exists {
							return opSlice[1], actualValue // expected, actual
						}
					}
				}
				// Check if second element is a variable
				if varRef, ok := opSlice[1].(map[string]interface{}); ok {
					if varName, ok := varRef["var"].(string); ok {
						if actualValue, exists := context.GetVariable(varName); exists {
							return opSlice[0], actualValue // expected, actual
						}
						// Try with underscore conversion
						safeVarName := strings.ReplaceAll(varName, ".", "_")
						if actualValue, exists := context.GetVariable(safeVarName); exists {
							return opSlice[0], actualValue // expected, actual
						}
					}
				}
			}
		}
	}

	// Fallback to JSONLogic result values
	return result.Expected, result.Actual
}

// analyzeFailureReason analyzes why an assertion failed and provides detailed reasoning
func (engine *DefaultAlignmentEngine) analyzeFailureReason(
	assertion map[string]interface{},
	result *AssertionResult,
	context *EvaluationContext,
) string {
	if result.Passed {
		return ""
	}

	// Analyze the type of failure
	var reasons []string

	// Type mismatch analysis
	if result.Expected != nil && result.Actual != nil {
		expectedType := reflect.TypeOf(result.Expected)
		actualType := reflect.TypeOf(result.Actual)

		if expectedType != actualType {
			reasons = append(reasons, fmt.Sprintf(
				"Type mismatch: expected %s but got %s",
				expectedType, actualType))
		}
	}

	// Null/nil value analysis
	if result.Expected != nil && result.Actual == nil {
		reasons = append(reasons, "Expected non-nil value but got nil")
	} else if result.Expected == nil && result.Actual != nil {
		reasons = append(reasons, "Expected nil value but got non-nil")
	}

	// Numeric comparison analysis
	if isNumeric(result.Expected) && isNumeric(result.Actual) {
		expectedNum := toFloat64(result.Expected)
		actualNum := toFloat64(result.Actual)
		diff := actualNum - expectedNum

		if diff != 0 {
			reasons = append(reasons, fmt.Sprintf(
				"Numeric difference: actual value is %.2f %s than expected",
				abs(diff),
				map[bool]string{true: "greater", false: "less"}[diff > 0]))
		}
	}

	// String comparison analysis
	if expectedStr, ok := result.Expected.(string); ok {
		if actualStr, ok := result.Actual.(string); ok {
			if len(expectedStr) != len(actualStr) {
				reasons = append(reasons, fmt.Sprintf(
					"String length mismatch: expected %d characters, got %d",
					len(expectedStr), len(actualStr)))
			}

			// Find first difference
			minLen := min(len(expectedStr), len(actualStr))
			for i := 0; i < minLen; i++ {
				if expectedStr[i] != actualStr[i] {
					reasons = append(reasons, fmt.Sprintf(
						"First difference at position %d: expected '%c', got '%c'",
						i, expectedStr[i], actualStr[i]))
					break
				}
			}
		}
	}

	// JSONLogic specific analysis
	if result.Error != nil {
		reasons = append(reasons, fmt.Sprintf("JSONLogic evaluation error: %v", result.Error))
	}

	// Variable resolution analysis
	if len(reasons) == 0 {
		reasons = append(reasons, engine.analyzeVariableResolution(assertion, context))
	}

	if len(reasons) == 0 {
		return "Assertion evaluated to false but specific reason could not be determined"
	}

	return strings.Join(reasons, "; ")
}

// analyzeVariableResolution analyzes potential variable resolution issues
func (engine *DefaultAlignmentEngine) analyzeVariableResolution(
	assertion map[string]interface{},
	context *EvaluationContext,
) string {
	variables := engine.extractVariablesFromAssertion(assertion)
	var issues []string

	for _, variable := range variables {
		if value, exists := context.GetVariable(variable); !exists {
			issues = append(issues, fmt.Sprintf("Variable '%s' not found in context", variable))
		} else if value == nil {
			issues = append(issues, fmt.Sprintf("Variable '%s' is nil", variable))
		}
	}

	if len(issues) > 0 {
		return "Variable resolution issues: " + strings.Join(issues, ", ")
	}

	return "Unknown assertion failure reason"
}

// extractVariablesFromAssertion extracts variable references from a JSONLogic assertion
func (engine *DefaultAlignmentEngine) extractVariablesFromAssertion(assertion map[string]interface{}) []string {
	var variables []string
	engine.extractVariablesRecursive(assertion, &variables)
	return variables
}

// extractVariablesRecursive recursively extracts variables from nested structures
func (engine *DefaultAlignmentEngine) extractVariablesRecursive(obj interface{}, variables *[]string) {
	switch v := obj.(type) {
	case map[string]interface{}:
		for key, value := range v {
			if key == "var" {
				if varName, ok := value.(string); ok {
					*variables = append(*variables, varName)
				}
			} else {
				engine.extractVariablesRecursive(value, variables)
			}
		}
	case []interface{}:
		for _, item := range v {
			engine.extractVariablesRecursive(item, variables)
		}
	}
}

// extractContextInfo extracts relevant context information for debugging
func (engine *DefaultAlignmentEngine) extractContextInfo(
	span *models.Span,
	context *EvaluationContext,
) map[string]interface{} {
	info := make(map[string]interface{})

	// Span information
	info["span"] = map[string]interface{}{
		"id":         span.SpanID,
		"name":       span.Name,
		"trace_id":   span.TraceID,
		"parent_id":  span.ParentID,
		"start_time": span.StartTime,
		"end_time":   span.EndTime,
		"duration":   span.GetDuration(),
		"status":     span.Status,
		"has_error":  span.HasError(),
		"is_root":    span.IsRoot(),
	}

	// Available attributes
	info["attributes"] = span.Attributes

	// Available events
	if len(span.Events) > 0 {
		events := make([]map[string]interface{}, len(span.Events))
		for i, event := range span.Events {
			events[i] = map[string]interface{}{
				"name":       event.Name,
				"timestamp":  event.Timestamp,
				"attributes": event.Attributes,
			}
		}
		info["events"] = events
	}

	// Context variables
	info["variables"] = context.GetAllVariables()

	// Trace information
	if context.TraceData != nil {
		info["trace"] = map[string]interface{}{
			"id":         context.TraceData.TraceID,
			"span_count": len(context.TraceData.Spans),
		}

		if context.TraceData.RootSpan != nil {
			if traceInfo, ok := info["trace"].(map[string]interface{}); ok {
				traceInfo["root_span"] = map[string]interface{}{
					"id":   context.TraceData.RootSpan.SpanID,
					"name": context.TraceData.RootSpan.Name,
				}
			}
		}
	}

	return info
}

// generateSuggestions generates actionable suggestions for fixing assertion failures
func (engine *DefaultAlignmentEngine) generateSuggestions(
	detailType string,
	assertion map[string]interface{},
	result *AssertionResult,
	span *models.Span,
) []string {
	if result.Passed {
		return nil
	}

	var suggestions []string

	// Type-specific suggestions
	if result.Expected != nil && result.Actual != nil {
		expectedType := reflect.TypeOf(result.Expected)
		actualType := reflect.TypeOf(result.Actual)

		if expectedType != actualType {
			suggestions = append(suggestions, fmt.Sprintf(
				"Consider converting the actual value to %s or updating the assertion to expect %s",
				expectedType, actualType))
		}
	}

	// Null value suggestions
	if result.Expected != nil && result.Actual == nil {
		suggestions = append(suggestions,
			"Check if the span attribute or variable exists and has a non-nil value")
	}

	// Numeric comparison suggestions
	if isNumeric(result.Expected) && isNumeric(result.Actual) {
		suggestions = append(suggestions,
			"Verify the expected numeric value or check if the span attribute contains the correct numeric data")
	}

	// String comparison suggestions
	if _, expectedIsString := result.Expected.(string); expectedIsString {
		if _, actualIsString := result.Actual.(string); actualIsString {
			suggestions = append(suggestions,
				"Check for case sensitivity, whitespace, or encoding differences in string values")
		}
	}

	// Span-specific suggestions
	if span.HasError() {
		suggestions = append(suggestions,
			"The span has an error status - consider checking if this affects the expected behavior")
	}

	// General suggestions
	suggestions = append(suggestions,
		"Review the span attributes and trace data to ensure the assertion logic matches the actual service behavior")

	if detailType == "precondition" {
		suggestions = append(suggestions,
			"Precondition failures may indicate that the service was called with unexpected input parameters")
	} else if detailType == "postcondition" {
		suggestions = append(suggestions,
			"Postcondition failures may indicate that the service behavior has changed or the assertion needs updating")
	}

	return suggestions
}

// populateEvaluationContext populates the evaluation context with span data
func (engine *DefaultAlignmentEngine) populateEvaluationContext(context *EvaluationContext, span *models.Span) {
	context.mu.Lock()
	defer context.mu.Unlock()

	// Add span attributes to context
	for key, value := range span.Attributes {
		// Keep original key for backward compatibility
		context.Variables[key] = value
		// Also add with underscores for JSONLogic compatibility
		safeKey := strings.ReplaceAll(key, ".", "_")
		if safeKey != key {
			context.Variables[safeKey] = value
		}
	}

	// Add span metadata
	context.Variables["span.id"] = span.SpanID
	context.Variables["span.name"] = span.Name
	context.Variables["span.start_time"] = span.StartTime
	context.Variables["span.end_time"] = span.EndTime
	context.Variables["span.duration"] = span.GetDuration()
	context.Variables["span.status.code"] = span.Status.Code
	context.Variables["span.status.message"] = span.Status.Message
	context.Variables["span.has_error"] = span.HasError()
	context.Variables["span.is_root"] = span.IsRoot()

	// Add trace metadata
	context.Variables["trace.id"] = span.TraceID
	if context.TraceData != nil {
		context.Variables["trace.span_count"] = len(context.TraceData.Spans)
		if context.TraceData.RootSpan != nil {
			context.Variables["trace.root_span.id"] = context.TraceData.RootSpan.SpanID
		}
	}
}

// EvaluationContext methods

// GetVariable gets a variable from the context
func (ctx *EvaluationContext) GetVariable(key string) (interface{}, bool) {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	value, exists := ctx.Variables[key]
	return value, exists
}

// SetVariable sets a variable in the context
func (ctx *EvaluationContext) SetVariable(key string, value interface{}) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.Variables[key] = value
}

// GetAllVariables returns a copy of all variables
func (ctx *EvaluationContext) GetAllVariables() map[string]interface{} {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	result := make(map[string]interface{})
	for key, value := range ctx.Variables {
		result[key] = value
	}
	return result
}

// SpecMatcher methods

// AddStrategy adds a matching strategy
func (sm *SpecMatcher) AddStrategy(strategy MatchStrategy) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.matchStrategies = append(sm.matchStrategies, strategy)
}

// FindMatchingSpans finds spans that match the given spec
func (sm *SpecMatcher) FindMatchingSpans(spec models.ServiceSpec, traceData *models.TraceData) ([]*models.Span, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Try each strategy in order of priority
	for _, strategy := range sm.matchStrategies {
		spans, err := strategy.Match(spec, traceData)
		if err != nil {
			continue // Try next strategy
		}

		if len(spans) > 0 {
			return spans, nil
		}
	}

	// No matching spans found
	return []*models.Span{}, nil
}

// MatchStrategy implementations

// OperationIDMatcher methods

// Match implements the MatchStrategy interface
func (matcher *OperationIDMatcher) Match(spec models.ServiceSpec, traceData *models.TraceData) ([]*models.Span, error) {
	var matchingSpans []*models.Span

	for _, span := range traceData.Spans {
		if operationID, ok := span.Attributes["operation.id"].(string); ok {
			if operationID == spec.OperationID {
				matchingSpans = append(matchingSpans, span)
			}
		}
	}

	return matchingSpans, nil
}

// GetName implements the MatchStrategy interface
func (matcher *OperationIDMatcher) GetName() string {
	return "operation_id"
}

// GetPriority implements the MatchStrategy interface
func (matcher *OperationIDMatcher) GetPriority() int {
	return 100 // Highest priority
}

// SpanNameMatcher methods

// Match implements the MatchStrategy interface
func (matcher *SpanNameMatcher) Match(spec models.ServiceSpec, traceData *models.TraceData) ([]*models.Span, error) {
	var matchingSpans []*models.Span

	// Try to match by span name (use operation ID as span name)
	for _, span := range traceData.Spans {
		if span.Name == spec.OperationID {
			matchingSpans = append(matchingSpans, span)
		}
	}

	return matchingSpans, nil
}

// GetName implements the MatchStrategy interface
func (matcher *SpanNameMatcher) GetName() string {
	return "span_name"
}

// GetPriority implements the MatchStrategy interface
func (matcher *SpanNameMatcher) GetPriority() int {
	return 80 // High priority
}

// AttributeMatcher methods

// Match implements the MatchStrategy interface
func (matcher *AttributeMatcher) Match(spec models.ServiceSpec, traceData *models.TraceData) ([]*models.Span, error) {
	var matchingSpans []*models.Span

	for _, span := range traceData.Spans {
		if value, ok := span.Attributes[matcher.attributeKey].(string); ok {
			if value == spec.OperationID {
				matchingSpans = append(matchingSpans, span)
			}
		}
	}

	return matchingSpans, nil
}

// GetName implements the MatchStrategy interface
func (matcher *AttributeMatcher) GetName() string {
	return fmt.Sprintf("attribute_%s", matcher.attributeKey)
}

// GetPriority implements the MatchStrategy interface
func (matcher *AttributeMatcher) GetPriority() int {
	return 60 // Medium priority
}

// ValidationContext methods

// GetSpec returns the spec being validated
func (ctx *ValidationContext) GetSpec() models.ServiceSpec {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.spec
}

// GetSpan returns the span being validated
func (ctx *ValidationContext) GetSpan() *models.Span {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.span
}

// GetTraceData returns the trace data
func (ctx *ValidationContext) GetTraceData() *models.TraceData {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return ctx.traceData
}

// GetElapsedTime returns the elapsed time since validation started
func (ctx *ValidationContext) GetElapsedTime() time.Duration {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	return time.Since(ctx.startTime)
}

// SetVariable sets a variable in the validation context
func (ctx *ValidationContext) SetVariable(key string, value interface{}) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.variables[key] = value
}

// GetVariable gets a variable from the validation context
func (ctx *ValidationContext) GetVariable(key string) (interface{}, bool) {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	value, exists := ctx.variables[key]
	return value, exists
}

// Utility functions

// isNumeric checks if a value is numeric
func isNumeric(value interface{}) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float32, float64:
		return true
	default:
		return false
	}
}

// toFloat64 converts a numeric value to float64
func toFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case int:
		return float64(v)
	case int8:
		return float64(v)
	case int16:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case uint:
		return float64(v)
	case uint8:
		return float64(v)
	case uint16:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return v
	default:
		return 0
	}
}

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getMemoryUsageMB returns the current memory usage in MB
func (engine *DefaultAlignmentEngine) getMemoryUsageMB() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// Return allocated memory in MB
	return float64(m.Alloc) / 1024 / 1024
}

// ValidateEngineConfig validates the engine configuration
func ValidateEngineConfig(config *EngineConfig) error {
	if config.MaxConcurrency <= 0 {
		return fmt.Errorf("MaxConcurrency must be positive, got %d", config.MaxConcurrency)
	}

	if config.Timeout <= 0 {
		return fmt.Errorf("Timeout must be positive, got %s", config.Timeout)
	}

	return nil
}
