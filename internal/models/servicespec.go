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
	"encoding/json"
	"fmt"
)

// ServiceSpec represents a service specification with preconditions and postconditions
type ServiceSpec struct {
	OperationID    string                 `json:"operationId"`
	Description    string                 `json:"description"`
	Preconditions  map[string]interface{} `json:"preconditions"`
	Postconditions map[string]interface{} `json:"postconditions"`
	SourceFile     string                 `json:"sourceFile"`
	LineNumber     int                    `json:"lineNumber"`
}

// ParseResult contains the results of parsing ServiceSpecs from source files
type ParseResult struct {
	Specs   []ServiceSpec          `json:"specs"`
	Errors  []ParseError           `json:"errors"`
	Metrics map[string]interface{} `json:"metrics,omitempty"`
}

// ParseError represents an error that occurred during parsing
type ParseError struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Message string `json:"message"`
}

// Validate checks if the ServiceSpec has all required fields
func (s *ServiceSpec) Validate() error {
	if s.OperationID == "" {
		return fmt.Errorf("operationId is required")
	}
	if s.Description == "" {
		return fmt.Errorf("description is required")
	}
	if s.SourceFile == "" {
		return fmt.Errorf("sourceFile is required")
	}
	if s.LineNumber <= 0 {
		return fmt.Errorf("lineNumber must be positive")
	}
	return nil
}

// ToJSON serializes the ServiceSpec to JSON
func (s *ServiceSpec) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

// FromJSON deserializes JSON data into a ServiceSpec
func (s *ServiceSpec) FromJSON(data []byte) error {
	return json.Unmarshal(data, s)
}

// String returns a string representation of the ServiceSpec
func (s *ServiceSpec) String() string {
	return fmt.Sprintf("ServiceSpec{OperationID: %s, Description: %s, SourceFile: %s, LineNumber: %d}",
		s.OperationID, s.Description, s.SourceFile, s.LineNumber)
}

// Error returns a string representation of the ParseError
func (e *ParseError) Error() string {
	return fmt.Sprintf("%s:%d: %s", e.File, e.Line, e.Message)
}

// Trace-related data structures

// TraceData represents a complete trace with all its spans organized for efficient querying
type TraceData struct {
	TraceID  string           `json:"traceId"`
	RootSpan *Span            `json:"rootSpan"`
	Spans    map[string]*Span `json:"spans"`
	SpanTree *SpanNode        `json:"spanTree"`
}

// Span represents a single span in an OpenTelemetry trace
type Span struct {
	SpanID     string                 `json:"spanId"`
	TraceID    string                 `json:"traceId"`
	ParentID   string                 `json:"parentSpanId,omitempty"`
	Name       string                 `json:"name"`
	StartTime  int64                  `json:"startTime"` // Unix timestamp in nanoseconds
	EndTime    int64                  `json:"endTime"`   // Unix timestamp in nanoseconds
	Status     SpanStatus             `json:"status"`
	Attributes map[string]interface{} `json:"attributes"`
	Events     []SpanEvent            `json:"events"`
}

// SpanStatus represents the status of a span
type SpanStatus struct {
	Code    string `json:"code"` // "OK", "ERROR", "TIMEOUT"
	Message string `json:"message"`
}

// SpanEvent represents an event within a span
type SpanEvent struct {
	Name       string                 `json:"name"`
	Timestamp  int64                  `json:"timestamp"` // Unix timestamp in nanoseconds
	Attributes map[string]interface{} `json:"attributes"`
}

// SpanNode represents a node in the span tree structure
type SpanNode struct {
	Span     *Span       `json:"span"`
	Children []*SpanNode `json:"children"`
}

// BuildSpanTree constructs a hierarchical tree structure from the spans in TraceData
func (td *TraceData) BuildSpanTree() error {
	// If there are no spans, there is no tree to build. This is a valid state.
	if len(td.Spans) == 0 {
		return nil
	}

	nodes := make(map[string]*SpanNode, len(td.Spans))
	var rootNodes []*SpanNode

	// First pass: create all nodes and find potential roots
	for _, span := range td.Spans {
		nodes[span.SpanID] = &SpanNode{Span: span}
		if span.ParentID == "" {
			rootNodes = append(rootNodes, nodes[span.SpanID])
		}
	}

	// Second pass: link children to their parents
	for _, span := range td.Spans {
		if span.ParentID != "" {
			if parentNode, ok := nodes[span.ParentID]; ok {
				childNode := nodes[span.SpanID]
				parentNode.Children = append(parentNode.Children, childNode)
			}
			// Note: Spans with non-existent parents will be ignored and not part of the tree.
		}
	}

	// Determine the final root. In a valid trace, there should be exactly one root.
	// We handle cases with multiple roots gracefully by picking the first one.
	if len(rootNodes) > 0 {
		td.RootSpan = rootNodes[0].Span
		td.SpanTree = rootNodes[0]
	} else if len(td.Spans) > 0 {
		// Handle cases where no span has an empty ParentID (e.g., circular dependencies or all spans have parents)
		// As a fallback, we could pick the span with the earliest start time.
		// For now, we'll consider this an invalid trace structure.
		return fmt.Errorf("no root span found (all spans have parents)")
	}
	// If len(td.Spans) is 0, we've already returned nil, so no error here.

	return nil
}

