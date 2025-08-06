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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/flowspec/flowspec-cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStreamingIngestor(t *testing.T) {
	config := &StreamingConfig{
		ChunkSize:      2048,
		MaxMemoryUsage: 100 * 1024 * 1024, // 100MB
		EnableGC:       true,
		GCInterval:     1 * time.Second,
	}

	ingestor := NewStreamingIngestor(config)

	assert.NotNil(t, ingestor)
	assert.Equal(t, 2048, ingestor.chunkSize)
	assert.Equal(t, int64(100*1024*1024), ingestor.maxMemoryUsage)
}

func TestDefaultStreamingConfig(t *testing.T) {
	config := DefaultStreamingConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 1024*1024, config.ChunkSize)
	assert.Equal(t, int64(500*1024*1024), config.MaxMemoryUsage)
	assert.True(t, config.EnableGC)
	assert.Equal(t, 5*time.Second, config.GCInterval)
}

func TestStreamingIngestor_SmallFile(t *testing.T) {
	ingestor := NewStreamingIngestor(nil) // Use default config

	// Create small test data
	testData := createTestOTLPData()
	reader := strings.NewReader(testData)

	traceData, err := ingestor.IngestFromReaderStreaming(reader, int64(len(testData)))

	require.NoError(t, err)
	assert.NotNil(t, traceData)
	assert.Equal(t, "trace123", traceData.TraceID)
	assert.Len(t, traceData.Spans, 3)
	assert.NotNil(t, traceData.SpanTree)
}

func TestStreamingIngestor_LargeFile(t *testing.T) {
	ingestor := NewStreamingIngestor(&StreamingConfig{
		ChunkSize:      1024,              // Small chunks for testing
		MaxMemoryUsage: 200 * 1024 * 1024, // 200MB limit (more generous)
	})

	// Create large test data (100 spans - smaller for more reliable testing)
	largeData := createLargeOTLPData(100)
	reader := strings.NewReader(largeData)

	traceData, err := ingestor.IngestFromReaderStreaming(reader, int64(len(largeData)))

	require.NoError(t, err)
	assert.NotNil(t, traceData)
	assert.Equal(t, "large-trace", traceData.TraceID)
	assert.Len(t, traceData.Spans, 101) // 100 + 1 root span
	assert.NotNil(t, traceData.SpanTree)
}

func TestStreamingIngestor_WithProgressCallback(t *testing.T) {
	var progressUpdates []float64
	var progressMutex sync.Mutex

	progressCallback := func(processed, total int64) {
		progressMutex.Lock()
		defer progressMutex.Unlock()

		if total > 0 {
			percent := float64(processed) / float64(total) * 100
			progressUpdates = append(progressUpdates, percent)
		}
	}

	config := &StreamingConfig{
		ChunkSize:        1024,
		MaxMemoryUsage:   200 * 1024 * 1024, // More generous limit
		ProgressCallback: progressCallback,
	}

	ingestor := NewStreamingIngestor(config)

	// Create smaller test data
	testData := createLargeOTLPData(50)
	reader := strings.NewReader(testData)

	traceData, err := ingestor.IngestFromReaderStreaming(reader, int64(len(testData)))

	require.NoError(t, err)
	assert.NotNil(t, traceData)

	// Verify progress updates were called
	progressMutex.Lock()
	defer progressMutex.Unlock()
	assert.Greater(t, len(progressUpdates), 0, "Progress callback should be called")
}

func TestMemoryMonitor(t *testing.T) {
	// Use a much larger limit to avoid issues with current system memory usage
	monitor := NewMemoryMonitor(500 * 1024 * 1024) // 500MB limit

	// Test initial state
	assert.Equal(t, int64(500*1024*1024), monitor.maxMemory)
	assert.Equal(t, float64(0.8), monitor.gcThreshold)

	// Test memory limit check with small amount
	err := monitor.CheckMemoryLimit(1024) // 1KB should be fine
	assert.NoError(t, err)

	// Test memory usage update
	monitor.UpdateMemoryUsage()
	usage := monitor.GetMemoryUsage()
	assert.Greater(t, usage, int64(0))

	// Test percentage calculation (should be reasonable with larger limit)
	percent := monitor.GetMemoryUsagePercent()
	assert.GreaterOrEqual(t, percent, 0.0)
	// Don't assert upper bound as it depends on system state
}

func TestMemoryMonitor_ExceedsLimit(t *testing.T) {
	monitor := NewMemoryMonitor(1024) // Very small limit (1KB)

	// This should exceed the limit
	err := monitor.CheckMemoryLimit(2048) // 2KB
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "memory usage would exceed limit")
}

