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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGoFileParser(t *testing.T) {
	parser := NewGoFileParser()

	assert.NotNil(t, parser)
	assert.NotNil(t, parser.BaseFileParser)
	assert.Equal(t, LanguageGo, parser.GetLanguage())
}

func TestGoFileParser_CanParse(t *testing.T) {
	parser := NewGoFileParser()

	testCases := []struct {
		filename string
		expected bool
	}{
		{"test.go", true},
		{"main.go", true},
		{"pkg/service/user.go", true},
		{"cmd/server/main.go", true},
		{"test.java", false},
		{"test.ts", false},
		{"test.py", false},
		{"test.txt", false},
		{"test", false},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			result := parser.CanParse(tc.filename)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGoFileParser_ParseFile_BasicAnnotation(t *testing.T) {
	parser := NewGoFileParser()

	// Create temporary Go file with basic ServiceSpec annotation
	tmpDir := t.TempDir()
	goFile := filepath.Join(tmpDir, "user_service.go")

	content := `package service

import (
	"context"
	"github.com/example/models"
)

type UserService struct {
	repo UserRepository
}

// @ServiceSpec
// operationId: "createUser"
// description: "Create a new user account"
// preconditions:
//   "email_validation": {
//     "and": [
//       {"!=": [{"var": "request.body.email"}, null]},
//       {"regex": [{"var": "request.body.email"}, "^[\\w\\.-]+@[\\w\\.-]+\\.[a-zA-Z]{2,}$"]}
//     ]
//   }
//   "password_strength": {
//     ">=": [{"strlen": [{"var": "request.body.password"}]}, 8]
//   }
// postconditions:
//   "successful_creation": {
//     "and": [
//       {"==": [{"var": "response.status"}, 201]},
//       {"!=": [{"var": "response.body.userId"}, null]}
//     ]
//   }
func (s *UserService) CreateUser(ctx context.Context, request *CreateUserRequest) (*User, error) {
	return s.repo.Save(ctx, models.NewUser(request))
}`

	err := os.WriteFile(goFile, []byte(content), 0644)
	require.NoError(t, err)

	specs, errors := parser.ParseFile(goFile)

	assert.Empty(t, errors)
	assert.Len(t, specs, 1)

	spec := specs[0]
	assert.Equal(t, "createUser", spec.OperationID)
	assert.Equal(t, "Create a new user account", spec.Description)
	assert.Equal(t, goFile, spec.SourceFile)
	assert.Equal(t, 12, spec.LineNumber) // Line where @ServiceSpec appears

	// Verify preconditions
	assert.NotNil(t, spec.Preconditions)
	assert.Contains(t, spec.Preconditions, "email_validation")
	assert.Contains(t, spec.Preconditions, "password_strength")

	// Verify postconditions
	assert.NotNil(t, spec.Postconditions)
	assert.Contains(t, spec.Postconditions, "successful_creation")
}

func TestGoFileParser_ParseFile_MultiLineComment(t *testing.T) {
	parser := NewGoFileParser()

	// Create temporary Go file with multi-line comment ServiceSpec annotation
	tmpDir := t.TempDir()
	goFile := filepath.Join(tmpDir, "order_service.go")

	content := `package service

/*
 * @ServiceSpec
 * operationId: "processOrder"
 * description: "Process customer order"
 * preconditions:
 *   "request.body.customerId": {"!=": null}
 *   "request.body.items": {">= ": 1}
 * postconditions:
 *   "response.status": {"==": 200}
 *   "response.body.orderId": {"!=": null}
 */
func ProcessOrder(request *OrderRequest) (*OrderResponse, error) {
	return orderProcessor.Process(request)
}`

	err := os.WriteFile(goFile, []byte(content), 0644)
	require.NoError(t, err)

	specs, errors := parser.ParseFile(goFile)

	assert.Empty(t, errors)
	assert.Len(t, specs, 1)

	spec := specs[0]
	assert.Equal(t, "processOrder", spec.OperationID)
	assert.Equal(t, "Process customer order", spec.Description)
	assert.Equal(t, goFile, spec.SourceFile)

	// Verify preconditions
	assert.NotNil(t, spec.Preconditions)
	assert.Contains(t, spec.Preconditions, "request.body.customerId")
	assert.Contains(t, spec.Preconditions, "request.body.items")

	// Verify postconditions
	assert.NotNil(t, spec.Postconditions)
	assert.Contains(t, spec.Postconditions, "response.status")
	assert.Contains(t, spec.Postconditions, "response.body.orderId")
}

func TestGoFileParser_ParseFile_MultipleAnnotations(t *testing.T) {
	parser := NewGoFileParser()

	// Create temporary Go file with multiple ServiceSpec annotations
	tmpDir := t.TempDir()
	goFile := filepath.Join(tmpDir, "user_handler.go")

	content := `package handler

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service UserService
}

// @ServiceSpec
// operationId: "createUser"
// description: "Create a new user"
// preconditions:
//   "request.body.email": {"!=": null}
// postconditions:
//   "response.status": {"==": 201}
func (h *UserHandler) CreateUser(c *gin.Context) {
	// Implementation
	c.JSON(http.StatusCreated, gin.H{"status": "created"})
}

// @ServiceSpec
// operationId: "getUser"
// description: "Retrieve user by ID"
// preconditions:
//   "request.path.userId": {"!=": null}
// postconditions:
//   "response.status": {"==": 200}
//   "response.body.id": {"!=": null}
func (h *UserHandler) GetUser(c *gin.Context) {
	// Implementation
	c.JSON(http.StatusOK, gin.H{"id": "123"})
}

// @ServiceSpec
// operationId: "updateUser"
// description: "Update existing user"
// preconditions:
//   "request.path.userId": {"!=": null}
//   "request.body.email": {"!=": null}
// postconditions:
//   "response.status": {"==": 200}
func (h *UserHandler) UpdateUser(c *gin.Context) {
	// Implementation
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}`

	err := os.WriteFile(goFile, []byte(content), 0644)
	require.NoError(t, err)

	specs, errors := parser.ParseFile(goFile)

	assert.Empty(t, errors)
	assert.Len(t, specs, 3)

	// Verify first spec
	assert.Equal(t, "createUser", specs[0].OperationID)
	assert.Equal(t, "Create a new user", specs[0].Description)

	// Verify second spec
	assert.Equal(t, "getUser", specs[1].OperationID)
	assert.Equal(t, "Retrieve user by ID", specs[1].Description)

	// Verify third spec
	assert.Equal(t, "updateUser", specs[2].OperationID)
	assert.Equal(t, "Update existing user", specs[2].Description)
}

func TestGoFileParser_ParseFile_ComplexJSONLogicExpressions(t *testing.T) {
	parser := NewGoFileParser()

	// Create temporary Go file with complex JSONLogic expressions
	tmpDir := t.TempDir()
	goFile := filepath.Join(tmpDir, "payment_service.go")

	content := `package service

import "context"

type PaymentService struct {
	processor PaymentProcessor
}

// @ServiceSpec
// operationId: "processPayment"
// description: "Process payment with complex validation"
// preconditions:
//   "amount_validation": {
//     "and": [
//       {"!=": [{"var": "request.body.amount"}, null]},
//       {">": [{"var": "request.body.amount"}, 0]},
//       {"<=": [{"var": "request.body.amount"}, 10000]}
//     ]
//   }
//   "payment_method_validation": {
//     "or": [
//       {"==": [{"var": "request.body.paymentMethod"}, "CREDIT_CARD"]},
//       {"==": [{"var": "request.body.paymentMethod"}, "DEBIT_CARD"]},
//       {"==": [{"var": "request.body.paymentMethod"}, "PAYPAL"]}
//     ]
//   }
// postconditions:
//   "success_response": {
//     "and": [
//       {"==": [{"var": "response.status"}, 200]},
//       {"!=": [{"var": "response.body.transactionId"}, null]},
//       {"in": [{"var": "response.body.status"}, ["SUCCESS", "PENDING"]]}
//     ]
//   }
func (s *PaymentService) ProcessPayment(ctx context.Context, request *PaymentRequest) (*PaymentResponse, error) {
	response, err := s.processor.Process(ctx, request)
	return response, err
}`

	err := os.WriteFile(goFile, []byte(content), 0644)
	require.NoError(t, err)

	specs, errors := parser.ParseFile(goFile)

	assert.Empty(t, errors)
	assert.Len(t, specs, 1)

	spec := specs[0]
	assert.Equal(t, "processPayment", spec.OperationID)
	assert.Equal(t, "Process payment with complex validation", spec.Description)

	// Verify complex preconditions
	assert.NotNil(t, spec.Preconditions)
	assert.Contains(t, spec.Preconditions, "amount_validation")
	assert.Contains(t, spec.Preconditions, "payment_method_validation")

	// Verify the structure of complex expressions
	amountValidation, ok := spec.Preconditions["amount_validation"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, amountValidation, "and")

	paymentMethodValidation, ok := spec.Preconditions["payment_method_validation"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, paymentMethodValidation, "or")

	// Verify complex postconditions
	assert.NotNil(t, spec.Postconditions)
	assert.Contains(t, spec.Postconditions, "success_response")

	successResponse, ok := spec.Postconditions["success_response"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, successResponse, "and")
}

func TestGoFileParser_ParseFile_MinimalAnnotation(t *testing.T) {
	parser := NewGoFileParser()

	// Create temporary Go file with minimal ServiceSpec annotation (no conditions)
	tmpDir := t.TempDir()
	goFile := filepath.Join(tmpDir, "health_service.go")

	content := `package service

// @ServiceSpec
// operationId: "healthCheck"
// description: "Health check endpoint"
func HealthCheck() map[string]string {
	return map[string]string{"status": "OK"}
}`

	err := os.WriteFile(goFile, []byte(content), 0644)
	require.NoError(t, err)

	specs, errors := parser.ParseFile(goFile)

	assert.Empty(t, errors)
	assert.Len(t, specs, 1)

	spec := specs[0]
	assert.Equal(t, "healthCheck", spec.OperationID)
	assert.Equal(t, "Health check endpoint", spec.Description)

	// Should have empty but non-nil conditions
	assert.NotNil(t, spec.Preconditions)
	assert.Empty(t, spec.Preconditions)
	assert.NotNil(t, spec.Postconditions)
	assert.Empty(t, spec.Postconditions)
}

func TestGoFileParser_ParseFile_ErrorHandling(t *testing.T) {
	parser := NewGoFileParser()

	testCases := []struct {
		name        string
		content     string
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing_operation_id",
			content: `// @ServiceSpec
// description: "Test operation"
func test() {}`,
			expectError: true,
			errorMsg:    "missing or invalid operationId",
		},
		{
			name: "missing_description",
			content: `// @ServiceSpec
// operationId: "test"
func test() {}`,
			expectError: true,
			errorMsg:    "missing or invalid description",
		},
		{
			name: "invalid_yaml",
			content: `// @ServiceSpec
// operationId: "test"
// description: "Test"
// preconditions: [unclosed bracket
func test() {}`,
			expectError: true,
			errorMsg:    "failed to parse annotation content",
		},
		{
			name: "invalid_preconditions_type",
			content: `// @ServiceSpec
// operationId: "test"
// description: "Test"
// preconditions: "invalid_string_type"
func test() {}`,
			expectError: true,
			errorMsg:    "preconditions must be a map/object",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			goFile := filepath.Join(tmpDir, "test.go")

			err := os.WriteFile(goFile, []byte(tc.content), 0644)
			require.NoError(t, err)

			specs, errors := parser.ParseFile(goFile)

			if tc.expectError {
				assert.Empty(t, specs)
				assert.NotEmpty(t, errors)
				assert.Contains(t, errors[0].Message, tc.errorMsg)
			} else {
				assert.NotEmpty(t, specs)
				assert.Empty(t, errors)
			}
		})
	}
}

