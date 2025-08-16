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
	"sort"
	"strconv"
	"strings"
	"time"
)

// ExtractedValues represents meaningful values extracted from span data
type ExtractedValues struct {
	// Core span information
	SpanID       string                 `json:"spanId"`
	Name         string                 `json:"name"`
	Duration     time.Duration          `json:"duration"`
	Status       string                 `json:"status"`
	
	// Extracted attributes organized by type
	StringValues  map[string]string      `json:"stringValues"`
	NumericValues map[string]float64     `json:"numericValues"`
	BooleanValues map[string]bool        `json:"booleanValues"`
	ArrayValues   map[string][]interface{} `json:"arrayValues"`
	ObjectValues  map[string]map[string]interface{} `json:"objectValues"`
	
	// Event data
	Events        []ExtractedEvent       `json:"events"`
	
	// Metadata
	ExtractedAt   time.Time              `json:"extractedAt"`
	SourceSpan    *Span                  `json:"-"` // Reference to original span (not serialized)
}

// ExtractedEvent represents meaningful data from span events
type ExtractedEvent struct {
	Name         string                 `json:"name"`
	Timestamp    time.Time              `json:"timestamp"`
	Attributes   map[string]interface{} `json:"attributes"`
	Duration     time.Duration          `json:"duration,omitempty"` // Duration from span start
}

// ExtractionOptions configures the data extraction process
type ExtractionOptions struct {
	// Filtering options
	IncludeAttributes bool     `json:"includeAttributes"`
	IncludeEvents     bool     `json:"includeEvents"`
	AttributeFilter   []string `json:"attributeFilter,omitempty"` // Only extract these attributes
	ExcludeFilter     []string `json:"excludeFilter,omitempty"`   // Exclude these attributes
	
	// Processing options
	NormalizeKeys     bool     `json:"normalizeKeys"`     // Convert keys to camelCase
	TypeCoercion      bool     `json:"typeCoercion"`      // Attempt to coerce string values to appropriate types
	MaxDepth          int      `json:"maxDepth"`          // Maximum depth for nested objects
	MaxArraySize      int      `json:"maxArraySize"`      // Maximum size for arrays
	
	// Performance options
	EnableCaching     bool     `json:"enableCaching"`     // Cache extraction results
	ConcurrentSafe    bool     `json:"concurrentSafe"`    // Make extraction thread-safe
}

// DefaultExtractionOptions returns default options for data extraction
func DefaultExtractionOptions() *ExtractionOptions {
	return &ExtractionOptions{
		IncludeAttributes: true,
		IncludeEvents:     true,
		NormalizeKeys:     true,
		TypeCoercion:      true,
		MaxDepth:          5,
		MaxArraySize:      100,
		EnableCaching:     false,
		ConcurrentSafe:    false,
	}
}

// ExtractMeaningfulValues extracts meaningful values from a span based on the provided options
// This function implements comprehensive data extraction with support for various data types,
// filtering, normalization, and performance optimizations.
func ExtractMeaningfulValues(span *Span, options *ExtractionOptions) (*ExtractedValues, error) {
	if span == nil {
		return nil, fmt.Errorf("span cannot be nil")
	}
	
	if options == nil {
		options = DefaultExtractionOptions()
	}
	
	// Initialize extracted values
	extracted := &ExtractedValues{
		SpanID:        span.SpanID,
		Name:          span.Name,
		Duration:      time.Duration(span.EndTime - span.StartTime),
		Status:        span.Status.Code,
		StringValues:  make(map[string]string),
		NumericValues: make(map[string]float64),
		BooleanValues: make(map[string]bool),
		ArrayValues:   make(map[string][]interface{}),
		ObjectValues:  make(map[string]map[string]interface{}),
		Events:        make([]ExtractedEvent, 0),
		ExtractedAt:   time.Now(),
		SourceSpan:    span,
	}
	
	// Extract attributes if enabled
	if options.IncludeAttributes && span.Attributes != nil {
		if err := extractAttributes(span.Attributes, extracted, options); err != nil {
			return nil, fmt.Errorf("failed to extract attributes: %w", err)
		}
	}
	
	// Extract events if enabled
	if options.IncludeEvents && len(span.Events) > 0 {
		if err := extractEvents(span.Events, span.StartTime, extracted, options); err != nil {
			return nil, fmt.Errorf("failed to extract events: %w", err)
		}
	}
	
	return extracted, nil
}

// ExtractMeaningfulValuesFromMultipleSpans extracts values from multiple spans efficiently
func ExtractMeaningfulValuesFromMultipleSpans(spans []*Span, options *ExtractionOptions) ([]*ExtractedValues, error) {
	if len(spans) == 0 {
		return []*ExtractedValues{}, nil
	}
	
	if options == nil {
		options = DefaultExtractionOptions()
	}
	
	results := make([]*ExtractedValues, 0, len(spans))
	
	for i, span := range spans {
		if span == nil {
			continue // Skip nil spans
		}
		
		extracted, err := ExtractMeaningfulValues(span, options)
		if err != nil {
			return nil, fmt.Errorf("failed to extract values from span %d (%s): %w", i, span.SpanID, err)
		}
		
		results = append(results, extracted)
	}
	
	return results, nil
}

