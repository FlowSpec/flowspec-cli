package traffic

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/flowspec/flowspec-cli/internal/ingestor"
	"github.com/klauspost/compress/zstd"
)

// NginxAccessIngestor implements TrafficIngestor for Nginx access logs
type NginxAccessIngestor struct {
	metrics     *IngestMetrics
	options     *IngestOptions
	regex       *regexp.Regexp
	logFormat   string
	timeLayout  string
}

// Predefined Nginx log formats with their corresponding regex patterns
var nginxLogFormats = map[string]struct {
	regex      string
	timeLayout string
}{
	"combined": {
		// Combined log format: $remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent"
		regex:      `^(\S+) - (\S+) \[([^\]]+)\] "([A-Z]+) ([^"]*) HTTP/[^"]*" (\d+) (\d+) "([^"]*)" "([^"]*)"`,
		timeLayout: "02/Jan/2006:15:04:05 -0700",
	},
	"common": {
		// Common log format: $remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent
		regex:      `^(\S+) - (\S+) \[([^\]]+)\] "([A-Z]+) ([^"]*) HTTP/[^"]*" (\d+) (\d+)`,
		timeLayout: "02/Jan/2006:15:04:05 -0700",
	},
}

// NewNginxAccessIngestor creates a new Nginx access log ingestor
func NewNginxAccessIngestor() *NginxAccessIngestor {
	return &NginxAccessIngestor{
		metrics: NewIngestMetrics(),
	}
}

// Supports checks if the ingestor can handle the given file path
func (n *NginxAccessIngestor) Supports(filePath string) bool {
	// Support common Nginx access log file patterns
	filename := strings.ToLower(filepath.Base(filePath))
	
	// Check for common access log naming patterns
	accessLogPatterns := []string{
		"access.log",
		"access_log",
		"nginx.log",
		"nginx_access.log",
	}
	
	for _, pattern := range accessLogPatterns {
		if strings.Contains(filename, pattern) {
			return true
		}
	}
	
	// Also support compressed versions
	if strings.HasSuffix(filename, ".gz") || strings.HasSuffix(filename, ".zst") {
		baseFilename := strings.TrimSuffix(strings.TrimSuffix(filename, ".gz"), ".zst")
		for _, pattern := range accessLogPatterns {
			if strings.Contains(baseFilename, pattern) {
				return true
			}
		}
	}
	
	return false
}

// Ingest processes the input files and returns an iterator of normalized records
func (n *NginxAccessIngestor) Ingest(inputs []string, options *IngestOptions) (ingestor.Iterator[*NormalizedRecord], error) {
	if options == nil {
		options = DefaultIngestOptions()
	}
	
	n.options = options
	n.metrics = NewIngestMetrics()
	
	// Setup regex pattern
	if err := n.setupRegex(); err != nil {
		return nil, fmt.Errorf("failed to setup regex pattern: %w", err)
	}
	
	// Create channel iterator with backpressure control
	iterator, dataCh, errCh := ingestor.NewChannelIterator[*NormalizedRecord](1000)
	
	// Start processing in a goroutine
	go n.processFiles(inputs, dataCh, errCh)
	
	return iterator, nil
}

// setupRegex configures the regex pattern based on options
func (n *NginxAccessIngestor) setupRegex() error {
	var regexPattern string
	var timeLayout string
	
	// Use custom regex if provided
	if n.options.CustomRegex != "" {
		regexPattern = n.options.CustomRegex
		timeLayout = "02/Jan/2006:15:04:05 -0700" // Default time layout
		n.logFormat = "custom"
	} else {
		// Use predefined format
		format, exists := nginxLogFormats[n.options.LogFormat]
		if !exists {
			return n.createFormatError()
		}
		regexPattern = format.regex
		timeLayout = format.timeLayout
		n.logFormat = n.options.LogFormat
	}
	
	// Compile regex
	var err error
	n.regex, err = regexp.Compile(regexPattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %w", err)
	}
	
	n.timeLayout = timeLayout
	return nil
}

