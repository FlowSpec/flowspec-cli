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
	"testing"

	"github.com/flowspec/flowspec-cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSchemaValidator(t *testing.T) {
	validator, err := NewSchemaValidator()
	require.NoError(t, err)
	assert.NotNil(t, validator)
	assert.NotNil(t, validator.schema)
}

func TestSchemaValidator_ValidateServiceSpec_ValidSpec(t *testing.T) {
	validator, err := NewSchemaValidator()
	require.NoError(t, err)

	spec := &models.ServiceSpec{
		APIVersion: "flowspec/v1alpha1",
		Kind:       "ServiceSpec",
		Metadata: &models.ServiceSpecMetadata{
			Name:    "user-service",
			Version: "v1.0.0",
		},
		Spec: &models.ServiceSpecDefinition{
			Endpoints: []models.EndpointSpec{
				{
					Path: "/api/users/{id}",
					Operations: []models.OperationSpec{
						{
							Method: "GET",
							Responses: models.ResponseSpec{
								StatusRanges: []string{"2xx", "4xx"},
								Aggregation:  "range",
							},
							Required: models.RequiredFieldsSpec{
								Headers: []string{"authorization"},
								Query:   []string{},
							},
						},
					},
				},
			},
		},
	}

	errors := validator.ValidateServiceSpec(spec)
	assert.Empty(t, errors)
}

func TestSchemaValidator_ValidateServiceSpec_MissingAPIVersion(t *testing.T) {
	validator, err := NewSchemaValidator()
	require.NoError(t, err)

	spec := &models.ServiceSpec{
		// APIVersion missing
		Kind: "ServiceSpec",
		Metadata: &models.ServiceSpecMetadata{
			Name:    "user-service",
			Version: "v1.0.0",
		},
		Spec: &models.ServiceSpecDefinition{
			Endpoints: []models.EndpointSpec{
				{
					Path: "/api/users/{id}",
					Operations: []models.OperationSpec{
						{
							Method: "GET",
							Responses: models.ResponseSpec{
								StatusRanges: []string{"2xx"},
							},
							Required: models.RequiredFieldsSpec{
								Headers: []string{},
								Query:   []string{},
							},
						},
					},
				},
			},
		},
	}

	errors := validator.ValidateServiceSpec(spec)
	assert.Len(t, errors, 1)
	assert.Equal(t, "apiVersion is required", errors[0].Message)
	assert.Equal(t, "/apiVersion", errors[0].JSONPointer)
}

func TestSchemaValidator_ValidateServiceSpec_InvalidAPIVersion(t *testing.T) {
	validator, err := NewSchemaValidator()
	require.NoError(t, err)

	spec := &models.ServiceSpec{
		APIVersion: "invalid/version",
		Kind:       "ServiceSpec",
		Metadata: &models.ServiceSpecMetadata{
			Name:    "user-service",
			Version: "v1.0.0",
		},
		Spec: &models.ServiceSpecDefinition{
			Endpoints: []models.EndpointSpec{
				{
					Path: "/api/users/{id}",
					Operations: []models.OperationSpec{
						{
							Method: "GET",
							Responses: models.ResponseSpec{
								StatusRanges: []string{"2xx"},
							},
							Required: models.RequiredFieldsSpec{
								Headers: []string{},
								Query:   []string{},
							},
						},
					},
				},
			},
		},
	}

	errors := validator.ValidateServiceSpec(spec)
	assert.Len(t, errors, 1)
	assert.Contains(t, errors[0].Message, "apiVersion 'invalid/version' is invalid")
	assert.Equal(t, "/apiVersion", errors[0].JSONPointer)
}

