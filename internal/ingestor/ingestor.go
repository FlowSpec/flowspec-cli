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

package ingestor

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/flowspec/flowspec-cli/internal/models"
)

// TraceIngestor defines the interface for ingesting OpenTelemetry traces
type TraceIngestor interface {
	IngestFromFile(filePath string) (*models.TraceData, error)
	IngestFromReader(reader io.Reader) (*models.TraceData, error)
	SetMemoryLimit(limitMB int64)
	GetMemoryUsage() int64
}

// TraceQuery defines the interface for querying trace data
type TraceQuery interface {
	FindSpanByID(spanID string) *models.Span
	FindSpansByName(name string) []*models.Span
	FindSpansByOperationID(operationID string) []*models.Span
	GetRootSpan() *models.Span
	GetAllSpans() []*models.Span
}

// TraceStore provides storage and querying capabilities for trace data
type TraceStore struct {
	traceData      *models.TraceData
	spanIndex      map[string]*models.Span   // spanID -> Span
	nameIndex      map[string][]*models.Span // span name -> Spans
	operationIndex map[string][]*models.Span // operation ID -> Spans
	mu             sync.RWMutex
}

// DefaultTraceIngestor implements the TraceIngestor interface
type DefaultTraceIngestor struct {
	memoryLimit   int64 // Memory limit in bytes
	currentMemory int64 // Current memory usage estimate
	mu            sync.RWMutex
}

// IngestorConfig holds configuration for the trace ingestor
type IngestorConfig struct {
	MemoryLimitMB   int64 // Memory limit in MB
	EnableStreaming bool  // Enable streaming for large files
	ChunkSize       int   // Chunk size for streaming
	MaxFileSize     int64 // Maximum file size in bytes
	EnableMetrics   bool  // Enable performance metrics
}

// IngestMetrics tracks ingestion performance
type IngestMetrics struct {
	StartTime      time.Time
	EndTime        time.Time
	TotalSpans     int
	ProcessedSpans int
	MemoryUsed     int64
	FileSize       int64
	ProcessingTime time.Duration
	mu             sync.RWMutex
}

// OTLPTrace represents the root structure of an OTLP JSON trace
type OTLPTrace struct {
	ResourceSpans []ResourceSpan `json:"resourceSpans"`
}

// ResourceSpan represents a resource span in OTLP format
type ResourceSpan struct {
	Resource   Resource    `json:"resource"`
	ScopeSpans []ScopeSpan `json:"scopeSpans"`
}

// Resource represents a resource in OTLP format
type Resource struct {
	Attributes []Attribute `json:"attributes"`
}

// ScopeSpan represents a scope span in OTLP format
type ScopeSpan struct {
	Scope Scope      `json:"scope"`
	Spans []OTLPSpan `json:"spans"`
}

// Scope represents a scope in OTLP format
type Scope struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// OTLPSpan represents a span in OTLP JSON format
type OTLPSpan struct {
	TraceID           string      `json:"traceId"`
	SpanID            string      `json:"spanId"`
	ParentSpanID      string      `json:"parentSpanId,omitempty"`
	Name              string      `json:"name"`
	Kind              SpanKind    `json:"kind"`
	StartTimeUnixNano string      `json:"startTimeUnixNano"`
	EndTimeUnixNano   string      `json:"endTimeUnixNano"`
	Attributes        []Attribute `json:"attributes"`
	Status            Status      `json:"status"`
	Events            []Event     `json:"events"`
}

// SpanKind represents the span kind that can be either string or int
type SpanKind int

// UnmarshalJSON implements custom unmarshaling for SpanKind
func (sk *SpanKind) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as int first
	var intVal int
	if err := json.Unmarshal(data, &intVal); err == nil {
		*sk = SpanKind(intVal)
		return nil
	}

	// Try to unmarshal as string
	var strVal string
	if err := json.Unmarshal(data, &strVal); err != nil {
		return err
	}

	// Convert string values to int
	switch strVal {
	case "SPAN_KIND_UNSPECIFIED":
		*sk = 0
	case "SPAN_KIND_INTERNAL":
		*sk = 1
	case "SPAN_KIND_SERVER":
		*sk = 2
	case "SPAN_KIND_CLIENT":
		*sk = 3
	case "SPAN_KIND_PRODUCER":
		*sk = 4
	case "SPAN_KIND_CONSUMER":
		*sk = 5
	default:
		*sk = 0 // Default to unspecified
	}

	return nil
}

// Attribute represents an attribute in OTLP format
type Attribute struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// Status represents span status in OTLP format
type Status struct {
	Code    StatusCode `json:"code"`
	Message string     `json:"message"`
}

// StatusCode represents the status code that can be either string or int
type StatusCode int

