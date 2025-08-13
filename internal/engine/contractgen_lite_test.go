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
	"testing"
	"time"

	"github.com/flowspec/flowspec-cli/internal/ingestor"
	"github.com/flowspec/flowspec-cli/internal/ingestor/traffic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultGenerationOptions(t *testing.T) {
	options := DefaultGenerationOptions()

	assert.NotNil(t, options)
	assert.Equal(t, 0.8, options.PathClusteringThreshold)
	assert.Equal(t, 20, options.MinSampleSize)
	assert.Equal(t, 0.95, options.RequiredFieldThreshold)
	assert.Equal(t, 5, options.MinEndpointSamples)
	assert.Equal(t, "auto", options.StatusAggregation)
	assert.Equal(t, 10000, options.MaxUniqueValues)
	assert.Equal(t, "generated-service", options.ServiceName)
	assert.Equal(t, "v1.0.0", options.ServiceVersion)
}

func TestNewContractGeneratorLite(t *testing.T) {
	generator := NewContractGeneratorLite()

	assert.NotNil(t, generator)
	assert.NotNil(t, generator.options)
	assert.Equal(t, DefaultGenerationOptions(), generator.options)
}

func TestContractGeneratorLite_SetOptions(t *testing.T) {
	generator := NewContractGeneratorLite()
	
	customOptions := &GenerationOptions{
		PathClusteringThreshold: 0.9,
		MinSampleSize:          30,
		ServiceName:            "test-service",
	}

	generator.SetOptions(customOptions)
	assert.Equal(t, customOptions, generator.options)

	// Test with nil options (should not crash)
	generator.SetOptions(nil)
	assert.Equal(t, customOptions, generator.options) // Should remain unchanged
}

func TestNewOperationPattern(t *testing.T) {
	pattern := NewOperationPattern("GET")

	assert.NotNil(t, pattern)
	assert.Equal(t, "GET", pattern.Method)
	assert.Empty(t, pattern.StatusCodes)
	assert.Empty(t, pattern.StatusRanges)
	assert.Empty(t, pattern.RequiredQuery)
	assert.Empty(t, pattern.RequiredHeaders)
	assert.Empty(t, pattern.OptionalQuery)
	assert.Empty(t, pattern.OptionalHeaders)
	assert.Equal(t, 0, pattern.SampleCount)
	assert.True(t, pattern.FirstSeen.IsZero())
	assert.True(t, pattern.LastSeen.IsZero())
	assert.NotNil(t, pattern.queryFieldCounts)
	assert.NotNil(t, pattern.headerFieldCounts)
}

func TestOperationPattern_AddRecord(t *testing.T) {
	pattern := NewOperationPattern("GET")
	
	timestamp1 := time.Now()
	timestamp2 := timestamp1.Add(1 * time.Hour)
	timestamp3 := timestamp1.Add(-1 * time.Hour) // Earlier than timestamp1

	record1 := &traffic.NormalizedRecord{
		Method:    "GET",
		Status:    200,
		Timestamp: timestamp1,
		Query: map[string][]string{
			"id":     {"123"},
			"format": {"json"},
		},
		Headers: map[string][]string{
			"authorization": {"Bearer token"},
			"accept":        {"application/json"},
		},
	}

	record2 := &traffic.NormalizedRecord{
		Method:    "GET",
		Status:    404,
		Timestamp: timestamp2,
		Query: map[string][]string{
			"id": {"456"},
		},
		Headers: map[string][]string{
			"authorization": {"Bearer token2"},
		},
	}

	record3 := &traffic.NormalizedRecord{
		Method:    "GET",
		Status:    200, // Duplicate status
		Timestamp: timestamp3,
		Query: map[string][]string{
			"id":     {"789"},
			"format": {"xml"},
		},
		Headers: map[string][]string{
			"authorization": {"Bearer token3"},
			"user-agent":    {"Mozilla/5.0"},
		},
	}

	// Add first record
	pattern.AddRecord(record1)
	assert.Equal(t, 1, pattern.SampleCount)
	assert.Equal(t, []int{200}, pattern.StatusCodes)
	assert.Equal(t, timestamp1, pattern.FirstSeen)
	assert.Equal(t, timestamp1, pattern.LastSeen)
	assert.Equal(t, 1, pattern.queryFieldCounts["id"])
	assert.Equal(t, 1, pattern.queryFieldCounts["format"])
	assert.Equal(t, 1, pattern.headerFieldCounts["authorization"])
	assert.Equal(t, 1, pattern.headerFieldCounts["accept"])

	// Add second record
	pattern.AddRecord(record2)
	assert.Equal(t, 2, pattern.SampleCount)
	assert.Equal(t, []int{200, 404}, pattern.StatusCodes)
	assert.Equal(t, timestamp1, pattern.FirstSeen) // Should remain the same
	assert.Equal(t, timestamp2, pattern.LastSeen)  // Should update to later time
	assert.Equal(t, 2, pattern.queryFieldCounts["id"])
	assert.Equal(t, 1, pattern.queryFieldCounts["format"]) // Only in record1
	assert.Equal(t, 2, pattern.headerFieldCounts["authorization"])
	assert.Equal(t, 1, pattern.headerFieldCounts["accept"]) // Only in record1

	// Add third record (with earlier timestamp)
	pattern.AddRecord(record3)
	assert.Equal(t, 3, pattern.SampleCount)
	assert.Equal(t, []int{200, 404}, pattern.StatusCodes) // 200 already exists, shouldn't duplicate
	assert.Equal(t, timestamp3, pattern.FirstSeen)        // Should update to earlier time
	assert.Equal(t, timestamp2, pattern.LastSeen)         // Should remain the latest
	assert.Equal(t, 3, pattern.queryFieldCounts["id"])
	assert.Equal(t, 2, pattern.queryFieldCounts["format"])
	assert.Equal(t, 3, pattern.headerFieldCounts["authorization"])
	assert.Equal(t, 1, pattern.headerFieldCounts["accept"])
	assert.Equal(t, 1, pattern.headerFieldCounts["user-agent"])
}

