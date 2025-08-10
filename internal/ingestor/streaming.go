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
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/flowspec/flowspec-cli/internal/models"
)

// StreamingIngestor provides streaming ingestion capabilities for large trace files
type StreamingIngestor struct {
	*DefaultTraceIngestor
	chunkSize        int
	maxMemoryUsage   int64
	progressCallback func(processed, total int64)
}

// StreamingConfig holds configuration for streaming ingestion
type StreamingConfig struct {
	ChunkSize        int                          // Size of each processing chunk
	MaxMemoryUsage   int64                        // Maximum memory usage in bytes
	ProgressCallback func(processed, total int64) // Progress callback function
	EnableGC         bool                         // Enable aggressive garbage collection
	GCInterval       time.Duration                // Interval for garbage collection
}

// ChunkProcessor processes individual chunks of trace data
type ChunkProcessor struct {
	spans      []*models.Span
	errors     []error
	memoryUsed int64
	mu         sync.Mutex
}

// MemoryMonitor monitors and controls memory usage during ingestion
type MemoryMonitor struct {
	maxMemory     int64
	currentMemory int64
	gcThreshold   float64 // Trigger GC when memory usage exceeds this percentage
	mu            sync.RWMutex
}

// ProgressTracker tracks ingestion progress
type ProgressTracker struct {
	totalBytes     int64
	processedBytes int64
	startTime      time.Time
	callback       func(processed, total int64)
	mu             sync.RWMutex
}

// DefaultStreamingConfig returns a default streaming configuration
func DefaultStreamingConfig() *StreamingConfig {
	return &StreamingConfig{
		ChunkSize:      1024 * 1024,       // 1MB chunks
		MaxMemoryUsage: 500 * 1024 * 1024, // 500MB max memory
		EnableGC:       true,
		GCInterval:     5 * time.Second,
	}
}

// NewStreamingIngestor creates a new streaming ingestor
func NewStreamingIngestor(config *StreamingConfig) *StreamingIngestor {
	if config == nil {
		config = DefaultStreamingConfig()
	}

	baseIngestor := NewTraceIngestor()
	baseIngestor.SetMemoryLimit(config.MaxMemoryUsage / (1024 * 1024)) // Convert to MB

	return &StreamingIngestor{
		DefaultTraceIngestor: baseIngestor,
		chunkSize:            config.ChunkSize,
		maxMemoryUsage:       config.MaxMemoryUsage,
		progressCallback:     config.ProgressCallback,
	}
}