// UnmarshalJSON implements custom unmarshaling for StatusCode
func (sc *StatusCode) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as int first
	var intVal int
	if err := json.Unmarshal(data, &intVal); err == nil {
		*sc = StatusCode(intVal)
		return nil
	}

	// Try to unmarshal as string
	var strVal string
	if err := json.Unmarshal(data, &strVal); err != nil {
		return err
	}

	// Convert string values to int
	switch strVal {
	case "STATUS_CODE_UNSET":
		*sc = 0
	case "STATUS_CODE_OK":
		*sc = 1
	case "STATUS_CODE_ERROR":
		*sc = 2
	default:
		*sc = 0 // Default to unset
	}

	return nil
}

// Event represents a span event in OTLP format
type Event struct {
	TimeUnixNano string      `json:"timeUnixNano"`
	Name         string      `json:"name"`
	Attributes   []Attribute `json:"attributes"`
}

// DefaultIngestorConfig returns a default ingestor configuration
func DefaultIngestorConfig() *IngestorConfig {
	return &IngestorConfig{
		MemoryLimitMB:   500, // 500MB default limit
		EnableStreaming: true,
		ChunkSize:       1024 * 1024,       // 1MB chunks
		MaxFileSize:     100 * 1024 * 1024, // 100MB max file size
		EnableMetrics:   true,
	}
}

// NewTraceIngestor creates a new trace ingestor with default configuration
func NewTraceIngestor() *DefaultTraceIngestor {
	config := DefaultIngestorConfig()
	return &DefaultTraceIngestor{
		memoryLimit: config.MemoryLimitMB * 1024 * 1024, // Convert to bytes
	}
}

// NewTraceIngestorWithConfig creates a new trace ingestor with custom configuration
func NewTraceIngestorWithConfig(config *IngestorConfig) *DefaultTraceIngestor {
	return &DefaultTraceIngestor{
		memoryLimit: config.MemoryLimitMB * 1024 * 1024, // Convert to bytes
	}
}

// NewTraceStore creates a new trace store
func NewTraceStore() *TraceStore {
	return &TraceStore{
		spanIndex:      make(map[string]*models.Span),
		nameIndex:      make(map[string][]*models.Span),
		operationIndex: make(map[string][]*models.Span),
	}
}

// NewIngestMetrics creates a new ingest metrics tracker
func NewIngestMetrics() *IngestMetrics {
	return &IngestMetrics{
		StartTime: time.Now(),
	}
}

// IngestFromFile implements the TraceIngestor interface
func (ti *DefaultTraceIngestor) IngestFromFile(filePath string) (*models.TraceData, error) {
	// Check if file exists and get size
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to access file %s: %w", filePath, err)
	}

	// Check file size limits
	if fileInfo.Size() > 100*1024*1024 { // 100MB limit
		return nil, fmt.Errorf("file size %d bytes exceeds maximum limit of 100MB", fileInfo.Size())
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	// Ingest from reader
	return ti.IngestFromReader(file)
}

// IngestFromReader implements the TraceIngestor interface
func (ti *DefaultTraceIngestor) IngestFromReader(reader io.Reader) (*models.TraceData, error) {
	metrics := NewIngestMetrics()
	defer metrics.Finish()

	// Check memory before starting
	if err := ti.checkMemoryLimit(); err != nil {
		return nil, err
	}

	// Read and parse JSON
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read trace data: %w", err)
	}

	// Update memory usage estimate
	ti.updateMemoryUsage(int64(len(data)))
	metrics.FileSize = int64(len(data))

	// Parse OTLP JSON
	var otlpTrace OTLPTrace
	if err := json.Unmarshal(data, &otlpTrace); err != nil {
		return nil, fmt.Errorf("failed to parse OTLP JSON: %w", err)
	}

	// Convert to internal format
	traceData, err := ti.convertOTLPToTraceData(otlpTrace, metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to convert OTLP data: %w", err)
	}

	// Build span tree
	if err := traceData.BuildSpanTree(); err != nil {
		return nil, fmt.Errorf("failed to build span tree: %w", err)
	}

	metrics.ProcessedSpans = len(traceData.Spans)
	metrics.MemoryUsed = ti.GetMemoryUsage()

	return traceData, nil
}

// SetMemoryLimit implements the TraceIngestor interface
func (ti *DefaultTraceIngestor) SetMemoryLimit(limitMB int64) {
	ti.mu.Lock()
	defer ti.mu.Unlock()
	ti.memoryLimit = limitMB * 1024 * 1024 // Convert to bytes
}

// GetMemoryUsage implements the TraceIngestor interface
func (ti *DefaultTraceIngestor) GetMemoryUsage() int64 {
	ti.mu.RLock()
	defer ti.mu.RUnlock()
	return ti.currentMemory
}