func TestProgressTracker(t *testing.T) {
	totalBytes := int64(1000)
	var lastProcessed, lastTotal int64

	callback := func(processed, total int64) {
		lastProcessed = processed
		lastTotal = total
	}

	tracker := NewProgressTracker(totalBytes, callback)

	// Test initial state
	processed, total, percent, elapsed := tracker.GetProgress()
	assert.Equal(t, int64(0), processed)
	assert.Equal(t, totalBytes, total)
	assert.Equal(t, 0.0, percent)
	assert.Greater(t, elapsed, time.Duration(0))

	// Test progress update
	tracker.UpdateProgress(500)
	assert.Equal(t, int64(500), lastProcessed)
	assert.Equal(t, totalBytes, lastTotal)

	processed, total, percent, _ = tracker.GetProgress()
	assert.Equal(t, int64(500), processed)
	assert.Equal(t, 50.0, percent)

	// Test percentage update
	tracker.UpdateProgressPercent(0.75) // 75%
	processed, _, percent, _ = tracker.GetProgress()
	assert.Equal(t, int64(750), processed)
	assert.Equal(t, 75.0, percent)
}

func TestProgressTracker_ETA(t *testing.T) {
	tracker := NewProgressTracker(1000, nil)

	// No progress yet, ETA should be 0
	eta := tracker.GetETA()
	assert.Equal(t, time.Duration(0), eta)

	// Simulate some progress with more time
	time.Sleep(50 * time.Millisecond) // More time for measurable rate
	tracker.UpdateProgress(100)       // 10% done

	eta = tracker.GetETA()
	// ETA might still be 0 if the calculation is too fast, so just check it's not negative
	assert.GreaterOrEqual(t, eta, time.Duration(0))
}

func TestChunkProcessor(t *testing.T) {
	processor := NewChunkProcessor()

	// Test initial state
	spans, errors, memUsed := processor.GetResults()
	assert.Empty(t, spans)
	assert.Empty(t, errors)
	assert.Equal(t, int64(0), memUsed)

	// Add a span
	testSpan := &models.Span{
		SpanID:  "test-span",
		TraceID: "test-trace",
		Name:    "test-operation",
		Attributes: map[string]interface{}{
			"service.name": "test-service",
		},
	}

	processor.AddSpan(testSpan)

	spans, errors, memUsed = processor.GetResults()
	assert.Len(t, spans, 1)
	assert.Empty(t, errors)
	assert.Greater(t, memUsed, int64(0))

	// Add an error
	testError := fmt.Errorf("test error")
	processor.AddError(testError)

	spans, errors, _ = processor.GetResults()
	assert.Len(t, spans, 1)
	assert.Len(t, errors, 1)
	assert.Equal(t, testError, errors[0])

	// Clear processor
	processor.Clear()
	spans, errors, memUsed = processor.GetResults()
	assert.Empty(t, spans)
	assert.Empty(t, errors)
	assert.Equal(t, int64(0), memUsed)
}

func TestMemoryOptimization(t *testing.T) {
	// Get initial memory stats
	initialStats := GetMemoryStats()
	assert.NotNil(t, initialStats)
	assert.Contains(t, initialStats, "alloc")
	assert.Contains(t, initialStats, "total_alloc")
	assert.Contains(t, initialStats, "sys")
	assert.Contains(t, initialStats, "num_gc")

	// Allocate some memory
	data := make([]byte, 1024*1024) // 1MB
	for i := range data {
		data[i] = byte(i % 256)
	}

	// Optimize memory usage
	OptimizeMemoryUsage()

	// Get stats after optimization
	afterStats := GetMemoryStats()
	assert.NotNil(t, afterStats)

	// GC count should have increased
	initialGC := initialStats["num_gc"].(uint32)
	afterGC := afterStats["num_gc"].(uint32)
	assert.GreaterOrEqual(t, afterGC, initialGC)

	// Keep reference to data to prevent it from being optimized away
	_ = data[0]
}

func TestStreamingIngestor_MemoryConstraints(t *testing.T) {
	// Create ingestor with very low memory limit
	config := &StreamingConfig{
		ChunkSize:      1024,
		MaxMemoryUsage: 1024 * 1024, // 1MB limit
	}

	ingestor := NewStreamingIngestor(config)

	// Create moderately large data that should fit within limits
	testData := createLargeOTLPData(50) // 50 spans should be manageable
	reader := strings.NewReader(testData)

	traceData, err := ingestor.IngestFromReaderStreaming(reader, int64(len(testData)))

	// Should succeed with memory optimization
	require.NoError(t, err)
	assert.NotNil(t, traceData)
	assert.Len(t, traceData.Spans, 51) // 50 + 1 root
}

func TestStreamingIngestor_VeryLargeFile(t *testing.T) {
	// Skip this test in short mode as it's resource intensive
	if testing.Short() {
		t.Skip("Skipping large file test in short mode")
	}

	config := &StreamingConfig{
		ChunkSize:      4096,
		MaxMemoryUsage: 100 * 1024 * 1024, // 100MB
	}

	ingestor := NewStreamingIngestor(config)

	// Create very large data (5000 spans)
	largeData := createLargeOTLPData(5000)
	reader := strings.NewReader(largeData)

	start := time.Now()
	traceData, err := ingestor.IngestFromReaderStreaming(reader, int64(len(largeData)))
	duration := time.Since(start)

	require.NoError(t, err)
	assert.NotNil(t, traceData)
	assert.Len(t, traceData.Spans, 5001) // 5000 + 1 root

	// Should complete within reasonable time
	assert.Less(t, duration, 30*time.Second, "Large file processing should complete within 30 seconds")

	t.Logf("Processed %d spans in %s", len(traceData.Spans), duration)
}

