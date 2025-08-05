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

package engine

import (
	"fmt"
	"testing"
	"time"

	"github.com/flowspec/flowspec-cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompleteAlignmentFlow tests the complete alignment flow from specs to report
func TestCompleteAlignmentFlow(t *testing.T) {
	engine := NewAlignmentEngine()

	// Create test specs with various scenarios
	specs := []models.ServiceSpec{
		{
			OperationID: "createUser",
			Description: "Create a new user",
			Preconditions: map[string]interface{}{
				"==": []interface{}{
					map[string]interface{}{"var": "http_method"},
					"POST",
				},
			},
			Postconditions: map[string]interface{}{
				"and": []interface{}{
					map[string]interface{}{
						"==": []interface{}{
							map[string]interface{}{"var": "span.status.code"},
							"OK",
						},
					},
					map[string]interface{}{
						">=": []interface{}{
							map[string]interface{}{"var": "http_status_code"},
							200,
						},
					},
				},
			},
		},
		{
			OperationID: "getUser",
			Description: "Get user by ID",
			Preconditions: map[string]interface{}{
				"==": []interface{}{
					map[string]interface{}{"var": "http_method"},
					"GET",
				},
			},
			Postconditions: map[string]interface{}{
				"==": []interface{}{
					map[string]interface{}{"var": "span.status.code"},
					"OK",
				},
			},
		},
		{
			OperationID: "deleteUser",
			Description: "Delete user by ID",
			Preconditions: map[string]interface{}{
				"==": []interface{}{
					map[string]interface{}{"var": "http_method"},
					"DELETE",
				},
			},
			Postconditions: map[string]interface{}{
				"==": []interface{}{
					map[string]interface{}{"var": "span.status.code"},
					"OK",
				},
			},
		},
		{
			OperationID: "nonExistentOperation",
			Description: "This operation has no matching spans",
			Preconditions: map[string]interface{}{
				"==": []interface{}{true, true},
			},
			Postconditions: map[string]interface{}{
				"==": []interface{}{true, true},
			},
		},
	}

	// Create test trace data with matching and non-matching spans
	traceData := &models.TraceData{
		TraceID: "test-trace-123",
		Spans: map[string]*models.Span{
			"span-1": {
				SpanID:    "span-1",
				TraceID:   "test-trace-123",
				Name:      "createUser",
				StartTime: time.Now().Add(-10 * time.Second).UnixNano(),
				EndTime:   time.Now().Add(-9 * time.Second).UnixNano(),
				Status: models.SpanStatus{
					Code:    "OK",
					Message: "Success",
				},
				Attributes: map[string]interface{}{
					"operation.id":     "createUser",
					"http.method":      "POST",
					"http.status_code": 201,
				},
			},
			"span-2": {
				SpanID:    "span-2",
				TraceID:   "test-trace-123",
				Name:      "getUser",
				StartTime: time.Now().Add(-8 * time.Second).UnixNano(),
				EndTime:   time.Now().Add(-7 * time.Second).UnixNano(),
				Status: models.SpanStatus{
					Code:    "OK",
					Message: "Success",
				},
				Attributes: map[string]interface{}{
					"operation.id":     "getUser",
					"http.method":      "GET",
					"http.status_code": 200,
				},
			},
			"span-3": {
				SpanID:    "span-3",
				TraceID:   "test-trace-123",
				Name:      "deleteUser",
				StartTime: time.Now().Add(-6 * time.Second).UnixNano(),
				EndTime:   time.Now().Add(-5 * time.Second).UnixNano(),
				Status: models.SpanStatus{
					Code:    "ERROR",
					Message: "Internal server error",
				},
				Attributes: map[string]interface{}{
					"operation.id":     "deleteUser",
					"http.method":      "DELETE",
					"http.status_code": 500,
				},
			},
		},
	}

	// Set root span
	traceData.RootSpan = traceData.Spans["span-1"]

	// Execute alignment
	report, err := engine.AlignSpecsWithTrace(specs, traceData)

	// Verify no errors
	require.NoError(t, err)
	require.NotNil(t, report)

	// Verify report structure
	assert.Equal(t, 4, len(report.Results))
	assert.Equal(t, 4, report.Summary.Total)

	// Verify timing information
	assert.Greater(t, report.ExecutionTime, int64(0))
	assert.Greater(t, report.StartTime, int64(0))
	assert.Greater(t, report.EndTime, int64(0))
	assert.GreaterOrEqual(t, report.EndTime, report.StartTime)

	// Verify performance information
	assert.Equal(t, 4, report.PerformanceInfo.SpecsProcessed)
	assert.Equal(t, 3, report.PerformanceInfo.SpansMatched) // 3 specs have matching spans
	assert.Greater(t, report.PerformanceInfo.AssertionsEvaluated, 0)
	assert.GreaterOrEqual(t, report.PerformanceInfo.MemoryUsageMB, 0.0)
	assert.Greater(t, report.PerformanceInfo.ProcessingRate, 0.0)

	// Verify individual results
	resultsByOperationID := make(map[string]*models.AlignmentResult)
	for i := range report.Results {
		result := &report.Results[i]
		resultsByOperationID[result.SpecOperationID] = result

		// Verify timing for each result
		assert.Greater(t, result.ExecutionTime, int64(0))
		assert.Greater(t, result.StartTime, int64(0))
		assert.Greater(t, result.EndTime, int64(0))
		assert.GreaterOrEqual(t, result.EndTime, result.StartTime)
	}

	// Verify createUser result (should succeed)
	createUserResult := resultsByOperationID["createUser"]
	require.NotNil(t, createUserResult)
	assert.Equal(t, models.StatusSuccess, createUserResult.Status)
	assert.Len(t, createUserResult.MatchedSpans, 1)
	assert.Equal(t, "span-1", createUserResult.MatchedSpans[0])
	assert.Equal(t, 2, createUserResult.AssertionsTotal) // precondition + postcondition
	assert.Equal(t, 2, createUserResult.AssertionsPassed)
	assert.Equal(t, 0, createUserResult.AssertionsFailed)

	// Verify getUser result (should succeed)
	getUserResult := resultsByOperationID["getUser"]
	require.NotNil(t, getUserResult)
	assert.Equal(t, models.StatusSuccess, getUserResult.Status)
	assert.Len(t, getUserResult.MatchedSpans, 1)
	assert.Equal(t, "span-2", getUserResult.MatchedSpans[0])
	assert.Equal(t, 2, getUserResult.AssertionsTotal)
	assert.Equal(t, 2, getUserResult.AssertionsPassed)
	assert.Equal(t, 0, getUserResult.AssertionsFailed)

	// Verify deleteUser result (should fail due to ERROR status)
	deleteUserResult := resultsByOperationID["deleteUser"]
	require.NotNil(t, deleteUserResult)
	assert.Equal(t, models.StatusFailed, deleteUserResult.Status)
	assert.Len(t, deleteUserResult.MatchedSpans, 1)
	assert.Equal(t, "span-3", deleteUserResult.MatchedSpans[0])
	assert.Equal(t, 2, deleteUserResult.AssertionsTotal)
	assert.Equal(t, 1, deleteUserResult.AssertionsPassed) // precondition passes
	assert.Equal(t, 1, deleteUserResult.AssertionsFailed) // postcondition fails

	// Verify nonExistentOperation result (should be skipped)
	nonExistentResult := resultsByOperationID["nonExistentOperation"]
	require.NotNil(t, nonExistentResult)
	assert.Equal(t, models.StatusSkipped, nonExistentResult.Status)
	assert.Empty(t, nonExistentResult.MatchedSpans)
	assert.Equal(t, 0, nonExistentResult.AssertionsTotal)
	assert.Equal(t, 0, nonExistentResult.AssertionsPassed)
	assert.Equal(t, 0, nonExistentResult.AssertionsFailed)

	// Verify summary statistics
	assert.Equal(t, 2, report.Summary.Success)        // createUser, getUser
	assert.Equal(t, 1, report.Summary.Failed)         // deleteUser
	assert.Equal(t, 1, report.Summary.Skipped)        // nonExistentOperation
	assert.Equal(t, 0.5, report.Summary.SuccessRate)  // 2/4
	assert.Equal(t, 0.25, report.Summary.FailureRate) // 1/4
	assert.Equal(t, 0.25, report.Summary.SkipRate)    // 1/4
	assert.Greater(t, report.Summary.AverageExecutionTime, int64(0))
	assert.Equal(t, 6, report.Summary.TotalAssertions)  // 2+2+2+0
	assert.Equal(t, 1, report.Summary.FailedAssertions) // 1 from deleteUser

	// Verify report has failures
	assert.True(t, report.HasFailures())
}

// TestAlignmentFlow_AllSuccess tests a scenario where all specs succeed
func TestAlignmentFlow_AllSuccess(t *testing.T) {
	engine := NewAlignmentEngine()

	specs := []models.ServiceSpec{
		{
			OperationID: "operation1",
			Description: "Test operation 1",
			Preconditions: map[string]interface{}{
				"==": []interface{}{true, true},
			},
			Postconditions: map[string]interface{}{
				"==": []interface{}{
					map[string]interface{}{"var": "span.status.code"},
					"OK",
				},
			},
		},
		{
			OperationID: "operation2",
			Description: "Test operation 2",
			Preconditions: map[string]interface{}{
				"==": []interface{}{true, true},
			},
			Postconditions: map[string]interface{}{
				"==": []interface{}{
					map[string]interface{}{"var": "span.status.code"},
					"OK",
				},
			},
		},
	}

	traceData := &models.TraceData{
		TraceID: "success-trace",
		Spans: map[string]*models.Span{
			"span-1": {
				SpanID:  "span-1",
				TraceID: "success-trace",
				Name:    "operation1",
				Status: models.SpanStatus{
					Code: "OK",
				},
				Attributes: map[string]interface{}{
					"operation.id": "operation1",
				},
			},
			"span-2": {
				SpanID:  "span-2",
				TraceID: "success-trace",
				Name:    "operation2",
				Status: models.SpanStatus{
					Code: "OK",
				},
				Attributes: map[string]interface{}{
					"operation.id": "operation2",
				},
			},
		},
	}
	traceData.RootSpan = traceData.Spans["span-1"]

	report, err := engine.AlignSpecsWithTrace(specs, traceData)

	require.NoError(t, err)
	require.NotNil(t, report)

	// Verify all success
	assert.Equal(t, 2, report.Summary.Total)
	assert.Equal(t, 2, report.Summary.Success)
	assert.Equal(t, 0, report.Summary.Failed)
	assert.Equal(t, 0, report.Summary.Skipped)
	assert.Equal(t, 1.0, report.Summary.SuccessRate)
	assert.Equal(t, 0.0, report.Summary.FailureRate)
	assert.Equal(t, 0.0, report.Summary.SkipRate)
	assert.False(t, report.HasFailures())
}

// TestAlignmentFlow_AllSkipped tests a scenario where all specs are skipped
func TestAlignmentFlow_AllSkipped(t *testing.T) {
	engine := NewAlignmentEngine()

	specs := []models.ServiceSpec{
		{
			OperationID: "nonExistent1",
			Description: "Non-existent operation 1",
			Preconditions: map[string]interface{}{
				"==": []interface{}{true, true},
			},
			Postconditions: map[string]interface{}{
				"==": []interface{}{true, true},
			},
		},
		{
			OperationID: "nonExistent2",
			Description: "Non-existent operation 2",
			Preconditions: map[string]interface{}{
				"==": []interface{}{true, true},
			},
			Postconditions: map[string]interface{}{
				"==": []interface{}{true, true},
			},
		},
	}

	traceData := &models.TraceData{
		TraceID: "empty-trace",
		Spans: map[string]*models.Span{
			"unrelated-span": {
				SpanID:  "unrelated-span",
				TraceID: "empty-trace",
				Name:    "unrelatedOperation",
				Status: models.SpanStatus{
					Code: "OK",
				},
				Attributes: map[string]interface{}{
					"operation.id": "unrelatedOperation",
				},
			},
		},
	}
	traceData.RootSpan = traceData.Spans["unrelated-span"]

	report, err := engine.AlignSpecsWithTrace(specs, traceData)

	require.NoError(t, err)
	require.NotNil(t, report)

	// Verify all skipped
	assert.Equal(t, 2, report.Summary.Total)
	assert.Equal(t, 0, report.Summary.Success)
	assert.Equal(t, 0, report.Summary.Failed)
	assert.Equal(t, 2, report.Summary.Skipped)
	assert.Equal(t, 0.0, report.Summary.SuccessRate)
	assert.Equal(t, 0.0, report.Summary.FailureRate)
	assert.Equal(t, 1.0, report.Summary.SkipRate)
	assert.False(t, report.HasFailures())
}

// TestAlignmentFlow_PerformanceMetrics tests performance metrics collection
func TestAlignmentFlow_PerformanceMetrics(t *testing.T) {
	// Create engine with metrics enabled
	config := DefaultEngineConfig()
	config.EnableMetrics = true
	engine := NewAlignmentEngineWithConfig(config)

	specs := []models.ServiceSpec{
		{
			OperationID: "perfTest1",
			Description: "Performance test 1",
			Preconditions: map[string]interface{}{
				"==": []interface{}{true, true},
			},
			Postconditions: map[string]interface{}{
				"==": []interface{}{true, true},
			},
		},
		{
			OperationID: "perfTest2",
			Description: "Performance test 2",
			Preconditions: map[string]interface{}{
				"==": []interface{}{true, true},
			},
			Postconditions: map[string]interface{}{
				"==": []interface{}{true, true},
			},
		},
	}

	traceData := &models.TraceData{
		TraceID: "perf-trace",
		Spans: map[string]*models.Span{
			"perf-span-1": {
				SpanID:  "perf-span-1",
				TraceID: "perf-trace",
				Name:    "perfTest1",
				Status: models.SpanStatus{
					Code: "OK",
				},
				Attributes: map[string]interface{}{
					"operation.id": "perfTest1",
				},
			},
			"perf-span-2": {
				SpanID:  "perf-span-2",
				TraceID: "perf-trace",
				Name:    "perfTest2",
				Status: models.SpanStatus{
					Code: "OK",
				},
				Attributes: map[string]interface{}{
					"operation.id": "perfTest2",
				},
			},
		},
	}
	traceData.RootSpan = traceData.Spans["perf-span-1"]

	report, err := engine.AlignSpecsWithTrace(specs, traceData)

	require.NoError(t, err)
	require.NotNil(t, report)

	// Verify performance metrics are populated
	perfInfo := report.PerformanceInfo
	assert.Equal(t, 2, perfInfo.SpecsProcessed)
	assert.Equal(t, 2, perfInfo.SpansMatched)
	assert.Equal(t, 4, perfInfo.AssertionsEvaluated) // 2 specs * 2 assertions each
	assert.Greater(t, perfInfo.ConcurrentWorkers, 0)
	assert.GreaterOrEqual(t, perfInfo.MemoryUsageMB, 0.0)
	assert.Greater(t, perfInfo.ProcessingRate, 0.0)
}

// TestAlignmentFlow_ConcurrentProcessing tests concurrent processing with multiple workers
func TestAlignmentFlow_ConcurrentProcessing(t *testing.T) {
	// Create engine with multiple workers
	config := DefaultEngineConfig()
	config.MaxConcurrency = 8
	config.EnableMetrics = true
	engine := NewAlignmentEngineWithConfig(config)

	// Create many specs to test concurrency
	specs := make([]models.ServiceSpec, 20)
	spans := make(map[string]*models.Span)

	for i := 0; i < 20; i++ {
		operationID := fmt.Sprintf("concurrentOp%d", i)
		spanID := fmt.Sprintf("concurrent-span-%d", i)

		specs[i] = models.ServiceSpec{
			OperationID: operationID,
			Description: fmt.Sprintf("Concurrent operation %d", i),
			Preconditions: map[string]interface{}{
				"==": []interface{}{true, true},
			},
			Postconditions: map[string]interface{}{
				"==": []interface{}{
					map[string]interface{}{"var": "span.status.code"},
					"OK",
				},
			},
		}

		spans[spanID] = &models.Span{
			SpanID:  spanID,
			TraceID: "concurrent-trace",
			Name:    operationID,
			Status: models.SpanStatus{
				Code: "OK",
			},
			Attributes: map[string]interface{}{
				"operation.id": operationID,
			},
		}
	}

	traceData := &models.TraceData{
		TraceID: "concurrent-trace",
		Spans:   spans,
	}
	// Set first span as root
	traceData.RootSpan = spans["concurrent-span-0"]

	report, err := engine.AlignSpecsWithTrace(specs, traceData)

	require.NoError(t, err)
	require.NotNil(t, report)

	// Verify all specs were processed successfully
	assert.Equal(t, 20, report.Summary.Total)
	assert.Equal(t, 20, report.Summary.Success)
	assert.Equal(t, 0, report.Summary.Failed)
	assert.Equal(t, 0, report.Summary.Skipped)

	// Verify performance metrics
	assert.Equal(t, 20, report.PerformanceInfo.SpecsProcessed)
	assert.Equal(t, 20, report.PerformanceInfo.SpansMatched)
	assert.Equal(t, 40, report.PerformanceInfo.AssertionsEvaluated) // 20 specs * 2 assertions each

	// Verify concurrent workers were used (should be limited by MaxConcurrency or number of specs)
	expectedWorkers := min(8, 20) // min(MaxConcurrency, number of specs)
	assert.Equal(t, expectedWorkers, report.PerformanceInfo.ConcurrentWorkers)

	// Verify all results have proper timing
	for _, result := range report.Results {
		assert.Greater(t, result.ExecutionTime, int64(0))
		assert.Greater(t, result.StartTime, int64(0))
		assert.Greater(t, result.EndTime, int64(0))
		assert.GreaterOrEqual(t, result.EndTime, result.StartTime)
		assert.Len(t, result.MatchedSpans, 1)
	}
}

// TestAlignmentFlow_ErrorHandling tests error handling in the alignment flow
func TestAlignmentFlow_ErrorHandling(t *testing.T) {
	engine := NewAlignmentEngine()

	// Test with nil trace data
	specs := []models.ServiceSpec{
		{
			OperationID: "testOp",
			Description: "Test operation",
		},
	}

	report, err := engine.AlignSpecsWithTrace(specs, nil)
	assert.Error(t, err)
	assert.Nil(t, report)
	assert.Contains(t, err.Error(), "trace data is empty or nil")

	// Test with empty trace data
	emptyTraceData := &models.TraceData{
		TraceID: "empty",
		Spans:   map[string]*models.Span{},
	}

	report, err = engine.AlignSpecsWithTrace(specs, emptyTraceData)
	assert.Error(t, err)
	assert.Nil(t, report)
	assert.Contains(t, err.Error(), "trace data is empty or nil")

	// Test with empty specs (should succeed)
	validTraceData := &models.TraceData{
		TraceID: "valid",
		Spans: map[string]*models.Span{
			"span-1": {
				SpanID:  "span-1",
				TraceID: "valid",
				Name:    "testOp",
			},
		},
	}

	report, err = engine.AlignSpecsWithTrace([]models.ServiceSpec{}, validTraceData)
	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, 0, report.Summary.Total)
}