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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOTLPJSONParsing_ComplexTrace(t *testing.T) {
	ingestor := NewTraceIngestor()

	// Create complex OTLP trace with multiple resource spans
	complexTrace := createComplexOTLPTrace()
	reader := strings.NewReader(complexTrace)

	traceData, err := ingestor.IngestFromReader(reader)

	require.NoError(t, err)
	assert.NotNil(t, traceData)
	assert.Equal(t, "complex-trace-123", traceData.TraceID)
	assert.Len(t, traceData.Spans, 5) // 1 root + 4 children across different services

	// Verify root span
	rootSpan := traceData.RootSpan
	assert.NotNil(t, rootSpan)
	assert.Equal(t, "api-gateway", rootSpan.Name)
	assert.Equal(t, "", rootSpan.ParentID)

	// Verify span tree structure
	assert.NotNil(t, traceData.SpanTree)
	assert.Equal(t, rootSpan.SpanID, traceData.SpanTree.Span.SpanID)
	assert.Len(t, traceData.SpanTree.Children, 2) // Gateway calls 2 services

	// Verify service spans
	userServiceSpan := traceData.FindSpanByID("span-user-service")
	assert.NotNil(t, userServiceSpan)
	assert.Equal(t, "user-service", userServiceSpan.Name)
	assert.Equal(t, "user-service", userServiceSpan.Attributes["service.name"])

	orderServiceSpan := traceData.FindSpanByID("span-order-service")
	assert.NotNil(t, orderServiceSpan)
	assert.Equal(t, "order-service", orderServiceSpan.Name)
	assert.Equal(t, "order-service", orderServiceSpan.Attributes["service.name"])
}

func TestOTLPJSONParsing_SpanAttributes(t *testing.T) {
	ingestor := NewTraceIngestor()

	// Create trace with various attribute types
	traceWithAttributes := createTraceWithAttributes()
	reader := strings.NewReader(traceWithAttributes)

	traceData, err := ingestor.IngestFromReader(reader)

	require.NoError(t, err)
	assert.Len(t, traceData.Spans, 1)

	span := traceData.GetAllSpans()[0]

	// Verify different attribute types
	assert.Equal(t, "test-service", span.Attributes["service.name"])
	assert.Equal(t, "1.0.0", span.Attributes["service.version"])
	assert.Equal(t, float64(200), span.Attributes["http.status_code"])
	assert.Equal(t, true, span.Attributes["http.success"])
	assert.Equal(t, "POST", span.Attributes["http.method"])
	assert.Equal(t, "/api/users", span.Attributes["http.url"])
}

func TestOTLPJSONParsing_SpanEvents(t *testing.T) {
	ingestor := NewTraceIngestor()

	// Create trace with span events
	traceWithEvents := createTraceWithEvents()
	reader := strings.NewReader(traceWithEvents)

	traceData, err := ingestor.IngestFromReader(reader)

	require.NoError(t, err)
	assert.Len(t, traceData.Spans, 1)

	span := traceData.GetAllSpans()[0]
	assert.Len(t, span.Events, 3)

	// Verify events
	events := span.Events
	assert.Equal(t, "request.start", events[0].Name)
	assert.Equal(t, "validation.complete", events[1].Name)
	assert.Equal(t, "response.sent", events[2].Name)

	// Verify event attributes
	assert.Equal(t, "info", events[0].Attributes["level"])
	assert.Equal(t, "user input validated", events[1].Attributes["message"])
	assert.Equal(t, float64(201), events[2].Attributes["status_code"])
}

func TestOTLPJSONParsing_SpanStatus(t *testing.T) {
	ingestor := NewTraceIngestor()

	testCases := []struct {
		name            string
		statusCode      int
		statusMessage   string
		expectedCode    string
		expectedMessage string
	}{
		{"Unset Status", 0, "", "UNSET", ""},
		{"OK Status", 1, "Success", "OK", "Success"},
		{"Error Status", 2, "Internal Server Error", "ERROR", "Internal Server Error"},
		{"Unknown Status", 99, "Unknown", "UNKNOWN", "Unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			traceWithStatus := createTraceWithStatus(tc.statusCode, tc.statusMessage)
			reader := strings.NewReader(traceWithStatus)

			traceData, err := ingestor.IngestFromReader(reader)

			require.NoError(t, err)
			assert.Len(t, traceData.Spans, 1)

			span := traceData.GetAllSpans()[0]
			assert.Equal(t, tc.expectedCode, span.Status.Code)
			assert.Equal(t, tc.expectedMessage, span.Status.Message)
		})
	}
}

