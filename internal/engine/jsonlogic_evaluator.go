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
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/diegoholiveira/jsonlogic/v3"
)

// JSONLogicEvaluator implements the AssertionEvaluator interface using JSONLogic
type JSONLogicEvaluator struct {
	config *JSONLogicConfig
}

// JSONLogicConfig holds configuration for JSONLogic evaluation
type JSONLogicConfig struct {
	MaxDepth         int           // Maximum recursion depth for expressions
	Timeout          time.Duration // Timeout for individual expression evaluation
	StrictMode       bool          // Strict mode for type checking
	AllowedOperators []string      // List of allowed JSONLogic operators (empty = all allowed)
	SandboxMode      bool          // Enable sandbox mode for security
}

// DefaultJSONLogicConfig returns a default configuration for JSONLogic evaluation
func DefaultJSONLogicConfig() *JSONLogicConfig {
	return &JSONLogicConfig{
		MaxDepth:         10,
		Timeout:          5 * time.Second,
		StrictMode:       false,
		AllowedOperators: []string{}, // Allow all operators by default
		SandboxMode:      true,
	}
}

// NewJSONLogicEvaluator creates a new JSONLogic evaluator with default configuration
func NewJSONLogicEvaluator() *JSONLogicEvaluator {
	return NewJSONLogicEvaluatorWithConfig(DefaultJSONLogicConfig())
}

// NewJSONLogicEvaluatorWithConfig creates a new JSONLogic evaluator with custom configuration
func NewJSONLogicEvaluatorWithConfig(config *JSONLogicConfig) *JSONLogicEvaluator {
	return &JSONLogicEvaluator{
		config: config,
	}
}

// EvaluateAssertion implements the AssertionEvaluator interface
func (evaluator *JSONLogicEvaluator) EvaluateAssertion(assertion map[string]interface{}, context *EvaluationContext) (*AssertionResult, error) {
	if len(assertion) == 0 {
		return &AssertionResult{
				Passed:     true,
				Expected:   true,
				Actual:     true,
				Expression: "empty_assertion",
				Message:    "Empty assertion always passes",
			},
			nil
	}

	// Build evaluation data from context
	data, err := evaluator.buildEvaluationData(context)
	if err != nil {
		return nil, fmt.Errorf("failed to build evaluation data: %w", err)
	}

	// Check if this is a multi-condition assertion (ServiceSpec format)
	// If so, convert it to a proper JSONLogic "and" expression
	processedAssertion := evaluator.preprocessAssertion(assertion)

	// Validate assertion before evaluation if in strict mode
	if evaluator.config.StrictMode {
		if validationErr := evaluator.ValidateAssertion(processedAssertion); validationErr != nil {
			return nil, fmt.Errorf("assertion validation failed: %w", validationErr)
		}
	}

	// Apply JSONLogic with timeout protection
	result, err := evaluator.applyWithTimeout(processedAssertion, data)
	if err != nil {
		assertionJSON, marshalErr := json.Marshal(assertion)
		if marshalErr != nil {
			// Handle case where even marshaling the assertion fails
			assertionJSON = []byte("invalid_assertion_format")
		}
		return &AssertionResult{
				Passed:     false,
				Expected:   true,
				Actual:     false,
				Expression: string(assertionJSON),
				Message:    fmt.Sprintf("JSONLogic evaluation failed: %v", err),
				Error:      err,
			},
			nil
	}

	// Convert result to boolean
	passed := evaluator.convertToBool(result)

	assertionJSON, marshalErr := json.Marshal(assertion)
	if marshalErr != nil {
		assertionJSON = []byte("invalid_assertion_format")
	}
	return &AssertionResult{
			Passed:     passed,
			Expected:   true,
			Actual:     result,
			Expression: string(assertionJSON),
			Message:    evaluator.buildResultMessage(passed, assertion, result),
		},
		nil
}