// checkMemoryLimit checks if current memory usage is within limits
func (ti *DefaultTraceIngestor) checkMemoryLimit() error {
	ti.mu.RLock()
	defer ti.mu.RUnlock()

	// Get current system memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	if int64(m.Alloc) > ti.memoryLimit {
		return fmt.Errorf("memory usage %d bytes exceeds limit %d bytes", m.Alloc, ti.memoryLimit)
	}

	return nil
}

// updateMemoryUsage updates the current memory usage estimate
func (ti *DefaultTraceIngestor) updateMemoryUsage(additionalBytes int64) {
	ti.mu.Lock()
	defer ti.mu.Unlock()
	ti.currentMemory += additionalBytes
}

// convertOTLPToTraceData converts OTLP format to internal TraceData format
func (ti *DefaultTraceIngestor) convertOTLPToTraceData(otlpTrace OTLPTrace, metrics *IngestMetrics) (*models.TraceData, error) {
	traceData := &models.TraceData{
		Spans: make(map[string]*models.Span),
	}

	// An empty ResourceSpans is valid, just means no data.
	if len(otlpTrace.ResourceSpans) == 0 {
		return traceData, nil
	}

	// Process all resource spans
	for _, resourceSpan := range otlpTrace.ResourceSpans {
		for _, scopeSpan := range resourceSpan.ScopeSpans {
			for _, otlpSpan := range scopeSpan.Spans {
				span, err := ti.convertOTLPSpan(otlpSpan)
				if err != nil {
					// Log or collect errors for spans that fail to convert?
					// For now, we fail the entire ingestion.
					return nil, fmt.Errorf("failed to convert span %s: %w", otlpSpan.SpanID, err)
				}

				// Set trace ID if not set
				if traceData.TraceID == "" {
					traceData.TraceID = span.TraceID
				}

				// Add to spans map
				traceData.Spans[span.SpanID] = span
				metrics.TotalSpans++
			}
		}
	}

	return traceData, nil
}

// convertOTLPSpan converts an OTLP span to internal Span format
func (ti *DefaultTraceIngestor) convertOTLPSpan(otlpSpan OTLPSpan) (*models.Span, error) {
	// Parse timestamps
	startTime, err := parseNanoTimestamp(otlpSpan.StartTimeUnixNano)
	if err != nil {
		return nil, fmt.Errorf("invalid start time: %w", err)
	}

	endTime, err := parseNanoTimestamp(otlpSpan.EndTimeUnixNano)
	if err != nil {
		return nil, fmt.Errorf("invalid end time: %w", err)
	}

	// Convert attributes
	attributes := make(map[string]interface{})
	for _, attr := range otlpSpan.Attributes {
		attributes[attr.Key] = extractAttributeValue(attr.Value)
	}

	// Convert status
	status := models.SpanStatus{
		Code:    convertStatusCode(otlpSpan.Status.Code),
		Message: otlpSpan.Status.Message,
	}

	// Convert events
	var events []models.SpanEvent
	for _, event := range otlpSpan.Events {
		eventTime, err := parseNanoTimestamp(event.TimeUnixNano)
		if err != nil {
			continue // Skip invalid events
		}

		eventAttrs := make(map[string]interface{})
		for _, attr := range event.Attributes {
			eventAttrs[attr.Key] = extractAttributeValue(attr.Value)
		}

		events = append(events, models.SpanEvent{
			Name:       event.Name,
			Timestamp:  eventTime,
			Attributes: eventAttrs,
		})
	}

	span := &models.Span{
		SpanID:     otlpSpan.SpanID,
		TraceID:    otlpSpan.TraceID,
		ParentID:   otlpSpan.ParentSpanID,
		Name:       otlpSpan.Name,
		StartTime:  startTime,
		EndTime:    endTime,
		Status:     status,
		Attributes: attributes,
		Events:     events,
	}

	return span, nil
}

// parseNanoTimestamp parses a nanosecond timestamp string
func parseNanoTimestamp(timestampStr string) (int64, error) {
	if timestampStr == "" {
		return 0, fmt.Errorf("empty timestamp")
	}

	// Try to parse as int64 directly (nanoseconds since epoch)
	var timestamp int64
	n, err := fmt.Sscanf(timestampStr, "%d", &timestamp)
	if err != nil {
		return 0, fmt.Errorf("failed to parse timestamp %s: %w", timestampStr, err)
	}
	if n != 1 {
		return 0, fmt.Errorf("failed to parse timestamp %s: invalid format", timestampStr)
	}

	// Validate that the entire string was consumed (no extra characters)
	var extra string
	if _, err := fmt.Sscanf(timestampStr, "%d%s", &timestamp, &extra); err == nil && extra != "" {
		return 0, fmt.Errorf("failed to parse timestamp %s: contains non-numeric characters", timestampStr)
	}

	return timestamp, nil
}