func TestSchemaValidator_ValidateServiceSpec_InvalidKind(t *testing.T) {
	validator, err := NewSchemaValidator()
	require.NoError(t, err)

	spec := &models.ServiceSpec{
		APIVersion: "flowspec/v1alpha1",
		Kind:       "InvalidKind",
		Metadata: &models.ServiceSpecMetadata{
			Name:    "user-service",
			Version: "v1.0.0",
		},
		Spec: &models.ServiceSpecDefinition{
			Endpoints: []models.EndpointSpec{
				{
					Path: "/api/users/{id}",
					Operations: []models.OperationSpec{
						{
							Method: "GET",
							Responses: models.ResponseSpec{
								StatusRanges: []string{"2xx"},
							},
							Required: models.RequiredFieldsSpec{
								Headers: []string{},
								Query:   []string{},
							},
						},
					},
				},
			},
		},
	}

	errors := validator.ValidateServiceSpec(spec)
	assert.Len(t, errors, 1)
	assert.Contains(t, errors[0].Message, "kind 'InvalidKind' is invalid")
	assert.Equal(t, "/kind", errors[0].JSONPointer)
}

func TestSchemaValidator_ValidateServiceSpec_MissingMetadata(t *testing.T) {
	validator, err := NewSchemaValidator()
	require.NoError(t, err)

	spec := &models.ServiceSpec{
		APIVersion: "flowspec/v1alpha1",
		Kind:       "ServiceSpec",
		// Metadata missing
		Spec: &models.ServiceSpecDefinition{
			Endpoints: []models.EndpointSpec{
				{
					Path: "/api/users/{id}",
					Operations: []models.OperationSpec{
						{
							Method: "GET",
							Responses: models.ResponseSpec{
								StatusRanges: []string{"2xx"},
							},
							Required: models.RequiredFieldsSpec{
								Headers: []string{},
								Query:   []string{},
							},
						},
					},
				},
			},
		},
	}

	errors := validator.ValidateServiceSpec(spec)
	assert.Len(t, errors, 1)
	assert.Equal(t, "metadata is required", errors[0].Message)
	assert.Equal(t, "/metadata", errors[0].JSONPointer)
}

func TestSchemaValidator_ValidateServiceSpec_MissingMetadataFields(t *testing.T) {
	validator, err := NewSchemaValidator()
	require.NoError(t, err)

	spec := &models.ServiceSpec{
		APIVersion: "flowspec/v1alpha1",
		Kind:       "ServiceSpec",
		Metadata: &models.ServiceSpecMetadata{
			// Name and Version missing
		},
		Spec: &models.ServiceSpecDefinition{
			Endpoints: []models.EndpointSpec{
				{
					Path: "/api/users/{id}",
					Operations: []models.OperationSpec{
						{
							Method: "GET",
							Responses: models.ResponseSpec{
								StatusRanges: []string{"2xx"},
							},
							Required: models.RequiredFieldsSpec{
								Headers: []string{},
								Query:   []string{},
							},
						},
					},
				},
			},
		},
	}

	errors := validator.ValidateServiceSpec(spec)
	assert.Len(t, errors, 2)
	
	// Check that both name and version errors are present
	errorMessages := make([]string, len(errors))
	errorPointers := make([]string, len(errors))
	for i, err := range errors {
		errorMessages[i] = err.Message
		errorPointers[i] = err.JSONPointer
	}
	
	assert.Contains(t, errorMessages, "metadata.name is required")
	assert.Contains(t, errorMessages, "metadata.version is required")
	assert.Contains(t, errorPointers, "/metadata/name")
	assert.Contains(t, errorPointers, "/metadata/version")
}

func TestSchemaValidator_ValidateServiceSpec_EmptyEndpoints(t *testing.T) {
	validator, err := NewSchemaValidator()
	require.NoError(t, err)

	spec := &models.ServiceSpec{
		APIVersion: "flowspec/v1alpha1",
		Kind:       "ServiceSpec",
		Metadata: &models.ServiceSpecMetadata{
			Name:    "user-service",
			Version: "v1.0.0",
		},
		Spec: &models.ServiceSpecDefinition{
			Endpoints: []models.EndpointSpec{}, // Empty endpoints
		},
	}

	errors := validator.ValidateServiceSpec(spec)
	assert.Len(t, errors, 1)
	assert.Equal(t, "spec.endpoints cannot be empty", errors[0].Message)
	assert.Equal(t, "/spec/endpoints", errors[0].JSONPointer)
}

