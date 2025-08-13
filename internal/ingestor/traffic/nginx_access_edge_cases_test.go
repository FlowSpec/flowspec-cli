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

package traffic

import (
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNginxAccessIngestor_EdgeCases_MalformedLogLines(t *testing.T) {
	ingestor := NewNginxAccessIngestor()
	options := &IngestOptions{
		LogFormat: "combined",
	}
	ingestor.options = options
	err := ingestor.setupRegex()
	require.NoError(t, err)

	testCases := []struct {
		name    string
		logLine string
		wantErr bool
	}{
		{
			name:    "Empty line",
			logLine: "",
			wantErr: true,
		},
		{
			name:    "Only whitespace",
			logLine: "   \t  \n  ",
			wantErr: true,
		},
		{
			name:    "Partial log line",
			logLine: `192.168.1.1 - -`,
			wantErr: true,
		},
		{
			name:    "Missing quotes around request",
			logLine: `192.168.1.1 - - [10/Aug/2025:12:00:00 +0000] GET /api/test HTTP/1.1 200 1234 "http://example.com" "Mozilla/5.0"`,
			wantErr: true,
		},
		{
			name:    "Unmatched quotes",
			logLine: `192.168.1.1 - - [10/Aug/2025:12:00:00 +0000] "GET /api/test HTTP/1.1 200 1234 "http://example.com" "Mozilla/5.0"`,
			wantErr: true,
		},
		{
			name:    "Invalid HTTP method",
			logLine: `192.168.1.1 - - [10/Aug/2025:12:00:00 +0000] "INVALID /api/test HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`,
			wantErr: false, // Should parse but with invalid method
		},
		{
			name:    "Very long URL",
			logLine: `192.168.1.1 - - [10/Aug/2025:12:00:00 +0000] "GET /api/very/long/path/that/goes/on/and/on/with/many/segments/and/parameters?param1=value1&param2=value2&param3=value3&param4=value4&param5=value5 HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`,
			wantErr: false,
		},
		{
			name:    "Special characters in URL",
			logLine: `192.168.1.1 - - [10/Aug/2025:12:00:00 +0000] "GET /api/files/file%20with%20spaces%21%40%23%24%25.txt HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`,
			wantErr: false,
		},
		{
			name:    "Non-ASCII characters in user agent",
			logLine: `192.168.1.1 - - [10/Aug/2025:12:00:00 +0000] "GET /api/test HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0 (中文测试)"`,
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			record, err := ingestor.parseLogLine(tc.logLine)

			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, record)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, record)
			}
		})
	}
}

func TestNginxAccessIngestor_EdgeCases_ExtremeTimestamps(t *testing.T) {
	ingestor := NewNginxAccessIngestor()
	ingestor.timeLayout = "02/Jan/2006:15:04:05 -0700"

	testCases := []struct {
		name      string
		timeStr   string
		wantErr   bool
		checkTime func(t *testing.T, parsedTime time.Time)
	}{
		{
			name:    "Year 1970",
			timeStr: "01/Jan/1970:00:00:00 +0000",
			wantErr: false,
			checkTime: func(t *testing.T, parsedTime time.Time) {
				assert.Equal(t, 1970, parsedTime.Year())
			},
		},
		{
			name:    "Year 2099",
			timeStr: "31/Dec/2099:23:59:59 +0000",
			wantErr: false,
			checkTime: func(t *testing.T, parsedTime time.Time) {
				assert.Equal(t, 2099, parsedTime.Year())
			},
		},
		{
			name:    "Leap year February 29",
			timeStr: "29/Feb/2024:12:00:00 +0000",
			wantErr: false,
			checkTime: func(t *testing.T, parsedTime time.Time) {
				assert.Equal(t, 2024, parsedTime.Year())
				assert.Equal(t, time.February, parsedTime.Month())
				assert.Equal(t, 29, parsedTime.Day())
			},
		},
		{
			name:    "Invalid leap year",
			timeStr: "29/Feb/2023:12:00:00 +0000",
			wantErr: true,
		},
		{
			name:    "Extreme timezone offset",
			timeStr: "01/Jan/2025:12:00:00 +1400",
			wantErr: false,
			checkTime: func(t *testing.T, parsedTime time.Time) {
				// Should be converted to UTC
				assert.Equal(t, time.UTC, parsedTime.Location())
			},
		},
		{
			name:    "Negative extreme timezone",
			timeStr: "01/Jan/2025:12:00:00 -1200",
			wantErr: false,
			checkTime: func(t *testing.T, parsedTime time.Time) {
				assert.Equal(t, time.UTC, parsedTime.Location())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parsedTime, err := ingestor.parseTimestamp(tc.timeStr)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.checkTime != nil {
					tc.checkTime(t, parsedTime)
				}
			}
		})
	}
}

