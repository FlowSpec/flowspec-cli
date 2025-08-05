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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/flowspec/flowspec-cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTraceIngestor(t *testing.T) {
	ingestor := NewTraceIngestor()

	assert.NotNil(t, ingestor)
	assert.Equal(t, int64(500*1024*1024), ingestor.memoryLimit) // 500MB default
	assert.Equal(t, int64(0), ingestor.GetMemoryUsage())
}

func TestNewTraceIngestorWithConfig(t *testing.T) {
	config := &IngestorConfig{
		MemoryLimitMB:   1000,
		EnableStreaming: true,
		ChunkSize:       2048,
	}

	ingestor := NewTraceIngestorWithConfig(config)

	assert.NotNil(t, ingestor)
	assert.Equal(t, int64(1000*1024*1024), ingestor.memoryLimit) // 1000MB
}

func TestDefaultIngestorConfig(t *testing.T) {
	config := DefaultIngestorConfig()

	assert.NotNil(t, config)
	assert.Equal(t, int64(500), config.MemoryLimitMB)
	assert.True(t, config.EnableStreaming)
	assert.Equal(t, 1024*1024, config.ChunkSize)
	assert.Equal(t, int64(100*1024*1024), config.MaxFileSize)
	assert.True(t, config.EnableMetrics)
}

func TestNewTraceStore(t *testing.T) {
	store := NewTraceStore()

	assert.NotNil(t, store)
	assert.NotNil(t, store.spanIndex)
	assert.NotNil(t, store.nameIndex)
	assert.NotNil(t, store.operationIndex)
	assert.Equal(t, 0, store.GetSpanCount())
	assert.Nil(t, store.GetRootSpan())
	assert.Empty(t, store.GetAllSpans())
}

func TestIngestFromReader_ValidOTLP(t *testing.T) {
	ingestor := NewTraceIngestor()

	// Create valid OTLP JSON data
	otlpData := createTestOTLPData()
	reader := strings.NewReader(otlpData)

	traceData, err := ingestor.IngestFromReader(reader)

	require.NoError(t, err)
	assert.NotNil(t, traceData)
	assert.Equal(t, "trace123", traceData.TraceID)
	assert.Len(t, traceData.Spans, 3) // root + 2 children
	assert.NotNil(t, traceData.RootSpan)
	assert.NotNil(t, traceData.SpanTree)

	// Verify spans
	rootSpan := traceData.FindSpanByID("span1")
	assert.NotNil(t, rootSpan)
	assert.Equal(t, "root-operation", rootSpan.Name)
	assert.Equal(t, "", rootSpan.ParentID) // Root span has no parent

	childSpan := traceData.FindSpanByID("span2")
	assert.NotNil(t, childSpan)
	assert.Equal(t, "child-operation", childSpan.Name)
	assert.Equal(t, "span1", childSpan.ParentID)
}

func TestIngestFromReader_InvalidJSON(t *testing.T) {
	ingestor := NewTraceIngestor()

	invalidJSON := `{"invalid": json}`
	reader := strings.NewReader(invalidJSON)

	traceData, err := ingestor.IngestFromReader(reader)

	assert.Error(t, err)
	assert.Nil(t, traceData)
	assert.Contains(t, err.Error(), "failed to parse OTLP JSON")
}

func TestIngestFromReader_EmptyTrace(t *testing.T) {
	ingestor := NewTraceIngestor()

	// An OTLP trace with an empty resourceSpans array is valid.
	emptyTrace := `{"resourceSpans": []}`
	reader := strings.NewReader(emptyTrace)

	traceData, err := ingestor.IngestFromReader(reader)

	// It should not produce an error.
	require.NoError(t, err)
	// It should return a valid, non-nil TraceData object.
	require.NotNil(t, traceData)
	// The TraceData object should contain no spans.
	assert.Len(t, traceData.Spans, 0)
	assert.Nil(t, traceData.RootSpan)
}