// createFormatError creates a detailed error message for unsupported formats
func (n *NginxAccessIngestor) createFormatError() error {
	supportedFormats := make([]string, 0, len(nginxLogFormats))
	for format := range nginxLogFormats {
		supportedFormats = append(supportedFormats, format)
	}
	
	return fmt.Errorf(`unsupported log format: "%s"

Supported formats: %s

Example log lines:
  combined: 192.168.1.1 - - [10/Aug/2025:12:00:00 +0000] "GET /api/users/123 HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"
  common:   192.168.1.1 - - [10/Aug/2025:12:00:00 +0000] "GET /api/users/123 HTTP/1.1" 200 1234

To use a custom format, specify --regex with your own regular expression pattern.
The regex should capture groups in this order: remote_addr, remote_user, time_local, method, request_uri, status, body_bytes_sent, [referer], [user_agent]`,
		n.options.LogFormat, strings.Join(supportedFormats, ", "))
}

// processFiles processes all input files and sends records to the channel
func (n *NginxAccessIngestor) processFiles(inputs []string, dataCh chan<- *NormalizedRecord, errCh chan<- error) {
	defer close(dataCh)
	
	startTime := time.Now()
	
	for _, input := range inputs {
		if err := n.processFile(input, dataCh); err != nil {
			errCh <- fmt.Errorf("failed to process file %s: %w", input, err)
			return
		}
	}
	
	n.metrics.SetDuration(time.Since(startTime))
}

// processFile processes a single file
func (n *NginxAccessIngestor) processFile(filePath string, dataCh chan<- *NormalizedRecord) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	// Create reader with compression support
	reader, err := n.createReader(file, filePath)
	if err != nil {
		return fmt.Errorf("failed to create reader: %w", err)
	}
	defer reader.Close()
	
	scanner := bufio.NewScanner(reader)
	
	// Set a larger buffer for long log lines
	const maxCapacity = 1024 * 1024 // 1MB
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, maxCapacity)
	
	for scanner.Scan() {
		line := scanner.Text()
		n.metrics.AddTotal()
		
		// Apply sampling if configured
		if n.options.SampleRate < 1.0 && n.shouldSkipLine() {
			continue
		}
		
		record, err := n.parseLogLine(line)
		if err != nil {
			n.metrics.AddError(line, n.options.MaxErrorSamples)
			continue
		}
		
		// Apply time filter if configured
		if n.options.TimeFilter != nil && !n.isWithinTimeRange(record.Timestamp) {
			continue
		}
		
		n.metrics.AddParsed()
		
		// Send record to channel (with context cancellation support)
		select {
		case dataCh <- record:
		case <-context.Background().Done():
			return context.Background().Err()
		}
	}
	
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}
	
	return nil
}

// createReader creates an appropriate reader based on file extension
func (n *NginxAccessIngestor) createReader(file *os.File, filePath string) (io.ReadCloser, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	
	switch ext {
	case ".gz":
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		return gzReader, nil
		
	case ".zst":
		zstReader, err := zstd.NewReader(file)
		if err != nil {
			return nil, fmt.Errorf("failed to create zstd reader: %w", err)
		}
		return io.NopCloser(zstReader), nil
		
	default:
		return io.NopCloser(file), nil
	}
}

// shouldSkipLine determines if a line should be skipped based on sampling rate
func (n *NginxAccessIngestor) shouldSkipLine() bool {
	// Simple sampling based on line count
	// In a real implementation, you might want to use a more sophisticated approach
	return float64(n.metrics.TotalLines%100)/100.0 >= n.options.SampleRate
}