func TestOTLPJSONParsing_TimestampHandling(t *testing.T) {
	ingestor := NewTraceIngestor()

	// Create trace with specific timestamps
	traceWithTimestamps := createTraceWithTimestamps()
	reader := strings.NewReader(traceWithTimestamps)

	traceData, err := ingestor.IngestFromReader(reader)

	require.NoError(t, err)
	assert.Len(t, traceData.Spans, 1)

	span := traceData.GetAllSpans()[0]

	// Verify timestamps (nanoseconds since epoch)
	expectedStart := int64(1640995200000000000) // 2022-01-01 00:00:00 UTC
	expectedEnd := int64(1640995201500000000)   // 2022-01-01 00:00:01.5 UTC

	assert.Equal(t, expectedStart, span.StartTime)
	assert.Equal(t, expectedEnd, span.EndTime)
	assert.Equal(t, int64(1500000000), span.GetDuration()) // 1.5 seconds in nanoseconds
}

func TestOTLPJSONParsing_MultipleResourceSpans(t *testing.T) {
	ingestor := NewTraceIngestor()

	// Create trace with multiple resource spans (different services)
	multiResourceTrace := createMultiResourceTrace()
	reader := strings.NewReader(multiResourceTrace)

	traceData, err := ingestor.IngestFromReader(reader)

	require.NoError(t, err)
	assert.Equal(t, "multi-resource-trace", traceData.TraceID)
	assert.Len(t, traceData.Spans, 4) // 2 spans from each resource

	// Verify spans from different services
	serviceNames := make(map[string]int)
	for _, span := range traceData.GetAllSpans() {
		if serviceName, ok := span.Attributes["service.name"].(string); ok {
			serviceNames[serviceName]++
		}
	}

	assert.Equal(t, 2, serviceNames["frontend-service"])
	assert.Equal(t, 2, serviceNames["backend-service"])
}

func TestOTLPJSONParsing_EmptyOptionalFields(t *testing.T) {
	ingestor := NewTraceIngestor()

	// Create minimal trace with only required fields
	minimalTrace := createMinimalTrace()
	reader := strings.NewReader(minimalTrace)

	traceData, err := ingestor.IngestFromReader(reader)

	require.NoError(t, err)
	assert.Len(t, traceData.Spans, 1)

	span := traceData.GetAllSpans()[0]

	// Verify minimal span
	assert.Equal(t, "minimal-span", span.Name)
	assert.Equal(t, "", span.ParentID)         // No parent
	assert.Empty(t, span.Attributes)           // No attributes
	assert.Empty(t, span.Events)               // No events
	assert.Equal(t, "UNSET", span.Status.Code) // Default status
	assert.Equal(t, "", span.Status.Message)
}

func TestOTLPJSONParsing_InvalidTimestamps(t *testing.T) {
	ingestor := NewTraceIngestor()

	// Create trace with invalid timestamps
	invalidTrace := createTraceWithInvalidTimestamps()
	reader := strings.NewReader(invalidTrace)

	traceData, err := ingestor.IngestFromReader(reader)

	assert.Error(t, err)
	assert.Nil(t, traceData)
	assert.Contains(t, err.Error(), "invalid start time")
}

func TestOTLPJSONParsing_MalformedJSON(t *testing.T) {
	ingestor := NewTraceIngestor()

	malformedJSON := `{
		"resourceSpans": [
			{
				"scopeSpans": [
					{
						"spans": [
							{
								"traceId": "trace123",
								"spanId": "span123",
								"name": "test-span",
								"startTimeUnixNano": "1640995200000000000",
								// This comment makes it invalid JSON
								"endTimeUnixNano": "1640995201000000000"
							}
						]
					}
				]
			}
		]
	}`

	reader := strings.NewReader(malformedJSON)
	traceData, err := ingestor.IngestFromReader(reader)

	assert.Error(t, err)
	assert.Nil(t, traceData)
	assert.Contains(t, err.Error(), "failed to parse OTLP JSON")
}