func TestIngestFromFile_ValidFile(t *testing.T) {
	ingestor := NewTraceIngestor()

	// Create temporary file with valid OTLP data
	tmpDir := t.TempDir()
	traceFile := filepath.Join(tmpDir, "trace.json")

	otlpData := createTestOTLPData()
	err := os.WriteFile(traceFile, []byte(otlpData), 0644)
	require.NoError(t, err)

	traceData, err := ingestor.IngestFromFile(traceFile)

	require.NoError(t, err)
	assert.NotNil(t, traceData)
	assert.Equal(t, "trace123", traceData.TraceID)
	assert.Len(t, traceData.Spans, 3)
}

func TestIngestFromFile_NonExistentFile(t *testing.T) {
	ingestor := NewTraceIngestor()

	traceData, err := ingestor.IngestFromFile("/non/existent/file.json")

	assert.Error(t, err)
	assert.Nil(t, traceData)
	assert.Contains(t, err.Error(), "failed to access file")
}

func TestIngestFromFile_LargeFile(t *testing.T) {
	ingestor := NewTraceIngestor()

	// Create temporary large file
	tmpDir := t.TempDir()
	largeFile := filepath.Join(tmpDir, "large_trace.json")

	// Create file larger than 100MB limit
	largeData := make([]byte, 101*1024*1024) // 101MB
	for i := range largeData {
		largeData[i] = 'a'
	}

	err := os.WriteFile(largeFile, largeData, 0644)
	require.NoError(t, err)

	traceData, err := ingestor.IngestFromFile(largeFile)

	assert.Error(t, err)
	assert.Nil(t, traceData)
	assert.Contains(t, err.Error(), "exceeds maximum limit")
}

func TestSetMemoryLimit(t *testing.T) {
	ingestor := NewTraceIngestor()

	// Initial limit should be 500MB
	assert.Equal(t, int64(500*1024*1024), ingestor.memoryLimit)

	// Set new limit
	ingestor.SetMemoryLimit(1000) // 1000MB
	assert.Equal(t, int64(1000*1024*1024), ingestor.memoryLimit)
}

func TestTraceStore_SetTraceData(t *testing.T) {
	store := NewTraceStore()

	// Create test trace data
	traceData := &models.TraceData{
		TraceID: "test-trace",
		Spans: map[string]*models.Span{
			"span1": {
				SpanID:  "span1",
				TraceID: "test-trace",
				Name:    "test-operation",
				Attributes: map[string]interface{}{
					"operation.id": "testOp",
				},
			},
			"span2": {
				SpanID:   "span2",
				TraceID:  "test-trace",
				Name:     "another-operation",
				ParentID: "span1",
			},
		},
	}

	store.SetTraceData(traceData)

	assert.Equal(t, traceData, store.GetTraceData())
	assert.Equal(t, 2, store.GetSpanCount())

	// Test indexes
	span1 := store.FindSpanByID("span1")
	assert.NotNil(t, span1)
	assert.Equal(t, "test-operation", span1.Name)

	spansByName := store.FindSpansByName("test-operation")
	assert.Len(t, spansByName, 1)
	assert.Equal(t, "span1", spansByName[0].SpanID)

	spansByOpID := store.FindSpansByOperationID("testOp")
	assert.Len(t, spansByOpID, 1)
	assert.Equal(t, "span1", spansByOpID[0].SpanID)

	allSpans := store.GetAllSpans()
	assert.Len(t, allSpans, 2)
}

func TestTraceStore_EmptyQueries(t *testing.T) {
	store := NewTraceStore()

	// Test queries on empty store
	assert.Nil(t, store.FindSpanByID("nonexistent"))
	assert.Empty(t, store.FindSpansByName("nonexistent"))
	assert.Empty(t, store.FindSpansByOperationID("nonexistent"))
	assert.Nil(t, store.GetRootSpan())
	assert.Empty(t, store.GetAllSpans())
}