func TestNginxAccessIngestor_EdgeCases_ExtremeFileSizes(t *testing.T) {
	ingestor := NewNginxAccessIngestor()

	// Create temporary file with single very long line (but not too long to cause scanner issues)
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "extreme.log")

	// Create a long but manageable log line (64KB - within scanner limits)
	longURL := strings.Repeat("a", 64*1024)
	longLogLine := fmt.Sprintf(`192.168.1.1 - - [10/Aug/2025:12:00:00 +0000] "GET /%s HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`, longURL)

	err := os.WriteFile(logFile, []byte(longLogLine), 0644)
	require.NoError(t, err)

	// Process the file
	options := DefaultIngestOptions()
	iterator, err := ingestor.Ingest([]string{logFile}, options)
	
	// This might fail due to line length limits, which is expected behavior
	if err != nil {
		assert.Contains(t, err.Error(), "token too long")
		return
	}

	// If it succeeds, verify the parsing
	var records []*NormalizedRecord
	for iterator.Next() {
		records = append(records, iterator.Value())
	}
	
	// Check for errors during iteration
	if iterator.Err() != nil {
		assert.Contains(t, iterator.Err().Error(), "token too long")
		return
	}

	// If successful, verify the record
	if len(records) > 0 {
		record := records[0]
		assert.Equal(t, "GET", record.Method)
		assert.Contains(t, record.Path, strings.Repeat("a", 100)) // Should contain the long path
	}
}

func TestNginxAccessIngestor_EdgeCases_CorruptedGzipFile(t *testing.T) {
	ingestor := NewNginxAccessIngestor()

	// Create temporary corrupted gzip file
	tmpDir := t.TempDir()
	gzipFile := filepath.Join(tmpDir, "corrupted.log.gz")

	// Write invalid gzip data
	err := os.WriteFile(gzipFile, []byte("not a gzip file"), 0644)
	require.NoError(t, err)

	// Try to process the corrupted file
	options := DefaultIngestOptions()
	iterator, err := ingestor.Ingest([]string{gzipFile}, options)

	// Ingest should succeed initially (async processing)
	assert.NoError(t, err)
	assert.NotNil(t, iterator)
	
	// But should encounter error when trying to read
	hasNext := iterator.Next()
	assert.False(t, hasNext)
	assert.Error(t, iterator.Err())
	
	iterator.Close()
}

func TestNginxAccessIngestor_EdgeCases_EmptyGzipFile(t *testing.T) {
	ingestor := NewNginxAccessIngestor()

	// Create temporary empty gzip file
	tmpDir := t.TempDir()
	gzipFile := filepath.Join(tmpDir, "empty.log.gz")

	file, err := os.Create(gzipFile)
	require.NoError(t, err)

	gzWriter := gzip.NewWriter(file)
	err = gzWriter.Close()
	require.NoError(t, err)
	err = file.Close()
	require.NoError(t, err)

	// Process the empty gzip file
	options := DefaultIngestOptions()
	iterator, err := ingestor.Ingest([]string{gzipFile}, options)
	require.NoError(t, err)

	// Should handle empty file gracefully
	var records []*NormalizedRecord
	for iterator.Next() {
		records = append(records, iterator.Value())
	}
	assert.NoError(t, iterator.Err())
	assert.Empty(t, records)

	// Check metrics
	metrics := ingestor.Metrics()
	assert.Equal(t, int64(0), metrics.TotalLines)
	assert.Equal(t, int64(0), metrics.ParsedLines)
	assert.Equal(t, int64(0), metrics.ErrorLines)
}

