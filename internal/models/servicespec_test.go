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
	"testing"
	"time"
)

func TestServiceSpec_IsYAMLFormat(t *testing.T) {
	// Test YAML format
	yamlSpec := ServiceSpec{
		APIVersion: "flowspec/v1alpha1",
		Kind:       "ServiceSpec",
		Metadata: &ServiceSpecMetadata{
			Name:    "test-service",
			Version: "v1.0.0",
		},
		Spec: &ServiceSpecDefinition{
			Endpoints: []EndpointSpec{
				{
					Path: "/api/users/{id}",
					Operations: []OperationSpec{
						{
							Method: "GET",
							Responses: ResponseSpec{
								StatusRanges: []string{"2xx", "4xx"},
								Aggregation:  "range",
							},
							Required: RequiredFieldsSpec{
								Headers: []string{"authorization"},
								Query:   []string{},
							},
						},
					},
				},
			},
		},
	}

	if !yamlSpec.IsYAMLFormat() {
		t.Error("Expected YAML format to be detected")
	}

	if yamlSpec.IsLegacyFormat() {
		t.Error("Expected legacy format to not be detected")
	}

	// Test legacy format
	legacySpec := ServiceSpec{
		OperationID:    "getUserById",
		Description:    "Get user by ID",
		Preconditions:  map[string]interface{}{"auth": true},
		Postconditions: map[string]interface{}{"status": 200},
		SourceFile:     "user.go",
		LineNumber:     42,
	}

	if legacySpec.IsYAMLFormat() {
		t.Error("Expected YAML format to not be detected")
	}

	if !legacySpec.IsLegacyFormat() {
		t.Error("Expected legacy format to be detected")
	}
}

func TestServiceSpec_ValidateYAMLFormat(t *testing.T) {
	// Valid YAML spec
	validSpec := ServiceSpec{
		APIVersion: "flowspec/v1alpha1",
		Kind:       "ServiceSpec",
		Metadata: &ServiceSpecMetadata{
			Name:    "test-service",
			Version: "v1.0.0",
		},
		Spec: &ServiceSpecDefinition{
			Endpoints: []EndpointSpec{
				{
					Path: "/api/users/{id}",
					Operations: []OperationSpec{
						{
							Method: "GET",
							Responses: ResponseSpec{
								StatusCodes: []int{200, 404},
								Aggregation: "exact",
							},
							Required: RequiredFieldsSpec{
								Headers: []string{"authorization"},
								Query:   []string{},
							},
						},
					},
				},
			},
		},
	}

	if err := validSpec.Validate(); err != nil {
		t.Errorf("Expected valid spec to pass validation, got: %v", err)
	}

	// Invalid YAML spec - missing apiVersion
	invalidSpec := ServiceSpec{
		Kind: "ServiceSpec",
		Metadata: &ServiceSpecMetadata{
			Name:    "test-service",
			Version: "v1.0.0",
		},
		Spec: &ServiceSpecDefinition{
			Endpoints: []EndpointSpec{},
		},
	}

	if err := invalidSpec.Validate(); err == nil {
		t.Error("Expected invalid spec to fail validation")
	}
}

func TestServiceSpec_ValidateLegacyFormat(t *testing.T) {
	// Valid legacy spec
	validSpec := ServiceSpec{
		OperationID:    "getUserById",
		Description:    "Get user by ID",
		Preconditions:  map[string]interface{}{"auth": true},
		Postconditions: map[string]interface{}{"status": 200},
		SourceFile:     "user.go",
		LineNumber:     42,
	}

	if err := validSpec.Validate(); err != nil {
		t.Errorf("Expected valid legacy spec to pass validation, got: %v", err)
	}

	// Invalid legacy spec - missing operationId
	invalidSpec := ServiceSpec{
		Description:    "Get user by ID",
		Preconditions:  map[string]interface{}{"auth": true},
		Postconditions: map[string]interface{}{"status": 200},
		SourceFile:     "user.go",
		LineNumber:     42,
	}

	if err := invalidSpec.Validate(); err == nil {
		t.Error("Expected invalid legacy spec to fail validation")
	}
}

func TestResponseSpec_Validate(t *testing.T) {
	// Valid with status codes
	validWithCodes := ResponseSpec{
		StatusCodes: []int{200, 404, 500},
		Aggregation: "exact",
	}

	if err := validWithCodes.Validate(); err != nil {
		t.Errorf("Expected valid ResponseSpec with codes to pass validation, got: %v", err)
	}

	// Valid with status ranges
	validWithRanges := ResponseSpec{
		StatusRanges: []string{"2xx", "4xx"},
		Aggregation:  "range",
	}

	if err := validWithRanges.Validate(); err != nil {
		t.Errorf("Expected valid ResponseSpec with ranges to pass validation, got: %v", err)
	}

	// Invalid - no status codes or ranges
	invalid := ResponseSpec{
		Aggregation: "auto",
	}

	if err := invalid.Validate(); err == nil {
		t.Error("Expected ResponseSpec without codes or ranges to fail validation")
	}

	// Invalid status code
	invalidCode := ResponseSpec{
		StatusCodes: []int{999},
	}

	if err := invalidCode.Validate(); err == nil {
		t.Error("Expected ResponseSpec with invalid status code to fail validation")
	}

	// Invalid status range
	invalidRange := ResponseSpec{
		StatusRanges: []string{"9xx"},
	}

	if err := invalidRange.Validate(); err == nil {
		t.Error("Expected ResponseSpec with invalid status range to fail validation")
	}

	// Invalid aggregation
	invalidAggregation := ResponseSpec{
		StatusCodes: []int{200},
		Aggregation: "invalid",
	}

	if err := invalidAggregation.Validate(); err == nil {
		t.Error("Expected ResponseSpec with invalid aggregation to fail validation")
	}
}

func TestOperationSpec_Validate(t *testing.T) {
	// Valid operation
	validOp := OperationSpec{
		Method: "GET",
		Responses: ResponseSpec{
			StatusCodes: []int{200},
		},
		Required: RequiredFieldsSpec{
			Headers: []string{"authorization"},
			Query:   []string{},
		},
	}

	if err := validOp.Validate(); err != nil {
		t.Errorf("Expected valid OperationSpec to pass validation, got: %v", err)
	}

	// Invalid method
	invalidMethod := OperationSpec{
		Method: "INVALID",
		Responses: ResponseSpec{
			StatusCodes: []int{200},
		},
		Required: RequiredFieldsSpec{
			Headers: []string{},
			Query:   []string{},
		},
	}

	if err := invalidMethod.Validate(); err == nil {
		t.Error("Expected OperationSpec with invalid method to fail validation")
	}
}

func TestEndpointStats(t *testing.T) {
	now := time.Now()
	stats := EndpointStats{
		SupportCount: 100,
		FirstSeen:    now.Add(-24 * time.Hour),
		LastSeen:     now,
	}

	if stats.SupportCount != 100 {
		t.Errorf("Expected SupportCount to be 100, got %d", stats.SupportCount)
	}

	if stats.LastSeen.Before(stats.FirstSeen) {
		t.Error("Expected LastSeen to be after FirstSeen")
	}
}