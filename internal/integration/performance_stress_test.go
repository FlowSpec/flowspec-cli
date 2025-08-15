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

package integration

import (
	"compress/gzip"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/flowspec/flowspec-cli/internal/engine"
	"github.com/flowspec/flowspec-cli/internal/ingestor/traffic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLargeFileProcessing tests processing of large log files (≥5GB simulation)
func TestLargeFileProcessing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file test in short mode")
	}
	
	tmpDir := t.TempDir()
	
	// Create a large log file (simulate 5GB by creating multiple smaller files)
	// We'll create 50MB files to simulate large file processing without actually using 5GB
	largeLogFile := filepath.Join(tmpDir, "large_access.log")
	
	t.Logf("Creating large log file simulation...")
	startTime := time.Now()
	
	// Create a 50MB log file with realistic data patterns
	createLargeLogFile(t, largeLogFile, 50*1024*1024) // 50MB
	
	creationTime := time.Since(startTime)
	t.Logf("Large log file created in %v", creationTime)
	
	// Test memory usage during processing
	var memBefore, memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)
	
	// Process the large file
	processingStart := time.Now()
	
	ingestor := traffic.NewNginxAccessIngestor()
	options := traffic.DefaultIngestOptions()
	
	iterator, err := ingestor.Ingest([]string{largeLogFile}, options)
	require.NoError(t, err)
	
	// Count records and measure processing speed
	recordCount := 0
	for iterator.Next() {
		recordCount++
		// Process every 10000th record to avoid excessive memory usage in test
		if recordCount%10000 == 0 {
			t.Logf("Processed %d records...", recordCount)
		}
	}
	require.NoError(t, iterator.Err())
	
	processingTime := time.Since(processingStart)
	
	runtime.GC()
	runtime.ReadMemStats(&memAfter)
	
	// Calculate performance metrics
	processingRate := float64(recordCount) / processingTime.Seconds()
	memoryUsedMB := float64(memAfter.Alloc-memBefore.Alloc) / (1024 * 1024)
	
	t.Logf("Large file processing results:")
	t.Logf("  - Records processed: %d", recordCount)
	t.Logf("  - Processing time: %v", processingTime)
	t.Logf("  - Processing rate: %.2f records/sec", processingRate)
	t.Logf("  - Memory used: %.2f MB", memoryUsedMB)
	
	// Performance assertions
	assert.Greater(t, recordCount, 100000, "Should process significant number of records")
	assert.Greater(t, processingRate, 1000.0, "Should process at least 1000 records/sec")
	assert.Less(t, memoryUsedMB, 500.0, "Should use less than 500MB memory")
	
	// Verify metrics
	metrics := ingestor.Metrics()
	assert.Equal(t, int64(recordCount), metrics.ParsedLines)
	assert.Equal(t, int64(0), metrics.ErrorLines, "Should have no parsing errors")
	assert.False(t, metrics.IsIncomplete(), "Should not be marked as incomplete")
}