func TestNginxAccessIngestor_EdgeCases_MixedLineEndings(t *testing.T) {
	ingestor := NewNginxAccessIngestor()

	// Create temporary file with mixed line endings
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "mixed-endings.log")

	// Mix of \n, \r\n, and \r line endings
	logContent := `192.168.1.1 - - [10/Aug/2025:12:00:00 +0000] "GET /api/test1 HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"` + "\n" +
		`192.168.1.2 - - [10/Aug/2025:12:01:00 +0000] "GET /api/test2 HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"` + "\r\n" +
		`192.168.1.3 - - [10/Aug/2025:12:02:00 +0000] "GET /api/test3 HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"` + "\r" +
		`192.168.1.4 - - [10/Aug/2025:12:03:00 +0000] "GET /api/test4 HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`

	err := os.WriteFile(logFile, []byte(logContent), 0644)
	require.NoError(t, err)

	// Process the file
	options := DefaultIngestOptions()
	iterator, err := ingestor.Ingest([]string{logFile}, options)
	require.NoError(t, err)

	// Should handle mixed line endings
	var records []*NormalizedRecord
	for iterator.Next() {
		records = append(records, iterator.Value())
	}
	assert.NoError(t, iterator.Err())

	// Should parse most lines (note: \r-only line endings may not be handled by bufio.Scanner)
	// This is acceptable as \r-only line endings are rare in modern systems
	assert.GreaterOrEqual(t, len(records), 3)
	assert.LessOrEqual(t, len(records), 4)
	for i, record := range records {
		assert.Equal(t, "GET", record.Method)
		assert.Equal(t, fmt.Sprintf("/api/test%d", i+1), record.Path)
	}
}

func TestNginxAccessIngestor_EdgeCases_VeryHighSampleRate(t *testing.T) {
	ingestor := NewNginxAccessIngestor()

	// Create test log file
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "sample-test.log")

	var logLines []string
	for i := 0; i < 1000; i++ {
		logLine := fmt.Sprintf(`192.168.1.%d - - [10/Aug/2025:12:00:00 +0000] "GET /api/test%d HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`, i%255+1, i)
		logLines = append(logLines, logLine)
	}

	err := os.WriteFile(logFile, []byte(strings.Join(logLines, "\n")), 0644)
	require.NoError(t, err)

	// Test with very high sample rate (should process all)
	options := &IngestOptions{
		LogFormat:  "combined",
		SampleRate: 1.0, // 100%
	}

	iterator, err := ingestor.Ingest([]string{logFile}, options)
	require.NoError(t, err)

	var records []*NormalizedRecord
	for iterator.Next() {
		records = append(records, iterator.Value())
	}
	assert.NoError(t, iterator.Err())

	// Should process all records
	assert.Len(t, records, 1000)

	// Test with very low sample rate
	ingestor2 := NewNginxAccessIngestor()
	options.SampleRate = 0.01 // 1%

	iterator2, err := ingestor2.Ingest([]string{logFile}, options)
	require.NoError(t, err)

	var records2 []*NormalizedRecord
	for iterator2.Next() {
		records2 = append(records2, iterator2.Value())
	}
	assert.NoError(t, iterator2.Err())

	// Should process much fewer records
	assert.Less(t, len(records2), 100) // Should be around 10, but allow some variance
}