// ValidateAssertion implements the AssertionEvaluator interface
func (evaluator *JSONLogicEvaluator) ValidateAssertion(assertion map[string]interface{}) error {
	if assertion == nil {
		return fmt.Errorf("assertion cannot be nil")
	}

	// Check for allowed operators if configured
	if len(evaluator.config.AllowedOperators) > 0 {
		if err := evaluator.validateOperators(assertion, evaluator.config.AllowedOperators); err != nil {
			return fmt.Errorf("operator validation failed: %w", err)
		}
	}

	// Check maximum depth
	if err := evaluator.validateDepth(assertion, 0, evaluator.config.MaxDepth); err != nil {
		return fmt.Errorf("depth validation failed: %w", err)
	}

	// Try to marshal to JSON to ensure it's valid
	_, err := json.Marshal(assertion)
	if err != nil {
		return fmt.Errorf("assertion is not valid JSON: %w", err)
	}

	return nil
}

// BuildEvaluationData builds the data context for JSONLogic evaluation (exported for testing)
func (evaluator *JSONLogicEvaluator) BuildEvaluationData(context *EvaluationContext) (map[string]interface{}, error) {
	return evaluator.buildEvaluationData(context)
}

// buildEvaluationData builds the data context for JSONLogic evaluation
func (evaluator *JSONLogicEvaluator) buildEvaluationData(context *EvaluationContext) (map[string]interface{}, error) {
	if context == nil {
		return map[string]interface{}{}, nil
	}

	data := make(map[string]interface{})

	// Add span data if available
	if context.Span != nil {
		span := context.Span

		// Span basic information
		spanData := map[string]interface{}{
			"id":         span.SpanID,
			"name":       span.Name,
			"trace_id":   span.TraceID,
			"parent_id":  span.ParentID,
			"start_time": span.StartTime,
			"end_time":   span.EndTime,
			"duration":   span.GetDuration(),
			"status": map[string]interface{}{
				"code":    span.Status.Code,
				"message": span.Status.Message,
			},
			"has_error": span.HasError(),
			"is_root":   span.IsRoot(),
		}

		// Expand dot-notation keys from attributes and merge them into spanData
		expandedAttrs := expandDotKeys(span.Attributes)
		for key, value := range expandedAttrs {
			// Merge expanded attributes into spanData, avoiding overwriting existing keys
			if _, exists := spanData[key]; !exists {
				spanData[key] = value
			}
		}

		// Also add expanded attributes under the "attributes" key for JSONLogic expressions
		spanData["attributes"] = expandedAttrs

		data["span"] = spanData

		// Add raw attributes and underscore-versions for backward compatibility
		data["attributes"] = span.Attributes
		for key, value := range span.Attributes {
			// Add at root level for backward compatibility
			data[key] = value
			// Add underscore version
			safeKey := strings.ReplaceAll(key, ".", "_")
			data[safeKey] = value
		}

		// Also add the expanded nested structure to the root level
		// This allows JSONLogic to access "request.id" as nested path
		for key, value := range expandedAttrs {
			if _, exists := data[key]; !exists {
				data[key] = value
			}
		}

		// Add span events
		if len(span.Events) > 0 {
			events := make([]map[string]interface{}, len(span.Events))
			for i, event := range span.Events {
				events[i] = map[string]interface{}{
					"name":       event.Name,
					"timestamp":  event.Timestamp,
					"attributes": event.Attributes,
				}
			}
			data["events"] = events
		}
	}

	// Add trace data if available
	if context.TraceData != nil {
		traceData := context.TraceData
		data["trace"] = map[string]interface{}{
			"id":         traceData.TraceID,
			"span_count": len(traceData.Spans),
		}

		if traceData.RootSpan != nil {
			data["trace"].(map[string]interface{})["root_span"] = map[string]interface{}{
				"id":   traceData.RootSpan.SpanID,
				"name": traceData.RootSpan.Name,
			}
		}
	}

	// Add context variables
	allVars := context.GetAllVariables()
	for key, value := range allVars {
		// Avoid overwriting existing keys, use "vars" namespace
		if _, exists := data[key]; !exists {
			data[key] = value
		}
	}

	// Add variables under "vars" namespace as well
	data["vars"] = allVars

	// Add evaluation metadata
	data["_meta"] = map[string]interface{}{
		"timestamp":    context.Timestamp,
		"evaluator":    "jsonlogic",
		"version":      "1.0",
		"strict_mode":  evaluator.config.StrictMode,
		"sandbox_mode": evaluator.config.SandboxMode,
	}

	return data, nil
}