func TestOperationPattern_FinalizeFields(t *testing.T) {
	pattern := NewOperationPattern("GET")
	
	// Simulate field counts
	pattern.SampleCount = 100
	pattern.queryFieldCounts = map[string]int{
		"id":       100, // 100% - should be required
		"format":   95,  // 95% - should be required (threshold is 95%)
		"optional": 50,  // 50% - should be optional
		"rare":     5,   // 5% - should be optional
	}
	pattern.headerFieldCounts = map[string]int{
		"authorization": 98,  // 98% - should be required
		"accept":        94,  // 94% - should be optional (below 95% threshold)
		"user-agent":    30,  // 30% - should be optional
	}

	pattern.FinalizeFields(0.95) // 95% threshold

	// Check required fields
	assert.Contains(t, pattern.RequiredQuery, "id")
	assert.Contains(t, pattern.RequiredQuery, "format")
	assert.NotContains(t, pattern.RequiredQuery, "optional")
	assert.NotContains(t, pattern.RequiredQuery, "rare")

	assert.Contains(t, pattern.RequiredHeaders, "authorization")
	assert.NotContains(t, pattern.RequiredHeaders, "accept")
	assert.NotContains(t, pattern.RequiredHeaders, "user-agent")

	// Check optional fields
	assert.Contains(t, pattern.OptionalQuery, "optional")
	assert.Contains(t, pattern.OptionalQuery, "rare")
	assert.NotContains(t, pattern.OptionalQuery, "id")
	assert.NotContains(t, pattern.OptionalQuery, "format")

	assert.Contains(t, pattern.OptionalHeaders, "accept")
	assert.Contains(t, pattern.OptionalHeaders, "user-agent")
	assert.NotContains(t, pattern.OptionalHeaders, "authorization")
}

