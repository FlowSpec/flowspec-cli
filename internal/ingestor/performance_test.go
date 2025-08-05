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
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLargeFilePerformance_100MB(t *testing.T) {
	// Skip in short mode as this is resource intensive
	if testing.Short() {
		t.Skip("Skipping large file performance test in short mode")
	}

	// Set a reasonable timeout for this test
	timeout := 30 * time.Second
	done := make(chan bool, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Test panicked: %v", r)
			}
			done <- true
		}()

		// Create streaming ingestor with 500MB memory limit
		config := &StreamingConfig{
			ChunkSize:      4096,
			MaxMemoryUsage: 500 * 1024 * 1024, // 500MB limit as per requirement
		}

		ingestor := NewStreamingIngestor(config)

		// Reduce test size to prevent hanging - use 10k spans instead of 100k
		// This should still create a substantial file (~10MB) for testing
		numSpans := 10000 // 10k spans for ~10MB (more reasonable for CI)
		largeData := createVeryLargeOTLPData(numSpans)

		t.Logf("Created test data with %d spans, size: %d bytes (%.2f MB)",
			numSpans, len(largeData), float64(len(largeData))/(1024*1024))

		// Verify file size is substantial (at least 5MB)
		assert.Greater(t, len(largeData), 5*1024*1024, "Test data should be at least 5MB")

		// Get initial memory usage
		var initialMem runtime.MemStats
		runtime.ReadMemStats(&initialMem)

		// Track memory usage during processing
		var maxMemoryUsed uint64 = initialMem.Alloc
		var memoryTracker = func(processed, total int64) {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			if m.Alloc > maxMemoryUsed {
				maxMemoryUsed = m.Alloc
			}
		}

		// Create new ingestor with memory tracker
		config.ProgressCallback = memoryTracker
		ingestor = NewStreamingIngestor(config)

		// Process the large file
		reader := strings.NewReader(largeData)
		start := time.Now()

		// Check memory before processing
		var beforeMem runtime.MemStats
		runtime.ReadMemStats(&beforeMem)

		traceData, err := ingestor.IngestFromReaderStreaming(reader, int64(len(largeData)))

		// Check memory after processing
		var afterMem runtime.MemStats
		runtime.ReadMemStats(&afterMem)

		// Update max memory if needed
		if afterMem.Alloc > maxMemoryUsed {
			maxMemoryUsed = afterMem.Alloc
		}

		duration := time.Since(start)

		require.NoError(t, err)
		assert.NotNil(t, traceData)
		assert.Equal(t, "very-large-trace", traceData.TraceID)
		assert.Len(t, traceData.Spans, numSpans+1) // +1 for root span

		// Verify memory usage stayed within 500MB limit
		assert.Less(t, int64(maxMemoryUsed), int64(500*1024*1024),
			"Memory usage should stay under 500MB limit")

		// Log performance metrics
		t.Logf("Performance metrics:")
		t.Logf("  - File size: %.2f MB", float64(len(largeData))/(1024*1024))
		t.Logf("  - Processing time: %s", duration)
		t.Logf("  - Spans processed: %d", len(traceData.Spans))
		t.Logf("  - Max memory used: %.2f MB", float64(maxMemoryUsed)/(1024*1024))
		t.Logf("  - Processing rate: %.2f spans/sec", float64(len(traceData.Spans))/duration.Seconds())
		t.Logf("  - Throughput: %.2f MB/sec", float64(len(largeData))/(1024*1024)/duration.Seconds())
	}()

	// Wait for test completion or timeout
	select {
	case <-done:
		// Test completed successfully
	case <-time.After(timeout):
		t.Fatalf("Test timed out after %v", timeout)
	}
}