func TestOTLPJSONParsing_LargeTrace(t *testing.T) {
	ingestor := NewTraceIngestor()

	// Create a large trace with many spans
	largeTrace := createLargeTrace(100) // 100 spans
	reader := strings.NewReader(largeTrace)

	traceData, err := ingestor.IngestFromReader(reader)

	require.NoError(t, err)
	assert.Equal(t, "large-trace", traceData.TraceID)
	assert.Len(t, traceData.Spans, 100)

	// Verify tree structure is built correctly
	assert.NotNil(t, traceData.SpanTree)
	assert.NotNil(t, traceData.RootSpan)

	// Root should have many children
	assert.Greater(t, len(traceData.SpanTree.Children), 0)
}

// Helper functions to create test OTLP data

func createComplexOTLPTrace() string {
	trace := OTLPTrace{
		ResourceSpans: []ResourceSpan{
			{
				Resource: Resource{
					Attributes: []Attribute{
						{Key: "service.name", Value: "api-gateway"},
						{Key: "service.version", Value: "1.0.0"},
					},
				},
				ScopeSpans: []ScopeSpan{
					{
						Spans: []OTLPSpan{
							{
								TraceID:           "complex-trace-123",
								SpanID:            "span-gateway",
								Name:              "api-gateway",
								StartTimeUnixNano: "1640995200000000000",
								EndTimeUnixNano:   "1640995205000000000",
								Attributes: []Attribute{
									{Key: "service.name", Value: "api-gateway"},
									{Key: "http.method", Value: "POST"},
									{Key: "http.url", Value: "/api/orders"},
								},
								Status: Status{Code: 1, Message: "OK"},
							},
						},
					},
				},
			},
			{
				Resource: Resource{
					Attributes: []Attribute{
						{Key: "service.name", Value: "user-service"},
					},
				},
				ScopeSpans: []ScopeSpan{
					{
						Spans: []OTLPSpan{
							{
								TraceID:           "complex-trace-123",
								SpanID:            "span-user-service",
								ParentSpanID:      "span-gateway",
								Name:              "user-service",
								StartTimeUnixNano: "1640995201000000000",
								EndTimeUnixNano:   "1640995202000000000",
								Attributes: []Attribute{
									{Key: "service.name", Value: "user-service"},
									{Key: "operation.id", Value: "validateUser"},
								},
								Status: Status{Code: 1, Message: "OK"},
							},
							{
								TraceID:           "complex-trace-123",
								SpanID:            "span-user-db",
								ParentSpanID:      "span-user-service",
								Name:              "user-database-query",
								StartTimeUnixNano: "1640995201200000000",
								EndTimeUnixNano:   "1640995201800000000",
								Attributes: []Attribute{
									{Key: "db.system", Value: "postgresql"},
									{Key: "db.statement", Value: "SELECT * FROM users WHERE id = $1"},
								},
								Status: Status{Code: 1, Message: "OK"},
							},
						},
					},
				},
			},
			{
				Resource: Resource{
					Attributes: []Attribute{
						{Key: "service.name", Value: "order-service"},
					},
				},
				ScopeSpans: []ScopeSpan{
					{
						Spans: []OTLPSpan{
							{
								TraceID:           "complex-trace-123",
								SpanID:            "span-order-service",
								ParentSpanID:      "span-gateway",
								Name:              "order-service",
								StartTimeUnixNano: "1640995202000000000",
								EndTimeUnixNano:   "1640995204000000000",
								Attributes: []Attribute{
									{Key: "service.name", Value: "order-service"},
									{Key: "operation.id", Value: "createOrder"},
								},
								Status: Status{Code: 1, Message: "OK"},
							},
							{
								TraceID:           "complex-trace-123",
								SpanID:            "span-order-db",
								ParentSpanID:      "span-order-service",
								Name:              "order-database-insert",
								StartTimeUnixNano: "1640995202500000000",
								EndTimeUnixNano:   "1640995203500000000",
								Attributes: []Attribute{
									{Key: "db.system", Value: "postgresql"},
									{Key: "db.statement", Value: "INSERT INTO orders (user_id, total) VALUES ($1, $2)"},
								},
								Status: Status{Code: 1, Message: "OK"},
							},
						},
					},
				},
			},
		},
	}

	data, _ := json.Marshal(trace)
	return string(data)
}

