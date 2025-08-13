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
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNginxAccessIngestor(t *testing.T) {
	ingestor := NewNginxAccessIngestor()

	assert.NotNil(t, ingestor)
	assert.NotNil(t, ingestor.metrics)
}

func TestNginxAccessIngestor_Supports(t *testing.T) {
	ingestor := NewNginxAccessIngestor()

	testCases := []struct {
		filename string
		expected bool
	}{
		// Supported patterns
		{"access.log", true},
		{"access_log", true},
		{"nginx.log", true},
		{"nginx_access.log", true},
		{"server_access.log", true},
		{"app_access_log", true},
		
		// Compressed versions
		{"access.log.gz", true},
		{"access_log.gz", true},
		{"nginx.log.zst", true},
		{"nginx_access.log.gz", true},
		
		// Case insensitive
		{"ACCESS.LOG", true},
		{"Nginx_Access.Log", true},
		
		// Not supported
		{"error.log", false},
		{"application.log", false},
		{"debug.log", false},
		{"random.txt", false},
		{"", false},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			result := ingestor.Supports(tc.filename)
			assert.Equal(t, tc.expected, result, "filename: %s", tc.filename)
		})
	}
}

func TestNginxAccessIngestor_setupRegex_Combined(t *testing.T) {
	ingestor := NewNginxAccessIngestor()
	options := &IngestOptions{
		LogFormat: "combined",
	}
	ingestor.options = options

	err := ingestor.setupRegex()

	assert.NoError(t, err)
	assert.NotNil(t, ingestor.regex)
	assert.Equal(t, "combined", ingestor.logFormat)
	assert.Equal(t, "02/Jan/2006:15:04:05 -0700", ingestor.timeLayout)
}

func TestNginxAccessIngestor_setupRegex_Common(t *testing.T) {
	ingestor := NewNginxAccessIngestor()
	options := &IngestOptions{
		LogFormat: "common",
	}
	ingestor.options = options

	err := ingestor.setupRegex()

	assert.NoError(t, err)
	assert.NotNil(t, ingestor.regex)
	assert.Equal(t, "common", ingestor.logFormat)
	assert.Equal(t, "02/Jan/2006:15:04:05 -0700", ingestor.timeLayout)
}

func TestNginxAccessIngestor_setupRegex_Custom(t *testing.T) {
	ingestor := NewNginxAccessIngestor()
	customRegex := `^(\S+) - (\S+) \[([^\]]+)\] "([A-Z]+) ([^"]*) HTTP/[^"]*" (\d+) (\d+)`
	options := &IngestOptions{
		CustomRegex: customRegex,
	}
	ingestor.options = options

	err := ingestor.setupRegex()

	assert.NoError(t, err)
	assert.NotNil(t, ingestor.regex)
	assert.Equal(t, "custom", ingestor.logFormat)
	assert.Equal(t, "02/Jan/2006:15:04:05 -0700", ingestor.timeLayout)
}

func TestNginxAccessIngestor_setupRegex_UnsupportedFormat(t *testing.T) {
	ingestor := NewNginxAccessIngestor()
	options := &IngestOptions{
		LogFormat: "unsupported",
	}
	ingestor.options = options

	err := ingestor.setupRegex()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported log format")
	assert.Contains(t, err.Error(), "Supported formats:")
	assert.Contains(t, err.Error(), "Example log lines:")
}

func TestNginxAccessIngestor_setupRegex_InvalidCustomRegex(t *testing.T) {
	ingestor := NewNginxAccessIngestor()
	options := &IngestOptions{
		CustomRegex: "[invalid regex",
	}
	ingestor.options = options

	err := ingestor.setupRegex()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid regex pattern")
}

