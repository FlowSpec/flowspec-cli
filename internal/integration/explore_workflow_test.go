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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/flowspec/flowspec-cli/internal/engine"
	"github.com/flowspec/flowspec-cli/internal/ingestor/traffic"
	"github.com/flowspec/flowspec-cli/internal/models"
	"github.com/flowspec/flowspec-cli/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExploreWorkflow_EndToEnd tests the complete explore workflow:
// Nginx logs → NormalizedRecords → ServiceSpec → YAML → Validation
func TestExploreWorkflow_EndToEnd(t *testing.T) {
	// Step 1: Create test Nginx log file
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "access.log")
	
	logContent := strings.Join([]string{
		// User API endpoints
		`192.168.1.1 - - [10/Aug/2025:12:00:00 +0000] "GET /api/users/123 HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`,
		`192.168.1.2 - - [10/Aug/2025:12:01:00 +0000] "GET /api/users/456 HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`,
		`192.168.1.3 - - [10/Aug/2025:12:02:00 +0000] "GET /api/users/789 HTTP/1.1" 404 567 "http://example.com" "Mozilla/5.0"`,
		`192.168.1.4 - - [10/Aug/2025:12:03:00 +0000] "POST /api/users HTTP/1.1" 201 890 "http://example.com" "curl/7.68.0"`,
		`192.168.1.5 - - [10/Aug/2025:12:04:00 +0000] "PUT /api/users/123 HTTP/1.1" 200 456 "http://example.com" "curl/7.68.0"`,
		
		// Posts API endpoints
		`192.168.1.6 - - [10/Aug/2025:12:05:00 +0000] "GET /api/posts?limit=10&offset=0 HTTP/1.1" 200 2345 "http://example.com" "Mozilla/5.0"`,
		`192.168.1.7 - - [10/Aug/2025:12:06:00 +0000] "GET /api/posts?limit=20&offset=10 HTTP/1.1" 200 2345 "http://example.com" "Mozilla/5.0"`,
		`192.168.1.8 - - [10/Aug/2025:12:07:00 +0000] "POST /api/posts HTTP/1.1" 201 1123 "http://example.com" "curl/7.68.0"`,
		
		// Health check
		`192.168.1.9 - - [10/Aug/2025:12:08:00 +0000] "GET /health HTTP/1.1" 200 45 "-" "kube-probe/1.0"`,
		`192.168.1.10 - - [10/Aug/2025:12:09:00 +0000] "GET /health HTTP/1.1" 200 45 "-" "kube-probe/1.0"`,
	}, "\n")
	
	err := os.WriteFile(logFile, []byte(logContent), 0644)
	require.NoError(t, err)
	
	// Step 2: Ingest traffic logs
	ingestor := traffic.NewNginxAccessIngestor()
	options := traffic.DefaultIngestOptions()
	
	iterator, err := ingestor.Ingest([]string{logFile}, options)
	require.NoError(t, err)
	
	// Step 3: Generate contract from traffic
	generator := engine.NewContractGeneratorLite()
	generationOptions := &engine.GenerationOptions{
		PathClusteringThreshold: 0.8,
		MinSampleSize:          2,
		RequiredFieldThreshold: 0.8,
		MinEndpointSamples:     2,
		StatusAggregation:      "auto",
		ServiceName:            "test-api",
		ServiceVersion:         "v1.0.0",
	}
	generator.SetOptions(generationOptions)
	
	spec, err := generator.GenerateSpec(iterator)
	require.NoError(t, err)
	require.NotNil(t, spec)
	
	// Step 4: Verify generated contract structure
	assert.Equal(t, "flowspec/v1alpha1", spec.APIVersion)
	assert.Equal(t, "ServiceSpec", spec.Kind)
	assert.Equal(t, "test-api", spec.Metadata.Name)
	assert.Equal(t, "v1.0.0", spec.Metadata.Version)
	assert.Greater(t, len(spec.Spec.Endpoints), 0)
	
	// Step 5: Write contract to YAML file
	yamlFile := filepath.Join(tmpDir, "service-spec.yaml")
	yamlContent := convertSpecToYAML(t, spec)
	err = os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	require.NoError(t, err)
	
	// Step 6: Parse the generated YAML back
	yamlParser := parser.NewYAMLFileParser()
	parsedSpecs, parseErrors := yamlParser.ParseFile(yamlFile)
	
	assert.Empty(t, parseErrors, "Generated YAML should be valid")
	require.Len(t, parsedSpecs, 1)
	
	parsedSpec := parsedSpecs[0]
	assert.Equal(t, spec.APIVersion, parsedSpec.APIVersion)
	assert.Equal(t, spec.Kind, parsedSpec.Kind)
	assert.Equal(t, spec.Metadata.Name, parsedSpec.Metadata.Name)
	
	// Step 7: Verify metrics
	metrics := ingestor.Metrics()
	assert.Equal(t, int64(10), metrics.TotalLines)
	assert.Equal(t, int64(10), metrics.ParsedLines)
	assert.Equal(t, int64(0), metrics.ErrorLines)
	assert.False(t, metrics.IsIncomplete())
}