func TestGoFileParser_ParseFile_EdgeCases(t *testing.T) {
	parser := NewGoFileParser()

	testCases := []struct {
		name     string
		content  string
		expected int // number of specs expected
	}{
		{
			name:     "empty_file",
			content:  "",
			expected: 0,
		},
		{
			name: "no_servicespec_annotations",
			content: `package main

func main() {
	// Regular comment
	println("Hello, World!")
}`,
			expected: 0,
		},
		{
			name: "commented_out_servicespec",
			content: `package main

func main() {
	// This is not a ServiceSpec annotation
	println("Hello, World!")
}`,
			expected: 0,
		},
		{
			name: "mixed_comment_styles",
			content: `package main

/* 
 * @ServiceSpec
 * operationId: "test1"
 * description: "Test with block comment style"
 */
func test1() {}

// @ServiceSpec
// operationId: "test2"
// description: "Test with single-line comment style"
func test2() {}`,
			expected: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			goFile := filepath.Join(tmpDir, "test.go")

			err := os.WriteFile(goFile, []byte(tc.content), 0644)
			require.NoError(t, err)

			specs, errors := parser.ParseFile(goFile)

			assert.Len(t, specs, tc.expected)
			assert.Empty(t, errors) // No errors expected for these cases
		})
	}
}

