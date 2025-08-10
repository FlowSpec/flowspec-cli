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

func TestNginxAccessIngestor_Supports(t *testing.T) {
	ingestor := NewNginxAccessIngestor()
	
	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{"access.log", "/var/log/nginx/access.log", true},
		{"access_log", "/var/log/nginx/access_log", true},
		{"nginx.log", "/var/log/nginx.log", true},
		{"nginx_access.log", "/var/log/nginx_access.log", true},
		{"access.log.gz", "/var/log/nginx/access.log.gz", true},
		{"access.log.zst", "/var/log/nginx/access.log.zst", true},
		{"error.log", "/var/log/nginx/error.log", false},
		{"random.txt", "/tmp/random.txt", false},
		{"access.log.1", "/var/log/nginx/access.log.1", true}, // Contains access.log
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ingestor.Supports(tt.filePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNginxAccessIngestor_ParseLogLine_Combined(t *testing.T) {
	ingestor := NewNginxAccessIngestor()
	options := &IngestOptions{
		LogFormat:       "combined",
		SensitiveKeys:   []string{"authorization"},
		RedactionPolicy: "drop",
	}
	
	ingestor.options = options
	err := ingestor.setupRegex()
	require.NoError(t, err)
	
	// Test combined format log line
	logLine := `192.168.1.1 - - [10/Aug/2025:12:34:56 +0000] "GET /api/users/123?include=profile HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"`
	
	record, err := ingestor.parseLogLine(logLine)
	require.NoError(t, err)
	
	assert.Equal(t, "GET", record.Method)
	assert.Equal(t, "/api/users/123", record.Path)
	assert.Equal(t, "/api/users/123?include=profile", record.RawPath)
	assert.Equal(t, 200, record.Status)
	assert.Equal(t, int64(1234), record.BodyBytes)
	assert.Equal(t, "192.168.1.1", record.Host)
	assert.Equal(t, "http", record.Scheme)
	
	// Check query parameters
	assert.Contains(t, record.Query, "include")
	assert.Equal(t, []string{"profile"}, record.Query["include"])
	
	// Check headers
	assert.Contains(t, record.Headers, "referer")
	assert.Equal(t, []string{"http://example.com"}, record.Headers["referer"])
	assert.Contains(t, record.Headers, "user-agent")
	
	// Check timestamp parsing
	expectedTime, _ := time.Parse("02/Jan/2006:15:04:05 -0700", "10/Aug/2025:12:34:56 +0000")
	assert.Equal(t, expectedTime.UTC(), record.Timestamp)
}

func TestNginxAccessIngestor_ParseLogLine_Common(t *testing.T) {
	ingestor := NewNginxAccessIngestor()
	options := &IngestOptions{
		LogFormat: "common",
	}
	
	ingestor.options = options
	err := ingestor.setupRegex()
	require.NoError(t, err)
	
	// Test common format log line
	logLine := `10.0.0.1 - user [25/Dec/2025:10:00:00 +0100] "POST /api/orders HTTP/1.1" 201 567`
	
	record, err := ingestor.parseLogLine(logLine)
	require.NoError(t, err)
	
	assert.Equal(t, "POST", record.Method)
	assert.Equal(t, "/api/orders", record.Path)
	assert.Equal(t, "/api/orders", record.RawPath)
	assert.Equal(t, 201, record.Status)
	assert.Equal(t, int64(567), record.BodyBytes)
	
	// Common format doesn't include referer and user-agent
	assert.Empty(t, record.Headers["referer"])
	assert.Empty(t, record.Headers["user-agent"])
}

func TestNginxAccessIngestor_ParseLogLine_CustomRegex(t *testing.T) {
	ingestor := NewNginxAccessIngestor()
	options := &IngestOptions{
		CustomRegex: `^(\S+) - (\S+) \[([^\]]+)\] "([A-Z]+) ([^"]*) HTTP/[^"]*" (\d+) (\d+)`,
	}
	
	ingestor.options = options
	err := ingestor.setupRegex()
	require.NoError(t, err)
	
	logLine := `127.0.0.1 - - [01/Jan/2025:00:00:00 +0000] "GET /health HTTP/1.1" 200 2`
	
	record, err := ingestor.parseLogLine(logLine)
	require.NoError(t, err)
	
	assert.Equal(t, "GET", record.Method)
	assert.Equal(t, "/health", record.Path)
	assert.Equal(t, 200, record.Status)
}

func TestNginxAccessIngestor_ParseLogLine_InvalidFormat(t *testing.T) {
	ingestor := NewNginxAccessIngestor()
	options := &IngestOptions{
		LogFormat: "combined",
	}
	
	ingestor.options = options
	err := ingestor.setupRegex()
	require.NoError(t, err)
	
	// Invalid log line that doesn't match the pattern
	logLine := `This is not a valid nginx log line`
	
	_, err = ingestor.parseLogLine(logLine)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "line does not match expected format")
}

func TestNginxAccessIngestor_PathNormalization(t *testing.T) {
	ingestor := NewNginxAccessIngestor()
	options := &IngestOptions{
		LogFormat: "combined",
	}
	
	ingestor.options = options
	err := ingestor.setupRegex()
	require.NoError(t, err)
	
	tests := []struct {
		name        string
		requestURI  string
		expectedPath string
	}{
		{
			name:        "simple path",
			requestURI:  "/api/users",
			expectedPath: "/api/users",
		},
		{
			name:        "path with trailing slash",
			requestURI:  "/api/users/",
			expectedPath: "/api/users",
		},
		{
			name:        "path with query string",
			requestURI:  "/api/users?page=1&limit=10",
			expectedPath: "/api/users",
		},
		{
			name:        "path with multiple slashes",
			requestURI:  "/api//users///123",
			expectedPath: "/api/users/123",
		},
		{
			name:        "URL encoded path",
			requestURI:  "/api/users/john%20doe",
			expectedPath: "/api/users/john doe",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logLine := `192.168.1.1 - - [10/Aug/2025:12:34:56 +0000] "GET ` + tt.requestURI + ` HTTP/1.1" 200 1234 "-" "-"`
			
			record, err := ingestor.parseLogLine(logLine)
			require.NoError(t, err)
			
			assert.Equal(t, tt.expectedPath, record.Path)
			assert.Equal(t, tt.requestURI, record.RawPath)
		})
	}
}