// TestExploreWorkflow_WithErrors tests the explore workflow with parsing errors
func TestExploreWorkflow_WithErrors(t *testing.T) {
	// Create test log file with some invalid lines
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "access_with_errors.log")
	
	logContent := strings.Join([]string{
		// Valid lines
		`192.168.1.1 - - [10/Aug/2025:12:00:00 +0000] "GET /api/users/123 HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`,
		`192.168.1.2 - - [10/Aug/2025:12:01:00 +0000] "GET /api/users/456 HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`,
		
		// Invalid lines
		`invalid log line format`,
		`another invalid line without proper structure`,
		`192.168.1.3 - - [invalid-timestamp] "GET /api/test HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`,
		
		// More valid lines
		`192.168.1.4 - - [10/Aug/2025:12:02:00 +0000] "POST /api/users HTTP/1.1" 201 890 "http://example.com" "curl/7.68.0"`,
		`192.168.1.5 - - [10/Aug/2025:12:03:00 +0000] "GET /health HTTP/1.1" 200 45 "-" "kube-probe/1.0"`,
	}, "\n")
	
	err := os.WriteFile(logFile, []byte(logContent), 0644)
	require.NoError(t, err)
	
	// Ingest with errors
	ingestor := traffic.NewNginxAccessIngestor()
	options := traffic.DefaultIngestOptions()
	
	iterator, err := ingestor.Ingest([]string{logFile}, options)
	require.NoError(t, err)
	
	// Collect records
	var records []*traffic.NormalizedRecord
	for iterator.Next() {
		records = append(records, iterator.Value())
	}
	assert.NoError(t, iterator.Err())
	
	// Should have parsed only valid lines
	assert.Len(t, records, 4) // 4 valid lines out of 7 total
	
	// Check metrics
	metrics := ingestor.Metrics()
	assert.Equal(t, int64(7), metrics.TotalLines)
	assert.Equal(t, int64(4), metrics.ParsedLines)
	assert.Equal(t, int64(3), metrics.ErrorLines)
	assert.Len(t, metrics.ErrorSamples, 3) // Should have collected error samples
	
	// Error rate should be high enough to mark as incomplete
	assert.True(t, metrics.IsIncomplete()) // 3/7 = ~43% error rate > 10% threshold
}

// TestExploreWorkflow_EmptyLogs tests handling of empty log files
func TestExploreWorkflow_EmptyLogs(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "empty.log")
	
	// Create empty log file
	err := os.WriteFile(logFile, []byte(""), 0644)
	require.NoError(t, err)
	
	// Ingest empty file
	ingestor := traffic.NewNginxAccessIngestor()
	options := traffic.DefaultIngestOptions()
	
	iterator, err := ingestor.Ingest([]string{logFile}, options)
	require.NoError(t, err)
	
	// Generate contract from empty data
	generator := engine.NewContractGeneratorLite()
	spec, err := generator.GenerateSpec(iterator)
	
	require.NoError(t, err)
	require.NotNil(t, spec)
	
	// Should generate empty contract
	assert.Equal(t, "flowspec/v1alpha1", spec.APIVersion)
	assert.Equal(t, "ServiceSpec", spec.Kind)
	assert.Empty(t, spec.Spec.Endpoints)
	
	// Check metrics
	metrics := ingestor.Metrics()
	assert.Equal(t, int64(0), metrics.TotalLines)
	assert.Equal(t, int64(0), metrics.ParsedLines)
	assert.Equal(t, int64(0), metrics.ErrorLines)
	assert.False(t, metrics.IsIncomplete())
}

// TestExploreWorkflow_CompressedLogs tests handling of compressed log files
func TestExploreWorkflow_CompressedLogs(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create compressed log file
	logContent := strings.Join([]string{
		`192.168.1.1 - - [10/Aug/2025:12:00:00 +0000] "GET /api/users/123 HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`,
		`192.168.1.2 - - [10/Aug/2025:12:01:00 +0000] "POST /api/users HTTP/1.1" 201 890 "http://example.com" "curl/7.68.0"`,
		`192.168.1.3 - - [10/Aug/2025:12:02:00 +0000] "GET /health HTTP/1.1" 200 45 "-" "kube-probe/1.0"`,
	}, "\n")
	
	// Create gzipped log file
	gzipFile := filepath.Join(tmpDir, "access.log.gz")
	createGzipFile(t, gzipFile, logContent)
	
	// Ingest compressed file
	ingestor := traffic.NewNginxAccessIngestor()
	options := traffic.DefaultIngestOptions()
	
	iterator, err := ingestor.Ingest([]string{gzipFile}, options)
	require.NoError(t, err)
	
	// Collect records
	var records []*traffic.NormalizedRecord
	for iterator.Next() {
		records = append(records, iterator.Value())
	}
	assert.NoError(t, iterator.Err())
	
	// Should parse all records from compressed file
	assert.Len(t, records, 3)
	
	// Verify record content
	assert.Equal(t, "GET", records[0].Method)
	assert.Equal(t, "/api/users/123", records[0].Path)
	assert.Equal(t, 200, records[0].Status)
	
	assert.Equal(t, "POST", records[1].Method)
	assert.Equal(t, "/api/users", records[1].Path)
	assert.Equal(t, 201, records[1].Status)
}