// TestMemoryLimitedProcessing tests memory usage limits and streaming processing
func TestMemoryLimitedProcessing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory limit test in short mode")
	}
	
	tmpDir := t.TempDir()
	
	// Create multiple log files to test streaming
	logFiles := make([]string, 5)
	for i := 0; i < 5; i++ {
		logFile := filepath.Join(tmpDir, fmt.Sprintf("access_%d.log", i))
		createMediumLogFile(t, logFile, 10*1024*1024) // 10MB each
		logFiles[i] = logFile
	}
	
	// Test streaming processing with memory monitoring
	var memStats []runtime.MemStats
	
	ingestor := traffic.NewNginxAccessIngestor()
	options := traffic.DefaultIngestOptions()
	
	iterator, err := ingestor.Ingest(logFiles, options)
	require.NoError(t, err)
	
	recordCount := 0
	maxMemoryMB := 0.0
	
	// Monitor memory usage during processing
	for iterator.Next() {
		recordCount++
		
		if recordCount%5000 == 0 {
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)
			memStats = append(memStats, mem)
			
			currentMemMB := float64(mem.Alloc) / (1024 * 1024)
			if currentMemMB > maxMemoryMB {
				maxMemoryMB = currentMemMB
			}
			
			t.Logf("Processed %d records, current memory: %.2f MB", recordCount, currentMemMB)
		}
	}
	require.NoError(t, iterator.Err())
	
	t.Logf("Memory-limited processing results:")
	t.Logf("  - Total records: %d", recordCount)
	t.Logf("  - Max memory usage: %.2f MB", maxMemoryMB)
	t.Logf("  - Memory samples: %d", len(memStats))
	
	// Memory usage should be bounded
	assert.Less(t, maxMemoryMB, 200.0, "Memory usage should be bounded under 200MB")
	assert.Greater(t, recordCount, 50000, "Should process significant number of records")
	
	// Verify streaming behavior - memory shouldn't grow linearly with input size
	if len(memStats) > 2 {
		firstMem := float64(memStats[0].Alloc) / (1024 * 1024)
		lastMem := float64(memStats[len(memStats)-1].Alloc) / (1024 * 1024)
		memGrowthRatio := lastMem / firstMem
		
		t.Logf("Memory growth ratio: %.2f", memGrowthRatio)
		assert.Less(t, memGrowthRatio, 3.0, "Memory growth should be bounded (streaming)")
	}
}

// TestConcurrentProcessing tests concurrent processing performance
func TestConcurrentProcessing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent processing test in short mode")
	}
	
	tmpDir := t.TempDir()
	
	// Create multiple log files for concurrent processing
	numFiles := 4
	logFiles := make([]string, numFiles)
	
	for i := 0; i < numFiles; i++ {
		logFile := filepath.Join(tmpDir, fmt.Sprintf("concurrent_%d.log", i))
		createMediumLogFile(t, logFile, 5*1024*1024) // 5MB each
		logFiles[i] = logFile
	}
	
	// Test sequential processing
	t.Logf("Testing sequential processing...")
	sequentialStart := time.Now()
	sequentialRecords := 0
	
	for _, logFile := range logFiles {
		ingestor := traffic.NewNginxAccessIngestor()
		options := traffic.DefaultIngestOptions()
		
		iterator, err := ingestor.Ingest([]string{logFile}, options)
		require.NoError(t, err)
		
		for iterator.Next() {
			sequentialRecords++
		}
		require.NoError(t, iterator.Err())
	}
	
	sequentialTime := time.Since(sequentialStart)
	sequentialRate := float64(sequentialRecords) / sequentialTime.Seconds()
	
	// Test concurrent processing
	t.Logf("Testing concurrent processing...")
	concurrentStart := time.Now()
	
	var wg sync.WaitGroup
	var mu sync.Mutex
	concurrentRecords := 0
	
	for _, logFile := range logFiles {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			
			ingestor := traffic.NewNginxAccessIngestor()
			options := traffic.DefaultIngestOptions()
			
			iterator, err := ingestor.Ingest([]string{file}, options)
			require.NoError(t, err)
			
			localCount := 0
			for iterator.Next() {
				localCount++
			}
			require.NoError(t, iterator.Err())
			
			mu.Lock()
			concurrentRecords += localCount
			mu.Unlock()
		}(logFile)
	}
	
	wg.Wait()
	concurrentTime := time.Since(concurrentStart)
	concurrentRate := float64(concurrentRecords) / concurrentTime.Seconds()
	
	t.Logf("Concurrent processing results:")
	t.Logf("  - Sequential: %d records in %v (%.2f records/sec)", sequentialRecords, sequentialTime, sequentialRate)
	t.Logf("  - Concurrent: %d records in %v (%.2f records/sec)", concurrentRecords, concurrentTime, concurrentRate)
	
	// Verify results
	assert.Equal(t, sequentialRecords, concurrentRecords, "Should process same number of records")
	
	// Concurrent processing should be faster (with some tolerance for test environment)
	speedupRatio := sequentialTime.Seconds() / concurrentTime.Seconds()
	t.Logf("  - Speedup ratio: %.2f", speedupRatio)
	
	// Allow for some overhead, but expect at least some improvement
	assert.Greater(t, speedupRatio, 1.1, "Concurrent processing should provide some speedup")
}