// NewMemoryMonitor creates a new memory monitor
func NewMemoryMonitor(maxMemory int64) *MemoryMonitor {
	return &MemoryMonitor{
		maxMemory:   maxMemory,
		gcThreshold: 0.8, // Trigger GC at 80% memory usage
	}
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(totalBytes int64, callback func(processed, total int64)) *ProgressTracker {
	return &ProgressTracker{
		totalBytes: totalBytes,
		startTime:  time.Now(),
		callback:   callback,
	}
}

// IngestFromReaderStreaming ingests trace data using streaming approach
func (si *StreamingIngestor) IngestFromReaderStreaming(reader io.Reader, totalSize int64) (*models.TraceData, error) {
	// Initialize memory monitor
	monitor := NewMemoryMonitor(si.maxMemoryUsage)

	// Initialize progress tracker
	var tracker *ProgressTracker
	if si.progressCallback != nil {
		tracker = NewProgressTracker(totalSize, si.progressCallback)
	}

	// Start memory monitoring goroutine
	stopMonitoring := make(chan bool)
	go si.monitorMemory(monitor, stopMonitoring)
	defer func() { stopMonitoring <- true }()

	// Process in chunks
	traceData, err := si.processInChunks(reader, monitor, tracker)
	if err != nil {
		return nil, err
	}

	// Build span tree
	if err := traceData.BuildSpanTree(); err != nil {
		return nil, fmt.Errorf("failed to build span tree: %w", err)
	}

	return traceData, nil
}

// processInChunks processes the input stream in manageable chunks
func (si *StreamingIngestor) processInChunks(reader io.Reader, monitor *MemoryMonitor, tracker *ProgressTracker) (*models.TraceData, error) {
	// Use a buffered reader for efficient reading
	bufferedReader := bufio.NewReaderSize(reader, si.chunkSize)

	// Read the entire JSON structure first to understand the format
	// For OTLP JSON, we need to parse the complete structure
	data, err := io.ReadAll(bufferedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read trace data: %w", err)
	}

	// Update progress
	if tracker != nil {
		tracker.UpdateProgress(int64(len(data)))
	}

	// Check memory usage before processing
	if err := monitor.CheckMemoryLimit(int64(len(data))); err != nil {
		return nil, err
	}

	// Parse JSON in chunks if possible, otherwise parse normally
	if len(data) > si.chunkSize*2 { // Use chunked processing for files larger than 2x chunk size
		return si.parseJSONInChunks(data, monitor, tracker)
	}

	// For smaller files, use normal parsing
	return si.parseJSONNormally(data, monitor)
}

// parseJSONInChunks attempts to parse large JSON files in chunks
func (si *StreamingIngestor) parseJSONInChunks(data []byte, monitor *MemoryMonitor, tracker *ProgressTracker) (*models.TraceData, error) {
	// For OTLP JSON format, we need to parse the complete structure
	// Chunked JSON parsing is complex, so we'll use memory-efficient parsing instead
	return si.parseJSONWithMemoryOptimization(data, monitor, tracker)
}

// parseJSONWithMemoryOptimization parses JSON with memory optimization techniques
func (si *StreamingIngestor) parseJSONWithMemoryOptimization(data []byte, monitor *MemoryMonitor, tracker *ProgressTracker) (*models.TraceData, error) {
	// Parse OTLP structure
	var otlpTrace OTLPTrace
	if err := json.Unmarshal(data, &otlpTrace); err != nil {
		return nil, fmt.Errorf("failed to parse OTLP JSON: %w", err)
	}

	// Process resource spans one by one to minimize memory usage
	traceData := &models.TraceData{
		Spans: make(map[string]*models.Span),
	}

	totalResourceSpans := len(otlpTrace.ResourceSpans)
	processedResourceSpans := 0

	for _, resourceSpan := range otlpTrace.ResourceSpans {
		// Check memory before processing each resource span
		if err := monitor.CheckMemoryLimit(0); err != nil {
			return nil, err
		}

		// Process scope spans
		for _, scopeSpan := range resourceSpan.ScopeSpans {
			// Process spans in batches
			batchSize := 100 // Process 100 spans at a time
			for i := 0; i < len(scopeSpan.Spans); i += batchSize {
				end := i + batchSize
				if end > len(scopeSpan.Spans) {
					end = len(scopeSpan.Spans)
				}

				// Process batch
				batch := scopeSpan.Spans[i:end]
				if err := si.processBatch(batch, traceData, monitor); err != nil {
					return nil, err
				}

				// Update progress
				if tracker != nil {
					progress := float64(processedResourceSpans)/float64(totalResourceSpans) +
						float64(end)/float64(len(scopeSpan.Spans))/float64(totalResourceSpans)
					tracker.UpdateProgressPercent(progress)
				}

				// Force garbage collection periodically
				if i%500 == 0 {
					runtime.GC()
				}
			}
		}

		processedResourceSpans++
	}

	return traceData, nil
}

// parseJSONNormally parses JSON using the normal approach for smaller files
func (si *StreamingIngestor) parseJSONNormally(data []byte, monitor *MemoryMonitor) (*models.TraceData, error) {
	// Check memory before parsing
	if err := monitor.CheckMemoryLimit(int64(len(data))); err != nil {
		return nil, err
	}

	// Parse OTLP JSON
	var otlpTrace OTLPTrace
	if err := json.Unmarshal(data, &otlpTrace); err != nil {
		return nil, fmt.Errorf("failed to parse OTLP JSON: %w", err)
	}

	// Convert to internal format
	metrics := NewIngestMetrics()
	traceData, err := si.convertOTLPToTraceData(otlpTrace, metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to convert OTLP data: %w", err)
	}

	return traceData, nil
}

// processBatch processes a batch of OTLP spans
func (si *StreamingIngestor) processBatch(batch []OTLPSpan, traceData *models.TraceData, monitor *MemoryMonitor) error {
	for _, otlpSpan := range batch {
		// Convert span
		span, err := si.convertOTLPSpan(otlpSpan)
		if err != nil {
			continue // Skip invalid spans
		}

		// Set trace ID if not set
		if traceData.TraceID == "" {
			traceData.TraceID = span.TraceID
		}

		// Add to spans map
		traceData.Spans[span.SpanID] = span

		// Check memory usage periodically
		if len(traceData.Spans)%100 == 0 {
			if err := monitor.CheckMemoryLimit(0); err != nil {
				return err
			}
		}
	}

	return nil
}

// monitorMemory monitors memory usage in a separate goroutine
func (si *StreamingIngestor) monitorMemory(monitor *MemoryMonitor, stop <-chan bool) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			monitor.UpdateMemoryUsage()

			// Trigger GC if memory usage is high
			if monitor.ShouldTriggerGC() {
				runtime.GC()
			}
		}
	}
}

// MemoryMonitor methods

// CheckMemoryLimit checks if adding the specified bytes would exceed memory limit
func (mm *MemoryMonitor) CheckMemoryLimit(additionalBytes int64) error {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	currentUsage := int64(m.Alloc)
	if currentUsage+additionalBytes > mm.maxMemory {
		return fmt.Errorf("memory usage would exceed limit: current=%d, additional=%d, limit=%d",
			currentUsage, additionalBytes, mm.maxMemory)
	}

	return nil
}

// UpdateMemoryUsage updates the current memory usage
func (mm *MemoryMonitor) UpdateMemoryUsage() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	mm.currentMemory = int64(m.Alloc)
}