func TestSchemaValidator_ValidateServiceSpec_InvalidHTTPMethod(t *testing.T) {
	validator, err := NewSchemaValidator()
	require.NoError(t, err)

	spec := &models.ServiceSpec{
		APIVersion: "flowspec/v1alpha1",
		Kind:       "ServiceSpec",
		Metadata: &models.ServiceSpecMetadata{
			Name:    "user-service",
			Version: "v1.0.0",
		},
		Spec: &models.ServiceSpecDefinition{
			Endpoints: []models.EndpointSpec{
				{
					Path: "/api/users/{id}",
					Operations: []models.OperationSpec{
						{
							Method: "INVALID", // Invalid HTTP method
							Responses: models.ResponseSpec{
								StatusRanges: []string{"2xx"},
							},
							Required: models.RequiredFieldsSpec{
								Headers: []string{},
								Query:   []string{},
							},
						},
					},
				},
			},
		},
	}

	errors := validator.ValidateServiceSpec(spec)
	assert.Len(t, errors, 1)
	assert.Contains(t, errors[0].Message, "method 'INVALID' is invalid")
	assert.Equal(t, "/spec/endpoints/0/operations/0/method", errors[0].JSONPointer)
}

func TestSchemaValidator_ValidateServiceSpec_InvalidStatusCode(t *testing.T) {
	validator, err := NewSchemaValidator()
	require.NoError(t, err)

	spec := &models.ServiceSpec{
		APIVersion: "flowspec/v1alpha1",
		Kind:       "ServiceSpec",
		Metadata: &models.ServiceSpecMetadata{
			Name:    "user-service",
			Version: "v1.0.0",
		},
		Spec: &models.ServiceSpecDefinition{
			Endpoints: []models.EndpointSpec{
				{
					Path: "/api/users/{id}",
					Operations: []models.OperationSpec{
						{
							Method: "GET",
							Responses: models.ResponseSpec{
								StatusCodes: []int{999}, // Invalid status code
							},
							Required: models.RequiredFieldsSpec{
								Headers: []string{},
								Query:   []string{},
							},
						},
					},
				},
			},
		},
	}

	errors := validator.ValidateServiceSpec(spec)
	assert.Len(t, errors, 1)
	assert.Contains(t, errors[0].Message, "status code 999 is not in valid range")
	assert.Equal(t, "/spec/endpoints/0/operations/0/responses/statusCodes/0", errors[0].JSONPointer)
}

func TestSchemaValidator_ValidateServiceSpec_InvalidStatusRange(t *testing.T) {
	validator, err := NewSchemaValidator()
	require.NoError(t, err)

	spec := &models.ServiceSpec{
		APIVersion: "flowspec/v1alpha1",
		Kind:       "ServiceSpec",
		Metadata: &models.ServiceSpecMetadata{
			Name:    "user-service",
			Version: "v1.0.0",
		},
		Spec: &models.ServiceSpecDefinition{
			Endpoints: []models.EndpointSpec{
				{
					Path: "/api/users/{id}",
					Operations: []models.OperationSpec{
						{
							Method: "GET",
							Responses: models.ResponseSpec{
								StatusRanges: []string{"9xx"}, // Invalid status range
							},
							Required: models.RequiredFieldsSpec{
								Headers: []string{},
								Query:   []string{},
							},
						},
					},
				},
			},
		},
	}

	errors := validator.ValidateServiceSpec(spec)
	assert.Len(t, errors, 1)
	assert.Contains(t, errors[0].Message, "status range '9xx' is not valid")
	assert.Equal(t, "/spec/endpoints/0/operations/0/responses/statusRanges/0", errors[0].JSONPointer)
}