// TestCompressedFilePerformance tests performance with compressed files
func TestCompressedFilePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping compressed file performance test in short mode")
	}
	
	tmpDir := t.TempDir()
	
	// Create uncompressed log file
	uncompressedFile := filepath.Join(tmpDir, "access.log")
	createMediumLogFile(t, uncompressedFile, 20*1024*1024) // 20MB
	
	// Create compressed version
	compressedFile := filepath.Join(tmpDir, "access.log.gz")
	compressLogFile(t, uncompressedFile, compressedFile)
	
	// Test uncompressed processing
	t.Logf("Testing uncompressed file processing...")
	uncompressedStart := time.Now()
	
	ingestor1 := traffic.NewNginxAccessIngestor()
	options1 := traffic.DefaultIngestOptions()
	
	iterator1, err := ingestor1.Ingest([]string{uncompressedFile}, options1)
	require.NoError(t, err)
	
	uncompressedRecords := 0
	for iterator1.Next() {
		uncompressedRecords++
	}
	require.NoError(t, iterator1.Err())
	
	uncompressedTime := time.Since(uncompressedStart)
	uncompressedRate := float64(uncompressedRecords) / uncompressedTime.Seconds()
	
	// Test compressed processing
	t.Logf("Testing compressed file processing...")
	compressedStart := time.Now()
	
	ingestor2 := traffic.NewNginxAccessIngestor()
	options2 := traffic.DefaultIngestOptions()
	
	iterator2, err := ingestor2.Ingest([]string{compressedFile}, options2)
	require.NoError(t, err)
	
	compressedRecords := 0
	for iterator2.Next() {
		compressedRecords++
	}
	require.NoError(t, iterator2.Err())
	
	compressedTime := time.Since(compressedStart)
	compressedRate := float64(compressedRecords) / compressedTime.Seconds()
	
	// Get file sizes
	uncompressedStat, err := os.Stat(uncompressedFile)
	require.NoError(t, err)
	compressedStat, err := os.Stat(compressedFile)
	require.NoError(t, err)
	
	compressionRatio := float64(uncompressedStat.Size()) / float64(compressedStat.Size())
	
	t.Logf("Compressed file performance results:")
	t.Logf("  - Uncompressed: %d records in %v (%.2f records/sec, %d bytes)", 
		uncompressedRecords, uncompressedTime, uncompressedRate, uncompressedStat.Size())
	t.Logf("  - Compressed: %d records in %v (%.2f records/sec, %d bytes)", 
		compressedRecords, compressedTime, compressedRate, compressedStat.Size())
	t.Logf("  - Compression ratio: %.2f:1", compressionRatio)
	
	// Verify results
	assert.Equal(t, uncompressedRecords, compressedRecords, "Should process same number of records")
	assert.Greater(t, compressionRatio, 2.0, "Should achieve reasonable compression")
	
	// Compressed processing might be slower due to decompression overhead, but should be reasonable
	performanceRatio := compressedRate / uncompressedRate
	t.Logf("  - Performance ratio (compressed/uncompressed): %.2f", performanceRatio)
	assert.Greater(t, performanceRatio, 0.3, "Compressed processing shouldn't be too much slower")
}