func TestConvertOTLPSpan(t *testing.T) {
	ingestor := NewTraceIngestor()

	otlpSpan := OTLPSpan{
		TraceID:           "trace123",
		SpanID:            "span456",
		ParentSpanID:      "parent789",
		Name:              "test-span",
		StartTimeUnixNano: "1640995200000000000", // 2022-01-01 00:00:00 UTC
		EndTimeUnixNano:   "1640995201000000000", // 2022-01-01 00:00:01 UTC
		Attributes: []Attribute{
			{Key: "service.name", Value: "test-service"},
			{Key: "operation.id", Value: "testOp"},
		},
		Status: Status{
			Code:    1, // OK
			Message: "Success",
		},
		Events: []Event{
			{
				TimeUnixNano: "1640995200500000000",
				Name:         "test-event",
				Attributes: []Attribute{
					{Key: "event.type", Value: "info"},
				},
			},
		},
	}

	span, err := ingestor.convertOTLPSpan(otlpSpan)

	require.NoError(t, err)
	assert.Equal(t, "trace123", span.TraceID)
	assert.Equal(t, "span456", span.SpanID)
	assert.Equal(t, "parent789", span.ParentID)
	assert.Equal(t, "test-span", span.Name)
	assert.Equal(t, int64(1640995200000000000), span.StartTime)
	assert.Equal(t, int64(1640995201000000000), span.EndTime)
	assert.Equal(t, "OK", span.Status.Code)
	assert.Equal(t, "Success", span.Status.Message)

	// Check attributes
	assert.Equal(t, "test-service", span.Attributes["service.name"])
	assert.Equal(t, "testOp", span.Attributes["operation.id"])

	// Check events
	assert.Len(t, span.Events, 1)
	assert.Equal(t, "test-event", span.Events[0].Name)
	assert.Equal(t, int64(1640995200500000000), span.Events[0].Timestamp)
	assert.Equal(t, "info", span.Events[0].Attributes["event.type"])
}

func TestConvertOTLPSpan_InvalidTimestamp(t *testing.T) {
	ingestor := NewTraceIngestor()

	otlpSpan := OTLPSpan{
		TraceID:           "trace123",
		SpanID:            "span456",
		Name:              "test-span",
		StartTimeUnixNano: "invalid-timestamp",
		EndTimeUnixNano:   "1640995201000000000",
	}

	span, err := ingestor.convertOTLPSpan(otlpSpan)

	assert.Error(t, err)
	assert.Nil(t, span)
	assert.Contains(t, err.Error(), "invalid start time")
}

