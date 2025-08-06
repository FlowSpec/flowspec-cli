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
	"fmt"
	"testing"
	"time"

	"github.com/flowspec/flowspec-cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTraceDataOrganization_SpanTreeBuilding(t *testing.T) {
	// Create trace data with hierarchical spans
	traceData := createHierarchicalTraceData()

	err := traceData.BuildSpanTree()
	require.NoError(t, err)

	// Verify root span identification
	assert.NotNil(t, traceData.RootSpan)
	assert.Equal(t, "root-span", traceData.RootSpan.SpanID)
	assert.Equal(t, "", traceData.RootSpan.ParentID)

	// Verify span tree structure
	assert.NotNil(t, traceData.SpanTree)
	assert.Equal(t, traceData.RootSpan.SpanID, traceData.SpanTree.Span.SpanID)

	// Root should have 2 direct children
	assert.Len(t, traceData.SpanTree.Children, 2)

	// Verify child relationships
	child1 := traceData.SpanTree.Children[0]
	child2 := traceData.SpanTree.Children[1]

	assert.Equal(t, "root-span", child1.Span.ParentID)
	assert.Equal(t, "root-span", child2.Span.ParentID)

	// One child should have its own child (grandchild of root)
	var grandchildFound bool
	for _, child := range traceData.SpanTree.Children {
		if len(child.Children) > 0 {
			grandchildFound = true
			grandchild := child.Children[0]
			assert.Equal(t, child.Span.SpanID, grandchild.Span.ParentID)
		}
	}
	assert.True(t, grandchildFound, "Should find at least one grandchild span")
}

func TestTraceDataOrganization_SpanTreeWithMultipleLevels(t *testing.T) {
	// Create deep hierarchy: root -> level1 -> level2 -> level3
	traceData := createDeepHierarchyTraceData()

	err := traceData.BuildSpanTree()
	require.NoError(t, err)

	// Verify 4-level hierarchy
	assert.NotNil(t, traceData.SpanTree)

	// Level 0 (root)
	root := traceData.SpanTree
	assert.Equal(t, "level0-span", root.Span.SpanID)
	assert.Len(t, root.Children, 1)

	// Level 1
	level1 := root.Children[0]
	assert.Equal(t, "level1-span", level1.Span.SpanID)
	assert.Equal(t, "level0-span", level1.Span.ParentID)
	assert.Len(t, level1.Children, 1)

	// Level 2
	level2 := level1.Children[0]
	assert.Equal(t, "level2-span", level2.Span.SpanID)
	assert.Equal(t, "level1-span", level2.Span.ParentID)
	assert.Len(t, level2.Children, 1)

	// Level 3 (leaf)
	level3 := level2.Children[0]
	assert.Equal(t, "level3-span", level3.Span.SpanID)
	assert.Equal(t, "level2-span", level3.Span.ParentID)
	assert.Len(t, level3.Children, 0) // Leaf node
}

func TestTraceDataOrganization_NoRootSpan(t *testing.T) {
	// Create trace data where all spans have parents (circular or missing root)
	traceData := &models.TraceData{
		TraceID: "no-root-trace",
		Spans: map[string]*models.Span{
			"span1": {
				SpanID:   "span1",
				TraceID:  "no-root-trace",
				ParentID: "span2", // Points to span2
				Name:     "span1",
			},
			"span2": {
				SpanID:   "span2",
				TraceID:  "no-root-trace",
				ParentID: "span1", // Points to span1 (circular)
				Name:     "span2",
			},
		},
	}

	err := traceData.BuildSpanTree()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no root span found")
}

func TestTraceDataOrganization_EmptyTrace(t *testing.T) {
	traceData := &models.TraceData{
		TraceID: "empty-trace",
		Spans:   map[string]*models.Span{},
	}

	// Building a tree from no spans should not produce an error.
	err := traceData.BuildSpanTree()
	assert.NoError(t, err)
	assert.Nil(t, traceData.RootSpan, "RootSpan should be nil for an empty trace")
	assert.Nil(t, traceData.SpanTree, "SpanTree should be nil for an empty trace")
}