func TestContractGeneratorLite_splitPath(t *testing.T) {
	generator := NewContractGeneratorLite()

	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Simple path",
			input:    "/api/users",
			expected: []string{"api", "users"},
		},
		{
			name:     "Root path",
			input:    "/",
			expected: []string{},
		},
		{
			name:     "Empty path",
			input:    "",
			expected: []string{},
		},
		{
			name:     "Path without leading slash",
			input:    "api/users",
			expected: []string{"api", "users"},
		},
		{
			name:     "Complex path",
			input:    "/api/v1/users/123/profile",
			expected: []string{"api", "v1", "users", "123", "profile"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := generator.splitPath(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestContractGeneratorLite_isParameter(t *testing.T) {
	generator := NewContractGeneratorLite()

	testCases := []struct {
		input    string
		expected bool
	}{
		{"{id}", true},
		{"{var}", true},
		{"{num}", true},
		{"api", false},
		{"users", false},
		{"{", false},
		{"}", false},
		{"{incomplete", false},
		{"incomplete}", false},
		{"", false},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := generator.isParameter(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestContractGeneratorLite_isNumeric(t *testing.T) {
	generator := NewContractGeneratorLite()

	testCases := []struct {
		input    string
		expected bool
	}{
		{"123", true},
		{"0", true},
		{"999999", true},
		{"abc", false},
		{"12a", false},
		{"a12", false},
		{"", false},
		{"12.34", false}, // Contains dot
		{"-123", false},  // Contains minus
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := generator.isNumeric(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestContractGeneratorLite_isUUIDLike(t *testing.T) {
	generator := NewContractGeneratorLite()

	testCases := []struct {
		input    string
		expected bool
	}{
		// Valid UUIDs with dashes
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{"6ba7b810-9dad-11d1-80b4-00c04fd430c8", true},
		
		// Valid UUIDs without dashes
		{"550e8400e29b41d4a716446655440000", true},
		{"6ba7b8109dad11d180b400c04fd430c8", true},
		
		// Invalid UUIDs
		{"550e8400-e29b-41d4-a716", false},           // Too short
		{"550e8400-e29b-41d4-a716-446655440000-extra", false}, // Too long
		{"550e8400-e29b-41d4-a716-44665544000g", false}, // Invalid hex character
		{"not-a-uuid", false},
		{"", false},
		{"123", false},
		{"550e8400e29b41d4a716446655440000extra", false}, // Too long without dashes
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := generator.isUUIDLike(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestContractGeneratorLite_isHex(t *testing.T) {
	generator := NewContractGeneratorLite()

	testCases := []struct {
		input    string
		expected bool
	}{
		{"abc123", true},
		{"ABC123", true},
		{"0123456789abcdef", true},
		{"0123456789ABCDEF", true},
		{"", true}, // Empty string is valid hex
		{"xyz", false},
		{"123g", false},
		{"12 34", false}, // Contains space
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := generator.isHex(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestContractGeneratorLite_generateParameterName(t *testing.T) {
	generator := NewContractGeneratorLite()

	testCases := []struct {
		name     string
		segment  string
		analysis *PathSegmentAnalysis
		expected string
	}{
		{
			name:    "Limited analysis (high cardinality)",
			segment: "123",
			analysis: &PathSegmentAnalysis{
				IsLimited: true,
			},
			expected: "{var}",
		},
		{
			name:    "Mostly numeric values",
			segment: "123",
			analysis: &PathSegmentAnalysis{
				UniqueValues: map[string]int{
					"123": 1,
					"456": 1,
					"789": 1,
					"101": 1,
					"202": 1,
					"303": 1,
					"404": 1,
					"505": 1,
					"606": 1,
					"707": 1, // 10 values, all numeric = 100%
				},
			},
			expected: "{num}",
		},
		{
			name:    "Some UUID-like values",
			segment: "550e8400-e29b-41d4-a716-446655440000",
			analysis: &PathSegmentAnalysis{
				UniqueValues: map[string]int{
					"550e8400-e29b-41d4-a716-446655440000": 1,
					"6ba7b810-9dad-11d1-80b4-00c04fd430c8": 1,
					"regular-string": 1,
				},
			},
			expected: "{id}",
		},
		{
			name:    "Mixed values (default)",
			segment: "abc",
			analysis: &PathSegmentAnalysis{
				UniqueValues: map[string]int{
					"abc": 1,
					"def": 1,
					"123": 1,
				},
			},
			expected: "{var}",
		},
		{
			name:    "Mostly numeric but below 90% threshold",
			segment: "123",
			analysis: &PathSegmentAnalysis{
				UniqueValues: map[string]int{
					"123": 1,
					"456": 1,
					"789": 1,
					"abc": 1, // 3/4 = 75% numeric, below 90% threshold
				},
			},
			expected: "{var}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := generator.generateParameterName(tc.segment, tc.analysis)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestContractGeneratorLite_statusCodesToRanges(t *testing.T) {
	generator := NewContractGeneratorLite()

	testCases := []struct {
		name     string
		input    []int
		expected []string
	}{
		{
			name:     "Single class",
			input:    []int{200, 201, 204},
			expected: []string{"2xx"},
		},
		{
			name:     "Multiple classes",
			input:    []int{200, 404, 500},
			expected: []string{"2xx", "4xx", "5xx"},
		},
		{
			name:     "Unsorted input",
			input:    []int{500, 200, 404},
			expected: []string{"2xx", "4xx", "5xx"},
		},
		{
			name:     "Empty input",
			input:    []int{},
			expected: []string{},
		},
		{
			name:     "Invalid status codes (ignored)",
			input:    []int{99, 200, 600},
			expected: []string{"2xx"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := generator.statusCodesToRanges(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestContractGeneratorLite_aggregateStatusCodes(t *testing.T) {
	generator := NewContractGeneratorLite()

	testCases := []struct {
		name           string
		statusCodes    []int
		strategy       string
		expectedCodes  []int
		expectedRanges []string
	}{
		{
			name:           "Exact strategy",
			statusCodes:    []int{200, 404, 500},
			strategy:       "exact",
			expectedCodes:  []int{200, 404, 500},
			expectedRanges: nil,
		},
		{
			name:           "Range strategy",
			statusCodes:    []int{200, 404, 500},
			strategy:       "range",
			expectedCodes:  nil,
			expectedRanges: []string{"2xx", "4xx", "5xx"},
		},
		{
			name:           "Auto strategy - single class",
			statusCodes:    []int{200, 201, 204},
			strategy:       "auto",
			expectedCodes:  nil,
			expectedRanges: []string{"2xx"},
		},
		{
			name:           "Auto strategy - multiple classes well represented",
			statusCodes:    []int{200, 201, 400, 404},
			strategy:       "auto",
			expectedCodes:  nil,
			expectedRanges: []string{"2xx", "4xx"},
		},
		{
			name:           "Auto strategy - sparse distribution",
			statusCodes:    []int{200, 403},
			strategy:       "auto",
			expectedCodes:  nil,
			expectedRanges: []string{"2xx", "4xx"}, // 200 and 403 are common codes, so use ranges
		},
		{
			name:           "Unknown strategy (defaults to auto)",
			statusCodes:    []int{200, 201, 204},
			strategy:       "unknown",
			expectedCodes:  nil,
			expectedRanges: []string{"2xx"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			codes, ranges := generator.aggregateStatusCodes(tc.statusCodes, tc.strategy)
			assert.Equal(t, tc.expectedCodes, codes)
			assert.Equal(t, tc.expectedRanges, ranges)
		})
	}
}

func TestContractGeneratorLite_calculateSpecificity(t *testing.T) {
	generator := NewContractGeneratorLite()

	testCases := []struct {
		name     string
		pattern  string
		expected int
	}{
		{
			name:     "All literal segments",
			pattern:  "/api/users/profile",
			expected: 3,
		},
		{
			name:     "Mixed literal and parameter segments",
			pattern:  "/api/users/{id}/profile",
			expected: 3, // api, users, profile
		},
		{
			name:     "All parameter segments",
			pattern:  "/{service}/{version}/{id}",
			expected: 0,
		},
		{
			name:     "Root path",
			pattern:  "/",
			expected: 0,
		},
		{
			name:     "Single literal segment",
			pattern:  "/api",
			expected: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := generator.calculateSpecificity(tc.pattern)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestContractGeneratorLite_patternsConflict(t *testing.T) {
	generator := NewContractGeneratorLite()

	testCases := []struct {
		name     string
		pattern1 string
		pattern2 string
		expected bool
	}{
		{
			name:     "Different segment count - no conflict",
			pattern1: "/api/users",
			pattern2: "/api/users/profile",
			expected: false,
		},
		{
			name:     "Different literal segments - no conflict",
			pattern1: "/api/users",
			pattern2: "/api/posts",
			expected: false,
		},
		{
			name:     "Same literal segments - conflict",
			pattern1: "/api/users",
			pattern2: "/api/users",
			expected: true,
		},
		{
			name:     "Parameter vs literal - conflict",
			pattern1: "/api/users/{id}",
			pattern2: "/api/users/123",
			expected: true,
		},
		{
			name:     "Both parameters - conflict",
			pattern1: "/api/users/{id}",
			pattern2: "/api/users/{userId}",
			expected: true,
		},
		{
			name:     "Mixed segments with different literals - no conflict",
			pattern1: "/api/users/{id}",
			pattern2: "/api/posts/{id}",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := generator.patternsConflict(tc.pattern1, tc.pattern2)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Integration test for the complete contract generation flow
func TestContractGeneratorLite_GenerateSpec_Integration(t *testing.T) {
	generator := NewContractGeneratorLite()
	
	// Set custom options for testing
	options := &GenerationOptions{
		PathClusteringThreshold: 0.8,
		MinSampleSize:          2, // Lower threshold for testing
		RequiredFieldThreshold: 0.8,
		MinEndpointSamples:     2,
		StatusAggregation:      "auto",
		ServiceName:            "test-service",
		ServiceVersion:         "v1.0.0",
	}
	generator.SetOptions(options)

	// Create test records with more data to ensure proper clustering
	baseTime := time.Now()
	records := []*traffic.NormalizedRecord{
		// Pattern: /api/users/{id} - GET (multiple IDs to trigger parameterization)
		{Method: "GET", Path: "/api/users/123", Status: 200, Timestamp: baseTime, Query: map[string][]string{"include": {"profile"}}, Headers: map[string][]string{"authorization": {"Bearer token1"}}},
		{Method: "GET", Path: "/api/users/456", Status: 200, Timestamp: baseTime.Add(1 * time.Minute), Query: map[string][]string{"include": {"profile"}}, Headers: map[string][]string{"authorization": {"Bearer token2"}}},
		{Method: "GET", Path: "/api/users/789", Status: 404, Timestamp: baseTime.Add(2 * time.Minute), Query: map[string][]string{"include": {"profile"}}, Headers: map[string][]string{"authorization": {"Bearer token3"}}},
		{Method: "GET", Path: "/api/users/101", Status: 200, Timestamp: baseTime.Add(3 * time.Minute), Query: map[string][]string{"include": {"profile"}}, Headers: map[string][]string{"authorization": {"Bearer token4"}}},
		{Method: "GET", Path: "/api/users/202", Status: 200, Timestamp: baseTime.Add(4 * time.Minute), Query: map[string][]string{"include": {"profile"}}, Headers: map[string][]string{"authorization": {"Bearer token5"}}},
		
		// Pattern: /api/users/{id} - POST
		{Method: "POST", Path: "/api/users/123", Status: 200, Timestamp: baseTime.Add(5 * time.Minute), Headers: map[string][]string{"authorization": {"Bearer token6"}, "content-type": {"application/json"}}},
		{Method: "POST", Path: "/api/users/456", Status: 201, Timestamp: baseTime.Add(6 * time.Minute), Headers: map[string][]string{"authorization": {"Bearer token7"}, "content-type": {"application/json"}}},
		{Method: "POST", Path: "/api/users/789", Status: 201, Timestamp: baseTime.Add(7 * time.Minute), Headers: map[string][]string{"authorization": {"Bearer token8"}, "content-type": {"application/json"}}},
		
		// Pattern: /api/posts (literal path - same path multiple times)
		{Method: "GET", Path: "/api/posts", Status: 200, Timestamp: baseTime.Add(8 * time.Minute), Query: map[string][]string{"limit": {"10"}, "offset": {"0"}}},
		{Method: "GET", Path: "/api/posts", Status: 200, Timestamp: baseTime.Add(9 * time.Minute), Query: map[string][]string{"limit": {"20"}, "offset": {"10"}}},
		{Method: "GET", Path: "/api/posts", Status: 200, Timestamp: baseTime.Add(10 * time.Minute), Query: map[string][]string{"limit": {"30"}, "offset": {"20"}}},
	}

	// Create iterator from records
	iterator := ingestor.NewSliceIterator(records)

	// Generate spec
	spec, err := generator.GenerateSpec(iterator)
	require.NoError(t, err)
	require.NotNil(t, spec)

	// Verify spec structure
	assert.Equal(t, "flowspec/v1alpha1", spec.APIVersion)
	assert.Equal(t, "ServiceSpec", spec.Kind)
	assert.Equal(t, "test-service", spec.Metadata.Name)
	assert.Equal(t, "v1.0.0", spec.Metadata.Version)

	// Should have 2 endpoints
	require.Len(t, spec.Spec.Endpoints, 2)

	// Debug: print all endpoint paths
	t.Logf("Generated endpoints:")
	for i, ep := range spec.Spec.Endpoints {
		t.Logf("  %d: %s", i, ep.Path)
	}

	// Basic verification that the contract generation works
	assert.Equal(t, "flowspec/v1alpha1", spec.APIVersion)
	assert.Equal(t, "ServiceSpec", spec.Kind)
	assert.Equal(t, "test-service", spec.Metadata.Name)
	assert.Equal(t, "v1.0.0", spec.Metadata.Version)

	// Should have generated some endpoints
	assert.Greater(t, len(spec.Spec.Endpoints), 0, "Should generate at least one endpoint")
	
	// Each endpoint should have operations
	for _, endpoint := range spec.Spec.Endpoints {
		assert.Greater(t, len(endpoint.Operations), 0, "Each endpoint should have at least one operation")
		
		// Each operation should have valid data
		for _, operation := range endpoint.Operations {
			assert.NotEmpty(t, operation.Method, "Operation should have a method")
			assert.NotNil(t, operation.Stats, "Operation should have stats")
			assert.Greater(t, operation.Stats.SupportCount, 0, "Operation should have support count > 0")
		}
	}
}