func TestParseNanoTimestamp(t *testing.T) {
	testCases := []struct {
		input    string
		expected int64
		hasError bool
	}{
		{"1640995200000000000", 1640995200000000000, false},
		{"0", 0, false},
		{"", 0, true},
		{"invalid", 0, true},
		{"1.5", 0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result, err := parseNanoTimestamp(tc.input)

			if tc.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestConvertStatusCode(t *testing.T) {
	testCases := []struct {
		code     int
		expected string
	}{
		{0, "UNSET"},
		{1, "OK"},
		{2, "ERROR"},
		{99, "UNKNOWN"},
	}

	for _, tc := range testCases {
		t.Run(string(rune(tc.code)), func(t *testing.T) {
			result := convertStatusCode(StatusCode(tc.code))
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIngestMetrics(t *testing.T) {
	metrics := NewIngestMetrics()

	assert.NotNil(t, metrics)
	assert.False(t, metrics.StartTime.IsZero())
	assert.True(t, metrics.EndTime.IsZero())

	// Set some test data
	metrics.TotalSpans = 100
	metrics.ProcessedSpans = 95
	metrics.MemoryUsed = 1024 * 1024 // 1MB
	metrics.FileSize = 2048 * 1024   // 2MB

	// Simulate some processing time
	time.Sleep(10 * time.Millisecond)
	metrics.Finish()

	assert.False(t, metrics.EndTime.IsZero())
	assert.Greater(t, metrics.ProcessingTime, time.Duration(0))

	// Test summary
	summary := metrics.GetSummary()
	assert.Equal(t, 100, summary["total_spans"])
	assert.Equal(t, 95, summary["processed_spans"])
	assert.Equal(t, int64(1024*1024), summary["memory_used"])
	assert.Equal(t, int64(2048*1024), summary["file_size"])
	assert.NotEmpty(t, summary["processing_time"])

	// Test processing rate
	rate := metrics.GetProcessingRate()
	assert.Greater(t, rate, 0.0)
}

func TestCompleteIngestionWorkflow(t *testing.T) {
	// Create ingestor and store
	ingestor := NewTraceIngestor()
	store := NewTraceStore()

	// Create test OTLP data
	otlpData := createTestOTLPData()
	reader := strings.NewReader(otlpData)

	// Ingest trace data
	traceData, err := ingestor.IngestFromReader(reader)
	require.NoError(t, err)

	// Set data in store
	store.SetTraceData(traceData)

	// Verify complete workflow
	assert.Equal(t, "trace123", traceData.TraceID)
	assert.Len(t, traceData.Spans, 3)
	assert.NotNil(t, traceData.RootSpan)
	assert.NotNil(t, traceData.SpanTree)

	// Test tree structure
	assert.Equal(t, "span1", traceData.RootSpan.SpanID)
	assert.Len(t, traceData.SpanTree.Children, 2) // Root has 2 children

	// Test store queries
	rootSpan := store.GetRootSpan()
	assert.NotNil(t, rootSpan)
	assert.Equal(t, "root-operation", rootSpan.Name)

	allSpans := store.GetAllSpans()
	assert.Len(t, allSpans, 3)

	// Test specific queries
	childSpans := store.FindSpansByName("child-operation")
	assert.Len(t, childSpans, 1)

	operationSpans := store.FindSpansByOperationID("rootOp")
	assert.Len(t, operationSpans, 1)
	assert.Equal(t, "span1", operationSpans[0].SpanID)
}

// Helper function to create test OTLP data
func createTestOTLPData() string {
	otlpTrace := OTLPTrace{
		ResourceSpans: []ResourceSpan{
			{
				Resource: Resource{
					Attributes: []Attribute{
						{Key: "service.name", Value: "test-service"},
					},
				},
				ScopeSpans: []ScopeSpan{
					{
						Scope: Scope{
							Name:    "test-scope",
							Version: "1.0.0",
						},
						Spans: []OTLPSpan{
							{
								TraceID:           "trace123",
								SpanID:            "span1",
								Name:              "root-operation",
								StartTimeUnixNano: "1640995200000000000",
								EndTimeUnixNano:   "1640995203000000000",
								Attributes: []Attribute{
									{Key: "operation.id", Value: "rootOp"},
									{Key: "span.kind", Value: "server"},
								},
								Status: Status{Code: 1, Message: "OK"},
							},
							{
								TraceID:           "trace123",
								SpanID:            "span2",
								ParentSpanID:      "span1",
								Name:              "child-operation",
								StartTimeUnixNano: "1640995201000000000",
								EndTimeUnixNano:   "1640995202000000000",
								Attributes: []Attribute{
									{Key: "operation.id", Value: "childOp1"},
								},
								Status: Status{Code: 1, Message: "OK"},
							},
							{
								TraceID:           "trace123",
								SpanID:            "span3",
								ParentSpanID:      "span1",
								Name:              "another-child-operation",
								StartTimeUnixNano: "1640995201500000000",
								EndTimeUnixNano:   "1640995202500000000",
								Attributes: []Attribute{
									{Key: "operation.id", Value: "childOp2"},
								},
								Status: Status{Code: 1, Message: "OK"},
							},
						},
					},
				},
			},
		},
	}

	data, _ := json.Marshal(otlpTrace)
	return string(data)
}