func createTraceWithAttributes() string {
	trace := OTLPTrace{
		ResourceSpans: []ResourceSpan{
			{
				ScopeSpans: []ScopeSpan{
					{
						Spans: []OTLPSpan{
							{
								TraceID:           "attr-trace-123",
								SpanID:            "span-with-attrs",
								Name:              "test-span",
								StartTimeUnixNano: "1640995200000000000",
								EndTimeUnixNano:   "1640995201000000000",
								Attributes: []Attribute{
									{Key: "service.name", Value: "test-service"},
									{Key: "service.version", Value: "1.0.0"},
									{Key: "http.status_code", Value: 200},
									{Key: "http.success", Value: true},
									{Key: "http.method", Value: "POST"},
									{Key: "http.url", Value: "/api/users"},
								},
								Status: Status{Code: 1, Message: "OK"},
							},
						},
					},
				},
			},
		},
	}

	data, _ := json.Marshal(trace)
	return string(data)
}

func createTraceWithEvents() string {
	trace := OTLPTrace{
		ResourceSpans: []ResourceSpan{
			{
				ScopeSpans: []ScopeSpan{
					{
						Spans: []OTLPSpan{
							{
								TraceID:           "event-trace-123",
								SpanID:            "span-with-events",
								Name:              "test-span",
								StartTimeUnixNano: "1640995200000000000",
								EndTimeUnixNano:   "1640995201000000000",
								Events: []Event{
									{
										TimeUnixNano: "1640995200100000000",
										Name:         "request.start",
										Attributes: []Attribute{
											{Key: "level", Value: "info"},
										},
									},
									{
										TimeUnixNano: "1640995200500000000",
										Name:         "validation.complete",
										Attributes: []Attribute{
											{Key: "message", Value: "user input validated"},
										},
									},
									{
										TimeUnixNano: "1640995200900000000",
										Name:         "response.sent",
										Attributes: []Attribute{
											{Key: "status_code", Value: 201},
										},
									},
								},
								Status: Status{Code: 1, Message: "OK"},
							},
						},
					},
				},
			},
		},
	}

	data, _ := json.Marshal(trace)
	return string(data)
}

func createTraceWithStatus(statusCode int, statusMessage string) string {
	trace := OTLPTrace{
		ResourceSpans: []ResourceSpan{
			{
				ScopeSpans: []ScopeSpan{
					{
						Spans: []OTLPSpan{
							{
								TraceID:           "status-trace-123",
								SpanID:            "span-with-status",
								Name:              "test-span",
								StartTimeUnixNano: "1640995200000000000",
								EndTimeUnixNano:   "1640995201000000000",
								Status: Status{
									Code:    StatusCode(statusCode),
									Message: statusMessage,
								},
							},
						},
					},
				},
			},
		},
	}

	data, _ := json.Marshal(trace)
	return string(data)
}

func createTraceWithTimestamps() string {
	trace := OTLPTrace{
		ResourceSpans: []ResourceSpan{
			{
				ScopeSpans: []ScopeSpan{
					{
						Spans: []OTLPSpan{
							{
								TraceID:           "timestamp-trace-123",
								SpanID:            "span-with-timestamps",
								Name:              "test-span",
								StartTimeUnixNano: "1640995200000000000", // 2022-01-01 00:00:00 UTC
								EndTimeUnixNano:   "1640995201500000000", // 2022-01-01 00:00:01.5 UTC
								Status:            Status{Code: 1, Message: "OK"},
							},
						},
					},
				},
			},
		},
	}

	data, _ := json.Marshal(trace)
	return string(data)
}