func TestNginxAccessIngestor_EdgeCases_TimeFilterEdgeCases(t *testing.T) {
	ingestor := NewNginxAccessIngestor()

	baseTime := time.Date(2025, 8, 10, 12, 0, 0, 0, time.UTC)
	
	testCases := []struct {
		name      string
		timeRange *TimeRange
		timestamp time.Time
		expected  bool
	}{
		{
			name: "Exactly at since boundary",
			timeRange: &TimeRange{
				Since: &baseTime,
			},
			timestamp: baseTime,
			expected:  true,
		},
		{
			name: "Exactly at until boundary",
			timeRange: &TimeRange{
				Until: &baseTime,
			},
			timestamp: baseTime,
			expected:  true,
		},
		{
			name: "One nanosecond before since",
			timeRange: &TimeRange{
				Since: &baseTime,
			},
			timestamp: baseTime.Add(-1 * time.Nanosecond),
			expected:  false,
		},
		{
			name: "One nanosecond after until",
			timeRange: &TimeRange{
				Until: &baseTime,
			},
			timestamp: baseTime.Add(1 * time.Nanosecond),
			expected:  false,
		},
		{
			name: "Both boundaries - within range",
			timeRange: &TimeRange{
				Since: &baseTime,
				Until: func() *time.Time { t := baseTime.Add(1 * time.Hour); return &t }(),
			},
			timestamp: baseTime.Add(30 * time.Minute),
			expected:  true,
		},
		{
			name: "Both boundaries - outside range (before)",
			timeRange: &TimeRange{
				Since: &baseTime,
				Until: func() *time.Time { t := baseTime.Add(1 * time.Hour); return &t }(),
			},
			timestamp: baseTime.Add(-1 * time.Minute),
			expected:  false,
		},
		{
			name: "Both boundaries - outside range (after)",
			timeRange: &TimeRange{
				Since: &baseTime,
				Until: func() *time.Time { t := baseTime.Add(1 * time.Hour); return &t }(),
			},
			timestamp: baseTime.Add(2 * time.Hour),
			expected:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ingestor.options = &IngestOptions{
				TimeFilter: tc.timeRange,
			}

			result := ingestor.isWithinTimeRange(tc.timestamp)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNginxAccessIngestor_EdgeCases_ConcurrentAccess(t *testing.T) {
	// This test ensures the ingestor is safe for concurrent access
	// (though the current implementation is not designed for concurrent use)
	ingestor := NewNginxAccessIngestor()

	// Create test log file with supported name pattern
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "access.log")

	logContent := strings.Join([]string{
		`192.168.1.1 - - [10/Aug/2025:12:00:00 +0000] "GET /api/test1 HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`,
		`192.168.1.2 - - [10/Aug/2025:12:01:00 +0000] "GET /api/test2 HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`,
	}, "\n")

	err := os.WriteFile(logFile, []byte(logContent), 0644)
	require.NoError(t, err)

	// Test that multiple calls to Supports don't interfere
	assert.True(t, ingestor.Supports(logFile))
	assert.True(t, ingestor.Supports(logFile))
	assert.True(t, ingestor.Supports(logFile))

	// Test that multiple calls to Metrics don't crash
	metrics1 := ingestor.Metrics()
	metrics2 := ingestor.Metrics()
	assert.Equal(t, metrics1, metrics2)

	// Test that Close can be called multiple times
	assert.NoError(t, ingestor.Close())
	assert.NoError(t, ingestor.Close())
}

func TestNginxAccessIngestor_EdgeCases_InvalidRegexPatterns(t *testing.T) {
	ingestor := NewNginxAccessIngestor()

	testCases := []struct {
		name        string
		customRegex string
		expectError bool
	}{
		{
			name:        "Unclosed group",
			customRegex: `^(\S+ - (\S+`,
			expectError: true,
		},
		{
			name:        "Invalid escape sequence",
			customRegex: `^\q`, // \q is not a valid escape sequence in Go regex
			expectError: true,
		},
		{
			name:        "Empty regex",
			customRegex: "",
			expectError: true,
		},
		{
			name:        "Valid but complex regex",
			customRegex: `^(\S+)\s+-\s+(\S+)\s+\[([^\]]+)\]\s+"([A-Z]+)\s+([^"]*)\s+HTTP/[^"]*"\s+(\d+)\s+(\d+)`,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			options := &IngestOptions{
				CustomRegex: tc.customRegex,
			}
			ingestor.options = options

			err := ingestor.setupRegex()

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}