// GetDuration returns the duration of the span in nanoseconds
func (s *Span) GetDuration() int64 {
	return s.EndTime - s.StartTime
}

// IsRoot returns true if this span is a root span (has no parent)
func (s *Span) IsRoot() bool {
	return s.ParentID == ""
}

// HasError returns true if the span has an error status
func (s *Span) HasError() bool {
	return s.Status.Code == "ERROR"
}

// GetAttribute returns the value of a span attribute, or nil if not found
func (s *Span) GetAttribute(key string) interface{} {
	if s.Attributes == nil {
		return nil
	}
	return s.Attributes[key]
}

// AddEvent adds an event to the span
func (s *Span) AddEvent(event SpanEvent) {
	if s.Events == nil {
		s.Events = []SpanEvent{}
	}
	s.Events = append(s.Events, event)
}

// String returns a string representation of the Span
func (s *Span) String() string {
	return fmt.Sprintf("Span{SpanID: %s, TraceID: %s, Name: %s, Status: %s}",
		s.SpanID, s.TraceID, s.Name, s.Status.Code)
}

// GetDepth returns the depth of this node in the span tree (root = 0)
func (sn *SpanNode) GetDepth() int {
	depth := 0
	// This would need to be calculated from the tree structure
	// For now, we'll implement a simple version
	return depth
}

// GetChildCount returns the number of direct children
func (sn *SpanNode) GetChildCount() int {
	return len(sn.Children)
}

// GetTotalDescendants returns the total number of descendants (children + grandchildren + ...)
func (sn *SpanNode) GetTotalDescendants() int {
	total := len(sn.Children)
	for _, child := range sn.Children {
		total += child.GetTotalDescendants()
	}
	return total
}

// FindSpanByID searches for a span by ID in the trace data
func (td *TraceData) FindSpanByID(spanID string) *Span {
	return td.Spans[spanID]
}

// FindSpansByName searches for spans by name in the trace data
func (td *TraceData) FindSpansByName(name string) []*Span {
	var result []*Span
	for _, span := range td.Spans {
		if span.Name == name {
			result = append(result, span)
		}
	}
	return result
}

// GetAllSpans returns all spans in the trace as a slice
func (td *TraceData) GetAllSpans() []*Span {
	spans := make([]*Span, 0, len(td.Spans))
	for _, span := range td.Spans {
		spans = append(spans, span)
	}
	return spans
}

// AlignmentReport-related data structures

// AlignmentReport represents the complete report of alignment verification
type AlignmentReport struct {
	Summary         AlignmentSummary  `json:"summary"`
	Results         []AlignmentResult `json:"results"`
	ExecutionTime   int64             `json:"executionTime"`   // Total execution time in nanoseconds
	StartTime       int64             `json:"startTime"`       // Start timestamp in Unix nanoseconds
	EndTime         int64             `json:"endTime"`         // End timestamp in Unix nanoseconds
	PerformanceInfo PerformanceInfo   `json:"performanceInfo"` // Performance monitoring data
}

// AlignmentSummary provides summary statistics for the alignment report
type AlignmentSummary struct {
	Total                int     `json:"total"`
	Success              int     `json:"success"`
	Failed               int     `json:"failed"`
	Skipped              int     `json:"skipped"`
	SuccessRate          float64 `json:"successRate"`          // Success rate as percentage (0.0 to 1.0)
	FailureRate          float64 `json:"failureRate"`          // Failure rate as percentage (0.0 to 1.0)
	SkipRate             float64 `json:"skipRate"`             // Skip rate as percentage (0.0 to 1.0)
	AverageExecutionTime int64   `json:"averageExecutionTime"` // Average execution time per spec in nanoseconds
	TotalAssertions      int     `json:"totalAssertions"`      // Total number of assertions evaluated
	FailedAssertions     int     `json:"failedAssertions"`     // Number of failed assertions
}

