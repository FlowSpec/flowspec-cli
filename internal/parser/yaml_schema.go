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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flowspec/flowspec-cli/internal/models"
)

// ServiceSpecSchema defines the JSON Schema for ServiceSpec validation
const ServiceSpecSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["apiVersion", "kind", "metadata", "spec"],
  "properties": {
    "apiVersion": {
      "type": "string",
      "pattern": "^flowspec/v[0-9]+[a-z]*[0-9]*$",
      "description": "API version, must be in format flowspec/v{version}"
    },
    "kind": {
      "type": "string",
      "enum": ["ServiceSpec"],
      "description": "Resource kind, must be ServiceSpec"
    },
    "metadata": {
      "type": "object",
      "required": ["name", "version"],
      "properties": {
        "name": {
          "type": "string",
          "minLength": 1,
          "description": "Service name"
        },
        "version": {
          "type": "string",
          "minLength": 1,
          "description": "Service version"
        }
      },
      "additionalProperties": false
    },
    "spec": {
      "type": "object",
      "required": ["endpoints"],
      "properties": {
        "endpoints": {
          "type": "array",
          "minItems": 1,
          "items": {
            "$ref": "#/definitions/endpoint"
          }
        }
      },
      "additionalProperties": false
    }
  },
  "additionalProperties": false,
  "definitions": {
    "endpoint": {
      "type": "object",
      "required": ["path", "operations"],
      "properties": {
        "path": {
          "type": "string",
          "minLength": 1,
          "description": "Endpoint path"
        },
        "operations": {
          "type": "array",
          "minItems": 1,
          "items": {
            "$ref": "#/definitions/operation"
          }
        },
        "stats": {
          "$ref": "#/definitions/endpointStats"
        }
      },
      "additionalProperties": false
    },
    "operation": {
      "type": "object",
      "required": ["method", "responses", "required"],
      "properties": {
        "method": {
          "type": "string",
          "enum": ["GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"],
          "description": "HTTP method"
        },
        "responses": {
          "$ref": "#/definitions/responseSpec"
        },
        "required": {
          "$ref": "#/definitions/requiredFields"
        },
        "optional": {
          "$ref": "#/definitions/optionalFields"
        },
        "stats": {
          "$ref": "#/definitions/operationStats"
        }
      },
      "additionalProperties": false
    },
    "responseSpec": {
      "type": "object",
      "properties": {
        "statusCodes": {
          "type": "array",
          "items": {
            "type": "integer",
            "minimum": 100,
            "maximum": 599
          }
        },
        "statusRanges": {
          "type": "array",
          "items": {
            "type": "string",
            "enum": ["1xx", "2xx", "3xx", "4xx", "5xx"]
          }
        },
        "aggregation": {
          "type": "string",
          "enum": ["range", "exact", "auto"]
        }
      },
      "anyOf": [
        {"required": ["statusCodes"]},
        {"required": ["statusRanges"]}
      ],
      "additionalProperties": false
    },
    "requiredFields": {
      "type": "object",
      "required": ["query", "headers"],
      "properties": {
        "query": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "headers": {
          "type": "array",
          "items": {
            "type": "string"
          }
        }
      },
      "additionalProperties": false
    },
    "optionalFields": {
      "type": "object",
      "required": ["query", "headers"],
      "properties": {
        "query": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "headers": {
          "type": "array",
          "items": {
            "type": "string"
          }
        }
      },
      "additionalProperties": false
    },
    "endpointStats": {
      "type": "object",
      "required": ["supportCount", "firstSeen", "lastSeen"],
      "properties": {
        "supportCount": {
          "type": "integer",
          "minimum": 0
        },
        "firstSeen": {
          "type": "string",
          "format": "date-time"
        },
        "lastSeen": {
          "type": "string",
          "format": "date-time"
        }
      },
      "additionalProperties": false
    },
    "operationStats": {
      "type": "object",
      "required": ["supportCount", "firstSeen", "lastSeen"],
      "properties": {
        "supportCount": {
          "type": "integer",
          "minimum": 0
        },
        "firstSeen": {
          "type": "string",
          "format": "date-time"
        },
        "lastSeen": {
          "type": "string",
          "format": "date-time"
        }
      },
      "additionalProperties": false
    }
  }
}`

// SchemaValidator provides JSON Schema validation for ServiceSpec
type SchemaValidator struct {
	schema map[string]interface{}
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator() (*SchemaValidator, error) {
	var schema map[string]interface{}
	if err := json.Unmarshal([]byte(ServiceSpecSchema), &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	return &SchemaValidator{
		schema: schema,
	}, nil
}

// ValidateServiceSpec validates a ServiceSpec against the JSON schema
func (sv *SchemaValidator) ValidateServiceSpec(spec *models.ServiceSpec) []models.ParseError {
	var errors []models.ParseError

	// Basic validation - check required top-level fields
	if spec.APIVersion == "" {
		errors = append(errors, models.ParseError{
			Message:     "apiVersion is required",
			JSONPointer: "/apiVersion",
		})
	} else if !sv.isValidAPIVersion(spec.APIVersion) {
		errors = append(errors, models.ParseError{
			Message:     fmt.Sprintf("apiVersion '%s' is invalid, must be in format flowspec/v{version}", spec.APIVersion),
			JSONPointer: "/apiVersion",
		})
	}

	if spec.Kind == "" {
		errors = append(errors, models.ParseError{
			Message:     "kind is required",
			JSONPointer: "/kind",
		})
	} else if spec.Kind != "ServiceSpec" {
		errors = append(errors, models.ParseError{
			Message:     fmt.Sprintf("kind '%s' is invalid, must be 'ServiceSpec'", spec.Kind),
			JSONPointer: "/kind",
		})
	}

	if spec.Metadata == nil {
		errors = append(errors, models.ParseError{
			Message:     "metadata is required",
			JSONPointer: "/metadata",
		})
	} else {
		errors = append(errors, sv.validateMetadata(spec.Metadata)...)
	}

	if spec.Spec == nil {
		errors = append(errors, models.ParseError{
			Message:     "spec is required",
			JSONPointer: "/spec",
		})
	} else {
		errors = append(errors, sv.validateSpec(spec.Spec)...)
	}

	return errors
}

// isValidAPIVersion checks if the API version follows the expected pattern
func (sv *SchemaValidator) isValidAPIVersion(version string) bool {
	// Simple validation for flowspec/v{version} pattern
	return strings.HasPrefix(version, "flowspec/v") && len(version) > len("flowspec/v")
}

// validateMetadata validates the metadata section
func (sv *SchemaValidator) validateMetadata(metadata *models.ServiceSpecMetadata) []models.ParseError {
	var errors []models.ParseError

	if metadata.Name == "" {
		errors = append(errors, models.ParseError{
			Message:     "metadata.name is required",
			JSONPointer: "/metadata/name",
		})
	}

	if metadata.Version == "" {
		errors = append(errors, models.ParseError{
			Message:     "metadata.version is required",
			JSONPointer: "/metadata/version",
		})
	}

	return errors
}

// validateSpec validates the spec section
func (sv *SchemaValidator) validateSpec(spec *models.ServiceSpecDefinition) []models.ParseError {
	var errors []models.ParseError

	if len(spec.Endpoints) == 0 {
		errors = append(errors, models.ParseError{
			Message:     "spec.endpoints cannot be empty",
			JSONPointer: "/spec/endpoints",
		})
	}

	for i, endpoint := range spec.Endpoints {
		errors = append(errors, sv.validateEndpoint(&endpoint, fmt.Sprintf("/spec/endpoints/%d", i))...)
	}

	return errors
}

// validateEndpoint validates an endpoint
func (sv *SchemaValidator) validateEndpoint(endpoint *models.EndpointSpec, basePath string) []models.ParseError {
	var errors []models.ParseError

	if endpoint.Path == "" {
		errors = append(errors, models.ParseError{
			Message:     "path is required",
			JSONPointer: basePath + "/path",
		})
	}

	if len(endpoint.Operations) == 0 {
		errors = append(errors, models.ParseError{
			Message:     "operations cannot be empty",
			JSONPointer: basePath + "/operations",
		})
	}

	for i, operation := range endpoint.Operations {
		errors = append(errors, sv.validateOperation(&operation, fmt.Sprintf("%s/operations/%d", basePath, i))...)
	}

	return errors
}

// validateOperation validates an operation
func (sv *SchemaValidator) validateOperation(operation *models.OperationSpec, basePath string) []models.ParseError {
	var errors []models.ParseError

	if operation.Method == "" {
		errors = append(errors, models.ParseError{
			Message:     "method is required",
			JSONPointer: basePath + "/method",
		})
	} else {
		validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
		methodValid := false
		for _, validMethod := range validMethods {
			if operation.Method == validMethod {
				methodValid = true
				break
			}
		}
		if !methodValid {
			errors = append(errors, models.ParseError{
				Message:     fmt.Sprintf("method '%s' is invalid, must be one of: %s", operation.Method, strings.Join(validMethods, ", ")),
				JSONPointer: basePath + "/method",
			})
		}
	}

	errors = append(errors, sv.validateResponseSpec(&operation.Responses, basePath+"/responses")...)

	return errors
}

// validateResponseSpec validates a response specification
func (sv *SchemaValidator) validateResponseSpec(responses *models.ResponseSpec, basePath string) []models.ParseError {
	var errors []models.ParseError

	// Must have either StatusCodes or StatusRanges
	if len(responses.StatusCodes) == 0 && len(responses.StatusRanges) == 0 {
		errors = append(errors, models.ParseError{
			Message:     "must specify either statusCodes or statusRanges",
			JSONPointer: basePath,
		})
	}

	// Validate status codes
	for i, code := range responses.StatusCodes {
		if code < 100 || code > 599 {
			errors = append(errors, models.ParseError{
				Message:     fmt.Sprintf("status code %d is not in valid range (100-599)", code),
				JSONPointer: fmt.Sprintf("%s/statusCodes/%d", basePath, i),
			})
		}
	}

	// Validate status ranges
	validRanges := []string{"1xx", "2xx", "3xx", "4xx", "5xx"}
	for i, rangeStr := range responses.StatusRanges {
		rangeValid := false
		for _, validRange := range validRanges {
			if rangeStr == validRange {
				rangeValid = true
				break
			}
		}
		if !rangeValid {
			errors = append(errors, models.ParseError{
				Message:     fmt.Sprintf("status range '%s' is not valid, must be one of: %s", rangeStr, strings.Join(validRanges, ", ")),
				JSONPointer: fmt.Sprintf("%s/statusRanges/%d", basePath, i),
			})
		}
	}

	// Validate aggregation strategy
	if responses.Aggregation != "" {
		validAggregations := []string{"range", "exact", "auto"}
		aggregationValid := false
		for _, validAggregation := range validAggregations {
			if responses.Aggregation == validAggregation {
				aggregationValid = true
				break
			}
		}
		if !aggregationValid {
			errors = append(errors, models.ParseError{
				Message:     fmt.Sprintf("aggregation '%s' is not valid, must be one of: %s", responses.Aggregation, strings.Join(validAggregations, ", ")),
				JSONPointer: basePath + "/aggregation",
			})
		}
	}

	return errors
}