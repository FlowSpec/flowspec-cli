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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flowspec/flowspec-cli/internal/models"
)

// TraceFileParser defines the interface for parsing trace files
type TraceFileParser interface {
	CanParse(filename string) bool
	ParseFile(filepath string) (*models.TraceData, error)
	GetSupportedFormats() []string
}

// TraceFormat represents a supported trace format
type TraceFormat string

const (
	FormatFlowSpecTrace TraceFormat = "flowspec-trace"
	FormatOTLP          TraceFormat = "otlp"
	FormatHAR           TraceFormat = "har"
)

// DefaultTraceFileParser implements the TraceFileParser interface
type DefaultTraceFileParser struct{}

// NewTraceFileParser creates a new trace file parser
func NewTraceFileParser() *DefaultTraceFileParser {
	return &DefaultTraceFileParser{}
}

// CanParse returns true if the file can be parsed as a trace file
func (p *DefaultTraceFileParser) CanParse(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".json"
}

// ParseFile parses a trace file and returns TraceData
func (p *DefaultTraceFileParser) ParseFile(filepath string) (*models.TraceData, error) {
	// Read the file
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read trace file: %w", err)
	}

	// Try to detect the format and parse accordingly
	format, err := p.detectFormat(data)
	if err != nil {
		return nil, fmt.Errorf("failed to detect trace format: %w", err)
	}

	switch format {
	case FormatFlowSpecTrace:
		return p.parseFlowSpecTrace(data)
	case FormatOTLP:
		return p.parseOTLPTrace(data)
	default:
		return nil, NewFormatDetectionError(string(format), p.GetSupportedFormats())
	}
}

// GetSupportedFormats returns a list of supported trace formats
func (p *DefaultTraceFileParser) GetSupportedFormats() []string {
	return []string{
		"flowspec-trace.json",
		"otlp.json",
	}
}

// detectFormat attempts to detect the trace file format
func (p *DefaultTraceFileParser) detectFormat(data []byte) (TraceFormat, error) {
	// Try to parse as JSON first
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return "", fmt.Errorf("invalid JSON format: %w", err)
	}

	// Check for FlowSpec trace format
	if p.isFlowSpecTrace(jsonData) {
		return FormatFlowSpecTrace, nil
	}

	// Check for OTLP format
	if p.isOTLPTrace(jsonData) {
		return FormatOTLP, nil
	}

	// Check for HAR format (basic detection)
	if p.isHARTrace(jsonData) {
		return FormatHAR, fmt.Errorf("HAR format detected but not yet supported")
	}

	return "", NewFormatDetectionError("unknown", p.GetSupportedFormats())
}

// isFlowSpecTrace checks if the JSON data is in FlowSpec trace format
func (p *DefaultTraceFileParser) isFlowSpecTrace(data map[string]interface{}) bool {
	// FlowSpec trace format should have traceId and spans fields
	_, hasTraceId := data["traceId"]
	_, hasSpans := data["spans"]
	
	// Optional: check for format version or type indicator
	if formatType, ok := data["format"]; ok {
		if formatStr, ok := formatType.(string); ok && formatStr == "flowspec" {
			return true
		}
	}
	
	return hasTraceId && hasSpans
}

// isOTLPTrace checks if the JSON data is in OTLP format
func (p *DefaultTraceFileParser) isOTLPTrace(data map[string]interface{}) bool {
	// OTLP format should have resourceSpans field
	_, hasResourceSpans := data["resourceSpans"]
	return hasResourceSpans
}

// isHARTrace checks if the JSON data is in HAR format
func (p *DefaultTraceFileParser) isHARTrace(data map[string]interface{}) bool {
	// HAR format should have log field with entries
	if log, ok := data["log"]; ok {
		if logMap, ok := log.(map[string]interface{}); ok {
			_, hasEntries := logMap["entries"]
			return hasEntries
		}
	}
	return false
}

