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

package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/flowspec/flowspec-cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultTraceFileParser_CanParse(t *testing.T) {
	parser := NewTraceFileParser()

	tests := []struct {
		filename string
		expected bool
	}{
		{"trace.json", true},
		{"trace.JSON", true},
		{"flowspec-trace.json", true},
		{"otlp-trace.json", true},
		{"trace.xml", false},
		{"trace.txt", false},
		{"trace", false},
	}

	for _, test := range tests {
		t.Run(test.filename, func(t *testing.T) {
			result := parser.CanParse(test.filename)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestDefaultTraceFileParser_ParseFile_FlowSpecTrace(t *testing.T) {
	parser := NewTraceFileParser()

	// Create a valid FlowSpec trace file
	traceContent := `{
  "format": "flowspec",
  "traceId": "test-trace-123",
  "spans": {
    "span-1": {
      "spanId": "span-1",
      "traceId": "test-trace-123",
      "name": "test-operation",
      "startTime": 1640995200000000000,
      "endTime": 1640995201000000000,
      "status": {
        "code": "OK",
        "message": ""
      },
      "attributes": {
        "http.method": "GET",
        "http.url": "/api/test"
      },
      "events": []
    },
    "span-2": {
      "spanId": "span-2",
      "traceId": "test-trace-123",
      "parentSpanId": "span-1",
      "name": "child-operation",
      "startTime": 1640995200500000000,
      "endTime": 1640995200800000000,
      "status": {
        "code": "OK",
        "message": ""
      },
      "attributes": {},
      "events": []
    }
  }
}`

	tmpDir := t.TempDir()
	traceFile := filepath.Join(tmpDir, "flowspec-trace.json")
	err := os.WriteFile(traceFile, []byte(traceContent), 0644)
	require.NoError(t, err)

	// Parse the file
	traceData, err := parser.ParseFile(traceFile)

	// Verify results
	require.NoError(t, err)
	assert.NotNil(t, traceData)
	assert.Equal(t, "test-trace-123", traceData.TraceID)
	assert.Len(t, traceData.Spans, 2)

	// Verify spans
	span1 := traceData.Spans["span-1"]
	assert.NotNil(t, span1)
	assert.Equal(t, "span-1", span1.SpanID)
	assert.Equal(t, "test-trace-123", span1.TraceID)
	assert.Equal(t, "test-operation", span1.Name)
	assert.Equal(t, int64(1640995200000000000), span1.StartTime)
	assert.Equal(t, int64(1640995201000000000), span1.EndTime)
	assert.Equal(t, "OK", span1.Status.Code)

	span2 := traceData.Spans["span-2"]
	assert.NotNil(t, span2)
	assert.Equal(t, "span-2", span2.SpanID)
	assert.Equal(t, "test-trace-123", span2.TraceID)
	assert.Equal(t, "span-1", span2.ParentID)
	assert.Equal(t, "child-operation", span2.Name)

	// Verify span tree was built
	assert.NotNil(t, traceData.SpanTree)
	assert.NotNil(t, traceData.RootSpan)
	assert.Equal(t, "span-1", traceData.RootSpan.SpanID)
}

func TestDefaultTraceFileParser_ParseFile_InvalidJSON(t *testing.T) {
	parser := NewTraceFileParser()

	// Create an invalid JSON file
	invalidContent := `{
  "traceId": "test-trace-123",
  "spans": {
    "span-1": {
      "spanId": "span-1"
      // Missing comma - invalid JSON
    }
  }
}`

	tmpDir := t.TempDir()
	traceFile := filepath.Join(tmpDir, "invalid.json")
	err := os.WriteFile(traceFile, []byte(invalidContent), 0644)
	require.NoError(t, err)

	// Parse the file
	traceData, err := parser.ParseFile(traceFile)

	// Verify error
	assert.Error(t, err)
	assert.Nil(t, traceData)
	assert.Contains(t, err.Error(), "invalid JSON format")
}

func TestDefaultTraceFileParser_ParseFile_UnsupportedFormat(t *testing.T) {
	parser := NewTraceFileParser()

	// Create a JSON file with unsupported format
	unsupportedContent := `{
  "version": "1.0",
  "data": {
    "some": "unknown format"
  }
}`

	tmpDir := t.TempDir()
	traceFile := filepath.Join(tmpDir, "unsupported.json")
	err := os.WriteFile(traceFile, []byte(unsupportedContent), 0644)
	require.NoError(t, err)

	// Parse the file
	traceData, err := parser.ParseFile(traceFile)

	// Verify error
	assert.Error(t, err)
	assert.Nil(t, traceData)
	assert.Contains(t, err.Error(), "unsupported trace format")
	assert.Contains(t, err.Error(), "Supported formats:")
	assert.Contains(t, err.Error(), "flowspec-trace.json")
	assert.Contains(t, err.Error(), "https://flowspec.dev/docs/trace-formats")
}

func TestDefaultTraceFileParser_ParseFile_MissingTraceId(t *testing.T) {
	parser := NewTraceFileParser()

	// Create a FlowSpec trace without traceId
	traceContent := `{
  "format": "flowspec",
  "spans": {
    "span-1": {
      "spanId": "span-1",
      "traceId": "test-trace-123",
      "name": "test-operation",
      "startTime": 1640995200000000000,
      "endTime": 1640995201000000000,
      "status": {"code": "OK"},
      "attributes": {},
      "events": []
    }
  }
}`

	tmpDir := t.TempDir()
	traceFile := filepath.Join(tmpDir, "missing-traceid.json")
	err := os.WriteFile(traceFile, []byte(traceContent), 0644)
	require.NoError(t, err)

	// Parse the file
	traceData, err := parser.ParseFile(traceFile)

	// Verify error
	assert.Error(t, err)
	assert.Nil(t, traceData)
	assert.Contains(t, err.Error(), "missing traceId")
}

func TestDefaultTraceFileParser_ParseFile_EmptySpans(t *testing.T) {
	parser := NewTraceFileParser()

	// Create a FlowSpec trace with empty spans
	traceContent := `{
  "format": "flowspec",
  "traceId": "test-trace-123",
  "spans": {}
}`

	tmpDir := t.TempDir()
	traceFile := filepath.Join(tmpDir, "empty-spans.json")
	err := os.WriteFile(traceFile, []byte(traceContent), 0644)
	require.NoError(t, err)

	// Parse the file
	traceData, err := parser.ParseFile(traceFile)

	// Verify error
	assert.Error(t, err)
	assert.Nil(t, traceData)
	assert.Contains(t, err.Error(), "no spans found")
}

func TestDefaultTraceFileParser_ParseFile_FileNotFound(t *testing.T) {
	parser := NewTraceFileParser()

	// Try to parse a non-existent file
	traceData, err := parser.ParseFile("/non/existent/trace.json")

	// Verify error
	assert.Error(t, err)
	assert.Nil(t, traceData)
	assert.Contains(t, err.Error(), "failed to read trace file")
}

func TestDefaultTraceFileParser_detectFormat(t *testing.T) {
	parser := NewTraceFileParser()

	tests := []struct {
		name     string
		data     string
		expected TraceFormat
		hasError bool
	}{
		{
			name: "FlowSpec format with format field",
			data: `{"format": "flowspec", "traceId": "123", "spans": {}}`,
			expected: FormatFlowSpecTrace,
			hasError: false,
		},
		{
			name: "FlowSpec format without format field",
			data: `{"traceId": "123", "spans": {}}`,
			expected: FormatFlowSpecTrace,
			hasError: false,
		},
		{
			name: "OTLP format",
			data: `{"resourceSpans": []}`,
			expected: FormatOTLP,
			hasError: false,
		},
		{
			name: "HAR format",
			data: `{"log": {"entries": []}}`,
			expected: FormatHAR,
			hasError: true, // HAR not supported yet
		},
		{
			name: "Unknown format",
			data: `{"unknown": "format"}`,
			expected: "",
			hasError: true,
		},
		{
			name: "Invalid JSON",
			data: `{invalid json}`,
			expected: "",
			hasError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			format, err := parser.detectFormat([]byte(test.data))
			
			if test.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, format)
			}
		})
	}
}

func TestDefaultTraceFileParser_GetSupportedFormats(t *testing.T) {
	parser := NewTraceFileParser()
	
	formats := parser.GetSupportedFormats()
	
	assert.Contains(t, formats, "flowspec-trace.json")
	assert.Contains(t, formats, "otlp.json")
	assert.True(t, len(formats) >= 2)
}

func TestFormatDetectionError(t *testing.T) {
	err := NewFormatDetectionError("har", []string{"flowspec-trace.json", "otlp.json"})
	
	errMsg := err.Error()
	assert.Contains(t, errMsg, "unsupported trace format")
	assert.Contains(t, errMsg, "detected: har")
	assert.Contains(t, errMsg, "Supported formats: flowspec-trace.json, otlp.json")
	assert.Contains(t, errMsg, "Use har2flowspec converter")
	assert.Contains(t, errMsg, "https://flowspec.dev/docs/trace-formats")
}

func TestDefaultTraceFileParser_ParseFile_FlowSpecTrace_ArrayFormat(t *testing.T) {
	parser := NewTraceFileParser()

	// Test array format (compatible with standard tracing systems)
	traceContent := `{
  "traceId": "test-trace-array",
  "spans": [
    {
      "spanId": "span-1",
      "traceId": "test-trace-array",
      "name": "test-operation",
      "startTime": 1640995200000000000,
      "endTime": 1640995201000000000,
      "status": {
        "code": "OK",
        "message": ""
      },
      "attributes": {
        "http.method": "GET",
        "http.url": "/api/test"
      },
      "events": []
    },
    {
      "spanId": "span-2",
      "traceId": "test-trace-array",
      "parentSpanId": "span-1",
      "name": "child-operation",
      "startTime": 1640995200500000000,
      "endTime": 1640995200800000000,
      "status": {
        "code": "OK",
        "message": ""
      },
      "attributes": {},
      "events": []
    }
  ]
}`

	tmpDir := t.TempDir()
	traceFile := filepath.Join(tmpDir, "flowspec-trace-array.json")
	err := os.WriteFile(traceFile, []byte(traceContent), 0644)
	require.NoError(t, err)

	// Parse the file
	traceData, err := parser.ParseFile(traceFile)

	// Verify results
	require.NoError(t, err)
	assert.NotNil(t, traceData)
	assert.Equal(t, "test-trace-array", traceData.TraceID)
	assert.Len(t, traceData.Spans, 2)

	// Verify spans are correctly converted to map format internally
	span1 := traceData.Spans["span-1"]
	assert.NotNil(t, span1)
	assert.Equal(t, "span-1", span1.SpanID)
	assert.Equal(t, "test-trace-array", span1.TraceID)
	assert.Equal(t, "test-operation", span1.Name)

	span2 := traceData.Spans["span-2"]
	assert.NotNil(t, span2)
	assert.Equal(t, "span-2", span2.SpanID)
	assert.Equal(t, "test-trace-array", span2.TraceID)
	assert.Equal(t, "span-1", span2.ParentID)
	assert.Equal(t, "child-operation", span2.Name)
}

func TestTraceData_ToCompatFormat(t *testing.T) {
	// Create TraceData with map format
	traceData := &models.TraceData{
		TraceID: "test-trace",
		Spans: map[string]*models.Span{
			"span-1": {
				SpanID:  "span-1",
				TraceID: "test-trace",
				Name:    "operation-1",
			},
			"span-2": {
				SpanID:   "span-2",
				TraceID:  "test-trace",
				ParentID: "span-1",
				Name:     "operation-2",
			},
		},
	}

	// Convert to compatible format
	compat := traceData.ToCompatFormat()

	// Verify conversion
	assert.Equal(t, "test-trace", compat.TraceID)
	assert.Len(t, compat.Spans, 2)

	// Verify spans are in array format
	spanIDs := make([]string, len(compat.Spans))
	for i, span := range compat.Spans {
		spanIDs[i] = span.SpanID
	}
	assert.Contains(t, spanIDs, "span-1")
	assert.Contains(t, spanIDs, "span-2")
}

func TestFromCompatFormat(t *testing.T) {
	// Create compatible format data
	compat := &models.TraceDataCompat{
		TraceID: "test-trace",
		Spans: []*models.Span{
			{
				SpanID:  "span-1",
				TraceID: "test-trace",
				Name:    "operation-1",
			},
			{
				SpanID:   "span-2",
				TraceID:  "test-trace",
				ParentID: "span-1",
				Name:     "operation-2",
			},
		},
	}

	// Convert to internal format
	traceData := models.FromCompatFormat(compat)

	// Verify conversion
	assert.Equal(t, "test-trace", traceData.TraceID)
	assert.Len(t, traceData.Spans, 2)

	// Verify spans are in map format
	assert.Contains(t, traceData.Spans, "span-1")
	assert.Contains(t, traceData.Spans, "span-2")

	span1 := traceData.Spans["span-1"]
	assert.Equal(t, "span-1", span1.SpanID)
	assert.Equal(t, "operation-1", span1.Name)

	span2 := traceData.Spans["span-2"]
	assert.Equal(t, "span-2", span2.SpanID)
	assert.Equal(t, "span-1", span2.ParentID)
	assert.Equal(t, "operation-2", span2.Name)
}