func TestGoFileParser_ParseFile_FileNotFound(t *testing.T) {
	parser := NewGoFileParser()

	specs, errors := parser.ParseFile("/non/existent/file.go")

	assert.Empty(t, specs)
	assert.Len(t, errors, 1)
	assert.Contains(t, errors[0].Message, "failed to open file")
}

func TestGoFileParser_ParseFile_RealWorldExample(t *testing.T) {
	parser := NewGoFileParser()

	// Create a realistic Go service file
	tmpDir := t.TempDir()
	goFile := filepath.Join(tmpDir, "user_management_service.go")

	content := `package service

import (
	"context"
	"errors"
	"time"
	
	"github.com/example/models"
	"github.com/example/repository"
)

type UserManagementService struct {
	userRepo     repository.UserRepository
	passwordHash PasswordHasher
}

func NewUserManagementService(userRepo repository.UserRepository, hasher PasswordHasher) *UserManagementService {
	return &UserManagementService{
		userRepo:     userRepo,
		passwordHash: hasher,
	}
}

// CreateUserAccount creates a new user account with email validation
//
// @ServiceSpec
// operationId: "createUserAccount"
// description: "Create a new user account with comprehensive validation"
// preconditions:
//   "email_validation": {
//     "and": [
//       {"!=": [{"var": "request.body.email"}, null]},
//       {"regex": [{"var": "request.body.email"}, "^[\\w\\.-]+@[\\w\\.-]+\\.[a-zA-Z]{2,}$"]}
//     ]
//   }
//   "password_strength": {
//     "and": [
//       {"!=": [{"var": "request.body.password"}, null]},
//       {">": [{"strlen": [{"var": "request.body.password"}]}, 8]},
//       {"regex": [{"var": "request.body.password"}, ".*[A-Z].*"]},
//       {"regex": [{"var": "request.body.password"}, ".*[0-9].*"]}
//     ]
//   }
//   "unique_email": {
//     "==": [{"var": "database.users.email_exists"}, false]
//   }
// postconditions:
//   "successful_creation": {
//     "and": [
//       {"==": [{"var": "response.status"}, 201]},
//       {"!=": [{"var": "response.body.userId"}, null]},
//       {"==": [{"var": "response.body.email"}, {"var": "request.body.email"}]},
//       {"!=": [{"var": "response.body.createdAt"}, null]}
//     ]
//   }
//   "password_security": {
//     "==": [{"var": "response.body.password"}, null]
//   }
func (s *UserManagementService) CreateUserAccount(ctx context.Context, request *CreateUserRequest) (*UserResponse, error) {
	// Validate email uniqueness
	existingUser, err := s.userRepo.FindByEmail(ctx, request.Email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, errors.New("email already registered")
	}
	
	// Create and save user
	user := &models.User{
		Email:        request.Email,
		PasswordHash: s.passwordHash.Hash(request.Password),
		CreatedAt:    time.Now(),
	}
	
	savedUser, err := s.userRepo.Save(ctx, user)
	if err != nil {
		return nil, err
	}
	
	// Return response without password
	return &UserResponse{
		UserID:    savedUser.ID,
		Email:     savedUser.Email,
		CreatedAt: savedUser.CreatedAt,
	}, nil
}

// @ServiceSpec
// operationId: "getUserProfile"
// description: "Retrieve user profile by ID"
// preconditions:
//   "valid_user_id": {
//     "and": [
//       {"!=": [{"var": "request.path.userId"}, null]},
//       {"regex": [{"var": "request.path.userId"}, "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"]}
//     ]
//   }
//   "user_exists": {
//     "==": [{"var": "database.users.exists"}, true]
//   }
// postconditions:
//   "successful_retrieval": {
//     "and": [
//       {"==": [{"var": "response.status"}, 200]},
//       {"==": [{"var": "response.body.userId"}, {"var": "request.path.userId"}]},
//       {"!=": [{"var": "response.body.email"}, null]}
//     ]
//   }
func (s *UserManagementService) GetUserProfile(ctx context.Context, userID string) (*UserResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}
	
	return &UserResponse{
		UserID:    user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}, nil
}`

	err := os.WriteFile(goFile, []byte(content), 0644)
	require.NoError(t, err)

	specs, errors := parser.ParseFile(goFile)

	assert.Empty(t, errors)
	assert.Len(t, specs, 2)

	// Verify first spec (createUserAccount)
	createUserSpec := specs[0]
	assert.Equal(t, "createUserAccount", createUserSpec.OperationID)
	assert.Equal(t, "Create a new user account with comprehensive validation", createUserSpec.Description)

	// Verify complex preconditions
	assert.Contains(t, createUserSpec.Preconditions, "email_validation")
	assert.Contains(t, createUserSpec.Preconditions, "password_strength")
	assert.Contains(t, createUserSpec.Preconditions, "unique_email")

	// Verify complex postconditions
	assert.Contains(t, createUserSpec.Postconditions, "successful_creation")
	assert.Contains(t, createUserSpec.Postconditions, "password_security")

	// Verify second spec (getUserProfile)
	getUserSpec := specs[1]
	assert.Equal(t, "getUserProfile", getUserSpec.OperationID)
	assert.Equal(t, "Retrieve user profile by ID", getUserSpec.Description)

	assert.Contains(t, getUserSpec.Preconditions, "valid_user_id")
	assert.Contains(t, getUserSpec.Preconditions, "user_exists")
	assert.Contains(t, getUserSpec.Postconditions, "successful_retrieval")
}