// extractAttributeValue extracts the actual value from OTLP attribute value format
func extractAttributeValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	// If it's already a simple value, return as-is
	switch v := value.(type) {
	case string, int, int64, float64, bool:
		return v
	case map[string]interface{}:
		// Handle OTLP wrapped values
		if stringVal, ok := v["stringValue"]; ok {
			return stringVal
		}
		if intVal, ok := v["intValue"]; ok {
			return intVal
		}
		if doubleVal, ok := v["doubleValue"]; ok {
			return doubleVal
		}
		if boolVal, ok := v["boolValue"]; ok {
			return boolVal
		}
		if bytesVal, ok := v["bytesValue"]; ok {
			return bytesVal
		}
		// If no known wrapper, return the map as-is
		return v
	default:
		// Return unknown types as-is
		return v
	}
}

// convertStatusCode converts OTLP status code to string
func convertStatusCode(code StatusCode) string {
	switch int(code) {
	case 0:
		return "UNSET"
	case 1:
		return "OK"
	case 2:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// TraceStore methods

// SetTraceData sets the trace data for the store
func (ts *TraceStore) SetTraceData(traceData *models.TraceData) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.traceData = traceData
	ts.buildIndexes()
}

// buildIndexes builds search indexes for efficient querying
func (ts *TraceStore) buildIndexes() {
	// Clear existing indexes
	ts.spanIndex = make(map[string]*models.Span)
	ts.nameIndex = make(map[string][]*models.Span)
	ts.operationIndex = make(map[string][]*models.Span)

	if ts.traceData == nil {
		return
	}

	// Build indexes
	for spanID, span := range ts.traceData.Spans {
		// Span ID index
		ts.spanIndex[spanID] = span

		// Name index
		ts.nameIndex[span.Name] = append(ts.nameIndex[span.Name], span)

		// Operation ID index (from attributes)
		if operationID, ok := span.Attributes["operation.id"].(string); ok {
			ts.operationIndex[operationID] = append(ts.operationIndex[operationID], span)
		}
	}
}

// FindSpanByID implements the TraceQuery interface
func (ts *TraceStore) FindSpanByID(spanID string) *models.Span {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.spanIndex[spanID]
}

// FindSpansByName implements the TraceQuery interface
func (ts *TraceStore) FindSpansByName(name string) []*models.Span {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.nameIndex[name]
}

// FindSpansByOperationID implements the TraceQuery interface
func (ts *TraceStore) FindSpansByOperationID(operationID string) []*models.Span {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.operationIndex[operationID]
}

// GetRootSpan implements the TraceQuery interface
func (ts *TraceStore) GetRootSpan() *models.Span {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if ts.traceData != nil {
		return ts.traceData.RootSpan
	}
	return nil
}

// GetAllSpans implements the TraceQuery interface
func (ts *TraceStore) GetAllSpans() []*models.Span {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if ts.traceData == nil {
		return []*models.Span{}
	}

	spans := make([]*models.Span, 0, len(ts.traceData.Spans))
	for _, span := range ts.traceData.Spans {
		spans = append(spans, span)
	}
	return spans
}

// GetTraceData returns the underlying trace data
func (ts *TraceStore) GetTraceData() *models.TraceData {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.traceData
}

// GetSpanCount returns the number of spans in the store
func (ts *TraceStore) GetSpanCount() int {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if ts.traceData == nil {
		return 0
	}
	return len(ts.traceData.Spans)
}

// IngestMetrics methods

// Finish marks the ingestion as finished and calculates final metrics
func (im *IngestMetrics) Finish() {
	im.mu.Lock()
	defer im.mu.Unlock()
	im.EndTime = time.Now()
	im.ProcessingTime = im.EndTime.Sub(im.StartTime)
}

// GetSummary returns a summary of ingestion metrics
func (im *IngestMetrics) GetSummary() map[string]interface{} {
	im.mu.RLock()
	defer im.mu.RUnlock()

	return map[string]interface{}{
		"total_spans":      im.TotalSpans,
		"processed_spans":  im.ProcessedSpans,
		"memory_used":      im.MemoryUsed,
		"file_size":        im.FileSize,
		"processing_time":  im.ProcessingTime.String(),
		"spans_per_second": float64(im.ProcessedSpans) / im.ProcessingTime.Seconds(),
	}
}

// GetProcessingRate returns the processing rate in spans per second
func (im *IngestMetrics) GetProcessingRate() float64 {
	im.mu.RLock()
	defer im.mu.RUnlock()

	if im.ProcessingTime.Seconds() == 0 {
		return 0
	}
	return float64(im.ProcessedSpans) / im.ProcessingTime.Seconds()
}