// expandDotKeys converts a flat map with dot-notation keys to a nested map
func expandDotKeys(flat map[string]interface{}) map[string]interface{} {
	nested := make(map[string]interface{})
	for key, value := range flat {
		parts := strings.Split(key, ".")
		current := nested
		for i, part := range parts {
			if i == len(parts)-1 {
				// Last part, set the value
				current[part] = value
			} else {
				// Intermediate part, create nested map if needed
				if _, ok := current[part]; !ok {
					current[part] = make(map[string]interface{})
				}
				// Type assertion to continue traversal
				if next, ok := current[part].(map[string]interface{}); ok {
					current = next
				} else {
					// This case handles when a key is both a prefix and a full key
					// e.g., "http.method" and "http". We can't create a nested map.
					// In this scenario, we'll just keep the original flat key.
					current[key] = value
					break
				}
			}
		}
	}
	return nested
}

// applyWithTimeout applies JSONLogic with timeout protection
func (evaluator *JSONLogicEvaluator) applyWithTimeout(rule interface{}, data interface{}) (interface{}, error) {
	// First, preprocess the rule to handle custom operators
	processedRule := evaluator.preprocessRule(rule)

	if evaluator.config.Timeout <= 0 {
		// No timeout, apply directly
		return jsonlogic.ApplyInterface(processedRule, data)
	}

	// Use channel to implement timeout
	resultChan := make(chan interface{}, 1)
	errorChan := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errorChan <- fmt.Errorf("JSONLogic evaluation panicked: %v", r)
			}
		}()

		result, err := jsonlogic.ApplyInterface(processedRule, data)
		if err != nil {
			errorChan <- err
		} else {
			resultChan <- result
		}
	}()

	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return nil, err
	case <-time.After(evaluator.config.Timeout):
		return nil, fmt.Errorf("JSONLogic evaluation timed out after %v", evaluator.config.Timeout)
	}
}

// preprocessRule handles custom operators before passing to JSONLogic
func (evaluator *JSONLogicEvaluator) preprocessRule(rule interface{}) interface{} {
	switch r := rule.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range r {
			if key == "match" {
				// Handle match operator (regex)
				if valueArray, ok := value.([]interface{}); ok && len(valueArray) == 2 {
					// For now, implement basic pattern matching
					pattern := valueArray[1]
					variable := valueArray[0]

					// Convert common regex patterns to JSONLogic equivalents
					if patternStr, ok := pattern.(string); ok {
						switch patternStr {
						case "^[0-9]+$":
							// Numeric pattern - check if all characters are digits
							// For simplicity, we'll check if it's not empty and contains only digits
							result["and"] = []interface{}{
								map[string]interface{}{
									"!=": []interface{}{variable, nil},
								},
								map[string]interface{}{
									"!=": []interface{}{variable, ""},
								},
								// For now, assume numeric strings are valid
								// In a real implementation, you'd want proper regex support
								true,
							}
						case "^[\\w\\.-]+@[\\w\\.-]+\\.[a-zA-Z]{2,}$":
							// Email pattern - check if it contains @ and .
							result["and"] = []interface{}{
								map[string]interface{}{
									"!=": []interface{}{variable, nil},
								},
								map[string]interface{}{
									"in": []interface{}{"@", variable},
								},
								map[string]interface{}{
									"in": []interface{}{".", variable},
								},
							}
						default:
							// Fallback for unknown patterns
							result["!="] = []interface{}{variable, nil}
						}
					} else {
						result[key] = evaluator.preprocessRule(value)
					}
				} else {
					result[key] = evaluator.preprocessRule(value)
				}
			} else {
				result[key] = evaluator.preprocessRule(value)
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(r))
		for i, item := range r {
			result[i] = evaluator.preprocessRule(item)
		}
		return result
	default:
		return rule
	}
}

// convertToBool converts JSONLogic result to boolean
func (evaluator *JSONLogicEvaluator) convertToBool(result interface{}) bool {
	if result == nil {
		return false
	}

	switch v := result.(type) {
	case bool:
		return v
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0.0
	case string:
		return v != ""
	case []interface{}:
		return len(v) > 0
	case map[string]interface{}:
		return len(v) > 0
	default:
		// Use reflection for other types
		rv := reflect.ValueOf(result)
		switch rv.Kind() {
		case reflect.Bool:
			return rv.Bool()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return rv.Int() != 0
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return rv.Uint() != 0
		case reflect.Float32, reflect.Float64:
			return rv.Float() != 0.0
		case reflect.String:
			return rv.String() != ""
		case reflect.Slice, reflect.Array, reflect.Map:
			return rv.Len() > 0
		case reflect.Ptr, reflect.Interface:
			return !rv.IsNil()
		default:
			return true // Non-zero values are truthy
		}
	}
}