// TestContractGenerationPerformance tests performance of contract generation
func TestContractGenerationPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping contract generation performance test in short mode")
	}
	
	tmpDir := t.TempDir()
	
	// Create log file with diverse patterns for clustering
	logFile := filepath.Join(tmpDir, "diverse_patterns.log")
	createDiversePatternLogFile(t, logFile, 100000) // 100k records
	
	// Process logs
	ingestor := traffic.NewNginxAccessIngestor()
	options := traffic.DefaultIngestOptions()
	
	iterator, err := ingestor.Ingest([]string{logFile}, options)
	require.NoError(t, err)
	
	// Test contract generation performance
	t.Logf("Testing contract generation performance...")
	generationStart := time.Now()
	
	generator := engine.NewContractGeneratorLite()
	generationOptions := &engine.GenerationOptions{
		PathClusteringThreshold: 0.8,
		MinSampleSize:          20,
		RequiredFieldThreshold: 0.95,
		MinEndpointSamples:     5,
		StatusAggregation:      "auto",
		ServiceName:            "performance-test",
		ServiceVersion:         "v1.0.0",
		MaxUniqueValues:        10000, // Test memory limits
	}
	generator.SetOptions(generationOptions)
	
	spec, err := generator.GenerateSpec(iterator)
	require.NoError(t, err)
	require.NotNil(t, spec)
	
	generationTime := time.Since(generationStart)
	
	// Get processing metrics
	metrics := ingestor.Metrics()
	processingRate := float64(metrics.ParsedLines) / generationTime.Seconds()
	
	t.Logf("Contract generation performance results:")
	t.Logf("  - Records processed: %d", metrics.ParsedLines)
	t.Logf("  - Generation time: %v", generationTime)
	t.Logf("  - Processing rate: %.2f records/sec", processingRate)
	t.Logf("  - Endpoints discovered: %d", len(spec.Spec.Endpoints))
	t.Logf("  - Error rate: %.2f%%", float64(metrics.ErrorLines)/float64(metrics.TotalLines)*100)
	
	// Performance assertions
	assert.Greater(t, processingRate, 5000.0, "Should process at least 5000 records/sec during generation")
	assert.Less(t, generationTime, 30*time.Second, "Generation should complete within 30 seconds")
	assert.Greater(t, len(spec.Spec.Endpoints), 5, "Should discover multiple endpoints")
	assert.Less(t, float64(metrics.ErrorLines)/float64(metrics.TotalLines), 0.01, "Error rate should be < 1%")
}

// TestStressTestWithErrorHandling tests system behavior under stress with errors
func TestStressTestWithErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}
	
	tmpDir := t.TempDir()
	
	// Create log file with mixed valid and invalid entries
	logFile := filepath.Join(tmpDir, "stress_with_errors.log")
	createStressLogFileWithErrors(t, logFile, 50000) // 50k records with 10% errors
	
	// Test processing under stress
	t.Logf("Testing stress processing with error handling...")
	stressStart := time.Now()
	
	var memBefore, memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)
	
	ingestor := traffic.NewNginxAccessIngestor()
	options := traffic.DefaultIngestOptions()
	options.MaxErrorSamples = 100 // Limit error samples
	
	iterator, err := ingestor.Ingest([]string{logFile}, options)
	require.NoError(t, err)
	
	recordCount := 0
	for iterator.Next() {
		recordCount++
		
		// Simulate some processing work
		if recordCount%1000 == 0 {
			runtime.GC() // Force GC periodically to test memory management
		}
	}
	require.NoError(t, iterator.Err())
	
	stressTime := time.Since(stressStart)
	
	runtime.GC()
	runtime.ReadMemStats(&memAfter)
	
	// Get metrics
	metrics := ingestor.Metrics()
	processingRate := float64(recordCount) / stressTime.Seconds()
	memoryUsedMB := float64(memAfter.Alloc-memBefore.Alloc) / (1024 * 1024)
	errorRate := float64(metrics.ErrorLines) / float64(metrics.TotalLines)
	
	t.Logf("Stress test with error handling results:")
	t.Logf("  - Total lines: %d", metrics.TotalLines)
	t.Logf("  - Parsed records: %d", recordCount)
	t.Logf("  - Error lines: %d", metrics.ErrorLines)
	t.Logf("  - Error rate: %.2f%%", errorRate*100)
	t.Logf("  - Processing time: %v", stressTime)
	t.Logf("  - Processing rate: %.2f records/sec", processingRate)
	t.Logf("  - Memory used: %.2f MB", memoryUsedMB)
	t.Logf("  - Error samples collected: %d", len(metrics.ErrorSamples))
	
	// Stress test assertions
	assert.Greater(t, processingRate, 2000.0, "Should maintain reasonable processing rate under stress")
	assert.Less(t, memoryUsedMB, 300.0, "Should manage memory efficiently under stress")
	assert.Greater(t, errorRate, 0.05, "Should have encountered expected error rate (~10%)")
	assert.Less(t, errorRate, 0.15, "Error rate shouldn't be too high")
	assert.LessOrEqual(t, len(metrics.ErrorSamples), 100, "Should limit error samples as configured")
	
	// System should remain stable
	assert.Equal(t, metrics.ParsedLines, int64(recordCount), "Parsed count should match iterator count")
	// With 10% error rate, it might be marked as incomplete depending on threshold
	// Just verify the error rate is reasonable
	assert.Less(t, errorRate, 0.5, "Error rate should be less than 50%")
}