// TestExploreWorkflow_TimeFiltering tests time-based filtering
func TestExploreWorkflow_TimeFiltering(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "access.log")
	
	logContent := strings.Join([]string{
		// Earlier logs (should be filtered out)
		`192.168.1.1 - - [10/Aug/2025:10:00:00 +0000] "GET /api/old/123 HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`,
		`192.168.1.2 - - [10/Aug/2025:11:00:00 +0000] "GET /api/old/456 HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`,
		
		// Target time range logs (should be included)
		`192.168.1.3 - - [10/Aug/2025:12:00:00 +0000] "GET /api/users/123 HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`,
		`192.168.1.4 - - [10/Aug/2025:12:30:00 +0000] "POST /api/users HTTP/1.1" 201 890 "http://example.com" "curl/7.68.0"`,
		`192.168.1.5 - - [10/Aug/2025:13:00:00 +0000] "GET /health HTTP/1.1" 200 45 "-" "kube-probe/1.0"`,
		
		// Later logs (should be filtered out)
		`192.168.1.6 - - [10/Aug/2025:14:00:00 +0000] "GET /api/future/789 HTTP/1.1" 200 1234 "http://example.com" "Mozilla/5.0"`,
	}, "\n")
	
	err := os.WriteFile(logFile, []byte(logContent), 0644)
	require.NoError(t, err)
	
	// Set up time filtering (12:00 to 13:30)
	since := time.Date(2025, 8, 10, 12, 0, 0, 0, time.UTC)
	until := time.Date(2025, 8, 10, 13, 30, 0, 0, time.UTC)
	
	options := &traffic.IngestOptions{
		LogFormat: "combined",
		TimeFilter: &traffic.TimeRange{
			Since: &since,
			Until: &until,
		},
		SampleRate:        1.0,
		RedactionPolicy:   "drop",
		MaxErrorSamples:   10,
		SensitiveKeys:     []string{},
	}
	
	// Ingest with time filtering
	ingestor := traffic.NewNginxAccessIngestor()
	iterator, err := ingestor.Ingest([]string{logFile}, options)
	require.NoError(t, err)
	
	// Collect records
	var records []*traffic.NormalizedRecord
	for iterator.Next() {
		records = append(records, iterator.Value())
	}
	assert.NoError(t, iterator.Err())
	
	// Should only have records within time range
	assert.Len(t, records, 3) // Only the middle 3 records
	
	// Verify timestamps are within range
	for _, record := range records {
		assert.True(t, record.Timestamp.After(since) || record.Timestamp.Equal(since))
		assert.True(t, record.Timestamp.Before(until) || record.Timestamp.Equal(until))
	}
}

// Helper functions

func convertSpecToYAML(t *testing.T, spec *models.ServiceSpec) string {
	// This is a simplified YAML conversion for testing
	// In a real implementation, you'd use a proper YAML marshaler
	yamlContent := fmt.Sprintf(`apiVersion: %s
kind: %s
metadata:
  name: %s
  version: %s
spec:
  endpoints:`, spec.APIVersion, spec.Kind, spec.Metadata.Name, spec.Metadata.Version)
	
	for _, endpoint := range spec.Spec.Endpoints {
		yamlContent += fmt.Sprintf(`
    - path: %s
      operations:`, endpoint.Path)
		
		for _, operation := range endpoint.Operations {
			yamlContent += fmt.Sprintf(`
        - method: %s
          responses:`, operation.Method)
			
			if len(operation.Responses.StatusCodes) > 0 {
				yamlContent += fmt.Sprintf(`
            statusCodes: %v`, operation.Responses.StatusCodes)
			}
			
			if len(operation.Responses.StatusRanges) > 0 {
				yamlContent += fmt.Sprintf(`
            statusRanges: %v`, operation.Responses.StatusRanges)
			}
			
			if operation.Responses.Aggregation != "" {
				yamlContent += fmt.Sprintf(`
            aggregation: %s`, operation.Responses.Aggregation)
			}
		}
	}
	
	return yamlContent
}

func createGzipFile(t *testing.T, filename, content string) {
	file, err := os.Create(filename)
	require.NoError(t, err)
	defer file.Close()
	
	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()
	
	_, err = gzWriter.Write([]byte(content))
	require.NoError(t, err)
}