func TestNginxAccessIngestor_parseLogLine_Combined(t *testing.T) {
	ingestor := NewNginxAccessIngestor()
	options := &IngestOptions{
		LogFormat: "combined",
	}
	ingestor.options = options
	err := ingestor.setupRegex()
	require.NoError(t, err)

	// Test valid combined log line
	logLine := `192.168.1.1 - - [10/Aug/2025:12:00:00 +0000] "GET /api/users/123?include=profile HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`

	record, err := ingestor.parseLogLine(logLine)

	assert.NoError(t, err)
	assert.NotNil(t, record)
	assert.Equal(t, "GET", record.Method)
	assert.Equal(t, "/api/users/123", record.Path)
	assert.Equal(t, "/api/users/123?include=profile", record.RawPath)
	assert.Equal(t, 200, record.Status)
	assert.Equal(t, int64(1234), record.BodyBytes)
	assert.Equal(t, "192.168.1.1", record.Host)
	assert.Equal(t, "http", record.Scheme)

	// Check timestamp parsing
	expectedTime, _ := time.Parse("02/Jan/2006:15:04:05 -0700", "10/Aug/2025:12:00:00 +0000")
	assert.Equal(t, expectedTime.UTC(), record.Timestamp)

	// Check query parameters
	assert.Contains(t, record.Query, "include")
	assert.Equal(t, []string{"profile"}, record.Query["include"])

	// Check headers
	assert.Contains(t, record.Headers, "referer")
	assert.Equal(t, []string{"http://example.com"}, record.Headers["referer"])
	assert.Contains(t, record.Headers, "user-agent")
	assert.Equal(t, []string{"Mozilla/5.0"}, record.Headers["user-agent"])
}

func TestNginxAccessIngestor_parseLogLine_Common(t *testing.T) {
	ingestor := NewNginxAccessIngestor()
	options := &IngestOptions{
		LogFormat: "common",
	}
	ingestor.options = options
	err := ingestor.setupRegex()
	require.NoError(t, err)

	// Test valid common log line
	logLine := `192.168.1.1 - - [10/Aug/2025:12:00:00 +0000] "POST /api/users HTTP/1.1" 201 567`

	record, err := ingestor.parseLogLine(logLine)

	assert.NoError(t, err)
	assert.NotNil(t, record)
	assert.Equal(t, "POST", record.Method)
	assert.Equal(t, "/api/users", record.Path)
	assert.Equal(t, "/api/users", record.RawPath)
	assert.Equal(t, 201, record.Status)
	assert.Equal(t, int64(567), record.BodyBytes)

	// Common format doesn't have referer/user-agent
	assert.NotContains(t, record.Headers, "referer")
	assert.NotContains(t, record.Headers, "user-agent")
}