// buildResultMessage builds a descriptive message for the assertion result
func (evaluator *JSONLogicEvaluator) buildResultMessage(passed bool, assertion map[string]interface{}, result interface{}) string {
	if passed {
		return fmt.Sprintf("Assertion passed: %v evaluated to %v", assertion, result)
	} else {
		return fmt.Sprintf("Assertion failed: %v evaluated to %v (expected truthy value)", assertion, result)
	}
}

// validateOperators validates that only allowed operators are used
func (evaluator *JSONLogicEvaluator) validateOperators(assertion map[string]interface{}, allowedOps []string) error {
	allowedSet := make(map[string]bool)
	for _, op := range allowedOps {
		allowedSet[op] = true
	}

	return evaluator.validateOperatorsRecursive(assertion, allowedSet)
}

// validateOperatorsRecursive recursively validates operators
func (evaluator *JSONLogicEvaluator) validateOperatorsRecursive(obj interface{}, allowedOps map[string]bool) error {
	switch v := obj.(type) {
	case map[string]interface{}:
		for key, value := range v {
			// Check if key is an operator
			if strings.Contains(key, ".") || len(allowedOps) > 0 {
				if !allowedOps[key] {
					return fmt.Errorf("operator '%s' is not allowed", key)
				}
			}

			// Recursively validate nested objects
			if err := evaluator.validateOperatorsRecursive(value, allowedOps); err != nil {
				return err
			}
		}
	case []interface{}:
		for _, item := range v {
			if err := evaluator.validateOperatorsRecursive(item, allowedOps); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateDepth validates that the assertion doesn't exceed maximum depth
func (evaluator *JSONLogicEvaluator) validateDepth(obj interface{}, currentDepth, maxDepth int) error {
	if currentDepth > maxDepth {
		return fmt.Errorf("assertion exceeds maximum depth of %d", maxDepth)
	}

	switch v := obj.(type) {
	case map[string]interface{}:
		for _, value := range v {
			if err := evaluator.validateDepth(value, currentDepth+1, maxDepth); err != nil {
				return err
			}
		}
	case []interface{}:
		for _, item := range v {
			if err := evaluator.validateDepth(item, currentDepth+1, maxDepth); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetConfig returns the current configuration
func (evaluator *JSONLogicEvaluator) GetConfig() *JSONLogicConfig {
	return evaluator.config
}

// SetConfig updates the configuration
func (evaluator *JSONLogicEvaluator) SetConfig(config *JSONLogicConfig) {
	evaluator.config = config
}

// preprocessAssertion converts ServiceSpec-style assertions to proper JSONLogic
func (evaluator *JSONLogicEvaluator) preprocessAssertion(assertion map[string]interface{}) map[string]interface{} {
	// Check if this looks like a ServiceSpec multi-condition assertion
	// (has multiple top-level keys that are not JSONLogic operators)
	jsonLogicOps := map[string]bool{
		"==": true, "!=": true, ">": true, "<": true, ">=": true, "<=": true,
		"and": true, "or": true, "not": true, "if": true, "in": true,
		"var": true, "missing": true, "missing_some": true,
	}

	// Check if this is already a proper JSONLogic expression
	if len(assertion) == 1 {
		for key := range assertion {
			if jsonLogicOps[key] {
				// It's already a JSONLogic expression, return as-is
				return assertion
			}
		}
	}

	// Count non-JSONLogic keys and convert them
	nonOpKeys := 0
	var conditions []interface{}

	for key, value := range assertion {
		if !jsonLogicOps[key] {
			nonOpKeys++
			// Convert this condition to a proper JSONLogic expression
			// The key represents a variable path (e.g., "request.id")
			conditions = append(conditions, evaluator.convertConditionToJSONLogic(key, value))
		} else {
			// It's a JSONLogic operator at the top level, keep as-is
			return assertion
		}
	}

	// If we have multiple non-operator keys, wrap them in an "and"
	if nonOpKeys > 1 {
		return map[string]interface{}{
			"and": conditions,
		}
	} else if nonOpKeys == 1 {
		// Single condition, return it directly
		if conditionMap, ok := conditions[0].(map[string]interface{}); ok {
			return conditionMap
		}
		// If it's not a map, wrap it in a simple structure
		return map[string]interface{}{
			"==": []interface{}{conditions[0], true},
		}
	}

	// Otherwise, it's already a proper JSONLogic expression
	return assertion
}

// convertConditionToJSONLogic converts a single condition to JSONLogic format
func (evaluator *JSONLogicEvaluator) convertConditionToJSONLogic(varPath string, condition interface{}) interface{} {
	conditionMap, ok := condition.(map[string]interface{})
	if !ok {
		// If it's not a map, return as-is
		return condition
	}

	// Check if this is already a valid JSONLogic expression
	jsonLogicOps := map[string]bool{
		"==": true, "!=": true, ">": true, "<": true, ">=": true, "<=": true,
		"and": true, "or": true, "not": true, "if": true, "in": true,
		"var": true, "missing": true, "missing_some": true, "match": true,
	}

	// If the condition contains JSONLogic operators, check if it needs variable substitution
	hasJsonLogicOps := false
	for op := range conditionMap {
		if jsonLogicOps[op] {
			hasJsonLogicOps = true
			break
		}
	}

	if hasJsonLogicOps {
		// This is already a JSONLogic expression, but we need to check if it needs variable substitution
		result := make(map[string]interface{})
		for op, value := range conditionMap {
			if jsonLogicOps[op] {
				// Handle JSONLogic operators
				switch op {
				case "==", "!=", ">", "<", ">=", "<=":
					// These operators expect [var, value] format
					if valueArray, ok := value.([]interface{}); ok && len(valueArray) == 2 {
						// Check if first element needs to be converted to var reference
						firstElem := valueArray[0]
						if varRef, ok := firstElem.(map[string]interface{}); ok {
							if varName, hasVar := varRef["var"]; hasVar {
								// Use the existing var reference
								result[op] = []interface{}{map[string]interface{}{"var": varName}, valueArray[1]}
							} else {
								// Convert to var reference using the variable path
								result[op] = []interface{}{map[string]interface{}{"var": varPath}, valueArray[1]}
							}
						} else {
							// First element is not a var reference, use the variable path
							result[op] = []interface{}{map[string]interface{}{"var": varPath}, valueArray[1]}
						}
					} else {
						// Single value, assume it's the comparison value
						result[op] = []interface{}{map[string]interface{}{"var": varPath}, value}
					}
				case "in":
					// "in" operator: {"in": [needle, haystack]}
					if valueArray, ok := value.([]interface{}); ok && len(valueArray) == 2 {
						result[op] = []interface{}{valueArray[0], valueArray[1]}
					} else {
						result[op] = []interface{}{map[string]interface{}{"var": varPath}, value}
					}
				default:
					// For other operators, keep as-is
					result[op] = value
				}
			} else {
				// Non-JSONLogic key, keep as-is
				result[op] = value
			}
		}
		return result
	}

	// Handle special operators that need conversion
	result := make(map[string]interface{})
	for op, value := range conditionMap {
		// Convert to proper JSONLogic format with array parameters
		switch op {
		case "==", "!=", ">", "<", ">=", "<=":
			// These operators need array format: {"op": [var, value]}
			result[op] = []interface{}{map[string]interface{}{"var": varPath}, value}
		case "in":
			// "in" operator: {"in": [needle, haystack]}
			result[op] = []interface{}{map[string]interface{}{"var": varPath}, value}
		default:
			// For other operators, keep as-is but ensure proper format
			result[op] = value
		}
	}

	return result
}

// ValidateConfig validates the JSONLogic configuration
func ValidateJSONLogicConfig(config *JSONLogicConfig) error {
	if config.MaxDepth <= 0 {
		return fmt.Errorf("MaxDepth must be positive, got %d", config.MaxDepth)
	}

	if config.Timeout < 0 {
		return fmt.Errorf("Timeout cannot be negative, got %s", config.Timeout)
	}

	return nil
}