// PerformanceInfo contains performance monitoring data
type PerformanceInfo struct {
	SpecsProcessed      int     `json:"specsProcessed"`      // Number of specs processed
	SpansMatched        int     `json:"spansMatched"`        // Number of spans matched
	AssertionsEvaluated int     `json:"assertionsEvaluated"` // Total assertions evaluated
	ConcurrentWorkers   int     `json:"concurrentWorkers"`   // Number of concurrent workers used
	MemoryUsageMB       float64 `json:"memoryUsageMB"`       // Peak memory usage in MB
	ProcessingRate      float64 `json:"processingRate"`      // Specs processed per second
}

// AlignmentResult represents the result of aligning a single ServiceSpec with trace data
type AlignmentResult struct {
	SpecOperationID  string             `json:"specOperationId"`
	Status           AlignmentStatus    `json:"status"`
	Details          []ValidationDetail `json:"details"`
	ExecutionTime    int64              `json:"executionTime"`          // Duration in nanoseconds
	StartTime        int64              `json:"startTime"`              // Start timestamp in Unix nanoseconds
	EndTime          int64              `json:"endTime"`                // End timestamp in Unix nanoseconds
	MatchedSpans     []string           `json:"matchedSpans"`           // IDs of spans that matched this spec
	AssertionsTotal  int                `json:"assertionsTotal"`        // Total number of assertions evaluated
	AssertionsPassed int                `json:"assertionsPassed"`       // Number of assertions that passed
	AssertionsFailed int                `json:"assertionsFailed"`       // Number of assertions that failed
	ErrorMessage     string             `json:"errorMessage,omitempty"` // Error message if processing failed
}

// AlignmentStatus represents the status of an alignment result
type AlignmentStatus string

const (
	StatusSuccess AlignmentStatus = "SUCCESS"
	StatusFailed  AlignmentStatus = "FAILED"
	StatusSkipped AlignmentStatus = "SKIPPED"
)

// ValidationDetail provides detailed information about a specific validation
type ValidationDetail struct {
	Type          string                 `json:"type"` // "precondition" | "postcondition"
	Expression    string                 `json:"expression"`
	Expected      interface{}            `json:"expected"`
	Actual        interface{}            `json:"actual"`
	Message       string                 `json:"message"`
	SpanContext   *Span                  `json:"spanContext,omitempty"`
	FailureReason string                 `json:"failureReason,omitempty"` // Detailed analysis of why the assertion failed
	ContextInfo   map[string]interface{} `json:"contextInfo,omitempty"`   // Additional context information for debugging
	Suggestions   []string               `json:"suggestions,omitempty"`   // Actionable suggestions for fixing the failure
}

// AddResult adds an alignment result to the report and updates the summary
func (ar *AlignmentReport) AddResult(result AlignmentResult) {
	ar.Results = append(ar.Results, result)
	ar.updateSummary()
}

// updateSummary recalculates the summary statistics based on current results
func (ar *AlignmentReport) updateSummary() {
	total := len(ar.Results)
	success := 0
	failed := 0
	skipped := 0
	totalExecutionTime := int64(0)
	totalAssertions := 0
	failedAssertions := 0

	for _, result := range ar.Results {
		switch result.Status {
		case StatusSuccess:
			success++
		case StatusFailed:
			failed++
		case StatusSkipped:
			skipped++
		}

		totalExecutionTime += result.ExecutionTime
		totalAssertions += result.AssertionsTotal
		failedAssertions += result.AssertionsFailed
	}

	ar.Summary = AlignmentSummary{
		Total:            total,
		Success:          success,
		Failed:           failed,
		Skipped:          skipped,
		TotalAssertions:  totalAssertions,
		FailedAssertions: failedAssertions,
	}

	// Calculate rates
	if total > 0 {
		ar.Summary.SuccessRate = float64(success) / float64(total)
		ar.Summary.FailureRate = float64(failed) / float64(total)
		ar.Summary.SkipRate = float64(skipped) / float64(total)
		ar.Summary.AverageExecutionTime = totalExecutionTime / int64(total)
	}
}

// HasFailures returns true if any alignment results have failed
func (ar *AlignmentReport) HasFailures() bool {
	return ar.Summary.Failed > 0
}