func TestNginxAccessIngestor_parseLogLine_EdgeCases(t *testing.T) {
	ingestor := NewNginxAccessIngestor()
	options := &IngestOptions{
		LogFormat: "combined",
	}
	ingestor.options = options
	err := ingestor.setupRegex()
	require.NoError(t, err)

	testCases := []struct {
		name     string
		logLine  string
		wantErr  bool
		checkFn  func(t *testing.T, record *NormalizedRecord)
	}{
		{
			name:    "Missing body bytes (dash)",
			logLine: `192.168.1.1 - - [10/Aug/2025:12:00:00 +0000] "GET /api/test HTTP/1.1" 200 - "http://example.com" "Mozilla/5.0"`,
			wantErr: true, // This should fail because the regex expects \d+ for body bytes, not -
		},
		{
			name:    "Missing referer (dash)",
			logLine: `192.168.1.1 - - [10/Aug/2025:12:00:00 +0000] "GET /api/test HTTP/1.1" 200 1234 "-" "Mozilla/5.0"`,
			wantErr: false,
			checkFn: func(t *testing.T, record *NormalizedRecord) {
				assert.NotContains(t, record.Headers, "referer")
			},
		},
		{
			name:    "Missing user-agent (dash)",
			logLine: `192.168.1.1 - - [10/Aug/2025:12:00:00 +0000] "GET /api/test HTTP/1.1" 200 1234 "http://example.com" "-"`,
			wantErr: false,
			checkFn: func(t *testing.T, record *NormalizedRecord) {
				assert.NotContains(t, record.Headers, "user-agent")
			},
		},
		{
			name:    "Complex URL with query parameters",
			logLine: `192.168.1.1 - - [10/Aug/2025:12:00:00 +0000] "GET /api/search?q=test&limit=10&offset=20 HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`,
			wantErr: false,
			checkFn: func(t *testing.T, record *NormalizedRecord) {
				assert.Equal(t, "/api/search", record.Path)
				assert.Contains(t, record.Query, "q")
				assert.Contains(t, record.Query, "limit")
				assert.Contains(t, record.Query, "offset")
				assert.Equal(t, []string{"test"}, record.Query["q"])
				assert.Equal(t, []string{"10"}, record.Query["limit"])
				assert.Equal(t, []string{"20"}, record.Query["offset"])
			},
		},
		{
			name:    "Invalid log line format",
			logLine: `invalid log line format`,
			wantErr: true,
		},
		{
			name:    "Invalid status code",
			logLine: `192.168.1.1 - - [10/Aug/2025:12:00:00 +0000] "GET /api/test HTTP/1.1" invalid 1234 "http://example.com" "Mozilla/5.0"`,
			wantErr: true,
		},
		{
			name:    "Invalid timestamp",
			logLine: `192.168.1.1 - - [invalid-timestamp] "GET /api/test HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`,
			wantErr: true,
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
				require.NotNil(t, record)
				if tc.checkFn != nil {
					tc.checkFn(t, record)
				}
			}
		})
	}
}