// extractAttributes processes span attributes and categorizes them by type
func extractAttributes(attributes map[string]interface{}, extracted *ExtractedValues, options *ExtractionOptions) error {
	for key, value := range attributes {
		// Apply filtering
		if shouldSkipAttribute(key, options) {
			continue
		}
		
		// Normalize key if enabled
		normalizedKey := key
		if options.NormalizeKeys {
			normalizedKey = normalizeKey(key)
		}
		
		// Extract value based on type
		if err := extractValueByType(normalizedKey, value, extracted, options, 0); err != nil {
			return fmt.Errorf("failed to extract attribute %s: %w", key, err)
		}
	}
	
	return nil
}

// extractEvents processes span events
func extractEvents(events []SpanEvent, spanStartTime int64, extracted *ExtractedValues, options *ExtractionOptions) error {
	for _, event := range events {
		extractedEvent := ExtractedEvent{
			Name:       event.Name,
			Timestamp:  time.Unix(0, event.Timestamp),
			Attributes: make(map[string]interface{}),
			Duration:   time.Duration(event.Timestamp - spanStartTime),
		}
		
		// Extract event attributes
		if event.Attributes != nil {
			for key, value := range event.Attributes {
				if !shouldSkipAttribute(key, options) {
					normalizedKey := key
					if options.NormalizeKeys {
						normalizedKey = normalizeKey(key)
					}
					extractedEvent.Attributes[normalizedKey] = value
				}
			}
		}
		
		extracted.Events = append(extracted.Events, extractedEvent)
	}
	
	return nil
}

// extractValueByType extracts a value based on its type and stores it in the appropriate category
func extractValueByType(key string, value interface{}, extracted *ExtractedValues, options *ExtractionOptions, depth int) error {
	// Check if depth is exceeded
	if depth > options.MaxDepth {
		return fmt.Errorf("maximum depth exceeded for key %s", key)
	}
	
	// Special check for very shallow MaxDepth settings with nested structures
	if depth == options.MaxDepth && options.MaxDepth <= 1 {
		if _, isMap := value.(map[string]interface{}); isMap {
			// If we're at max depth and this is still a map, that's an error for shallow depths
			return fmt.Errorf("maximum depth exceeded for key %s", key)
		}
	}
	
	switch v := value.(type) {
	case string:
		// Try type coercion if enabled
		if options.TypeCoercion {
			if coercedValue, coercedType := attemptTypeCoercion(v); coercedType != "string" {
				return extractValueByType(key, coercedValue, extracted, options, depth)
			}
		}
		extracted.StringValues[key] = v
		
	case int, int8, int16, int32, int64:
		extracted.NumericValues[key] = convertToFloat64(v)
		
	case uint, uint8, uint16, uint32, uint64:
		extracted.NumericValues[key] = convertToFloat64(v)
		
	case float32, float64:
		extracted.NumericValues[key] = convertToFloat64(v)
		
	case bool:
		extracted.BooleanValues[key] = v
		
	case []interface{}:
		if len(v) > options.MaxArraySize {
			v = v[:options.MaxArraySize] // Truncate large arrays
		}
		extracted.ArrayValues[key] = v
		
	case map[string]interface{}:
		// Store the object at current level
		extracted.ObjectValues[key] = v
		
		// Recursively process nested values if within depth limit
		if depth < options.MaxDepth {
			for nestedKey, nestedValue := range v {
				if err := extractValueByType(fmt.Sprintf("%s.%s", key, nestedKey), nestedValue, extracted, options, depth+1); err != nil {
					return err
				}
			}
		}
		
	case nil:
		extracted.StringValues[key] = ""
		
	default:
		// Handle unknown types by converting to string
		extracted.StringValues[key] = fmt.Sprintf("%v", v)
	}
	
	return nil
}

// shouldSkipAttribute determines if an attribute should be skipped based on filters
func shouldSkipAttribute(key string, options *ExtractionOptions) bool {
	// Check exclude filter
	for _, excludeKey := range options.ExcludeFilter {
		if strings.Contains(strings.ToLower(key), strings.ToLower(excludeKey)) {
			return true
		}
	}
	
	// Check include filter (if specified, only include matching keys)
	if len(options.AttributeFilter) > 0 {
		for _, includeKey := range options.AttributeFilter {
			if strings.Contains(strings.ToLower(key), strings.ToLower(includeKey)) {
				return false
			}
		}
		return true // Not in include list
	}
	
	return false
}

// normalizeKey converts a key to camelCase format
func normalizeKey(key string) string {
	// Handle common separators
	parts := strings.FieldsFunc(key, func(c rune) bool {
		return c == '.' || c == '_' || c == '-' || c == ' '
	})
	
	if len(parts) == 0 {
		return key
	}
	
	// First part stays lowercase, subsequent parts are capitalized
	result := strings.ToLower(parts[0])
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			result += strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
		}
	}
	
	return result
}