// GetSuccessRate returns the success rate as a percentage (0.0 to 1.0)
func (ar *AlignmentReport) GetSuccessRate() float64 {
	if ar.Summary.Total == 0 {
		return 0.0
	}
	return float64(ar.Summary.Success) / float64(ar.Summary.Total)
}

// GetFailureRate returns the failure rate as a percentage (0.0 to 1.0)
func (ar *AlignmentReport) GetFailureRate() float64 {
	if ar.Summary.Total == 0 {
		return 0.0
	}
	return float64(ar.Summary.Failed) / float64(ar.Summary.Total)
}

// String returns a string representation of the AlignmentStatus
func (as AlignmentStatus) String() string {
	return string(as)
}

// IsValid returns true if the AlignmentStatus is one of the valid values
func (as AlignmentStatus) IsValid() bool {
	switch as {
	case StatusSuccess, StatusFailed, StatusSkipped:
		return true
	default:
		return false
	}
}

// AddValidationDetail adds a validation detail to the alignment result
func (ar *AlignmentResult) AddValidationDetail(detail ValidationDetail) {
	ar.Details = append(ar.Details, detail)

	// Update status based on validation details
	ar.updateStatus()
}

// updateStatus updates the alignment result status based on validation details
func (ar *AlignmentResult) updateStatus() {
	if len(ar.Details) == 0 {
		ar.Status = StatusSkipped
		ar.AssertionsTotal = 0
		ar.AssertionsPassed = 0
		ar.AssertionsFailed = 0
		return
	}

	totalAssertions := 0
	passedAssertions := 0
	failedAssertions := 0
	hasFailure := false

	for _, detail := range ar.Details {
		// Skip "matching" type details as they are not assertions
		if detail.Type == "matching" {
			continue
		}

		totalAssertions++

		// Check if this assertion passed or failed
		if detail.IsPassed() {
			passedAssertions++
		} else {
			failedAssertions++
			hasFailure = true
		}
	}

	ar.AssertionsTotal = totalAssertions
	ar.AssertionsPassed = passedAssertions
	ar.AssertionsFailed = failedAssertions

	// Determine overall status
	if hasFailure {
		ar.Status = StatusFailed
	} else if totalAssertions > 0 {
		ar.Status = StatusSuccess
	} else {
		ar.Status = StatusSkipped
	}
}

// GetFailedDetails returns only the validation details that failed
func (ar *AlignmentResult) GetFailedDetails() []ValidationDetail {
	var failed []ValidationDetail
	for _, detail := range ar.Details {
		if detail.Expected != detail.Actual {
			failed = append(failed, detail)
		}
	}
	return failed
}

// GetPreconditionDetails returns only the precondition validation details
func (ar *AlignmentResult) GetPreconditionDetails() []ValidationDetail {
	var preconditions []ValidationDetail
	for _, detail := range ar.Details {
		if detail.Type == "precondition" {
			preconditions = append(preconditions, detail)
		}
	}
	return preconditions
}

// GetPostconditionDetails returns only the postcondition validation details
func (ar *AlignmentResult) GetPostconditionDetails() []ValidationDetail {
	var postconditions []ValidationDetail
	for _, detail := range ar.Details {
		if detail.Type == "postcondition" {
			postconditions = append(postconditions, detail)
		}
	}
	return postconditions
}

// String returns a string representation of the ValidationDetail
func (vd *ValidationDetail) String() string {
	return fmt.Sprintf("ValidationDetail{Type: %s, Expression: %s, Expected: %v, Actual: %v}",
		vd.Type, vd.Expression, vd.Expected, vd.Actual)
}

// IsPassed returns true if the validation detail passed (expected equals actual)
func (vd *ValidationDetail) IsPassed() bool {
	return vd.Expected == vd.Actual
}

// NewAlignmentReport creates a new empty alignment report
func NewAlignmentReport() *AlignmentReport {
	return &AlignmentReport{
		Summary: AlignmentSummary{},
		Results: []AlignmentResult{},
	}
}

// NewAlignmentResult creates a new alignment result with the given operation ID
func NewAlignmentResult(operationID string) *AlignmentResult {
	return &AlignmentResult{
		SpecOperationID: operationID,
		Status:          StatusSkipped, // Default to skipped until validation details are added
		Details:         []ValidationDetail{},
		ExecutionTime:   0,
	}
}

// NewValidationDetail creates a new validation detail
func NewValidationDetail(detailType, expression string, expected, actual interface{}, message string) *ValidationDetail {
	return &ValidationDetail{
		Type:       detailType,
		Expression: expression,
		Expected:   expected,
		Actual:     actual,
		Message:    message,
	}
}