func TestNginxAccessIngestor_SensitiveFieldRedaction(t *testing.T) {
	ingestor := NewNginxAccessIngestor()
	
	tests := []struct {
		name            string
		sensitiveKeys   []string
		redactionPolicy string
		logLine         string
		expectHeader    bool
	}{
		{
			name:            "drop sensitive headers",
			sensitiveKeys:   []string{"token"},
			redactionPolicy: "drop",
			logLine:         `192.168.1.1 - - [10/Aug/2025:12:34:56 +0000] "GET /api/users?token=secret HTTP/1.1" 200 1234 "-" "-"`,
			expectHeader:    false,
		},
		{
			name:            "mask sensitive headers",
			sensitiveKeys:   []string{"user-agent"},
			redactionPolicy: "mask",
			logLine:         `192.168.1.1 - - [10/Aug/2025:12:34:56 +0000] "GET /api/users HTTP/1.1" 200 1234 "-" "Mozilla/5.0"`,
			expectHeader:    true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &IngestOptions{
				LogFormat:       "combined",
				SensitiveKeys:   tt.sensitiveKeys,
				RedactionPolicy: tt.redactionPolicy,
			}
			
			ingestor.options = options
			err := ingestor.setupRegex()
			require.NoError(t, err)
			
			record, err := ingestor.parseLogLine(tt.logLine)
			require.NoError(t, err)
			
			if tt.redactionPolicy == "drop" {
				// Check that sensitive query parameters are dropped
				assert.NotContains(t, record.Query, "token")
			} else if tt.redactionPolicy == "mask" && tt.expectHeader {
				// Check that sensitive headers are masked
				if userAgent, exists := record.Headers["user-agent"]; exists {
					assert.Equal(t, []string{"***"}, userAgent)
				}
			}
		})
	}
}

func TestNginxAccessIngestor_UnsupportedFormat(t *testing.T) {
	ingestor := NewNginxAccessIngestor()
	options := &IngestOptions{
		LogFormat: "unsupported",
	}
	
	ingestor.options = options
	err := ingestor.setupRegex()
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported log format")
	assert.Contains(t, err.Error(), "Supported formats:")
	assert.Contains(t, err.Error(), "combined")
	assert.Contains(t, err.Error(), "common")
	assert.Contains(t, err.Error(), "Example log lines:")
}

