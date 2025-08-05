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

package models

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceSpec_Validate(t *testing.T) {
	tests := []struct {
		name    string
		spec    ServiceSpec
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid spec",
			spec: ServiceSpec{
				OperationID:    "createUser",
				Description:    "Create a new user",
				Preconditions:  map[string]interface{}{"request.body.email": map[string]interface{}{"!=": nil}},
				Postconditions: map[string]interface{}{"response.status": map[string]interface{}{"==": 201}},
				SourceFile:     "user.go",
				LineNumber:     10,
			},
			wantErr: false,
		},
		{
			name: "missing operationId",
			spec: ServiceSpec{
				Description:    "Create a new user",
				Preconditions:  map[string]interface{}{"request.body.email": map[string]interface{}{"!=": nil}},
				Postconditions: map[string]interface{}{"response.status": map[string]interface{}{"==": 201}},
				SourceFile:     "user.go",
				LineNumber:     10,
			},
			wantErr: true,
			errMsg:  "operationId is required",
		},
		{
			name: "missing description",
			spec: ServiceSpec{
				OperationID:    "createUser",
				Preconditions:  map[string]interface{}{"request.body.email": map[string]interface{}{"!=": nil}},
				Postconditions: map[string]interface{}{"response.status": map[string]interface{}{"==": 201}},
				SourceFile:     "user.go",
				LineNumber:     10,
			},
			wantErr: true,
			errMsg:  "description is required",
		},
		{
			name: "missing sourceFile",
			spec: ServiceSpec{
				OperationID:    "createUser",
				Description:    "Create a new user",
				Preconditions:  map[string]interface{}{"request.body.email": map[string]interface{}{"!=": nil}},
				Postconditions: map[string]interface{}{"response.status": map[string]interface{}{"==": 201}},
				LineNumber:     10,
			},
			wantErr: true,
			errMsg:  "sourceFile is required",
		},
		{
			name: "invalid lineNumber",
			spec: ServiceSpec{
				OperationID:    "createUser",
				Description:    "Create a new user",
				Preconditions:  map[string]interface{}{"request.body.email": map[string]interface{}{"!=": nil}},
				Postconditions: map[string]interface{}{"response.status": map[string]interface{}{"==": 201}},
				SourceFile:     "user.go",
				LineNumber:     0,
			},
			wantErr: true,
			errMsg:  "lineNumber must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServiceSpec_JSONSerialization(t *testing.T) {
	spec := ServiceSpec{
		OperationID: "createUser",
		Description: "Create a new user",
		Preconditions: map[string]interface{}{
			"request.body.email":    map[string]interface{}{"!=": nil},
			"request.body.password": map[string]interface{}{">=": 8},
		},
		Postconditions: map[string]interface{}{
			"response.status":      map[string]interface{}{"==": 201},
			"response.body.userId": map[string]interface{}{"!=": nil},
		},
		SourceFile: "user.go",
		LineNumber: 15,
	}

	// Test ToJSON
	jsonData, err := spec.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Verify JSON structure
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	require.NoError(t, err)
	assert.Equal(t, "createUser", jsonMap["operationId"])
	assert.Equal(t, "Create a new user", jsonMap["description"])
	assert.Equal(t, "user.go", jsonMap["sourceFile"])
	assert.Equal(t, float64(15), jsonMap["lineNumber"])

	// Test FromJSON
	var newSpec ServiceSpec
	err = newSpec.FromJSON(jsonData)
	require.NoError(t, err)
	assert.Equal(t, spec.OperationID, newSpec.OperationID)
	assert.Equal(t, spec.Description, newSpec.Description)
	assert.Equal(t, spec.SourceFile, newSpec.SourceFile)
	assert.Equal(t, spec.LineNumber, newSpec.LineNumber)

	// Note: JSON unmarshaling converts numbers to float64, so we need to check structure rather than exact equality
	assert.NotNil(t, newSpec.Preconditions)
	assert.NotNil(t, newSpec.Postconditions)
	assert.Len(t, newSpec.Preconditions, 2)
	assert.Len(t, newSpec.Postconditions, 2)
}

func TestServiceSpec_JSONDeserialization_InvalidJSON(t *testing.T) {
	var spec ServiceSpec
	err := spec.FromJSON([]byte("invalid json"))
	assert.Error(t, err)
}

func TestServiceSpec_String(t *testing.T) {
	spec := ServiceSpec{
		OperationID: "createUser",
		Description: "Create a new user",
		SourceFile:  "user.go",
		LineNumber:  15,
	}

	str := spec.String()
	assert.Contains(t, str, "createUser")
	assert.Contains(t, str, "Create a new user")
	assert.Contains(t, str, "user.go")
	assert.Contains(t, str, "15")
}

func TestParseResult_Structure(t *testing.T) {
	spec := ServiceSpec{
		OperationID: "createUser",
		Description: "Create a new user",
		SourceFile:  "user.go",
		LineNumber:  15,
	}

	parseError := ParseError{
		File:    "invalid.go",
		Line:    20,
		Message: "syntax error",
	}

	result := ParseResult{
		Specs:  []ServiceSpec{spec},
		Errors: []ParseError{parseError},
	}

	// Test JSON serialization of ParseResult
	jsonData, err := json.Marshal(result)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test JSON deserialization of ParseResult
	var newResult ParseResult
	err = json.Unmarshal(jsonData, &newResult)
	require.NoError(t, err)
	assert.Len(t, newResult.Specs, 1)
	assert.Len(t, newResult.Errors, 1)
	assert.Equal(t, spec.OperationID, newResult.Specs[0].OperationID)
	assert.Equal(t, parseError.File, newResult.Errors[0].File)
}

func TestParseError_Error(t *testing.T) {
	parseError := ParseError{
		File:    "test.go",
		Line:    42,
		Message: "unexpected token",
	}

	errStr := parseError.Error()
	assert.Equal(t, "test.go:42: unexpected token", errStr)
}

func TestServiceSpec_EmptyConditions(t *testing.T) {
	spec := ServiceSpec{
		OperationID:    "testOp",
		Description:    "Test operation",
		Preconditions:  nil,
		Postconditions: make(map[string]interface{}),
		SourceFile:     "test.go",
		LineNumber:     1,
	}

	// Should validate successfully even with empty conditions
	err := spec.Validate()
	assert.NoError(t, err)

	// Should serialize/deserialize correctly
	jsonData, err := spec.ToJSON()
	require.NoError(t, err)

	var newSpec ServiceSpec
	err = newSpec.FromJSON(jsonData)
	require.NoError(t, err)
	assert.Nil(t, newSpec.Preconditions)
	assert.NotNil(t, newSpec.Postconditions)
	assert.Len(t, newSpec.Postconditions, 0)
}

func TestServiceSpec_ComplexConditions(t *testing.T) {
	spec := ServiceSpec{
		OperationID: "complexOp",
		Description: "Complex operation with nested conditions",
		Preconditions: map[string]interface{}{
			"and": []interface{}{
				map[string]interface{}{"!=": []interface{}{map[string]interface{}{"var": "span.attributes.http.method"}, nil}},
				map[string]interface{}{"==": []interface{}{map[string]interface{}{"var": "span.attributes.http.method"}, "POST"}},
			},
		},
		Postconditions: map[string]interface{}{
			"response_validation": map[string]interface{}{
				"and": []interface{}{
					map[string]interface{}{"==": []interface{}{map[string]interface{}{"var": "status.code"}, "OK"}},
					map[string]interface{}{">=": []interface{}{map[string]interface{}{"var": "span.attributes.http.status_code"}, 200}},
					map[string]interface{}{"<": []interface{}{map[string]interface{}{"var": "span.attributes.http.status_code"}, 300}},
				},
			},
		},
		SourceFile: "complex.go",
		LineNumber: 100,
	}

	// Should validate successfully
	err := spec.Validate()
	assert.NoError(t, err)

	// Should serialize/deserialize correctly with complex nested structures
	jsonData, err := spec.ToJSON()
	require.NoError(t, err)

	var newSpec ServiceSpec
	err = newSpec.FromJSON(jsonData)
	require.NoError(t, err)
	assert.Equal(t, spec.OperationID, newSpec.OperationID)

	// Verify structure is preserved (JSON unmarshaling converts numbers to float64)
	assert.NotNil(t, newSpec.Preconditions)
	assert.NotNil(t, newSpec.Postconditions)
	assert.Contains(t, newSpec.Preconditions, "and")
	assert.Contains(t, newSpec.Postconditions, "response_validation")
}

// Tests for Trace-related data structures

func TestSpan_GetDuration(t *testing.T) {
	span := Span{
		StartTime: 1000000000, // 1 second in nanoseconds
		EndTime:   2000000000, // 2 seconds in nanoseconds
	}

	duration := span.GetDuration()
	assert.Equal(t, int64(1000000000), duration) // 1 second
}

func TestSpan_IsRoot(t *testing.T) {
	rootSpan := Span{ParentID: ""}
	childSpan := Span{ParentID: "parent123"}

	assert.True(t, rootSpan.IsRoot())
	assert.False(t, childSpan.IsRoot())
}

func TestSpan_HasError(t *testing.T) {
	errorSpan := Span{Status: SpanStatus{Code: "ERROR"}}
	okSpan := Span{Status: SpanStatus{Code: "OK"}}

	assert.True(t, errorSpan.HasError())
	assert.False(t, okSpan.HasError())
}

func TestSpan_GetAttribute(t *testing.T) {
	span := Span{
		Attributes: map[string]interface{}{
			"http.method":      "POST",
			"http.status_code": 200,
		},
	}

	assert.Equal(t, "POST", span.GetAttribute("http.method"))
	assert.Equal(t, 200, span.GetAttribute("http.status_code"))
	assert.Nil(t, span.GetAttribute("nonexistent"))

	// Test with nil attributes
	emptySpan := Span{}
	assert.Nil(t, emptySpan.GetAttribute("any"))
}

func TestSpan_AddEvent(t *testing.T) {
	span := Span{}
	event := SpanEvent{
		Name:      "test.event",
		Timestamp: 1000000000,
		Attributes: map[string]interface{}{
			"key": "value",
		},
	}

	span.AddEvent(event)
	assert.Len(t, span.Events, 1)
	assert.Equal(t, "test.event", span.Events[0].Name)
	assert.Equal(t, int64(1000000000), span.Events[0].Timestamp)
}

func TestSpan_String(t *testing.T) {
	span := Span{
		SpanID:  "span123",
		TraceID: "trace456",
		Name:    "test.operation",
		Status:  SpanStatus{Code: "OK"},
	}

	str := span.String()
	assert.Contains(t, str, "span123")
	assert.Contains(t, str, "trace456")
	assert.Contains(t, str, "test.operation")
	assert.Contains(t, str, "OK")
}

func TestTraceData_FindSpanByID(t *testing.T) {
	span1 := &Span{SpanID: "span1", Name: "operation1"}
	span2 := &Span{SpanID: "span2", Name: "operation2"}

	traceData := TraceData{
		Spans: map[string]*Span{
			"span1": span1,
			"span2": span2,
		},
	}

	found := traceData.FindSpanByID("span1")
	assert.NotNil(t, found)
	assert.Equal(t, "operation1", found.Name)

	notFound := traceData.FindSpanByID("nonexistent")
	assert.Nil(t, notFound)
}

func TestTraceData_FindSpansByName(t *testing.T) {
	span1 := &Span{SpanID: "span1", Name: "operation1"}
	span2 := &Span{SpanID: "span2", Name: "operation2"}
	span3 := &Span{SpanID: "span3", Name: "operation1"} // Same name as span1

	traceData := TraceData{
		Spans: map[string]*Span{
			"span1": span1,
			"span2": span2,
			"span3": span3,
		},
	}

	found := traceData.FindSpansByName("operation1")
	assert.Len(t, found, 2)

	notFound := traceData.FindSpansByName("nonexistent")
	assert.Len(t, notFound, 0)
}

func TestTraceData_GetAllSpans(t *testing.T) {
	span1 := &Span{SpanID: "span1"}
	span2 := &Span{SpanID: "span2"}

	traceData := TraceData{
		Spans: map[string]*Span{
			"span1": span1,
			"span2": span2,
		},
	}

	allSpans := traceData.GetAllSpans()
	assert.Len(t, allSpans, 2)
}

func TestTraceData_BuildSpanTree(t *testing.T) {
	// Create a simple tree: root -> child1 -> grandchild1
	//                            -> child2
	rootSpan := &Span{SpanID: "root", TraceID: "trace1", Name: "root", ParentID: ""}
	child1 := &Span{SpanID: "child1", TraceID: "trace1", Name: "child1", ParentID: "root"}
	child2 := &Span{SpanID: "child2", TraceID: "trace1", Name: "child2", ParentID: "root"}
	grandchild1 := &Span{SpanID: "grandchild1", TraceID: "trace1", Name: "grandchild1", ParentID: "child1"}

	traceData := TraceData{
		TraceID: "trace1",
		Spans: map[string]*Span{
			"root":        rootSpan,
			"child1":      child1,
			"child2":      child2,
			"grandchild1": grandchild1,
		},
	}

	err := traceData.BuildSpanTree()
	require.NoError(t, err)

	// Verify root span
	assert.NotNil(t, traceData.RootSpan)
	assert.Equal(t, "root", traceData.RootSpan.SpanID)

	// Verify tree structure
	assert.NotNil(t, traceData.SpanTree)
	assert.Equal(t, "root", traceData.SpanTree.Span.SpanID)
	assert.Len(t, traceData.SpanTree.Children, 2) // child1 and child2

	// Find child1 and verify it has grandchild1
	var child1Node *SpanNode
	for _, child := range traceData.SpanTree.Children {
		if child.Span.SpanID == "child1" {
			child1Node = child
			break
		}
	}
	require.NotNil(t, child1Node)
	assert.Len(t, child1Node.Children, 1) // grandchild1
	assert.Equal(t, "grandchild1", child1Node.Children[0].Span.SpanID)
}

func TestTraceData_BuildSpanTree_NoSpans(t *testing.T) {
	traceData := TraceData{
		TraceID: "trace1",
		Spans:   map[string]*Span{},
	}

	// Building a tree from no spans should not produce an error.
	err := traceData.BuildSpanTree()
	assert.NoError(t, err)
	assert.Nil(t, traceData.RootSpan, "RootSpan should be nil for an empty trace")
	assert.Nil(t, traceData.SpanTree, "SpanTree should be nil for an empty trace")
}

func TestTraceData_BuildSpanTree_NoRootSpan(t *testing.T) {
	// All spans have parents - no root span
	child1 := &Span{SpanID: "child1", ParentID: "nonexistent"}
	child2 := &Span{SpanID: "child2", ParentID: "child1"}

	traceData := TraceData{
		TraceID: "trace1",
		Spans: map[string]*Span{
			"child1": child1,
			"child2": child2,
		},
	}

	err := traceData.BuildSpanTree()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no root span found")
}

func TestSpanNode_GetChildCount(t *testing.T) {
	rootNode := &SpanNode{
		Span: &Span{SpanID: "root"},
		Children: []*SpanNode{
			{Span: &Span{SpanID: "child1"}},
			{Span: &Span{SpanID: "child2"}},
		},
	}

	assert.Equal(t, 2, rootNode.GetChildCount())

	leafNode := &SpanNode{
		Span:     &Span{SpanID: "leaf"},
		Children: []*SpanNode{},
	}

	assert.Equal(t, 0, leafNode.GetChildCount())
}

func TestSpanNode_GetTotalDescendants(t *testing.T) {
	// Create tree: root -> child1 -> grandchild1
	//                   -> child2
	grandchild1 := &SpanNode{
		Span:     &Span{SpanID: "grandchild1"},
		Children: []*SpanNode{},
	}

	child1 := &SpanNode{
		Span:     &Span{SpanID: "child1"},
		Children: []*SpanNode{grandchild1},
	}

	child2 := &SpanNode{
		Span:     &Span{SpanID: "child2"},
		Children: []*SpanNode{},
	}

	rootNode := &SpanNode{
		Span:     &Span{SpanID: "root"},
		Children: []*SpanNode{child1, child2},
	}

	// Root has 2 children + 1 grandchild = 3 total descendants
	assert.Equal(t, 3, rootNode.GetTotalDescendants())

	// Child1 has 1 grandchild = 1 total descendant
	assert.Equal(t, 1, child1.GetTotalDescendants())

	// Child2 has no children = 0 total descendants
	assert.Equal(t, 0, child2.GetTotalDescendants())
}

func TestSpanStatus_JSONSerialization(t *testing.T) {
	status := SpanStatus{
		Code:    "ERROR",
		Message: "Internal server error",
	}

	jsonData, err := json.Marshal(status)
	require.NoError(t, err)

	var newStatus SpanStatus
	err = json.Unmarshal(jsonData, &newStatus)
	require.NoError(t, err)

	assert.Equal(t, status.Code, newStatus.Code)
	assert.Equal(t, status.Message, newStatus.Message)
}

func TestSpanEvent_JSONSerialization(t *testing.T) {
	event := SpanEvent{
		Name:      "exception",
		Timestamp: 1000000000,
		Attributes: map[string]interface{}{
			"exception.type":    "RuntimeError",
			"exception.message": "Something went wrong",
		},
	}

	jsonData, err := json.Marshal(event)
	require.NoError(t, err)

	var newEvent SpanEvent
	err = json.Unmarshal(jsonData, &newEvent)
	require.NoError(t, err)

	assert.Equal(t, event.Name, newEvent.Name)
	assert.Equal(t, event.Timestamp, newEvent.Timestamp)
	assert.NotNil(t, newEvent.Attributes)
	assert.Len(t, newEvent.Attributes, 2)
}

func TestTraceData_JSONSerialization(t *testing.T) {
	rootSpan := &Span{
		SpanID:    "root",
		TraceID:   "trace1",
		Name:      "root_operation",
		StartTime: 1000000000,
		EndTime:   2000000000,
		Status:    SpanStatus{Code: "OK"},
		Attributes: map[string]interface{}{
			"service.name": "test-service",
		},
	}

	traceData := TraceData{
		TraceID:  "trace1",
		RootSpan: rootSpan,
		Spans: map[string]*Span{
			"root": rootSpan,
		},
		SpanTree: &SpanNode{
			Span:     rootSpan,
			Children: []*SpanNode{},
		},
	}

	jsonData, err := json.Marshal(traceData)
	require.NoError(t, err)

	var newTraceData TraceData
	err = json.Unmarshal(jsonData, &newTraceData)
	require.NoError(t, err)

	assert.Equal(t, traceData.TraceID, newTraceData.TraceID)
	assert.NotNil(t, newTraceData.RootSpan)
	assert.Equal(t, "root", newTraceData.RootSpan.SpanID)
	assert.NotNil(t, newTraceData.Spans)
	assert.Len(t, newTraceData.Spans, 1)
	assert.NotNil(t, newTraceData.SpanTree)
}

// Tests for AlignmentReport-related data structures

func TestAlignmentStatus_String(t *testing.T) {
	assert.Equal(t, "SUCCESS", StatusSuccess.String())
	assert.Equal(t, "FAILED", StatusFailed.String())
	assert.Equal(t, "SKIPPED", StatusSkipped.String())
}

func TestAlignmentStatus_IsValid(t *testing.T) {
	assert.True(t, StatusSuccess.IsValid())
	assert.True(t, StatusFailed.IsValid())
	assert.True(t, StatusSkipped.IsValid())
	assert.False(t, AlignmentStatus("INVALID").IsValid())
}

func TestValidationDetail_IsPassed(t *testing.T) {
	passedDetail := ValidationDetail{
		Expected: "OK",
		Actual:   "OK",
	}
	assert.True(t, passedDetail.IsPassed())

	failedDetail := ValidationDetail{
		Expected: "OK",
		Actual:   "ERROR",
	}
	assert.False(t, failedDetail.IsPassed())
}

func TestValidationDetail_String(t *testing.T) {
	detail := ValidationDetail{
		Type:       "precondition",
		Expression: "span.status == 'OK'",
		Expected:   "OK",
		Actual:     "ERROR",
	}

	str := detail.String()
	assert.Contains(t, str, "precondition")
	assert.Contains(t, str, "span.status == 'OK'")
	assert.Contains(t, str, "OK")
	assert.Contains(t, str, "ERROR")
}

func TestAlignmentResult_AddValidationDetail(t *testing.T) {
	result := NewAlignmentResult("testOp")

	// Initially should be skipped
	assert.Equal(t, StatusSkipped, result.Status)

	// Add a passing validation detail
	passingDetail := ValidationDetail{
		Type:     "precondition",
		Expected: "OK",
		Actual:   "OK",
	}
	result.AddValidationDetail(passingDetail)

	assert.Equal(t, StatusSuccess, result.Status)
	assert.Len(t, result.Details, 1)

	// Add a failing validation detail
	failingDetail := ValidationDetail{
		Type:     "postcondition",
		Expected: 200,
		Actual:   500,
	}
	result.AddValidationDetail(failingDetail)

	assert.Equal(t, StatusFailed, result.Status)
	assert.Len(t, result.Details, 2)
}

func TestAlignmentResult_GetFailedDetails(t *testing.T) {
	result := NewAlignmentResult("testOp")

	passingDetail := ValidationDetail{Expected: "OK", Actual: "OK"}
	failingDetail1 := ValidationDetail{Expected: 200, Actual: 500}
	failingDetail2 := ValidationDetail{Expected: "success", Actual: "error"}

	result.AddValidationDetail(passingDetail)
	result.AddValidationDetail(failingDetail1)
	result.AddValidationDetail(failingDetail2)

	failedDetails := result.GetFailedDetails()
	assert.Len(t, failedDetails, 2)
}

func TestAlignmentResult_GetPreconditionDetails(t *testing.T) {
	result := NewAlignmentResult("testOp")

	precondition1 := ValidationDetail{Type: "precondition", Expected: "OK", Actual: "OK"}
	precondition2 := ValidationDetail{Type: "precondition", Expected: 200, Actual: 200}
	postcondition := ValidationDetail{Type: "postcondition", Expected: "success", Actual: "success"}

	result.AddValidationDetail(precondition1)
	result.AddValidationDetail(precondition2)
	result.AddValidationDetail(postcondition)

	preconditions := result.GetPreconditionDetails()
	assert.Len(t, preconditions, 2)
	for _, detail := range preconditions {
		assert.Equal(t, "precondition", detail.Type)
	}
}

func TestAlignmentResult_GetPostconditionDetails(t *testing.T) {
	result := NewAlignmentResult("testOp")

	precondition := ValidationDetail{Type: "precondition", Expected: "OK", Actual: "OK"}
	postcondition1 := ValidationDetail{Type: "postcondition", Expected: 200, Actual: 200}
	postcondition2 := ValidationDetail{Type: "postcondition", Expected: "success", Actual: "success"}

	result.AddValidationDetail(precondition)
	result.AddValidationDetail(postcondition1)
	result.AddValidationDetail(postcondition2)

	postconditions := result.GetPostconditionDetails()
	assert.Len(t, postconditions, 2)
	for _, detail := range postconditions {
		assert.Equal(t, "postcondition", detail.Type)
	}
}

func TestAlignmentReport_AddResult(t *testing.T) {
	report := NewAlignmentReport()

	// Initially empty
	assert.Equal(t, 0, report.Summary.Total)

	// Add a successful result
	successResult := NewAlignmentResult("op1")
	successResult.Status = StatusSuccess
	report.AddResult(*successResult)

	assert.Equal(t, 1, report.Summary.Total)
	assert.Equal(t, 1, report.Summary.Success)
	assert.Equal(t, 0, report.Summary.Failed)
	assert.Equal(t, 0, report.Summary.Skipped)

	// Add a failed result
	failedResult := NewAlignmentResult("op2")
	failedResult.Status = StatusFailed
	report.AddResult(*failedResult)

	assert.Equal(t, 2, report.Summary.Total)
	assert.Equal(t, 1, report.Summary.Success)
	assert.Equal(t, 1, report.Summary.Failed)
	assert.Equal(t, 0, report.Summary.Skipped)

	// Add a skipped result
	skippedResult := NewAlignmentResult("op3")
	skippedResult.Status = StatusSkipped
	report.AddResult(*skippedResult)

	assert.Equal(t, 3, report.Summary.Total)
	assert.Equal(t, 1, report.Summary.Success)
	assert.Equal(t, 1, report.Summary.Failed)
	assert.Equal(t, 1, report.Summary.Skipped)
}

func TestAlignmentReport_HasFailures(t *testing.T) {
	report := NewAlignmentReport()

	// No failures initially
	assert.False(t, report.HasFailures())

	// Add successful result - still no failures
	successResult := NewAlignmentResult("op1")
	successResult.Status = StatusSuccess
	report.AddResult(*successResult)
	assert.False(t, report.HasFailures())

	// Add failed result - now has failures
	failedResult := NewAlignmentResult("op2")
	failedResult.Status = StatusFailed
	report.AddResult(*failedResult)
	assert.True(t, report.HasFailures())
}

func TestAlignmentReport_GetSuccessRate(t *testing.T) {
	report := NewAlignmentReport()

	// Empty report should have 0.0 success rate
	assert.Equal(t, 0.0, report.GetSuccessRate())

	// Add 2 successful, 1 failed
	for i := 0; i < 2; i++ {
		result := NewAlignmentResult(fmt.Sprintf("success%d", i))
		result.Status = StatusSuccess
		report.AddResult(*result)
	}

	failedResult := NewAlignmentResult("failed")
	failedResult.Status = StatusFailed
	report.AddResult(*failedResult)

	// Success rate should be 2/3 ≈ 0.667
	successRate := report.GetSuccessRate()
	assert.InDelta(t, 0.667, successRate, 0.001)
}

func TestAlignmentReport_GetFailureRate(t *testing.T) {
	report := NewAlignmentReport()

	// Empty report should have 0.0 failure rate
	assert.Equal(t, 0.0, report.GetFailureRate())

	// Add 1 successful, 2 failed
	successResult := NewAlignmentResult("success")
	successResult.Status = StatusSuccess
	report.AddResult(*successResult)

	for i := 0; i < 2; i++ {
		result := NewAlignmentResult(fmt.Sprintf("failed%d", i))
		result.Status = StatusFailed
		report.AddResult(*result)
	}

	// Failure rate should be 2/3 ≈ 0.667
	failureRate := report.GetFailureRate()
	assert.InDelta(t, 0.667, failureRate, 0.001)
}

func TestAlignmentReport_JSONSerialization(t *testing.T) {
	report := NewAlignmentReport()

	// Create a result with validation details
	result := NewAlignmentResult("testOp")
	result.ExecutionTime = 1000000 // 1ms in nanoseconds

	detail := ValidationDetail{
		Type:       "precondition",
		Expression: "span.status == 'OK'",
		Expected:   "OK",
		Actual:     "ERROR",
		Message:    "Status check failed",
	}
	result.AddValidationDetail(detail)

	report.AddResult(*result)

	// Test JSON serialization
	jsonData, err := json.Marshal(report)
	require.NoError(t, err)

	// Test JSON deserialization
	var newReport AlignmentReport
	err = json.Unmarshal(jsonData, &newReport)
	require.NoError(t, err)

	assert.Equal(t, report.Summary.Total, newReport.Summary.Total)
	assert.Equal(t, report.Summary.Failed, newReport.Summary.Failed)
	assert.Len(t, newReport.Results, 1)
	assert.Equal(t, "testOp", newReport.Results[0].SpecOperationID)
	assert.Equal(t, StatusFailed, newReport.Results[0].Status)
	assert.Len(t, newReport.Results[0].Details, 1)
	assert.Equal(t, "precondition", newReport.Results[0].Details[0].Type)
}

func TestNewAlignmentReport(t *testing.T) {
	report := NewAlignmentReport()

	assert.NotNil(t, report)
	assert.Equal(t, 0, report.Summary.Total)
	assert.Len(t, report.Results, 0)
}

func TestNewAlignmentResult(t *testing.T) {
	result := NewAlignmentResult("testOp")

	assert.NotNil(t, result)
	assert.Equal(t, "testOp", result.SpecOperationID)
	assert.Equal(t, StatusSkipped, result.Status)
	assert.Len(t, result.Details, 0)
	assert.Equal(t, int64(0), result.ExecutionTime)
}

func TestNewValidationDetail(t *testing.T) {
	detail := NewValidationDetail("precondition", "span.status == 'OK'", "OK", "ERROR", "Status mismatch")

	assert.NotNil(t, detail)
	assert.Equal(t, "precondition", detail.Type)
	assert.Equal(t, "span.status == 'OK'", detail.Expression)
	assert.Equal(t, "OK", detail.Expected)
	assert.Equal(t, "ERROR", detail.Actual)
	assert.Equal(t, "Status mismatch", detail.Message)
}

func TestAlignmentReport_ComplexScenario(t *testing.T) {
	report := NewAlignmentReport()

	// Create multiple results with different outcomes

	// Successful operation
	successResult := NewAlignmentResult("createUser")
	successResult.ExecutionTime = 500000 // 0.5ms
	successResult.AddValidationDetail(ValidationDetail{
		Type: "precondition", Expected: "POST", Actual: "POST",
	})
	successResult.AddValidationDetail(ValidationDetail{
		Type: "postcondition", Expected: 201, Actual: 201,
	})
	report.AddResult(*successResult)

	// Failed operation
	failedResult := NewAlignmentResult("updateUser")
	failedResult.ExecutionTime = 750000 // 0.75ms
	failedResult.AddValidationDetail(ValidationDetail{
		Type: "precondition", Expected: "PUT", Actual: "PUT",
	})
	failedResult.AddValidationDetail(ValidationDetail{
		Type: "postcondition", Expected: 200, Actual: 500, // This will cause failure
	})
	report.AddResult(*failedResult)

	// Skipped operation (no validation details)
	skippedResult := NewAlignmentResult("deleteUser")
	report.AddResult(*skippedResult)

	// Verify summary
	assert.Equal(t, 3, report.Summary.Total)
	assert.Equal(t, 1, report.Summary.Success)
	assert.Equal(t, 1, report.Summary.Failed)
	assert.Equal(t, 1, report.Summary.Skipped)

	// Verify rates
	assert.InDelta(t, 0.333, report.GetSuccessRate(), 0.001)
	assert.InDelta(t, 0.333, report.GetFailureRate(), 0.001)

	// Verify has failures
	assert.True(t, report.HasFailures())

	// Verify individual results
	assert.Equal(t, StatusSuccess, report.Results[0].Status)
	assert.Equal(t, StatusFailed, report.Results[1].Status)
	assert.Equal(t, StatusSkipped, report.Results[2].Status)

	// Verify failed details
	failedDetails := report.Results[1].GetFailedDetails()
	assert.Len(t, failedDetails, 1)
	assert.Equal(t, "postcondition", failedDetails[0].Type)
}