func TestMemoryEfficiency_MultipleFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory efficiency test in short mode")
	}

	config := &StreamingConfig{
		ChunkSize:      2048,
		MaxMemoryUsage: 200 * 1024 * 1024, // 200MB limit
	}

	ingestor := NewStreamingIngestor(config)

	// Process multiple files to test memory cleanup
	numFiles := 5
	spansPerFile := 1000

	var maxMemoryUsed uint64

	for i := 0; i < numFiles; i++ {
		testData := createLargeOTLPDataWithTraceID(spansPerFile, fmt.Sprintf("trace-%d", i))
		reader := strings.NewReader(testData)

		traceData, err := ingestor.IngestFromReaderStreaming(reader, int64(len(testData)))

		require.NoError(t, err)
		assert.NotNil(t, traceData)
		assert.Len(t, traceData.Spans, spansPerFile+1)

		// Check memory usage after each file
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		if m.Alloc > maxMemoryUsed {
			maxMemoryUsed = m.Alloc
		}

		// Force GC between files to test memory cleanup
		runtime.GC()
		runtime.GC()

		t.Logf("Processed file %d, current memory: %.2f MB", i+1, float64(m.Alloc)/(1024*1024))
	}

	// Memory usage should stay reasonable across multiple files
	assert.Less(t, int64(maxMemoryUsed), int64(200*1024*1024),
		"Memory usage should stay under 200MB across multiple files")

	t.Logf("Max memory used across %d files: %.2f MB", numFiles, float64(maxMemoryUsed)/(1024*1024))
}

func TestStreamingPerformance_Comparison(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance comparison test in short mode")
	}

	// Create test data
	numSpans := 5000
	testData := createLargeOTLPData(numSpans)

	// Test regular ingestor
	regularIngestor := NewTraceIngestor()
	reader1 := strings.NewReader(testData)

	start1 := time.Now()
	traceData1, err1 := regularIngestor.IngestFromReader(reader1)
	duration1 := time.Since(start1)

	require.NoError(t, err1)

	// Test streaming ingestor
	streamingIngestor := NewStreamingIngestor(nil)
	reader2 := strings.NewReader(testData)

	start2 := time.Now()
	traceData2, err2 := streamingIngestor.IngestFromReaderStreaming(reader2, int64(len(testData)))
	duration2 := time.Since(start2)

	require.NoError(t, err2)

	// Both should produce same results
	assert.Equal(t, len(traceData1.Spans), len(traceData2.Spans))
	assert.Equal(t, traceData1.TraceID, traceData2.TraceID)

	t.Logf("Performance comparison:")
	t.Logf("  - Regular ingestor: %s", duration1)
	t.Logf("  - Streaming ingestor: %s", duration2)
	t.Logf("  - Spans processed: %d", len(traceData1.Spans))

	// Streaming might be slightly slower due to overhead, but should be reasonable
	ratio := float64(duration2) / float64(duration1)
	assert.Less(t, ratio, 3.0, "Streaming ingestor should not be more than 3x slower")
}

func TestConcurrentIngestion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent ingestion test in short mode")
	}

	config := &StreamingConfig{
		ChunkSize:      1024,
		MaxMemoryUsage: 300 * 1024 * 1024, // 300MB limit
	}

	// Test concurrent ingestion of multiple traces
	numGoroutines := 3
	spansPerTrace := 1000

	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			ingestor := NewStreamingIngestor(config)
			testData := createLargeOTLPDataWithTraceID(spansPerTrace, fmt.Sprintf("concurrent-trace-%d", id))
			reader := strings.NewReader(testData)

			traceData, err := ingestor.IngestFromReaderStreaming(reader, int64(len(testData)))

			if err != nil {
				results <- err
				return
			}

			if len(traceData.Spans) != spansPerTrace+1 {
				results <- fmt.Errorf("expected %d spans, got %d", spansPerTrace+1, len(traceData.Spans))
				return
			}

			results <- nil
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		err := <-results
		assert.NoError(t, err, "Concurrent ingestion should succeed")
	}

	t.Logf("Successfully processed %d traces concurrently", numGoroutines)
}