func TestGoFileParser_Integration_WithSpecParser(t *testing.T) {
	// Test integration with the main SpecParser
	specParser := NewSpecParser()
	goParser := NewGoFileParser()
	specParser.RegisterFileParser(LanguageGo, goParser)

	// Create test directory with Go files
	tmpDir := t.TempDir()

	// Create first Go file
	goFile1 := filepath.Join(tmpDir, "service1.go")
	content1 := `// @ServiceSpec
// operationId: "service1Operation"
// description: "Service 1 operation"
func Service1Operation() {}`
	err := os.WriteFile(goFile1, []byte(content1), 0644)
	require.NoError(t, err)

	// Create second Go file
	goFile2 := filepath.Join(tmpDir, "service2.go")
	content2 := `// @ServiceSpec
// operationId: "service2Operation"
// description: "Service 2 operation"
func Service2Operation() {}`
	err = os.WriteFile(goFile2, []byte(content2), 0644)
	require.NoError(t, err)

	// Parse the directory
	result, err := specParser.ParseFromSource(tmpDir)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Specs, 2)
	assert.Empty(t, result.Errors)

	// Verify both specs were parsed
	operationIds := []string{result.Specs[0].OperationID, result.Specs[1].OperationID}
	assert.Contains(t, operationIds, "service1Operation")
	assert.Contains(t, operationIds, "service2Operation")
}
