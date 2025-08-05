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

func TestNewJavaFileParser(t *testing.T) {
	parser := NewJavaFileParser()

	assert.NotNil(t, parser)
	assert.NotNil(t, parser.BaseFileParser)
	assert.Equal(t, LanguageJava, parser.GetLanguage())
}

func TestJavaFileParser_CanParse(t *testing.T) {
	parser := NewJavaFileParser()

	testCases := []struct {
		filename string
		expected bool
	}{
		{"Test.java", true},
		{"com/example/Service.java", true},
		{"test.ts", false},
		{"test.go", false},
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

func TestJavaFileParser_ParseFile_BasicAnnotation(t *testing.T) {
	parser := NewJavaFileParser()

	// Create temporary Java file with basic ServiceSpec annotation
	tmpDir := t.TempDir()
	javaFile := filepath.Join(tmpDir, "UserService.java")

	content := `package com.example.service;

import com.example.model.User;

/**
 * @ServiceSpec
 * operationId: "createUser"
 * description: "Create a new user account"
 * preconditions:
 *   "request.body.email": {"!=": null}
 *   "request.body.password": {">=": 8}
 * postconditions:
 *   "response.status": {"==": 201}
 *   "response.body.userId": {"!=": null}
 */
public User createUser(CreateUserRequest request) {
    // Implementation
    return userRepository.save(new User(request));
}`

	err := os.WriteFile(javaFile, []byte(content), 0644)
	require.NoError(t, err)

	specs, errors := parser.ParseFile(javaFile)

	assert.Empty(t, errors)
	assert.Len(t, specs, 1)

	spec := specs[0]
	assert.Equal(t, "createUser", spec.OperationID)
	assert.Equal(t, "Create a new user account", spec.Description)
	assert.Equal(t, javaFile, spec.SourceFile)
	assert.Equal(t, 6, spec.LineNumber) // Line where @ServiceSpec appears

	// Verify preconditions
	assert.NotNil(t, spec.Preconditions)
	assert.Contains(t, spec.Preconditions, "request.body.email")
	assert.Contains(t, spec.Preconditions, "request.body.password")

	// Verify postconditions
	assert.NotNil(t, spec.Postconditions)
	assert.Contains(t, spec.Postconditions, "response.status")
	assert.Contains(t, spec.Postconditions, "response.body.userId")
}

func TestJavaFileParser_ParseFile_JSONFormat(t *testing.T) {
	parser := NewJavaFileParser()

	// Create temporary Java file with JSON format ServiceSpec annotation
	tmpDir := t.TempDir()
	javaFile := filepath.Join(tmpDir, "OrderService.java")

	content := `package com.example.service;

/**
 * @ServiceSpec
 * {
 *   "operationId": "processOrder",
 *   "description": "Process customer order",
 *   "preconditions": {
 *     "request.body.customerId": {"!=": null},
 *     "request.body.items": {">=": 1}
 *   },
 *   "postconditions": {
 *     "response.status": {"==": 200},
 *     "response.body.orderId": {"!=": null}
 *   }
 * }
 */
@PostMapping("/orders")
public ResponseEntity<Order> processOrder(@RequestBody OrderRequest request) {
    return ResponseEntity.ok(orderService.process(request));
}`

	err := os.WriteFile(javaFile, []byte(content), 0644)
	require.NoError(t, err)

	specs, errors := parser.ParseFile(javaFile)

	assert.Empty(t, errors)
	assert.Len(t, specs, 1)

	spec := specs[0]
	assert.Equal(t, "processOrder", spec.OperationID)
	assert.Equal(t, "Process customer order", spec.Description)
	assert.Equal(t, javaFile, spec.SourceFile)

	// Verify preconditions
	assert.NotNil(t, spec.Preconditions)
	assert.Contains(t, spec.Preconditions, "request.body.customerId")
	assert.Contains(t, spec.Preconditions, "request.body.items")

	// Verify postconditions
	assert.NotNil(t, spec.Postconditions)
	assert.Contains(t, spec.Postconditions, "response.status")
	assert.Contains(t, spec.Postconditions, "response.body.orderId")
}

func TestJavaFileParser_ParseFile_MultipleAnnotations(t *testing.T) {
	parser := NewJavaFileParser()

	// Create temporary Java file with multiple ServiceSpec annotations
	tmpDir := t.TempDir()
	javaFile := filepath.Join(tmpDir, "UserController.java")

	content := `package com.example.controller;

@RestController
@RequestMapping("/users")
public class UserController {

    /**
     * @ServiceSpec
     * operationId: "createUser"
     * description: "Create a new user"
     * preconditions:
     *   "request.body.email": {"!=": null}
     * postconditions:
     *   "response.status": {"==": 201}
     */
    @PostMapping
    public ResponseEntity<User> createUser(@RequestBody CreateUserRequest request) {
        return ResponseEntity.status(201).body(userService.create(request));
    }

    /**
     * @ServiceSpec
     * operationId: "getUser"
     * description: "Retrieve user by ID"
     * preconditions:
     *   "request.path.userId": {"!=": null}
     * postconditions:
     *   "response.status": {"==": 200}
     *   "response.body.id": {"!=": null}
     */
    @GetMapping("/{userId}")
    public ResponseEntity<User> getUser(@PathVariable String userId) {
        return ResponseEntity.ok(userService.findById(userId));
    }

    /**
     * @ServiceSpec
     * operationId: "updateUser"
     * description: "Update existing user"
     * preconditions:
     *   "request.path.userId": {"!=": null}
     *   "request.body.email": {"!=": null}
     * postconditions:
     *   "response.status": {"==": 200}
     */
    @PutMapping("/{userId}")
    public ResponseEntity<User> updateUser(@PathVariable String userId, @RequestBody UpdateUserRequest request) {
        return ResponseEntity.ok(userService.update(userId, request));
    }
}`

	err := os.WriteFile(javaFile, []byte(content), 0644)
	require.NoError(t, err)

	specs, errors := parser.ParseFile(javaFile)

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

func TestJavaFileParser_ParseFile_ComplexJSONLogicExpressions(t *testing.T) {
	parser := NewJavaFileParser()

	// Create temporary Java file with complex JSONLogic expressions
	tmpDir := t.TempDir()
	javaFile := filepath.Join(tmpDir, "PaymentService.java")

	content := `package com.example.service;

/**
 * @ServiceSpec
 * operationId: "processPayment"
 * description: "Process payment with complex validation"
 * preconditions:
 *   "amount_validation": {
 *     "and": [
 *       {"!=": [{"var": "request.body.amount"}, null]},
 *       {">": [{"var": "request.body.amount"}, 0]},
 *       {"<=": [{"var": "request.body.amount"}, 10000]}
 *     ]
 *   }
 *   "payment_method_validation": {
 *     "or": [
 *       {"==": [{"var": "request.body.paymentMethod"}, "CREDIT_CARD"]},
 *       {"==": [{"var": "request.body.paymentMethod"}, "DEBIT_CARD"]},
 *       {"==": [{"var": "request.body.paymentMethod"}, "PAYPAL"]}
 *     ]
 *   }
 * postconditions:
 *   "success_response": {
 *     "and": [
 *       {"==": [{"var": "response.status"}, 200]},
 *       {"!=": [{"var": "response.body.transactionId"}, null]},
 *       {"in": [{"var": "response.body.status"}, ["SUCCESS", "PENDING"]]}
 *     ]
 *   }
 */
@PostMapping("/payments")
public ResponseEntity<PaymentResponse> processPayment(@RequestBody PaymentRequest request) {
    PaymentResponse response = paymentProcessor.process(request);
    return ResponseEntity.ok(response);
}`

	err := os.WriteFile(javaFile, []byte(content), 0644)
	require.NoError(t, err)

	specs, errors := parser.ParseFile(javaFile)

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

func TestJavaFileParser_ParseFile_MinimalAnnotation(t *testing.T) {
	parser := NewJavaFileParser()

	// Create temporary Java file with minimal ServiceSpec annotation (no conditions)
	tmpDir := t.TempDir()
	javaFile := filepath.Join(tmpDir, "HealthService.java")

	content := `package com.example.service;

/**
 * @ServiceSpec
 * operationId: "healthCheck"
 * description: "Health check endpoint"
 */
@GetMapping("/health")
public ResponseEntity<String> healthCheck() {
    return ResponseEntity.ok("OK");
}`

	err := os.WriteFile(javaFile, []byte(content), 0644)
	require.NoError(t, err)

	specs, errors := parser.ParseFile(javaFile)

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

func TestJavaFileParser_ParseFile_ErrorHandling(t *testing.T) {
	parser := NewJavaFileParser()

	testCases := []struct {
		name        string
		content     string
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing_operation_id",
			content: `/**
 * @ServiceSpec
 * description: "Test operation"
 */
public void test() {}`,
			expectError: true,
			errorMsg:    "missing or invalid operationId",
		},
		{
			name: "missing_description",
			content: `/**
 * @ServiceSpec
 * operationId: "test"
 */
public void test() {}`,
			expectError: true,
			errorMsg:    "missing or invalid description",
		},
		{
			name: "invalid_yaml",
			content: `/**
 * @ServiceSpec
 * operationId: "test"
 * description: "Test"
 * preconditions: [unclosed bracket
 */
public void test() {}`,
			expectError: true,
			errorMsg:    "failed to parse annotation content",
		},
		{
			name: "invalid_preconditions_type",
			content: `/**
 * @ServiceSpec
 * operationId: "test"
 * description: "Test"
 * preconditions: "invalid_string_type"
 */
public void test() {}`,
			expectError: true,
			errorMsg:    "preconditions must be a map/object",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			javaFile := filepath.Join(tmpDir, "Test.java")

			err := os.WriteFile(javaFile, []byte(tc.content), 0644)
			require.NoError(t, err)

			specs, errors := parser.ParseFile(javaFile)

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

func TestJavaFileParser_ParseFile_EdgeCases(t *testing.T) {
	parser := NewJavaFileParser()

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
			content: `package com.example;
public class Test {
    // Regular comment
    public void method() {}
}`,
			expected: 0,
		},
		{
			name: "commented_out_servicespec",
			content: `package com.example;
public class Test {
    // This is not a ServiceSpec annotation
    public void method() {}
}`,
			expected: 0,
		},
		{
			name: "mixed_comment_styles",
			content: `package com.example;

/* 
 * @ServiceSpec
 * operationId: "test1"
 * description: "Test with block comment style"
 */
public void test1() {}

/**
 * @ServiceSpec
 * operationId: "test2"
 * description: "Test with javadoc comment style"
 */
public void test2() {}`,
			expected: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			javaFile := filepath.Join(tmpDir, "Test.java")

			err := os.WriteFile(javaFile, []byte(tc.content), 0644)
			require.NoError(t, err)

			specs, errors := parser.ParseFile(javaFile)

			assert.Len(t, specs, tc.expected)
			assert.Empty(t, errors) // No errors expected for these cases
		})
	}
}

func TestJavaFileParser_ParseFile_FileNotFound(t *testing.T) {
	parser := NewJavaFileParser()

	specs, errors := parser.ParseFile("/non/existent/file.java")

	assert.Empty(t, specs)
	assert.Len(t, errors, 1)
	assert.Contains(t, errors[0].Message, "failed to open file")
}

func TestJavaFileParser_ParseFile_RealWorldExample(t *testing.T) {
	parser := NewJavaFileParser()

	// Create a realistic Java service file
	tmpDir := t.TempDir()
	javaFile := filepath.Join(tmpDir, "UserManagementService.java")

	content := `package com.example.service;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;
import org.springframework.web.bind.annotation.*;
import com.example.model.User;
import com.example.repository.UserRepository;

@Service
@RestController
@RequestMapping("/api/v1/users")
public class UserManagementService {

    @Autowired
    private UserRepository userRepository;

    /**
     * Creates a new user account with email validation
     * 
     * @ServiceSpec
     * operationId: "createUserAccount"
     * description: "Create a new user account with comprehensive validation"
     * preconditions:
     *   "email_validation": {
     *     "and": [
     *       {"!=": [{"var": "request.body.email"}, null]},
     *       {"regex": [{"var": "request.body.email"}, "^[\\w\\.-]+@[\\w\\.-]+\\.[a-zA-Z]{2,}$"]}
     *     ]
     *   }
     *   "password_strength": {
     *     "and": [
     *       {"!=": [{"var": "request.body.password"}, null]},
     *       {">=": [{"strlen": [{"var": "request.body.password"}]}, 8]},
     *       {"regex": [{"var": "request.body.password"}, ".*[A-Z].*"]},
     *       {"regex": [{"var": "request.body.password"}, ".*[0-9].*"]}
     *     ]
     *   }
     *   "unique_email": {
     *     "==": [{"var": "database.users.email_exists"}, false]
     *   }
     * postconditions:
     *   "successful_creation": {
     *     "and": [
     *       {"==": [{"var": "response.status"}, 201]},
     *       {"!=": [{"var": "response.body.userId"}, null]},
     *       {"==": [{"var": "response.body.email"}, {"var": "request.body.email"}]},
     *       {"!=": [{"var": "response.body.createdAt"}, null]}
     *     ]
     *   }
     *   "password_security": {
     *     "==": [{"var": "response.body.password"}, null]
     *   }
     */
    @PostMapping
    public ResponseEntity<UserResponse> createUserAccount(@RequestBody CreateUserRequest request) {
        // Validate email uniqueness
        if (userRepository.existsByEmail(request.getEmail())) {
            throw new EmailAlreadyExistsException("Email already registered");
        }
        
        // Create and save user
        User user = new User();
        user.setEmail(request.getEmail());
        user.setPasswordHash(passwordEncoder.encode(request.getPassword()));
        user.setCreatedAt(Instant.now());
        
        User savedUser = userRepository.save(user);
        
        // Return response without password
        UserResponse response = new UserResponse();
        response.setUserId(savedUser.getId());
        response.setEmail(savedUser.getEmail());
        response.setCreatedAt(savedUser.getCreatedAt());
        
        return ResponseEntity.status(201).body(response);
    }

    /**
     * @ServiceSpec
     * operationId: "getUserProfile"
     * description: "Retrieve user profile by ID"
     * preconditions:
     *   "valid_user_id": {
     *     "and": [
     *       {"!=": [{"var": "request.path.userId"}, null]},
     *       {"regex": [{"var": "request.path.userId"}, "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"]}
     *     ]
     *   }
     *   "user_exists": {
     *     "==": [{"var": "database.users.exists"}, true]
     *   }
     * postconditions:
     *   "successful_retrieval": {
     *     "and": [
     *       {"==": [{"var": "response.status"}, 200]},
     *       {"==": [{"var": "response.body.userId"}, {"var": "request.path.userId"}]},
     *       {"!=": [{"var": "response.body.email"}, null]}
     *     ]
     *   }
     */
    @GetMapping("/{userId}")
    public ResponseEntity<UserResponse> getUserProfile(@PathVariable String userId) {
        User user = userRepository.findById(userId)
            .orElseThrow(() -> new UserNotFoundException("User not found"));
        
        UserResponse response = new UserResponse();
        response.setUserId(user.getId());
        response.setEmail(user.getEmail());
        response.setCreatedAt(user.getCreatedAt());
        
        return ResponseEntity.ok(response);
    }
}`

	err := os.WriteFile(javaFile, []byte(content), 0644)
	require.NoError(t, err)

	specs, errors := parser.ParseFile(javaFile)

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

func TestJavaFileParser_Integration_WithSpecParser(t *testing.T) {
	// Test integration with the main SpecParser
	specParser := NewSpecParser()
	javaParser := NewJavaFileParser()
	specParser.RegisterFileParser(LanguageJava, javaParser)

	// Create test directory with Java files
	tmpDir := t.TempDir()

	// Create first Java file
	javaFile1 := filepath.Join(tmpDir, "Service1.java")
	content1 := `/**
 * @ServiceSpec
 * operationId: "service1Operation"
 * description: "Service 1 operation"
 */
public void service1Operation() {}`
	err := os.WriteFile(javaFile1, []byte(content1), 0644)
	require.NoError(t, err)

	// Create second Java file
	javaFile2 := filepath.Join(tmpDir, "Service2.java")
	content2 := `/**
 * @ServiceSpec
 * operationId: "service2Operation"
 * description: "Service 2 operation"
 */
public void service2Operation() {}`
	err = os.WriteFile(javaFile2, []byte(content2), 0644)
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