func createMultiResourceTrace() string {
	trace := OTLPTrace{
		ResourceSpans: []ResourceSpan{
			{
				Resource: Resource{
					Attributes: []Attribute{
						{Key: "service.name", Value: "frontend-service"},
					},
				},
				ScopeSpans: []ScopeSpan{
					{
						Spans: []OTLPSpan{
							{
								TraceID:           "multi-resource-trace",
								SpanID:            "frontend-span-1",
								Name:              "frontend-operation-1",
								StartTimeUnixNano: "1640995200000000000",
								EndTimeUnixNano:   "1640995201000000000",
								Attributes: []Attribute{
									{Key: "service.name", Value: "frontend-service"},
								},
								Status: Status{Code: 1, Message: "OK"},
							},
							{
								TraceID:           "multi-resource-trace",
								SpanID:            "frontend-span-2",
								ParentSpanID:      "frontend-span-1",
								Name:              "frontend-operation-2",
								StartTimeUnixNano: "1640995200200000000",
								EndTimeUnixNano:   "1640995200800000000",
								Attributes: []Attribute{
									{Key: "service.name", Value: "frontend-service"},
								},
								Status: Status{Code: 1, Message: "OK"},
							},
						},
					},
				},
			},
			{
				Resource: Resource{
					Attributes: []Attribute{
						{Key: "service.name", Value: "backend-service"},
					},
				},
				ScopeSpans: []ScopeSpan{
					{
						Spans: []OTLPSpan{
							{
								TraceID:           "multi-resource-trace",
								SpanID:            "backend-span-1",
								ParentSpanID:      "frontend-span-1",
								Name:              "backend-operation-1",
								StartTimeUnixNano: "1640995200300000000",
								EndTimeUnixNano:   "1640995200700000000",
								Attributes: []Attribute{
									{Key: "service.name", Value: "backend-service"},
								},
								Status: Status{Code: 1, Message: "OK"},
							},
							{
								TraceID:           "multi-resource-trace",
								SpanID:            "backend-span-2",
								ParentSpanID:      "backend-span-1",
								Name:              "backend-operation-2",
								StartTimeUnixNano: "1640995200400000000",
								EndTimeUnixNano:   "1640995200600000000",
								Attributes: []Attribute{
									{Key: "service.name", Value: "backend-service"},
								},
								Status: Status{Code: 1, Message: "OK"},
							},
						},
					},
				},
			},
		},
	}

	data, _ := json.Marshal(trace)
	return string(data)
}

func createMinimalTrace() string {
	trace := OTLPTrace{
		ResourceSpans: []ResourceSpan{
			{
				ScopeSpans: []ScopeSpan{
					{
						Spans: []OTLPSpan{
							{
								TraceID:           "minimal-trace-123",
								SpanID:            "minimal-span-123",
								Name:              "minimal-span",
								StartTimeUnixNano: "1640995200000000000",
								EndTimeUnixNano:   "1640995201000000000",
								Status:            Status{Code: 0, Message: ""}, // UNSET status
							},
						},
					},
				},
			},
		},
	}

	data, _ := json.Marshal(trace)
	return string(data)
}

func createTraceWithInvalidTimestamps() string {
	trace := OTLPTrace{
		ResourceSpans: []ResourceSpan{
			{
				ScopeSpans: []ScopeSpan{
					{
						Spans: []OTLPSpan{
							{
								TraceID:           "invalid-trace-123",
								SpanID:            "invalid-span-123",
								Name:              "invalid-span",
								StartTimeUnixNano: "not-a-number",
								EndTimeUnixNano:   "1640995201000000000",
								Status:            Status{Code: 1, Message: "OK"},
							},
						},
					},
				},
			},
		},
	}

	data, _ := json.Marshal(trace)
	return string(data)
}

func createLargeTrace(numSpans int) string {
	spans := make([]OTLPSpan, numSpans)

	// Create root span
	spans[0] = OTLPSpan{
		TraceID:           "large-trace",
		SpanID:            "root-span",
		Name:              "root-operation",
		StartTimeUnixNano: "1640995200000000000",
		EndTimeUnixNano:   "1640995210000000000", // 10 seconds
		Status:            Status{Code: 1, Message: "OK"},
	}

	// Create child spans
	for i := 1; i < numSpans; i++ {
		spans[i] = OTLPSpan{
			TraceID:           "large-trace",
			SpanID:            fmt.Sprintf("span-%d", i),
			ParentSpanID:      "root-span",
			Name:              fmt.Sprintf("operation-%d", i),
			StartTimeUnixNano: fmt.Sprintf("%d", 1640995200000000000+int64(i)*100000000),          // 100ms apart
			EndTimeUnixNano:   fmt.Sprintf("%d", 1640995200000000000+int64(i)*100000000+50000000), // 50ms duration
			Status:            Status{Code: 1, Message: "OK"},
		}
	}

	trace := OTLPTrace{
		ResourceSpans: []ResourceSpan{
			{
				ScopeSpans: []ScopeSpan{
					{
						Spans: spans,
					},
				},
			},
		},
	}

	data, _ := json.Marshal(trace)
	return string(data)
}