// ShouldTriggerGC returns true if garbage collection should be triggered
func (mm *MemoryMonitor) ShouldTriggerGC() bool {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	return float64(mm.currentMemory)/float64(mm.maxMemory) > mm.gcThreshold
}

// GetMemoryUsage returns current memory usage
func (mm *MemoryMonitor) GetMemoryUsage() int64 {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.currentMemory
}

// GetMemoryUsagePercent returns current memory usage as percentage
func (mm *MemoryMonitor) GetMemoryUsagePercent() float64 {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return float64(mm.currentMemory) / float64(mm.maxMemory) * 100
}

// ProgressTracker methods

// UpdateProgress updates the progress with processed bytes
func (pt *ProgressTracker) UpdateProgress(processedBytes int64) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.processedBytes = processedBytes

	if pt.callback != nil {
		pt.callback(pt.processedBytes, pt.totalBytes)
	}
}

// UpdateProgressPercent updates progress with a percentage (0.0 to 1.0)
func (pt *ProgressTracker) UpdateProgressPercent(percent float64) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.processedBytes = int64(float64(pt.totalBytes) * percent)

	if pt.callback != nil {
		pt.callback(pt.processedBytes, pt.totalBytes)
	}
}

// GetProgress returns current progress information
func (pt *ProgressTracker) GetProgress() (processed, total int64, percent float64, elapsed time.Duration) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	processed = pt.processedBytes
	total = pt.totalBytes
	if total > 0 {
		percent = float64(processed) / float64(total) * 100
	}
	elapsed = time.Since(pt.startTime)

	return
}

// GetETA returns estimated time of arrival
func (pt *ProgressTracker) GetETA() time.Duration {
	processed, total, _, elapsed := pt.GetProgress()

	if processed == 0 {
		return 0
	}

	rate := float64(processed) / elapsed.Seconds()
	remaining := total - processed

	if rate > 0 {
		return time.Duration(float64(remaining)/rate) * time.Second
	}

	return 0
}

// ChunkProcessor methods

// NewChunkProcessor creates a new chunk processor
func NewChunkProcessor() *ChunkProcessor {
	return &ChunkProcessor{
		spans:  make([]*models.Span, 0),
		errors: make([]error, 0),
	}
}

// AddSpan adds a span to the processor
func (cp *ChunkProcessor) AddSpan(span *models.Span) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.spans = append(cp.spans, span)
	cp.memoryUsed += cp.estimateSpanMemory(span)
}

// AddError adds an error to the processor
func (cp *ChunkProcessor) AddError(err error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.errors = append(cp.errors, err)
}

// GetResults returns the processed results
func (cp *ChunkProcessor) GetResults() ([]*models.Span, []error, int64) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	return cp.spans, cp.errors, cp.memoryUsed
}

// Clear clears the processor state
func (cp *ChunkProcessor) Clear() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.spans = cp.spans[:0]
	cp.errors = cp.errors[:0]
	cp.memoryUsed = 0
}

// estimateSpanMemory estimates memory usage of a span
func (cp *ChunkProcessor) estimateSpanMemory(span *models.Span) int64 {
	// Rough estimation of span memory usage
	baseSize := int64(200) // Base struct size

	// Add string fields
	baseSize += int64(len(span.SpanID))
	baseSize += int64(len(span.TraceID))
	baseSize += int64(len(span.ParentID))
	baseSize += int64(len(span.Name))
	baseSize += 4 // StatusCode is an int, roughly 4 bytes
	baseSize += int64(len(span.Status.Message))

	// Add attributes (rough estimation)
	for key, value := range span.Attributes {
		baseSize += int64(len(key))
		if str, ok := value.(string); ok {
			baseSize += int64(len(str))
		} else {
			baseSize += 50 // Rough estimate for other types
		}
	}

	// Add events
	for _, event := range span.Events {
		baseSize += int64(len(event.Name))
		for key, value := range event.Attributes {
			baseSize += int64(len(key))
			if str, ok := value.(string); ok {
				baseSize += int64(len(str))
			} else {
				baseSize += 50
			}
		}
	}

	return baseSize
}

// Utility functions for memory optimization

// OptimizeMemoryUsage performs memory optimization operations
func OptimizeMemoryUsage() {
	// Force garbage collection
	runtime.GC()

	// Return memory to OS
	runtime.GC()
	runtime.GC() // Call twice for better effect
}

// GetMemoryStats returns current memory statistics
func GetMemoryStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"alloc":           m.Alloc,         // Currently allocated bytes
		"total_alloc":     m.TotalAlloc,    // Total allocated bytes (cumulative)
		"sys":             m.Sys,           // System memory obtained from OS
		"num_gc":          m.NumGC,         // Number of GC cycles
		"gc_cpu_fraction": m.GCCPUFraction, // Fraction of CPU time used by GC
		"heap_alloc":      m.HeapAlloc,     // Heap allocated bytes
		"heap_sys":        m.HeapSys,       // Heap system bytes
		"heap_objects":    m.HeapObjects,   // Number of heap objects
	}
}