// Helper functions for creating test data

func createLargeLogFile(t *testing.T, filename string, targetSize int64) {
	file, err := os.Create(filename)
	require.NoError(t, err)
	defer file.Close()
	
	// Generate realistic log patterns
	patterns := []string{
		`%s - - [10/Aug/2025:12:%02d:%02d +0000] "GET /api/users/%d HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`,
		`%s - - [10/Aug/2025:12:%02d:%02d +0000] "POST /api/users HTTP/1.1" 201 890 "http://example.com" "curl/7.68.0"`,
		`%s - - [10/Aug/2025:12:%02d:%02d +0000] "GET /api/posts/%s HTTP/1.1" 200 2345 "http://example.com" "Mozilla/5.0"`,
		`%s - - [10/Aug/2025:12:%02d:%02d +0000] "PUT /api/users/%d HTTP/1.1" 200 456 "http://example.com" "curl/7.68.0"`,
		`%s - - [10/Aug/2025:12:%02d:%02d +0000] "DELETE /api/posts/%s HTTP/1.1" 204 0 "http://example.com" "curl/7.68.0"`,
		`%s - - [10/Aug/2025:12:%02d:%02d +0000] "GET /health HTTP/1.1" 200 45 "-" "kube-probe/1.0"`,
		`%s - - [10/Aug/2025:12:%02d:%02d +0000] "GET /metrics HTTP/1.1" 200 1024 "-" "prometheus/2.0"`,
	}
	
	var currentSize int64
	lineCount := 0
	
	for currentSize < targetSize {
		pattern := patterns[rand.Intn(len(patterns))]
		
		// Generate realistic values
		ip := fmt.Sprintf("192.168.%d.%d", rand.Intn(256), rand.Intn(256))
		minute := rand.Intn(60)
		second := rand.Intn(60)
		userID := rand.Intn(10000)
		postID := fmt.Sprintf("post_%d", rand.Intn(10000))
		
		var line string
		switch {
		case strings.Contains(pattern, "/api/users/%d"):
			line = fmt.Sprintf(pattern, ip, minute, second, userID)
		case strings.Contains(pattern, "/api/posts/%s"):
			line = fmt.Sprintf(pattern, ip, minute, second, postID)
		default:
			line = fmt.Sprintf(pattern, ip, minute, second)
		}
		
		line += "\n"
		n, err := file.WriteString(line)
		require.NoError(t, err)
		
		currentSize += int64(n)
		lineCount++
		
		if lineCount%10000 == 0 {
			t.Logf("Generated %d lines, %d bytes", lineCount, currentSize)
		}
	}
	
	t.Logf("Created large log file with %d lines, %d bytes", lineCount, currentSize)
}

func createMediumLogFile(t *testing.T, filename string, targetSize int64) {
	file, err := os.Create(filename)
	require.NoError(t, err)
	defer file.Close()
	
	// Simpler pattern for medium files
	pattern := `192.168.1.%d - - [10/Aug/2025:12:%02d:%02d +0000] "GET /api/test/%d HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`
	
	var currentSize int64
	counter := 1
	
	for currentSize < targetSize {
		line := fmt.Sprintf(pattern, counter%256, (counter/60)%60, counter%60, counter) + "\n"
		n, err := file.WriteString(line)
		require.NoError(t, err)
		
		currentSize += int64(n)
		counter++
	}
}