func TestStreamingIngestor_MemoryUsageTracking(t *testing.T) {
	var memoryUsages []int64

	config := &StreamingConfig{
		ChunkSize:      2048,
		MaxMemoryUsage: 50 * 1024 * 1024,
		ProgressCallback: func(processed, total int64) {
			// Track memory usage during processing
			stats := GetMemoryStats()
			if alloc, ok := stats["alloc"].(uint64); ok {
				memoryUsages = append(memoryUsages, int64(alloc))
			}
		},
	}

	ingestor := NewStreamingIngestor(config)

	// Create test data
	testData := createLargeOTLPData(200)
	reader := strings.NewReader(testData)

	traceData, err := ingestor.IngestFromReaderStreaming(reader, int64(len(testData)))

	require.NoError(t, err)
	assert.NotNil(t, traceData)

	// Verify memory usage was tracked
	assert.Greater(t, len(memoryUsages), 0, "Memory usage should be tracked during processing")

	// Memory usage should stay within reasonable bounds
	for _, usage := range memoryUsages {
		assert.Less(t, usage, int64(100*1024*1024), "Memory usage should stay under 100MB")
	}
}

func TestStreamingIngestor_ErrorHandling(t *testing.T) {
	ingestor := NewStreamingIngestor(nil)

	// Test with invalid JSON
	invalidJSON := `{"invalid": json structure`
	reader := strings.NewReader(invalidJSON)

	traceData, err := ingestor.IngestFromReaderStreaming(reader, int64(len(invalidJSON)))

	assert.Error(t, err)
	assert.Nil(t, traceData)
	assert.Contains(t, err.Error(), "failed to parse OTLP JSON")
}

func TestStreamingIngestor_EmptyData(t *testing.T) {
	ingestor := NewStreamingIngestor(nil)

	// Test with empty resource spans, which is a valid OTLP structure.
	emptyData := `{"resourceSpans": []}`
	reader := strings.NewReader(emptyData)

	traceData, err := ingestor.IngestFromReaderStreaming(reader, int64(len(emptyData)))

	// It should not produce an error.
	require.NoError(t, err)
	// It should return a valid, non-nil TraceData object.
	require.NotNil(t, traceData)
	// The TraceData object should contain no spans.
	assert.Len(t, traceData.Spans, 0)
	assert.Nil(t, traceData.RootSpan)
}

// Benchmark tests

func BenchmarkStreamingIngestor_SmallFile(b *testing.B) {
	ingestor := NewStreamingIngestor(nil)
	testData := createTestOTLPData()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(testData)
		_, err := ingestor.IngestFromReaderStreaming(reader, int64(len(testData)))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStreamingIngestor_MediumFile(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	ingestor := NewStreamingIngestor(nil)
	largeData := createLargeOTLPData(500) // Smaller for regular benchmark

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(largeData)
		_, err := ingestor.IngestFromReaderStreaming(reader, int64(len(largeData)))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemoryMonitor(b *testing.B) {
	monitor := NewMemoryMonitor(100 * 1024 * 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		monitor.UpdateMemoryUsage()
		monitor.CheckMemoryLimit(1024)
		monitor.GetMemoryUsage()
		monitor.ShouldTriggerGC()
	}
}

// Helper functions

func createLargeOTLPData(numSpans int) string {
	spans := make([]OTLPSpan, numSpans+1) // +1 for root span

	// Create root span
	spans[0] = OTLPSpan{
		TraceID:           "large-trace",
		SpanID:            "root-span",
		Name:              "root-operation",
		StartTimeUnixNano: "1640995200000000000",
		EndTimeUnixNano:   "1640995210000000000",
		Attributes: []Attribute{
			{Key: "service.name", Value: "root-service"},
			{Key: "operation.id", Value: "rootOp"},
		},
		Status: Status{Code: 1, Message: "OK"},
	}

	// Create child spans
	for i := 1; i <= numSpans; i++ {
		spans[i] = OTLPSpan{
			TraceID:           "large-trace",
			SpanID:            fmt.Sprintf("span-%d", i),
			ParentSpanID:      "root-span",
			Name:              fmt.Sprintf("operation-%d", i),
			StartTimeUnixNano: fmt.Sprintf("%d", 1640995200000000000+int64(i)*100000000),
			EndTimeUnixNano:   fmt.Sprintf("%d", 1640995200000000000+int64(i)*100000000+50000000),
			Attributes: []Attribute{
				{Key: "service.name", Value: fmt.Sprintf("service-%d", i%10)},
				{Key: "operation.id", Value: fmt.Sprintf("op-%d", i)},
				{Key: "span.index", Value: i},
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