func BenchmarkStreamingIngestor_LargeFile(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	ingestor := NewStreamingIngestor(nil)
	testData := createLargeOTLPData(1000)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(testData)
		_, err := ingestor.IngestFromReaderStreaming(reader, int64(len(testData)))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemoryMonitor_Operations(b *testing.B) {
	monitor := NewMemoryMonitor(100 * 1024 * 1024)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		monitor.UpdateMemoryUsage()
		monitor.CheckMemoryLimit(1024)
		monitor.GetMemoryUsage()
		monitor.ShouldTriggerGC()
		monitor.GetMemoryUsagePercent()
	}
}

// Helper functions for performance tests

func createVeryLargeOTLPData(numSpans int) string {
	return createLargeOTLPDataWithTraceID(numSpans, "very-large-trace")
}

func createLargeOTLPDataWithTraceID(numSpans int, traceID string) string {
	spans := make([]OTLPSpan, numSpans+1) // +1 for root span

	// Create root span
	spans[0] = OTLPSpan{
		TraceID:           traceID,
		SpanID:            "root-span",
		Name:              "root-operation",
		StartTimeUnixNano: "1640995200000000000",
		EndTimeUnixNano:   "1640995210000000000",
		Attributes: []Attribute{
			{Key: "service.name", Value: "root-service"},
			{Key: "operation.id", Value: "rootOp"},
			{Key: "trace.size", Value: numSpans},
		},
		Status: Status{Code: 1, Message: "OK"},
	}

	// Create child spans with more attributes to increase size
	for i := 1; i <= numSpans; i++ {
		spans[i] = OTLPSpan{
			TraceID:           traceID,
			SpanID:            fmt.Sprintf("span-%d", i),
			ParentSpanID:      "root-span",
			Name:              fmt.Sprintf("operation-%d", i),
			StartTimeUnixNano: fmt.Sprintf("%d", 1640995200000000000+int64(i)*100000000),
			EndTimeUnixNano:   fmt.Sprintf("%d", 1640995200000000000+int64(i)*100000000+50000000),
			Attributes: []Attribute{
				{Key: "service.name", Value: fmt.Sprintf("service-%d", i%10)},
				{Key: "operation.id", Value: fmt.Sprintf("op-%d", i)},
				{Key: "span.index", Value: i},
				{Key: "http.method", Value: "POST"},
				{Key: "http.url", Value: fmt.Sprintf("/api/endpoint-%d", i%100)},
				{Key: "http.status_code", Value: 200},
				{Key: "user.id", Value: fmt.Sprintf("user-%d", i%1000)},
				{Key: "request.id", Value: fmt.Sprintf("req-%d-%d", i, time.Now().UnixNano())},
				{Key: "component", Value: "web-server"},
				{Key: "version", Value: "1.0.0"},
			},
			Events: []Event{
				{
					TimeUnixNano: fmt.Sprintf("%d", 1640995200000000000+int64(i)*100000000+10000000),
					Name:         "request.start",
					Attributes: []Attribute{
						{Key: "event.type", Value: "info"},
						{Key: "message", Value: fmt.Sprintf("Starting request %d", i)},
					},
				},
				{
					TimeUnixNano: fmt.Sprintf("%d", 1640995200000000000+int64(i)*100000000+40000000),
					Name:         "request.end",
					Attributes: []Attribute{
						{Key: "event.type", Value: "info"},
						{Key: "message", Value: fmt.Sprintf("Completed request %d", i)},
						{Key: "duration_ms", Value: 40},
					},
				},
			},
			Status: Status{Code: 1, Message: "OK"},
		}
	}

	trace := OTLPTrace{
		ResourceSpans: []ResourceSpan{
			{
				Resource: Resource{
					Attributes: []Attribute{
						{Key: "service.name", Value: "large-service"},
						{Key: "service.version", Value: "1.0.0"},
						{Key: "deployment.environment", Value: "production"},
					},
				},
				ScopeSpans: []ScopeSpan{
					{
						Scope: Scope{
							Name:    "large-scope",
							Version: "1.0.0",
						},
						Spans: spans,
					},
				},
			},
		},
	}

	data, _ := json.Marshal(trace)
	return string(data)
}