func TestNginxAccessIngestor_TimeFilter(t *testing.T) {
	ingestor := NewNginxAccessIngestor()
	
	// Create test time range
	since, _ := time.Parse(time.RFC3339, "2025-08-10T12:00:00Z")
	until, _ := time.Parse(time.RFC3339, "2025-08-10T13:00:00Z")
	
	options := &IngestOptions{
		LogFormat: "combined",
		TimeFilter: &TimeRange{
			Since: &since,
			Until: &until,
		},
	}
	
	ingestor.options = options
	
	tests := []struct {
		name      string
		timestamp time.Time
		expected  bool
	}{
		{
			name:      "before range",
			timestamp: time.Date(2025, 8, 10, 11, 30, 0, 0, time.UTC),
			expected:  false,
		},
		{
			name:      "within range",
			timestamp: time.Date(2025, 8, 10, 12, 30, 0, 0, time.UTC),
			expected:  true,
		},
		{
			name:      "after range",
			timestamp: time.Date(2025, 8, 10, 14, 0, 0, 0, time.UTC),
			expected:  false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ingestor.isWithinTimeRange(tt.timestamp)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNginxAccessIngestor_CompressedFiles(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "nginx_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// Test data
	logData := `192.168.1.1 - - [10/Aug/2025:12:34:56 +0000] "GET /api/test HTTP/1.1" 200 100 "-" "-"`
	
	// Test gzip compression
	t.Run("gzip compression", func(t *testing.T) {
		gzipFile := filepath.Join(tempDir, "access.log.gz")
		
		// Create gzipped file
		file, err := os.Create(gzipFile)
		require.NoError(t, err)
		
		gzWriter := gzip.NewWriter(file)
		_, err = gzWriter.Write([]byte(logData))
		require.NoError(t, err)
		gzWriter.Close()
		file.Close()
		
		// Test reading
		ingestor := NewNginxAccessIngestor()
		file, err = os.Open(gzipFile)
		require.NoError(t, err)
		defer file.Close()
		
		reader, err := ingestor.createReader(file, gzipFile)
		require.NoError(t, err)
		defer reader.Close()
		
		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, logData, string(data))
	})
	
	// Test zstd compression
	t.Run("zstd compression", func(t *testing.T) {
		zstFile := filepath.Join(tempDir, "access.log.zst")
		
		// Create zstd compressed file
		file, err := os.Create(zstFile)
		require.NoError(t, err)
		
		zstWriter, err := zstd.NewWriter(file)
		require.NoError(t, err)
		_, err = zstWriter.Write([]byte(logData))
		require.NoError(t, err)
		zstWriter.Close()
		file.Close()
		
		// Test reading
		ingestor := NewNginxAccessIngestor()
		file, err = os.Open(zstFile)
		require.NoError(t, err)
		defer file.Close()
		
		reader, err := ingestor.createReader(file, zstFile)
		require.NoError(t, err)
		defer reader.Close()
		
		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, logData, string(data))
	})
	
	// Test uncompressed file
	t.Run("uncompressed file", func(t *testing.T) {
		plainFile := filepath.Join(tempDir, "access.log")
		
		err := os.WriteFile(plainFile, []byte(logData), 0644)
		require.NoError(t, err)
		
		ingestor := NewNginxAccessIngestor()
		file, err := os.Open(plainFile)
		require.NoError(t, err)
		defer file.Close()
		
		reader, err := ingestor.createReader(file, plainFile)
		require.NoError(t, err)
		defer reader.Close()
		
		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, logData, string(data))
	})
}

func TestNginxAccessIngestor_Integration(t *testing.T) {
	// Create temporary log file
	tempDir, err := os.MkdirTemp("", "nginx_integration_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	logFile := filepath.Join(tempDir, "access.log")
	logContent := strings.Join([]string{
		`192.168.1.1 - - [10/Aug/2025:12:34:56 +0000] "GET /api/users/123 HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`,
		`192.168.1.2 - user [10/Aug/2025:12:35:00 +0000] "POST /api/orders HTTP/1.1" 201 567 "-" "curl/7.68.0"`,
		`192.168.1.3 - - [10/Aug/2025:12:35:30 +0000] "GET /api/products?category=electronics HTTP/1.1" 200 2048 "http://shop.com" "Chrome/91.0"`,
		`invalid log line that should be skipped`,
		`192.168.1.4 - - [10/Aug/2025:12:36:00 +0000] "DELETE /api/users/456 HTTP/1.1" 204 0 "-" "-"`,
	}, "\n")
	
	err = os.WriteFile(logFile, []byte(logContent), 0644)
	require.NoError(t, err)
	
	// Test ingestion
	ingestor := NewNginxAccessIngestor()
	options := DefaultIngestOptions()
	
	iterator, err := ingestor.Ingest([]string{logFile}, options)
	require.NoError(t, err)
	defer iterator.Close()
	
	// Collect all records
	var records []*NormalizedRecord
	for iterator.Next() {
		records = append(records, iterator.Value())
	}
	
	require.NoError(t, iterator.Err())
	
	// Verify results
	assert.Len(t, records, 4) // 4 valid lines, 1 invalid should be skipped
	
	// Check first record
	assert.Equal(t, "GET", records[0].Method)
	assert.Equal(t, "/api/users/123", records[0].Path)
	assert.Equal(t, 200, records[0].Status)
	
	// Check second record
	assert.Equal(t, "POST", records[1].Method)
	assert.Equal(t, "/api/orders", records[1].Path)
	assert.Equal(t, 201, records[1].Status)
	
	// Check third record with query parameters
	assert.Equal(t, "GET", records[2].Method)
	assert.Equal(t, "/api/products", records[2].Path)
	assert.Contains(t, records[2].Query, "category")
	assert.Equal(t, []string{"electronics"}, records[2].Query["category"])
	
	// Check fourth record
	assert.Equal(t, "DELETE", records[3].Method)
	assert.Equal(t, "/api/users/456", records[3].Path)
	assert.Equal(t, 204, records[3].Status)
	
	// Check metrics
	metrics := ingestor.Metrics()
	assert.Equal(t, int64(5), metrics.TotalLines)  // 5 total lines
	assert.Equal(t, int64(4), metrics.ParsedLines) // 4 successfully parsed
	assert.Equal(t, int64(1), metrics.ErrorLines)  // 1 error line
	assert.Len(t, metrics.ErrorSamples, 1)         // 1 error sample collected
	assert.Contains(t, metrics.ErrorSamples[0], "invalid log line")
}