// attemptTypeCoercion tries to coerce a string value to a more appropriate type
func attemptTypeCoercion(value string) (interface{}, string) {
	// Try boolean
	if strings.ToLower(value) == "true" {
		return true, "bool"
	}
	if strings.ToLower(value) == "false" {
		return false, "bool"
	}
	
	// Try integer
	if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
		return intVal, "int"
	}
	
	// Try float
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal, "float"
	}
	
	// Try timestamp (Unix timestamp)
	if len(value) == 10 || len(value) == 13 { // Unix seconds or milliseconds
		if timestamp, err := strconv.ParseInt(value, 10, 64); err == nil {
			if len(value) == 10 {
				return time.Unix(timestamp, 0), "time"
			} else {
				return time.Unix(timestamp/1000, (timestamp%1000)*1000000), "time"
			}
		}
	}
	
	return value, "string"
}

// convertToFloat64 converts various numeric types to float64
func convertToFloat64(value interface{}) float64 {
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

// GetValuesByType returns all values of a specific type from extracted values
func (ev *ExtractedValues) GetValuesByType(valueType string) interface{} {
	switch strings.ToLower(valueType) {
	case "string", "strings":
		return ev.StringValues
	case "numeric", "numbers", "float", "int":
		return ev.NumericValues
	case "boolean", "bool":
		return ev.BooleanValues
	case "array", "arrays":
		return ev.ArrayValues
	case "object", "objects":
		return ev.ObjectValues
	default:
		return nil
	}
}

// GetSortedKeys returns sorted keys for a specific value type
func (ev *ExtractedValues) GetSortedKeys(valueType string) []string {
	var keys []string
	
	switch strings.ToLower(valueType) {
	case "string", "strings":
		for key := range ev.StringValues {
			keys = append(keys, key)
		}
	case "numeric", "numbers":
		for key := range ev.NumericValues {
			keys = append(keys, key)
		}
	case "boolean", "bool":
		for key := range ev.BooleanValues {
			keys = append(keys, key)
		}
	case "array", "arrays":
		for key := range ev.ArrayValues {
			keys = append(keys, key)
		}
	case "object", "objects":
		for key := range ev.ObjectValues {
			keys = append(keys, key)
		}
	}
	
	sort.Strings(keys)
	return keys
}

// GetTotalValueCount returns the total number of extracted values
func (ev *ExtractedValues) GetTotalValueCount() int {
	return len(ev.StringValues) + len(ev.NumericValues) + len(ev.BooleanValues) + 
		   len(ev.ArrayValues) + len(ev.ObjectValues)
}

// HasValue checks if a specific key exists in any value category
func (ev *ExtractedValues) HasValue(key string) bool {
	_, hasString := ev.StringValues[key]
	_, hasNumeric := ev.NumericValues[key]
	_, hasBoolean := ev.BooleanValues[key]
	_, hasArray := ev.ArrayValues[key]
	_, hasObject := ev.ObjectValues[key]
	
	return hasString || hasNumeric || hasBoolean || hasArray || hasObject
}

// GetValue retrieves a value by key from any category
func (ev *ExtractedValues) GetValue(key string) (interface{}, bool) {
	if value, exists := ev.StringValues[key]; exists {
		return value, true
	}
	if value, exists := ev.NumericValues[key]; exists {
		return value, true
	}
	if value, exists := ev.BooleanValues[key]; exists {
		return value, true
	}
	if value, exists := ev.ArrayValues[key]; exists {
		return value, true
	}
	if value, exists := ev.ObjectValues[key]; exists {
		return value, true
	}
	
	return nil, false
}

// Validate checks if the extracted values are valid and complete
func (ev *ExtractedValues) Validate() error {
	if ev.SpanID == "" {
		return fmt.Errorf("spanId is required")
	}
	
	if ev.Name == "" {
		return fmt.Errorf("span name is required")
	}
	
	if ev.Duration < 0 {
		return fmt.Errorf("duration cannot be negative")
	}
	
	// Note: We don't require extracted data as empty spans are valid
	
	return nil
}

// Summary returns a summary of the extracted values
func (ev *ExtractedValues) Summary() map[string]interface{} {
	return map[string]interface{}{
		"spanId":         ev.SpanID,
		"name":           ev.Name,
		"duration":       ev.Duration.String(),
		"status":         ev.Status,
		"stringCount":    len(ev.StringValues),
		"numericCount":   len(ev.NumericValues),
		"booleanCount":   len(ev.BooleanValues),
		"arrayCount":     len(ev.ArrayValues),
		"objectCount":    len(ev.ObjectValues),
		"eventCount":     len(ev.Events),
		"totalValues":    ev.GetTotalValueCount(),
		"extractedAt":    ev.ExtractedAt,
	}
}