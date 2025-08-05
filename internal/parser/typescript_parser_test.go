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

func TestNewTypeScriptFileParser(t *testing.T) {
	parser := NewTypeScriptFileParser()

	assert.NotNil(t, parser)
	assert.NotNil(t, parser.BaseFileParser)
	assert.Equal(t, LanguageTypeScript, parser.GetLanguage())
}

func TestTypeScriptFileParser_CanParse(t *testing.T) {
	parser := NewTypeScriptFileParser()

	testCases := []struct {
		filename string
		expected bool
	}{
		{"test.ts", true},
		{"test.tsx", true},
		{"components/UserService.ts", true},
		{"types/User.d.ts", true},
		{"test.js", false},
		{"test.java", false},
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

func TestTypeScriptFileParser_ParseFile_BasicAnnotation(t *testing.T) {
	parser := NewTypeScriptFileParser()

	// Create temporary TypeScript file with basic ServiceSpec annotation
	tmpDir := t.TempDir()
	tsFile := filepath.Join(tmpDir, "UserService.ts")

	content := `import { Injectable } from '@nestjs/common';
import { User } from './types/User';

@Injectable()
export class UserService {

    /**
     * @ServiceSpec
     * operationId: "createUser"
     * description: "Create a new user account"
     * preconditions:
     *   "email_validation": {
     *     "and": [
     *       {"!=": [{"var": "request.body.email"}, null]},
     *       {"regex": [{"var": "request.body.email"}, "^[\\w\\.-]+@[\\w\\.-]+\\.[a-zA-Z]{2,}$"]}
     *     ]
     *   }
     *   "password_strength": {
     *     ">=": [{"strlen": [{"var": "request.body.password"}]}, 8]
     *   }
     * postconditions:
     *   "successful_creation": {
     *     "and": [
     *       {"==": [{"var": "response.status"}, 201]},
     *       {"!=": [{"var": "response.body.userId"}, null]}
     *     ]
     *   }
     */
    async createUser(request: CreateUserRequest): Promise<User> {
        return this.userRepository.save(new User(request));
    }
}`

	err := os.WriteFile(tsFile, []byte(content), 0644)
	require.NoError(t, err)

	specs, errors := parser.ParseFile(tsFile)

	assert.Empty(t, errors)
	assert.Len(t, specs, 1)

	spec := specs[0]
	assert.Equal(t, "createUser", spec.OperationID)
	assert.Equal(t, "Create a new user account", spec.Description)
	assert.Equal(t, tsFile, spec.SourceFile)
	assert.Equal(t, 8, spec.LineNumber) // Line where @ServiceSpec appears

	// Verify preconditions
	assert.NotNil(t, spec.Preconditions)
	assert.Contains(t, spec.Preconditions, "email_validation")
	assert.Contains(t, spec.Preconditions, "password_strength")

	// Verify postconditions
	assert.NotNil(t, spec.Postconditions)
	assert.Contains(t, spec.Postconditions, "successful_creation")
}

func TestTypeScriptFileParser_ParseFile_JSONFormat(t *testing.T) {
	parser := NewTypeScriptFileParser()

	// Create temporary TypeScript file with JSON format ServiceSpec annotation
	tmpDir := t.TempDir()
	tsFile := filepath.Join(tmpDir, "OrderService.ts")

	content := `import { Controller, Post, Body } from '@nestjs/common';

@Controller('orders')
export class OrderController {

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
    @Post()
    async processOrder(@Body() request: OrderRequest): Promise<Order> {
        return this.orderService.process(request);
    }
}`

	err := os.WriteFile(tsFile, []byte(content), 0644)
	require.NoError(t, err)

	specs, errors := parser.ParseFile(tsFile)

	assert.Empty(t, errors)
	assert.Len(t, specs, 1)

	spec := specs[0]
	assert.Equal(t, "processOrder", spec.OperationID)
	assert.Equal(t, "Process customer order", spec.Description)
	assert.Equal(t, tsFile, spec.SourceFile)

	// Verify preconditions
	assert.NotNil(t, spec.Preconditions)
	assert.Contains(t, spec.Preconditions, "request.body.customerId")
	assert.Contains(t, spec.Preconditions, "request.body.items")

	// Verify postconditions
	assert.NotNil(t, spec.Postconditions)
	assert.Contains(t, spec.Postconditions, "response.status")
	assert.Contains(t, spec.Postconditions, "response.body.orderId")
}

func TestTypeScriptFileParser_ParseFile_SingleLineComments(t *testing.T) {
	parser := NewTypeScriptFileParser()

	// Create temporary TypeScript file with single-line comment ServiceSpec annotation
	tmpDir := t.TempDir()
	tsFile := filepath.Join(tmpDir, "HealthService.ts")

	content := `export class HealthService {

    // @ServiceSpec
    // operationId: "healthCheck"
    // description: "Health check endpoint"
    // preconditions:
    //   "always_true": {"==": [1, 1]}
    // postconditions:
    //   "success_response": {"==": [{"var": "response.status"}, 200]}
    async healthCheck(): Promise<{ status: string }> {
        return { status: 'OK' };
    }
}`

	err := os.WriteFile(tsFile, []byte(content), 0644)
	require.NoError(t, err)

	specs, errors := parser.ParseFile(tsFile)

	assert.Empty(t, errors)
	assert.Len(t, specs, 1)

	spec := specs[0]
	assert.Equal(t, "healthCheck", spec.OperationID)
	assert.Equal(t, "Health check endpoint", spec.Description)
	assert.Equal(t, tsFile, spec.SourceFile)
	assert.Equal(t, 3, spec.LineNumber) // Line where @ServiceSpec appears

	// Verify preconditions
	assert.NotNil(t, spec.Preconditions)
	assert.Contains(t, spec.Preconditions, "always_true")

	// Verify postconditions
	assert.NotNil(t, spec.Postconditions)
	assert.Contains(t, spec.Postconditions, "success_response")
}

func TestTypeScriptFileParser_ParseFile_MultipleAnnotations(t *testing.T) {
	parser := NewTypeScriptFileParser()

	// Create temporary TypeScript file with multiple ServiceSpec annotations
	tmpDir := t.TempDir()
	tsFile := filepath.Join(tmpDir, "UserController.ts")

	content := `import { Controller, Get, Post, Put, Param, Body } from '@nestjs/common';

@Controller('users')
export class UserController {

    /**
     * @ServiceSpec
     * operationId: "createUser"
     * description: "Create a new user"
     * preconditions:
     *   "request.body.email": {"!=": null}
     * postconditions:
     *   "response.status": {"==": 201}
     */
    @Post()
    async createUser(@Body() request: CreateUserRequest): Promise<User> {
        return this.userService.create(request);
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
    @Get(':userId')
    async getUser(@Param('userId') userId: string): Promise<User> {
        return this.userService.findById(userId);
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
    @Put(':userId')
    async updateUser(
        @Param('userId') userId: string, 
        @Body() request: UpdateUserRequest
    ): Promise<User> {
        return this.userService.update(userId, request);
    }
}`

	err := os.WriteFile(tsFile, []byte(content), 0644)
	require.NoError(t, err)

	specs, errors := parser.ParseFile(tsFile)

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

func TestTypeScriptFileParser_ParseFile_ComplexJSONLogicExpressions(t *testing.T) {
	parser := NewTypeScriptFileParser()

	// Create temporary TypeScript file with complex JSONLogic expressions
	tmpDir := t.TempDir()
	tsFile := filepath.Join(tmpDir, "PaymentService.ts")

	content := `import { Injectable } from '@nestjs/common';

@Injectable()
export class PaymentService {

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
    async processPayment(request: PaymentRequest): Promise<PaymentResponse> {
        const response = await this.paymentProcessor.process(request);
        return response;
    }
}`

	err := os.WriteFile(tsFile, []byte(content), 0644)
	require.NoError(t, err)

	specs, errors := parser.ParseFile(tsFile)

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

func TestTypeScriptFileParser_ParseFile_TSXFile(t *testing.T) {
	parser := NewTypeScriptFileParser()

	// Create temporary TSX file with ServiceSpec annotation
	tmpDir := t.TempDir()
	tsxFile := filepath.Join(tmpDir, "UserComponent.tsx")

	content := `import React from 'react';

interface UserComponentProps {
    userId: string;
}

export const UserComponent: React.FC<UserComponentProps> = ({ userId }) => {

    /**
     * @ServiceSpec
     * operationId: "fetchUserData"
     * description: "Fetch user data for display"
     * preconditions:
     *   "valid_user_id": {
     *     "and": [
     *       {"!=": [{"var": "props.userId"}, null]},
     *       {"!=": [{"var": "props.userId"}, ""]}
     *     ]
     *   }
     * postconditions:
     *   "data_loaded": {
     *     "!=": [{"var": "response.data"}, null]
     *   }
     */
    const fetchUserData = async (userId: string): Promise<UserData> => {
        const response = await fetch('/api/users/' + userId);
        return response.json();
    };

    return (
        <div>
            <h1>User Profile</h1>
        </div>
    );
};
`

	err := os.WriteFile(tsxFile, []byte(content), 0644)
	require.NoError(t, err)

	specs, errors := parser.ParseFile(tsxFile)

	assert.Empty(t, errors)
	assert.Len(t, specs, 1)

	spec := specs[0]
	assert.Equal(t, "fetchUserData", spec.OperationID)
	assert.Equal(t, "Fetch user data for display", spec.Description)
	assert.Equal(t, tsxFile, spec.SourceFile)

	// Verify preconditions
	assert.NotNil(t, spec.Preconditions)
	assert.Contains(t, spec.Preconditions, "valid_user_id")

	// Verify postconditions
	assert.NotNil(t, spec.Postconditions)
	assert.Contains(t, spec.Postconditions, "data_loaded")
}

func TestTypeScriptFileParser_ParseFile_ErrorHandling(t *testing.T) {
	parser := NewTypeScriptFileParser()

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
function test() {}`,
			expectError: true,
			errorMsg:    "missing or invalid operationId",
		},
		{
			name: "missing_description",
			content: `/**
 * @ServiceSpec
 * operationId: "test"
 */
function test() {}`,
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
function test() {}`,
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
function test() {}`,
			expectError: true,
			errorMsg:    "preconditions must be a map/object",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tsFile := filepath.Join(tmpDir, "Test.ts")

			err := os.WriteFile(tsFile, []byte(tc.content), 0644)
			require.NoError(t, err)

			specs, errors := parser.ParseFile(tsFile)

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

func TestTypeScriptFileParser_ParseFile_EdgeCases(t *testing.T) {
	parser := NewTypeScriptFileParser()

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
			content: `export class Test {
    // Regular comment
    method() {}
}`,
			expected: 0,
		},
		{
			name: "commented_out_servicespec",
			content: `export class Test {
    // This is not a ServiceSpec annotation
    method() {}
}`,
			expected: 0,
		},
		{
			name: "mixed_comment_styles",
			content: `export class Test {

    /* 
     * @ServiceSpec
     * operationId: "test1"
     * description: "Test with block comment style"
     */
    test1() {}

    /**
     * @ServiceSpec
     * operationId: "test2"
     * description: "Test with JSDoc comment style"
     */
    test2() {}

    // @ServiceSpec
    // operationId: "test3"
    // description: "Test with single-line comment style"
    test3() {}
}`,
			expected: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tsFile := filepath.Join(tmpDir, "Test.ts")

			err := os.WriteFile(tsFile, []byte(tc.content), 0644)
			require.NoError(t, err)

			specs, errors := parser.ParseFile(tsFile)

			assert.Len(t, specs, tc.expected)
			assert.Empty(t, errors) // No errors expected for these cases
		})
	}
}

func TestTypeScriptFileParser_ParseFile_FileNotFound(t *testing.T) {
	parser := NewTypeScriptFileParser()

	specs, errors := parser.ParseFile("/non/existent/file.ts")

	assert.Empty(t, specs)
	assert.Len(t, errors, 1)
	assert.Contains(t, errors[0].Message, "failed to open file")
}

func TestTypeScriptFileParser_ParseFile_RealWorldExample(t *testing.T) {
	parser := NewTypeScriptFileParser()

	// Create a realistic TypeScript service file
	tmpDir := t.TempDir()
	tsFile := filepath.Join(tmpDir, "UserManagementService.ts")

	content := `import { Injectable, BadRequestException, NotFoundException } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import { User } from '../entities/User';
import { CreateUserRequest, UpdateUserRequest, UserResponse } from '../dto/user.dto';

@Injectable()
export class UserManagementService {

    constructor(
        @InjectRepository(User)
        private readonly userRepository: Repository<User>
    ) {}

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
     *       {"regex": [{"var": "request.body.password"}, ".*\\\[A-Z\\].*"]},
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
    async createUserAccount(request: CreateUserRequest): Promise<UserResponse> {
        // Validate email uniqueness
        const existingUser = await this.userRepository.findOne({ 
            where: { email: request.email } 
        });
        
        if (existingUser) {
            throw new BadRequestException('Email already registered');
        }
        
        // Create and save user
        const user = new User();
        user.email = request.email;
        user.passwordHash = await this.hashPassword(request.password);
        user.createdAt = new Date();
        
        const savedUser = await this.userRepository.save(user);
        
        // Return response without password
        return {
            userId: savedUser.id,
            email: savedUser.email,
            createdAt: savedUser.createdAt
        };
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
    async getUserProfile(userId: string): Promise<UserResponse> {
        const user = await this.userRepository.findOne({ where: { id: userId } });
        
        if (!user) {
            throw new NotFoundException('User not found');
        }
        
        return {
            userId: user.id,
            email: user.email,
            createdAt: user.createdAt
        };
    }

    private async hashPassword(password: string): Promise<string> {
        // Implementation would use bcrypt or similar
        return 'hashed_' + password;
    }
}`

	err := os.WriteFile(tsFile, []byte(content), 0644)
	require.NoError(t, err)

	specs, errors := parser.ParseFile(tsFile)

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

func TestTypeScriptFileParser_Integration_WithSpecParser(t *testing.T) {
	// Test integration with the main SpecParser
	specParser := NewSpecParser()
	tsParser := NewTypeScriptFileParser()
	specParser.RegisterFileParser(LanguageTypeScript, tsParser)

	// Create test directory with TypeScript files
	tmpDir := t.TempDir()

	// Create first TypeScript file
	tsFile1 := filepath.Join(tmpDir, "Service1.ts")
	content1 := `/**
 * @ServiceSpec
 * operationId: "service1Operation"
 * description: "Service 1 operation"
 */
export function service1Operation() {}`
	err := os.WriteFile(tsFile1, []byte(content1), 0644)
	require.NoError(t, err)

	// Create second TypeScript file
	tsFile2 := filepath.Join(tmpDir, "Service2.tsx")
	content2 := `/**
 * @ServiceSpec
 * operationId: "service2Operation"
 * description: "Service 2 operation"
 */
export const service2Operation = () => {}`
	err = os.WriteFile(tsFile2, []byte(content2), 0644)
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