// parseFlowSpecTrace parses a FlowSpec trace format
func (p *DefaultTraceFileParser) parseFlowSpecTrace(data []byte) (*models.TraceData, error) {
	// First try to parse as the internal map format
	var traceData models.TraceData
	if err := json.Unmarshal(data, &traceData); err == nil {
		// Successfully parsed as map format, validate it
		if err := p.validateTraceData(&traceData); err != nil {
			return nil, fmt.Errorf("invalid FlowSpec trace data: %w", err)
		}
		
		// Build span tree
		if err := traceData.BuildSpanTree(); err != nil {
			return nil, fmt.Errorf("failed to build span tree: %w", err)
		}
		
		return &traceData, nil
	}
	
	// Try to parse as the compatible array format
	var compatData models.TraceDataCompat
	if err := json.Unmarshal(data, &compatData); err != nil {
		return nil, fmt.Errorf("failed to parse FlowSpec trace in either format: %w", err)
	}
	
	// Convert to internal format
	traceData = *models.FromCompatFormat(&compatData)
	
	// Validate the trace data
	if err := p.validateTraceData(&traceData); err != nil {
		return nil, fmt.Errorf("invalid FlowSpec trace data: %w", err)
	}

	// Build span tree
	if err := traceData.BuildSpanTree(); err != nil {
		return nil, fmt.Errorf("failed to build span tree: %w", err)
	}

	return &traceData, nil
}

// parseOTLPTrace parses an OTLP trace format (delegates to existing ingestor)
func (p *DefaultTraceFileParser) parseOTLPTrace(data []byte) (*models.TraceData, error) {
	// For OTLP format, we can delegate to the existing ingestor
	// This is a simplified approach - in a real implementation, we might want to
	// refactor the ingestor to be more modular
	return nil, fmt.Errorf("OTLP format parsing should be handled by the existing TraceIngestor")
}

// validateTraceData validates the parsed trace data
func (p *DefaultTraceFileParser) validateTraceData(traceData *models.TraceData) error {
	if traceData.TraceID == "" {
		return fmt.Errorf("missing traceId")
	}

	if len(traceData.Spans) == 0 {
		return fmt.Errorf("no spans found")
	}

	// Validate each span
	for spanID, span := range traceData.Spans {
		if span.SpanID == "" {
			return fmt.Errorf("span missing spanId")
		}
		if span.SpanID != spanID {
			return fmt.Errorf("span ID mismatch: key=%s, span.spanId=%s", spanID, span.SpanID)
		}
		if span.TraceID == "" {
			return fmt.Errorf("span %s missing traceId", spanID)
		}
		if span.TraceID != traceData.TraceID {
			return fmt.Errorf("span %s has different traceId: expected=%s, actual=%s", spanID, traceData.TraceID, span.TraceID)
		}
		if span.Name == "" {
			return fmt.Errorf("span %s missing name", spanID)
		}
		if span.StartTime <= 0 {
			return fmt.Errorf("span %s has invalid startTime: %d", spanID, span.StartTime)
		}
		if span.EndTime <= 0 {
			return fmt.Errorf("span %s has invalid endTime: %d", spanID, span.EndTime)
		}
		if span.EndTime <= span.StartTime {
			return fmt.Errorf("span %s has endTime <= startTime: start=%d, end=%d", spanID, span.StartTime, span.EndTime)
		}
	}

	return nil
}

// FormatDetectionError represents an error in format detection with suggestions
type FormatDetectionError struct {
	DetectedFormat string
	SupportedFormats []string
	ConversionSuggestions []string
}

// Error implements the error interface
func (e *FormatDetectionError) Error() string {
	msg := fmt.Sprintf("unsupported trace format")
	if e.DetectedFormat != "" {
		msg += fmt.Sprintf(" (detected: %s)", e.DetectedFormat)
	}
	
	if len(e.SupportedFormats) > 0 {
		msg += fmt.Sprintf(". Supported formats: %s", strings.Join(e.SupportedFormats, ", "))
	}
	
	if len(e.ConversionSuggestions) > 0 {
		msg += fmt.Sprintf(". Conversion suggestions: %s", strings.Join(e.ConversionSuggestions, "; "))
	}
	
	msg += ". For more information, visit: https://flowspec.dev/docs/trace-formats"
	
	return msg
}

// NewFormatDetectionError creates a new format detection error with suggestions
func NewFormatDetectionError(detectedFormat string, supportedFormats []string) *FormatDetectionError {
	suggestions := []string{}
	
	switch detectedFormat {
	case "har":
		suggestions = append(suggestions, "Use har2flowspec converter")
	case "jaeger":
		suggestions = append(suggestions, "Export from Jaeger as OTLP JSON")
	case "zipkin":
		suggestions = append(suggestions, "Use zipkin2otlp converter")
	default:
		suggestions = append(suggestions, "Convert to FlowSpec trace format or OTLP JSON")
	}
	
	return &FormatDetectionError{
		DetectedFormat: detectedFormat,
		SupportedFormats: supportedFormats,
		ConversionSuggestions: suggestions,
	}
}