func createDiversePatternLogFile(t *testing.T, filename string, numRecords int) {
	file, err := os.Create(filename)
	require.NoError(t, err)
	defer file.Close()
	
	// Create diverse patterns for clustering algorithm testing
	endpoints := []string{
		"/api/users/%d",
		"/api/posts/%s",
		"/api/comments/%d",
		"/api/categories/%s",
		"/api/tags/%d",
		"/health",
		"/metrics",
		"/api/v1/users/%d",
		"/api/v2/users/%d",
		"/static/images/%s.jpg",
		"/static/css/%s.css",
		"/static/js/%s.js",
	}
	
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	statusCodes := []int{200, 201, 204, 400, 401, 403, 404, 500}
	
	for i := 0; i < numRecords; i++ {
		endpoint := endpoints[rand.Intn(len(endpoints))]
		method := methods[rand.Intn(len(methods))]
		status := statusCodes[rand.Intn(len(statusCodes))]
		
		var path string
		if strings.Contains(endpoint, "%d") {
			path = fmt.Sprintf(endpoint, rand.Intn(10000))
		} else if strings.Contains(endpoint, "%s") {
			path = fmt.Sprintf(endpoint, fmt.Sprintf("item_%d", rand.Intn(1000)))
		} else {
			path = endpoint
		}
		
		ip := fmt.Sprintf("192.168.%d.%d", rand.Intn(256), rand.Intn(256))
		minute := rand.Intn(60)
		second := rand.Intn(60)
		
		line := fmt.Sprintf(`%s - - [10/Aug/2025:12:%02d:%02d +0000] "%s %s HTTP/1.1" %d 1234 "http://example.com" "Mozilla/5.0"`,
			ip, minute, second, method, path, status) + "\n"
		
		_, err := file.WriteString(line)
		require.NoError(t, err)
	}
}

func createStressLogFileWithErrors(t *testing.T, filename string, numRecords int) {
	file, err := os.Create(filename)
	require.NoError(t, err)
	defer file.Close()
	
	validPattern := `192.168.1.%d - - [10/Aug/2025:12:%02d:%02d +0000] "GET /api/test/%d HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`
	
	// Invalid patterns to test error handling
	invalidPatterns := []string{
		"invalid log line without proper format",
		"192.168.1.1 - - [invalid-timestamp] \"GET /api/test HTTP/1.1\" 200 1234",
		"incomplete log line",
		"192.168.1.1 - - [10/Aug/2025:12:00:00 +0000] \"INVALID_METHOD /api/test HTTP/1.1\" 999 1234",
		"", // empty line
	}
	
	for i := 0; i < numRecords; i++ {
		var line string
		
		// 10% error rate
		if rand.Float32() < 0.1 {
			line = invalidPatterns[rand.Intn(len(invalidPatterns))]
		} else {
			line = fmt.Sprintf(validPattern, i%256, (i/60)%60, i%60, i)
		}
		
		line += "\n"
		_, err := file.WriteString(line)
		require.NoError(t, err)
	}
}

func compressLogFile(t *testing.T, srcFile, dstFile string) {
	src, err := os.Open(srcFile)
	require.NoError(t, err)
	defer src.Close()
	
	dst, err := os.Create(dstFile)
	require.NoError(t, err)
	defer dst.Close()
	
	gzWriter := gzip.NewWriter(dst)
	defer gzWriter.Close()
	
	buffer := make([]byte, 64*1024) // 64KB buffer
	for {
		n, err := src.Read(buffer)
		if n > 0 {
			_, writeErr := gzWriter.Write(buffer[:n])
			require.NoError(t, writeErr)
		}
		if err != nil {
			break
		}
	}
}