func TestTraceIndexing_SpanIDIndex(t *testing.T) {
	store := NewTraceStore()
	traceData := createTestTraceDataForIndexing()

	store.SetTraceData(traceData)

	// Test span ID lookups
	testCases := []struct {
		spanID   string
		expected bool
		spanName string
	}{
		{"service-a-span", true, "service-a-operation"},
		{"service-b-span", true, "service-b-operation"},
		{"database-span", true, "database-query"},
		{"nonexistent-span", false, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.spanID, func(t *testing.T) {
			span := store.FindSpanByID(tc.spanID)
			if tc.expected {
				assert.NotNil(t, span)
				assert.Equal(t, tc.spanName, span.Name)
				assert.Equal(t, tc.spanID, span.SpanID)
			} else {
				assert.Nil(t, span)
			}
		})
	}
}

func TestTraceIndexing_NameIndex(t *testing.T) {
	store := NewTraceStore()
	traceData := createTestTraceDataForIndexing()

	store.SetTraceData(traceData)

	// Test name-based lookups
	testCases := []struct {
		name          string
		expectedCount int
		expectedSpans []string
	}{
		{"service-a-operation", 1, []string{"service-a-span"}},
		{"service-b-operation", 1, []string{"service-b-span"}},
		{"database-query", 2, []string{"database-span", "database-span-2"}}, // Multiple spans with same name
		{"nonexistent-operation", 0, []string{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			spans := store.FindSpansByName(tc.name)
			assert.Len(t, spans, tc.expectedCount)

			if tc.expectedCount > 0 {
				spanIDs := make([]string, len(spans))
				for i, span := range spans {
					spanIDs[i] = span.SpanID
				}

				for _, expectedSpanID := range tc.expectedSpans {
					assert.Contains(t, spanIDs, expectedSpanID)
				}
			}
		})
	}
}

func TestTraceIndexing_OperationIDIndex(t *testing.T) {
	store := NewTraceStore()
	traceData := createTestTraceDataForIndexing()

	store.SetTraceData(traceData)

	// Test operation ID lookups
	testCases := []struct {
		operationID   string
		expectedCount int
		expectedSpans []string
	}{
		{"createUser", 1, []string{"service-a-span"}},
		{"processOrder", 1, []string{"service-b-span"}},
		{"queryDatabase", 2, []string{"database-span", "database-span-2"}}, // Multiple spans with same operation
		{"nonexistentOp", 0, []string{}},
	}

	for _, tc := range testCases {
		t.Run(tc.operationID, func(t *testing.T) {
			spans := store.FindSpansByOperationID(tc.operationID)
			assert.Len(t, spans, tc.expectedCount)

			if tc.expectedCount > 0 {
				spanIDs := make([]string, len(spans))
				for i, span := range spans {
					spanIDs[i] = span.SpanID
				}

				for _, expectedSpanID := range tc.expectedSpans {
					assert.Contains(t, spanIDs, expectedSpanID)
				}
			}
		})
	}
}

func TestTraceIndexing_IndexRebuildOnDataChange(t *testing.T) {
	store := NewTraceStore()

	// Set initial data
	initialData := createTestTraceDataForIndexing()
	store.SetTraceData(initialData)

	// Verify initial state
	assert.Equal(t, 5, store.GetSpanCount())
	assert.NotNil(t, store.FindSpanByID("service-a-span"))

	// Change data
	newData := &models.TraceData{
		TraceID: "new-trace",
		Spans: map[string]*models.Span{
			"new-span": {
				SpanID:  "new-span",
				TraceID: "new-trace",
				Name:    "new-operation",
				Attributes: map[string]interface{}{
					"operation.id": "newOp",
				},
			},
		},
	}

	store.SetTraceData(newData)

	// Verify indexes were rebuilt
	assert.Equal(t, 1, store.GetSpanCount())
	assert.Nil(t, store.FindSpanByID("service-a-span")) // Old span should be gone
	assert.NotNil(t, store.FindSpanByID("new-span"))    // New span should be found

	// Verify new indexes work
	spans := store.FindSpansByName("new-operation")
	assert.Len(t, spans, 1)

	opSpans := store.FindSpansByOperationID("newOp")
	assert.Len(t, opSpans, 1)
}