// isWithinTimeRange checks if a timestamp is within the configured time range
func (n *NginxAccessIngestor) isWithinTimeRange(timestamp time.Time) bool {
	if n.options.TimeFilter == nil {
		return true
	}
	if n.options.TimeFilter.Since != nil && timestamp.Before(*n.options.TimeFilter.Since) {
		return false
	}
	if n.options.TimeFilter.Until != nil && timestamp.After(*n.options.TimeFilter.Until) {
		return false
	}
	return true
}

// parseLogLine parses a single log line into a NormalizedRecord
func (n *NginxAccessIngestor) parseLogLine(line string) (*NormalizedRecord, error) {
	matches := n.regex.FindStringSubmatch(line)
	if matches == nil {
		return nil, fmt.Errorf("line does not match expected format")
	}
	
	// Extract fields based on the regex groups
	// The exact mapping depends on the regex pattern, but we'll handle the common cases
	
	var (
		remoteAddr   string
		timeLocal    string
		method       string
		requestURI   string
		status       string
		bodyBytes    string
		referer      string
		userAgent    string
	)
	
	// Map regex groups to fields (this assumes the standard nginx formats)
	if len(matches) >= 7 {
		remoteAddr = matches[1]
		// remoteUser = matches[2] // Not currently used, but available for future enhancement
		timeLocal = matches[3]
		method = matches[4]
		requestURI = matches[5]
		status = matches[6]
		bodyBytes = matches[7]
		
		// Additional fields for combined format
		if len(matches) >= 9 {
			referer = matches[8]
			userAgent = matches[9]
		}
	} else {
		return nil, fmt.Errorf("insufficient regex groups captured")
	}
	
	// Parse timestamp
	timestamp, err := n.parseTimestamp(timeLocal)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %w", err)
	}
	
	// Parse status code
	statusCode, err := strconv.Atoi(status)
	if err != nil {
		return nil, fmt.Errorf("invalid status code: %w", err)
	}
	
	// Parse body bytes
	bodyBytesInt, err := strconv.ParseInt(bodyBytes, 10, 64)
	if err != nil {
		// Some logs use "-" for missing body bytes
		if bodyBytes == "-" {
			bodyBytesInt = 0
		} else {
			return nil, fmt.Errorf("invalid body bytes: %w", err)
		}
	}
	
	// Extract query string from request URI
	queryString := ExtractQueryString(requestURI)
	
	// Create headers map from available data
	headers := make(map[string]string)
	if referer != "" && referer != "-" {
		headers["referer"] = referer
	}
	if userAgent != "" && userAgent != "-" {
		headers["user-agent"] = userAgent
	}
	
	// Create the normalized record
	record := &NormalizedRecord{
		Method:    strings.ToUpper(method),
		Path:      NormalizePath(requestURI),
		RawPath:   requestURI,
		Status:    statusCode,
		Timestamp: timestamp,
		Query:     NormalizeQuery(queryString),
		Headers:   NormalizeHeaders(headers),
		Host:      remoteAddr, // Using remote addr as host for now
		Scheme:    "http",     // Default to http, could be enhanced to detect https
		BodyBytes: bodyBytesInt,
	}
	
	// Apply redaction policy
	record.Headers, record.Query = ApplyRedactionPolicy(
		record.Headers,
		record.Query,
		n.options.SensitiveKeys,
		n.options.RedactionPolicy,
	)
	
	return record, nil
}

// parseTimestamp parses the timestamp from the log line and converts it to RFC3339
func (n *NginxAccessIngestor) parseTimestamp(timeStr string) (time.Time, error) {
	// Parse using the configured time layout
	parsedTime, err := time.Parse(n.timeLayout, timeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse time '%s' with layout '%s': %w", timeStr, n.timeLayout, err)
	}
	
	// Convert to UTC for consistency
	return parsedTime.UTC(), nil
}

// Metrics returns the current ingestion metrics
func (n *NginxAccessIngestor) Metrics() *IngestMetrics {
	return n.metrics
}

// Close releases any resources held by the ingestor
func (n *NginxAccessIngestor) Close() error {
	// No resources to clean up for this implementation
	return nil
}