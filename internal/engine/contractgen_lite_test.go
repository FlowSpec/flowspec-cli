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

	"github.com/flowspec/flowspec-cli/internal/ingestor"
	"github.com/flowspec/flowspec-cli/internal/ingestor/traffic"
	"github.com/flowspec/flowspec-cli/internal/models"
)

func TestContractGeneratorLite_GenerateSpec(t *testing.T) {
	generator := NewContractGeneratorLite()
	
	// Adjust options for testing
	options := DefaultGenerationOptions()
	options.MinEndpointSamples = 1  // Lower threshold for testing
	options.MinSampleSize = 1       // Lower threshold for testing
	generator.SetOptions(options)
	
	// Create test data
	records := []*traffic.NormalizedRecord{
		{
			Method:    "GET",
			Path:      "/api/users/123",
			Status:    200,
			Timestamp: time.Now(),
			Query:     map[string][]string{"include": {"profile"}},
			Headers:   map[string][]string{"authorization": {"Bearer token"}},
		},
		{
			Method:    "GET",
			Path:      "/api/users/456",
			Status:    200,
			Timestamp: time.Now(),
			Query:     map[string][]string{"include": {"profile"}},
			Headers:   map[string][]string{"authorization": {"Bearer token"}},
		},
		{
			Method:    "POST",
			Path:      "/api/users",
			Status:    201,
			Timestamp: time.Now(),
			Headers:   map[string][]string{"authorization": {"Bearer token"}, "content-type": {"application/json"}},
		},
	}
	
	// Create iterator
	iterator := ingestor.NewSliceIterator(records)
	
	// Generate spec
	spec, err := generator.GenerateSpec(iterator)
	if err != nil {
		t.Fatalf("GenerateSpec failed: %v", err)
	}
	
	// Verify basic structure
	if spec == nil {
		t.Fatal("Generated spec is nil")
	}
	
	if spec.APIVersion != "flowspec/v1alpha1" {
		t.Errorf("Expected APIVersion 'flowspec/v1alpha1', got '%s'", spec.APIVersion)
	}
	
	if spec.Kind != "ServiceSpec" {
		t.Errorf("Expected Kind 'ServiceSpec', got '%s'", spec.Kind)
	}
	
	if spec.Spec == nil {
		t.Fatal("Spec definition is nil")
	}
	
	// Should have 2 endpoints: /api/users/{num} and /api/users
	if len(spec.Spec.Endpoints) != 2 {
		t.Errorf("Expected 2 endpoints, got %d", len(spec.Spec.Endpoints))
	}
	
	// Verify parameterized endpoint
	var parameterizedEndpoint *models.EndpointSpec
	var literalEndpoint *models.EndpointSpec
	
	for i := range spec.Spec.Endpoints {
		if spec.Spec.Endpoints[i].Path == "/api/users/{num}" {
			parameterizedEndpoint = &spec.Spec.Endpoints[i]
		} else if spec.Spec.Endpoints[i].Path == "/api/users" {
			literalEndpoint = &spec.Spec.Endpoints[i]
		}
	}
	
	if parameterizedEndpoint == nil {
		t.Error("Expected parameterized endpoint /api/users/{num} not found")
	} else {
		// Should have GET operation
		if len(parameterizedEndpoint.Operations) != 1 {
			t.Errorf("Expected 1 operation for parameterized endpoint, got %d", len(parameterizedEndpoint.Operations))
		} else {
			op := parameterizedEndpoint.Operations[0]
			if op.Method != "GET" {
				t.Errorf("Expected GET method, got %s", op.Method)
			}
			
			// Should have authorization as required header
			found := false
			for _, header := range op.Required.Headers {
				if header == "authorization" {
					found = true
					break
				}
			}
			if !found {
				t.Error("Expected 'authorization' in required headers")
			}
		}
	}
	
	if literalEndpoint == nil {
		t.Error("Expected literal endpoint /api/users not found")
	} else {
		// Should have POST operation
		if len(literalEndpoint.Operations) != 1 {
			t.Errorf("Expected 1 operation for literal endpoint, got %d", len(literalEndpoint.Operations))
		} else {
			op := literalEndpoint.Operations[0]
			if op.Method != "POST" {
				t.Errorf("Expected POST method, got %s", op.Method)
			}
		}
	}
}

func TestPathParameterization(t *testing.T) {
	generator := NewContractGeneratorLite()
	
	tests := []struct {
		name     string
		segment  string
		analysis *PathSegmentAnalysis
		expected bool
	}{
		{
			name:    "high cardinality numeric",
			segment: "123",
			analysis: &PathSegmentAnalysis{
				UniqueValues: map[string]int{"123": 1, "456": 1, "789": 1},
				TotalCount:   3,
			},
			expected: false, // Not enough samples
		},
		{
			name:    "sufficient samples and high ratio",
			segment: "123",
			analysis: &PathSegmentAnalysis{
				UniqueValues: make(map[string]int),
				TotalCount:   25,
			},
			expected: true, // Should parameterize with high ratio
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Populate unique values for the second test
			if tt.name == "sufficient samples and high ratio" {
				for i := 0; i < 21; i++ {
					tt.analysis.UniqueValues[fmt.Sprintf("%d", i)] = 1
				}
			}
			
			result := generator.shouldParameterize(tt.segment, tt.analysis)
			if result != tt.expected {
				t.Errorf("shouldParameterize() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestStatusCodeAggregation(t *testing.T) {
	generator := NewContractGeneratorLite()
	
	tests := []struct {
		name           string
		statusCodes    []int
		strategy       string
		expectedCodes  []int
		expectedRanges []string
	}{
		{
			name:           "exact strategy",
			statusCodes:    []int{200, 404, 500},
			strategy:       "exact",
			expectedCodes:  []int{200, 404, 500},
			expectedRanges: nil,
		},
		{
			name:           "range strategy",
			statusCodes:    []int{200, 201, 404, 500},
			strategy:       "range",
			expectedCodes:  nil,
			expectedRanges: []string{"2xx", "4xx", "5xx"},
		},
		{
			name:           "auto strategy - same class",
			statusCodes:    []int{200, 201, 204},
			strategy:       "auto",
			expectedCodes:  nil,
			expectedRanges: []string{"2xx"},
		},
		{
			name:           "auto strategy - mixed classes",
			statusCodes:    []int{200, 404},
			strategy:       "auto",
			expectedCodes:  nil,
			expectedRanges: []string{"2xx", "4xx"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codes, ranges := generator.aggregateStatusCodes(tt.statusCodes, tt.strategy)
			
			if len(codes) != len(tt.expectedCodes) {
				t.Errorf("Expected %d codes, got %d", len(tt.expectedCodes), len(codes))
			}
			
			if len(ranges) != len(tt.expectedRanges) {
				t.Errorf("Expected %d ranges, got %d", len(tt.expectedRanges), len(ranges))
			}
			
			// Check codes match
			for i, expected := range tt.expectedCodes {
				if i < len(codes) && codes[i] != expected {
					t.Errorf("Expected code %d, got %d", expected, codes[i])
				}
			}
			
			// Check ranges match
			for i, expected := range tt.expectedRanges {
				if i < len(ranges) && ranges[i] != expected {
					t.Errorf("Expected range %s, got %s", expected, ranges[i])
				}
			}
		})
	}
}