func TestNginxAccessIngestor_parseTimestamp(t *testing.T) {
	ingestor := NewNginxAccessIngestor()
	ingestor.timeLayout = "02/Jan/2006:15:04:05 -0700"

	testCases := []struct {
		name      string
		timeStr   string
		wantErr   bool
		checkTime func(t *testing.T, parsedTime time.Time)
	}{
		{
			name:    "Valid timestamp with timezone",
			timeStr: "10/Aug/2025:12:00:00 +0000",
			wantErr: false,
			checkTime: func(t *testing.T, parsedTime time.Time) {
				assert.Equal(t, 2025, parsedTime.Year())
				assert.Equal(t, time.August, parsedTime.Month())
				assert.Equal(t, 10, parsedTime.Day())
				assert.Equal(t, 12, parsedTime.Hour())
				assert.Equal(t, time.UTC, parsedTime.Location())
			},
		},
		{
			name:    "Valid timestamp with negative timezone",
			timeStr: "10/Aug/2025:12:00:00 -0500",
			wantErr: false,
			checkTime: func(t *testing.T, parsedTime time.Time) {
				// Should be converted to UTC
				assert.Equal(t, time.UTC, parsedTime.Location())
				assert.Equal(t, 17, parsedTime.Hour()) // 12 + 5 = 17 UTC
			},
		},
		{
			name:    "Invalid timestamp format",
			timeStr: "invalid-timestamp",
			wantErr: true,
		},
		{
			name:    "Empty timestamp",
			timeStr: "",
			wantErr: true,
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

func TestNginxAccessIngestor_createReader_PlainFile(t *testing.T) {
	ingestor := NewNginxAccessIngestor()

	// Create temporary plain file
	tmpDir := t.TempDir()
	plainFile := filepath.Join(tmpDir, "access.log")
	content := "test content"
	err := os.WriteFile(plainFile, []byte(content), 0644)
	require.NoError(t, err)

	file, err := os.Open(plainFile)
	require.NoError(t, err)
	defer file.Close()

	reader, err := ingestor.createReader(file, plainFile)
	require.NoError(t, err)
	defer reader.Close()

	// Read content
	buf := make([]byte, len(content))
	n, err := reader.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, len(content), n)
	assert.Equal(t, content, string(buf))
}

func TestNginxAccessIngestor_createReader_GzipFile(t *testing.T) {
	ingestor := NewNginxAccessIngestor()

	// Create temporary gzip file
	tmpDir := t.TempDir()
	gzipFile := filepath.Join(tmpDir, "access.log.gz")
	content := "test content for gzip"

	file, err := os.Create(gzipFile)
	require.NoError(t, err)

	gzWriter := gzip.NewWriter(file)
	_, err = gzWriter.Write([]byte(content))
	require.NoError(t, err)
	err = gzWriter.Close()
	require.NoError(t, err)
	err = file.Close()
	require.NoError(t, err)

	// Open and read
	file, err = os.Open(gzipFile)
	require.NoError(t, err)
	defer file.Close()

	reader, err := ingestor.createReader(file, gzipFile)
	require.NoError(t, err)
	defer reader.Close()

	// Read content
	data, err := io.ReadAll(reader)
	assert.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestNginxAccessIngestor_createReader_ZstdFile(t *testing.T) {
	ingestor := NewNginxAccessIngestor()

	// Create temporary zstd file
	tmpDir := t.TempDir()
	zstdFile := filepath.Join(tmpDir, "access.log.zst")
	content := "test content for zstd"

	file, err := os.Create(zstdFile)
	require.NoError(t, err)

	zstWriter, err := zstd.NewWriter(file)
	require.NoError(t, err)
	_, err = zstWriter.Write([]byte(content))
	require.NoError(t, err)
	err = zstWriter.Close()
	require.NoError(t, err)
	err = file.Close()
	require.NoError(t, err)

	// Open and read
	file, err = os.Open(zstdFile)
	require.NoError(t, err)
	defer file.Close()

	reader, err := ingestor.createReader(file, zstdFile)
	require.NoError(t, err)
	defer reader.Close()

	// Read content
	buf := make([]byte, len(content))
	n, err := reader.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, len(content), n)
	assert.Equal(t, content, string(buf))
}

func TestNginxAccessIngestor_shouldSkipLine(t *testing.T) {
	ingestor := NewNginxAccessIngestor()
	ingestor.options = &IngestOptions{
		SampleRate: 0.5, // 50% sampling
	}
	ingestor.metrics = NewIngestMetrics()

	// Test sampling behavior
	skippedCount := 0
	totalTests := 1000

	for i := 0; i < totalTests; i++ {
		ingestor.metrics.TotalLines = int64(i)
		if ingestor.shouldSkipLine() {
			skippedCount++
		}
	}

	// With 50% sampling, we expect roughly half to be skipped
	// Allow some variance due to the simple modulo-based sampling
	expectedSkipped := totalTests / 2
	tolerance := totalTests / 10 // 10% tolerance
	assert.InDelta(t, expectedSkipped, skippedCount, float64(tolerance))
}

func TestNginxAccessIngestor_isWithinTimeRange(t *testing.T) {
	ingestor := NewNginxAccessIngestor()
	
	baseTime := time.Date(2025, 8, 10, 12, 0, 0, 0, time.UTC)
	since := baseTime.Add(-1 * time.Hour)
	until := baseTime.Add(1 * time.Hour)

	testCases := []struct {
		name      string
		timeRange *TimeRange
		timestamp time.Time
		expected  bool
	}{
		{
			name:      "No time filter",
			timeRange: nil,
			timestamp: baseTime,
			expected:  true,
		},
		{
			name: "Within range",
			timeRange: &TimeRange{
				Since: &since,
				Until: &until,
			},
			timestamp: baseTime,
			expected:  true,
		},
		{
			name: "Before since",
			timeRange: &TimeRange{
				Since: &since,
				Until: &until,
			},
			timestamp: since.Add(-1 * time.Minute),
			expected:  false,
		},
		{
			name: "After until",
			timeRange: &TimeRange{
				Since: &since,
				Until: &until,
			},
			timestamp: until.Add(1 * time.Minute),
			expected:  false,
		},
		{
			name: "Only since filter - within",
			timeRange: &TimeRange{
				Since: &since,
			},
			timestamp: baseTime,
			expected:  true,
		},
		{
			name: "Only since filter - before",
			timeRange: &TimeRange{
				Since: &since,
			},
			timestamp: since.Add(-1 * time.Minute),
			expected:  false,
		},
		{
			name: "Only until filter - within",
			timeRange: &TimeRange{
				Until: &until,
			},
			timestamp: baseTime,
			expected:  true,
		},
		{
			name: "Only until filter - after",
			timeRange: &TimeRange{
				Until: &until,
			},
			timestamp: until.Add(1 * time.Minute),
			expected:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.timeRange == nil {
				ingestor.options = &IngestOptions{}
			} else {
				ingestor.options = &IngestOptions{
					TimeFilter: tc.timeRange,
				}
			}

			result := ingestor.isWithinTimeRange(tc.timestamp)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNginxAccessIngestor_Metrics(t *testing.T) {
	ingestor := NewNginxAccessIngestor()

	metrics := ingestor.Metrics()
	assert.NotNil(t, metrics)
	assert.Equal(t, ingestor.metrics, metrics)
}

func TestNginxAccessIngestor_Close(t *testing.T) {
	ingestor := NewNginxAccessIngestor()

	err := ingestor.Close()
	assert.NoError(t, err)
}

func TestNginxAccessIngestor_Integration_ProcessFile(t *testing.T) {
	ingestor := NewNginxAccessIngestor()

	// Create test log file
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "access.log")

	logContent := strings.Join([]string{
		`192.168.1.1 - - [10/Aug/2025:12:00:00 +0000] "GET /api/users/123 HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`,
		`192.168.1.2 - - [10/Aug/2025:12:01:00 +0000] "POST /api/users HTTP/1.1" 201 567 "-" "curl/7.68.0"`,
		`192.168.1.3 - - [10/Aug/2025:12:02:00 +0000] "GET /api/users/456?include=profile HTTP/1.1" 200 890 "http://example.com" "Mozilla/5.0"`,
		`invalid log line that should be skipped`,
		`192.168.1.4 - - [10/Aug/2025:12:03:00 +0000] "DELETE /api/users/789 HTTP/1.1" 204 0 "-" "curl/7.68.0"`,
	}, "\n")

	err := os.WriteFile(logFile, []byte(logContent), 0644)
	require.NoError(t, err)

	// Process the file
	options := DefaultIngestOptions()
	iterator, err := ingestor.Ingest([]string{logFile}, options)
	require.NoError(t, err)

	// Collect all records
	var records []*NormalizedRecord
	for iterator.Next() {
		records = append(records, iterator.Value())
	}
	assert.NoError(t, iterator.Err())

	// Verify results
	assert.Len(t, records, 4) // 4 valid lines, 1 invalid skipped

	// Check metrics
	metrics := ingestor.Metrics()
	assert.Equal(t, int64(5), metrics.TotalLines)  // 5 total lines
	assert.Equal(t, int64(4), metrics.ParsedLines) // 4 successfully parsed
	assert.Equal(t, int64(1), metrics.ErrorLines)  // 1 error line
	assert.Len(t, metrics.ErrorSamples, 1)
	assert.Contains(t, metrics.ErrorSamples[0], "invalid log line")

	// Verify first record
	record1 := records[0]
	assert.Equal(t, "GET", record1.Method)
	assert.Equal(t, "/api/users/123", record1.Path)
	assert.Equal(t, 200, record1.Status)
	assert.Equal(t, int64(1234), record1.BodyBytes)

	// Verify record with query parameters
	record3 := records[2]
	assert.Equal(t, "GET", record3.Method)
	assert.Equal(t, "/api/users/456", record3.Path)
	assert.Contains(t, record3.Query, "include")
	assert.Equal(t, []string{"profile"}, record3.Query["include"])
}