func TestTraceIndexing_ConcurrentAccess(t *testing.T) {
	store := NewTraceStore()
	traceData := createTestTraceDataForIndexing()
	store.SetTraceData(traceData)

	// Test concurrent reads
	done := make(chan bool, 10)

	// Start multiple goroutines doing concurrent reads
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Perform various read operations
			for j := 0; j < 100; j++ {
				_ = store.FindSpanByID("service-a-span")
				_ = store.FindSpansByName("service-a-operation")
				_ = store.FindSpansByOperationID("createUser")
				_ = store.GetAllSpans()
				_ = store.GetSpanCount()
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify data integrity after concurrent access
	assert.Equal(t, 5, store.GetSpanCount())
	assert.NotNil(t, store.FindSpanByID("service-a-span"))
}

func TestTraceIndexing_QueryPerformance(t *testing.T) {
	store := NewTraceStore()

	// Create large trace data for performance testing
	largeTraceData := createLargeTraceDataForIndexing(1000) // 1000 spans
	store.SetTraceData(largeTraceData)

	// Measure query performance
	start := time.Now()

	// Perform many queries
	for i := 0; i < 1000; i++ {
		spanID := fmt.Sprintf("span-%d", i%1000)
		span := store.FindSpanByID(spanID)
		assert.NotNil(t, span)
	}

	duration := time.Since(start)

	// Should be able to perform 1000 queries quickly
	assert.Less(t, duration, 100*time.Millisecond, "1000 span ID queries should complete within 100ms")

	// Test name-based queries
	start = time.Now()
	for i := 0; i < 100; i++ {
		operationName := fmt.Sprintf("operation-%d", i%10) // 10 different operation names
		spans := store.FindSpansByName(operationName)
		assert.NotEmpty(t, spans)
	}
	duration = time.Since(start)

	assert.Less(t, duration, 50*time.Millisecond, "100 name queries should complete within 50ms")
}

func TestSpanTreeUtilities(t *testing.T) {
	traceData := createHierarchicalTraceData()
	err := traceData.BuildSpanTree()
	require.NoError(t, err)

	// Test SpanNode utilities
	root := traceData.SpanTree

	// Test child count
	assert.Equal(t, 2, root.GetChildCount())

	// Test total descendants
	totalDescendants := root.GetTotalDescendants()
	assert.Equal(t, 3, totalDescendants) // 2 direct children + 1 grandchild

	// Test on leaf nodes
	for _, child := range root.Children {
		if len(child.Children) == 0 {
			assert.Equal(t, 0, child.GetChildCount())
			assert.Equal(t, 0, child.GetTotalDescendants())
		}
	}
}

func TestTraceDataQueries(t *testing.T) {
	traceData := createTestTraceDataForIndexing()

	// Test FindSpanByID
	span := traceData.FindSpanByID("service-a-span")
	assert.NotNil(t, span)
	assert.Equal(t, "service-a-operation", span.Name)

	// Test FindSpansByName
	spans := traceData.FindSpansByName("database-query")
	assert.Len(t, spans, 2) // Two database spans with same name

	// Test GetAllSpans
	allSpans := traceData.GetAllSpans()
	assert.Len(t, allSpans, 5)

	// Verify all spans are included
	spanIDs := make(map[string]bool)
	for _, span := range allSpans {
		spanIDs[span.SpanID] = true
	}

	expectedSpanIDs := []string{"root-span", "service-a-span", "service-b-span", "database-span", "database-span-2"}
	for _, expectedID := range expectedSpanIDs {
		assert.True(t, spanIDs[expectedID], "Should find span ID: %s", expectedID)
	}
}

// Helper functions to create test data

func createHierarchicalTraceData() *models.TraceData {
	return &models.TraceData{
		TraceID: "hierarchical-trace",
		Spans: map[string]*models.Span{
			"root-span": {
				SpanID:    "root-span",
				TraceID:   "hierarchical-trace",
				ParentID:  "", // Root span
				Name:      "root-operation",
				StartTime: 1640995200000000000,
				EndTime:   1640995205000000000,
			},
			"child-span-1": {
				SpanID:    "child-span-1",
				TraceID:   "hierarchical-trace",
				ParentID:  "root-span",
				Name:      "child-operation-1",
				StartTime: 1640995201000000000,
				EndTime:   1640995202000000000,
			},
			"child-span-2": {
				SpanID:    "child-span-2",
				TraceID:   "hierarchical-trace",
				ParentID:  "root-span",
				Name:      "child-operation-2",
				StartTime: 1640995202000000000,
				EndTime:   1640995203000000000,
			},
			"grandchild-span": {
				SpanID:    "grandchild-span",
				TraceID:   "hierarchical-trace",
				ParentID:  "child-span-1",
				Name:      "grandchild-operation",
				StartTime: 1640995201200000000,
				EndTime:   1640995201800000000,
			},
		},
	}
}

func createDeepHierarchyTraceData() *models.TraceData {
	return &models.TraceData{
		TraceID: "deep-hierarchy-trace",
		Spans: map[string]*models.Span{
			"level0-span": {
				SpanID:   "level0-span",
				TraceID:  "deep-hierarchy-trace",
				ParentID: "", // Root
				Name:     "level0-operation",
			},
			"level1-span": {
				SpanID:   "level1-span",
				TraceID:  "deep-hierarchy-trace",
				ParentID: "level0-span",
				Name:     "level1-operation",
			},
			"level2-span": {
				SpanID:   "level2-span",
				TraceID:  "deep-hierarchy-trace",
				ParentID: "level1-span",
				Name:     "level2-operation",
			},
			"level3-span": {
				SpanID:   "level3-span",
				TraceID:  "deep-hierarchy-trace",
				ParentID: "level2-span",
				Name:     "level3-operation",
			},
		},
	}
}

func createTestTraceDataForIndexing() *models.TraceData {
	return &models.TraceData{
		TraceID: "indexing-test-trace",
		Spans: map[string]*models.Span{
			"root-span": {
				SpanID:   "root-span",
				TraceID:  "indexing-test-trace",
				ParentID: "",
				Name:     "root-operation",
				Attributes: map[string]interface{}{
					"operation.id": "rootOp",
				},
			},
			"service-a-span": {
				SpanID:   "service-a-span",
				TraceID:  "indexing-test-trace",
				ParentID: "root-span",
				Name:     "service-a-operation",
				Attributes: map[string]interface{}{
					"operation.id": "createUser",
					"service.name": "user-service",
				},
			},
			"service-b-span": {
				SpanID:   "service-b-span",
				TraceID:  "indexing-test-trace",
				ParentID: "root-span",
				Name:     "service-b-operation",
				Attributes: map[string]interface{}{
					"operation.id": "processOrder",
					"service.name": "order-service",
				},
			},
			"database-span": {
				SpanID:   "database-span",
				TraceID:  "indexing-test-trace",
				ParentID: "service-a-span",
				Name:     "database-query",
				Attributes: map[string]interface{}{
					"operation.id": "queryDatabase",
					"db.system":    "postgresql",
				},
			},
			"database-span-2": {
				SpanID:   "database-span-2",
				TraceID:  "indexing-test-trace",
				ParentID: "service-b-span",
				Name:     "database-query", // Same name as database-span
				Attributes: map[string]interface{}{
					"operation.id": "queryDatabase", // Same operation ID
					"db.system":    "postgresql",
				},
			},
		},
	}
}

func createLargeTraceDataForIndexing(numSpans int) *models.TraceData {
	spans := make(map[string]*models.Span)

	// Create root span
	spans["root"] = &models.Span{
		SpanID:   "root",
		TraceID:  "large-trace",
		ParentID: "",
		Name:     "root-operation",
		Attributes: map[string]interface{}{
			"operation.id": "rootOp",
		},
	}

	// Create many child spans
	for i := 0; i < numSpans; i++ {
		spanID := fmt.Sprintf("span-%d", i)
		operationName := fmt.Sprintf("operation-%d", i%10) // 10 different operation names
		operationID := fmt.Sprintf("op-%d", i%20)          // 20 different operation IDs

		spans[spanID] = &models.Span{
			SpanID:   spanID,
			TraceID:  "large-trace",
			ParentID: "root",
			Name:     operationName,
			Attributes: map[string]interface{}{
				"operation.id": operationID,
				"span.index":   i,
			},
		}
	}

	return &models.TraceData{
		TraceID: "large-trace",
		Spans:   spans,
	}
}
