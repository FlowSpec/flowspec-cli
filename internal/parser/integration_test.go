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
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/flowspec/flowspec-cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiLanguageIntegration(t *testing.T) {
	// Create a temporary directory with mixed language files
	tmpDir := t.TempDir()

	// Create Java file with ServiceSpec
	javaFile := filepath.Join(tmpDir, "UserService.java")
	javaContent := `package com.example.service;

/**
 * @ServiceSpec
 * operationId: "createUser"
 * description: "Creates a new user in the system"
 * preconditions:
 *   "request.email": {"!=": [null, null]}
 *   "request.password": {"and": [{"!=": [null, null]}, {">": [{"var": "request.password.length"}, 8]}]}
 * postconditions:
 *   "response.status": {"==": [201, 201]}
 *   "response.user.id": {"!=": [null, null]}
 */
public class UserService {
    public User createUser(CreateUserRequest request) {
        // Implementation
        return new User();
    }
}`

	err := os.WriteFile(javaFile, []byte(javaContent), 0644)
	require.NoError(t, err)

	// Create TypeScript file with ServiceSpec
	tsFile := filepath.Join(tmpDir, "orderService.ts")
	tsContent := `/**
 * @ServiceSpec
 * operationId: "processOrder"
 * description: "Processes an order and updates inventory"
 * preconditions:
 *   "order.items": {"!=": [null, null]}
 *   "order.total": {">": [0, 0]}
 * postconditions:
 *   "response.orderId": {"!=": [null, null]}
 *   "response.status": {"==": ["processed", "processed"]}
 */
export class OrderService {
    processOrder(order: Order): OrderResponse {
        // Implementation
        return { orderId: "123", status: "processed" };
    }
}`

	err = os.WriteFile(tsFile, []byte(tsContent), 0644)
	require.NoError(t, err)

	// Create Go file with ServiceSpec
	goFile := filepath.Join(tmpDir, "payment_service.go")
	goContent := `package payment

// @ServiceSpec
// operationId: "processPayment"
// description: "Processes a payment transaction"
// preconditions:
//   "payment.amount": {">": [0, 0]}
//   "payment.method": {"in": [{"var": "payment.method"}, ["card", "bank", "wallet"]]}
// postconditions:
//   "response.transactionId": {"!=": [null, null]}
//   "response.status": {"==": ["success", "success"]}
func ProcessPayment(payment PaymentRequest) PaymentResponse {
	// Implementation
	return PaymentResponse{TransactionID: "txn123", Status: "success"}
}`

	err = os.WriteFile(goFile, []byte(goContent), 0644)
	require.NoError(t, err)

	// Create subdirectory with more files
	subDir := filepath.Join(tmpDir, "services")
	err = os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	// Create another Java file in subdirectory
	javaFile2 := filepath.Join(subDir, "NotificationService.java")
	javaContent2 := `package com.example.service.notification;

/**
 * @ServiceSpec
 * operationId: "sendNotification"
 * description: "Sends notification to user"
 * preconditions:
 *   "notification.recipient": {"!=": [null, null]}
 *   "notification.message": {"!=": [null, null]}
 * postconditions:
 *   "response.sent": {"==": [true, true]}
 */
public class NotificationService {
    public void sendNotification(Notification notification) {
        // Implementation
    }
}`

	err = os.WriteFile(javaFile2, []byte(javaContent2), 0644)
	require.NoError(t, err)

	// Create TypeScript file with multiple ServiceSpecs
	tsFile2 := filepath.Join(subDir, "inventoryService.ts")
	tsContent2 := `/**
 * @ServiceSpec
 * operationId: "checkStock"
 * description: "Checks stock availability for a product"
 * preconditions:
 *   "productId": {"!=": [null, null]}
 * postconditions:
 *   "response.available": {"!=": [null, null]}
 */
export function checkStock(productId: string): StockResponse {
    return { available: true, quantity: 10 };
}

/**
 * @ServiceSpec
 * operationId: "updateStock"
 * description: "Updates stock quantity for a product"
 * preconditions:
 *   "productId": {"!=": [null, null]}
 *   "quantity": {">=": [0, 0]}
 * postconditions:
 *   "response.updated": {"==": [true, true]}
 */
export function updateStock(productId: string, quantity: number): UpdateResponse {
    return { updated: true };
}`

	err = os.WriteFile(tsFile2, []byte(tsContent2), 0644)
	require.NoError(t, err)

	// Create some non-ServiceSpec files that should be ignored
	readmeFile := filepath.Join(tmpDir, "README.md")
	err = os.WriteFile(readmeFile, []byte("# Project README"), 0644)
	require.NoError(t, err)

	configFile := filepath.Join(tmpDir, "config.json")
	err = os.WriteFile(configFile, []byte(`{"setting": "value"}`), 0644)
	require.NoError(t, err)

	// Test parsing with default parser
	parser := NewSpecParser()

	startTime := time.Now()
	result, err := parser.ParseFromSource(tmpDir)
	duration := time.Since(startTime)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify we found all ServiceSpecs from all languages
	assert.Len(t, result.Specs, 6, "Should find 6 ServiceSpecs total")
	assert.Empty(t, result.Errors, "Should have no parsing errors")

	// Verify specs by operation ID
	specsByOpID := make(map[string]models.ServiceSpec)
	for _, spec := range result.Specs {
		specsByOpID[spec.OperationID] = spec
	}

	// Check Java specs
	createUserSpec, exists := specsByOpID["createUser"]
	assert.True(t, exists, "Should find createUser spec from Java")
	assert.Equal(t, "Creates a new user in the system", createUserSpec.Description)
	assert.Contains(t, createUserSpec.SourceFile, "UserService.java")

	sendNotificationSpec, exists := specsByOpID["sendNotification"]
	assert.True(t, exists, "Should find sendNotification spec from Java")
	assert.Contains(t, sendNotificationSpec.SourceFile, "NotificationService.java")

	// Check TypeScript specs
	processOrderSpec, exists := specsByOpID["processOrder"]
	assert.True(t, exists, "Should find processOrder spec from TypeScript")
	assert.Equal(t, "Processes an order and updates inventory", processOrderSpec.Description)
	assert.Contains(t, processOrderSpec.SourceFile, "orderService.ts")

	checkStockSpec, exists := specsByOpID["checkStock"]
	assert.True(t, exists, "Should find checkStock spec from TypeScript")
	assert.Contains(t, checkStockSpec.SourceFile, "inventoryService.ts")

	updateStockSpec, exists := specsByOpID["updateStock"]
	assert.True(t, exists, "Should find updateStock spec from TypeScript")
	assert.Contains(t, updateStockSpec.SourceFile, "inventoryService.ts")

	// Check Go spec
	processPaymentSpec, exists := specsByOpID["processPayment"]
	assert.True(t, exists, "Should find processPayment spec from Go")
	assert.Equal(t, "Processes a payment transaction", processPaymentSpec.Description)
	assert.Contains(t, processPaymentSpec.SourceFile, "payment_service.go")

	// Verify preconditions and postconditions are properly parsed
	assert.NotEmpty(t, createUserSpec.Preconditions, "Java spec should have preconditions")
	assert.NotEmpty(t, createUserSpec.Postconditions, "Java spec should have postconditions")
	assert.NotEmpty(t, processOrderSpec.Preconditions, "TypeScript spec should have preconditions")
	assert.NotEmpty(t, processOrderSpec.Postconditions, "TypeScript spec should have postconditions")
	assert.NotEmpty(t, processPaymentSpec.Preconditions, "Go spec should have preconditions")
	assert.NotEmpty(t, processPaymentSpec.Postconditions, "Go spec should have postconditions")

	// Verify performance metrics
	assert.NotNil(t, result.Metrics)
	assert.Equal(t, 5, result.Metrics["total_files"]) // 5 source files (inventoryService.ts has 2 specs)
	assert.Equal(t, 5, result.Metrics["processed_files"])
	assert.Equal(t, 6, result.Metrics["total_specs"]) // 6 total ServiceSpecs
	assert.Equal(t, 0, result.Metrics["total_errors"])

	// Performance should be reasonable
	assert.Less(t, duration, 5*time.Second, "Multi-language parsing should complete quickly")

	filesPerSecond := result.Metrics["files_per_second"].(float64)
	assert.Greater(t, filesPerSecond, 1.0, "Should process at least 1 file per second")

	t.Logf("Multi-language integration test results:")
	t.Logf("  - Total specs found: %d", len(result.Specs))
	t.Logf("  - Parse duration: %s", duration)
	t.Logf("  - Files per second: %.2f", filesPerSecond)
	t.Logf("  - Languages: Java, TypeScript, Go")
}

func TestMultiLanguageWithErrors(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid Java file
	validJavaFile := filepath.Join(tmpDir, "ValidService.java")
	validJavaContent := `/**
 * @ServiceSpec
 * operationId: "validOperation"
 * description: "Valid operation"
 */
public class ValidService {}`

	err := os.WriteFile(validJavaFile, []byte(validJavaContent), 0644)
	require.NoError(t, err)

	// Create invalid Java file (malformed YAML)
	invalidJavaFile := filepath.Join(tmpDir, "InvalidService.java")
	invalidJavaContent := `/**
 * @ServiceSpec
 * operationId: "invalidOperation"
 * description: "Invalid operation"
 * preconditions:
 *   invalid yaml: [unclosed bracket
 */
public class InvalidService {}`

	err = os.WriteFile(invalidJavaFile, []byte(invalidJavaContent), 0644)
	require.NoError(t, err)

	// Create valid TypeScript file
	validTsFile := filepath.Join(tmpDir, "validService.ts")
	validTsContent := `/**
 * @ServiceSpec
 * operationId: "validTsOperation"
 * description: "Valid TypeScript operation"
 */
export function validTsOperation() {}`

	err = os.WriteFile(validTsFile, []byte(validTsContent), 0644)
	require.NoError(t, err)

	// Create TypeScript file with missing required field
	invalidTsFile := filepath.Join(tmpDir, "invalidService.ts")
	invalidTsContent := `/**
 * @ServiceSpec
 * operationId: "invalidTsOperation"
 */
export function invalidTsOperation() {}`

	err = os.WriteFile(invalidTsFile, []byte(invalidTsContent), 0644)
	require.NoError(t, err)

	parser := NewSpecParser()
	result, err := parser.ParseFromSource(tmpDir)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Should find valid specs
	assert.Len(t, result.Specs, 2, "Should find 2 valid specs")

	// Should have errors for invalid specs (allow for potential duplicates)
	assert.GreaterOrEqual(t, len(result.Errors), 2, "Should have at least 2 parsing errors")
	assert.LessOrEqual(t, len(result.Errors), 5, "Should have at most 5 parsing errors")

	// Log errors for debugging
	if len(result.Errors) > 3 {
		t.Logf("Found %d parsing errors:", len(result.Errors))
		for i, err := range result.Errors {
			t.Logf("  [%d] %s", i+1, err.Error())
		}
	}

	// Verify valid specs
	specsByOpID := make(map[string]models.ServiceSpec)
	for _, spec := range result.Specs {
		specsByOpID[spec.OperationID] = spec
	}

	_, hasValidJava := specsByOpID["validOperation"]
	assert.True(t, hasValidJava, "Should find valid Java spec")

	_, hasValidTs := specsByOpID["validTsOperation"]
	assert.True(t, hasValidTs, "Should find valid TypeScript spec")

	// Verify errors contain expected information
	errorsByFile := make(map[string][]models.ParseError)
	for _, parseError := range result.Errors {
		filename := filepath.Base(parseError.File)
		errorsByFile[filename] = append(errorsByFile[filename], parseError)
	}

	assert.Contains(t, errorsByFile, "InvalidService.java", "Should have error for invalid Java file")
	assert.Contains(t, errorsByFile, "invalidService.ts", "Should have error for invalid TypeScript file")
}

func TestMultiLanguagePerformanceOptimization(t *testing.T) {
	tmpDir := t.TempDir()

	// Create many files to test parallel processing
	const numFiles = 50
	const specsPerFile = 2

	for i := 0; i < numFiles; i++ {
		// Create Java files
		if i%3 == 0 {
			javaFile := filepath.Join(tmpDir, fmt.Sprintf("Service%d.java", i))
			content := fmt.Sprintf(`/**
 * @ServiceSpec
 * operationId: "javaOp%d_1"
 * description: "Java operation %d part 1"
 */
/**
 * @ServiceSpec
 * operationId: "javaOp%d_2"
 * description: "Java operation %d part 2"
 */
public class Service%d {}`, i, i, i, i, i)

			err := os.WriteFile(javaFile, []byte(content), 0644)
			require.NoError(t, err)
		}

		// Create TypeScript files
		if i%3 == 1 {
			tsFile := filepath.Join(tmpDir, fmt.Sprintf("service%d.ts", i))
			content := fmt.Sprintf(`/**
 * @ServiceSpec
 * operationId: "tsOp%d_1"
 * description: "TypeScript operation %d part 1"
 */
export function tsOp%d_1() {}

/**
 * @ServiceSpec
 * operationId: "tsOp%d_2"
 * description: "TypeScript operation %d part 2"
 */
export function tsOp%d_2() {}`, i, i, i, i, i, i)

			err := os.WriteFile(tsFile, []byte(content), 0644)
			require.NoError(t, err)
		}

		// Create Go files
		if i%3 == 2 {
			goFile := filepath.Join(tmpDir, fmt.Sprintf("service%d.go", i))
			content := fmt.Sprintf(`package service%d

// @ServiceSpec
// operationId: "goOp%d_1"
// description: "Go operation %d part 1"
func GoOp%d_1() {}

// @ServiceSpec
// operationId: "goOp%d_2"
// description: "Go operation %d part 2"
func GoOp%d_2() {}`, i, i, i, i, i, i, i)

			err := os.WriteFile(goFile, []byte(content), 0644)
			require.NoError(t, err)
		}
	}

	// Test with different worker configurations
	testCases := []struct {
		name       string
		maxWorkers int
	}{
		{"Single Worker", 1},
		{"Default Workers", 4},
		{"Many Workers", 8},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := DefaultParserConfig()
			config.MaxWorkers = tc.maxWorkers
			parser := NewSpecParserWithConfig(config)

			startTime := time.Now()
			result, err := parser.ParseFromSource(tmpDir)
			duration := time.Since(startTime)

			require.NoError(t, err)
			assert.NotNil(t, result)

			// Should find all specs
			expectedSpecs := numFiles * specsPerFile
			assert.Len(t, result.Specs, expectedSpecs, "Should find all ServiceSpecs")
			assert.Empty(t, result.Errors, "Should have no errors")

			// Performance should improve with more workers (up to a point)
			filesPerSecond := result.Metrics["files_per_second"].(float64)
			assert.Greater(t, filesPerSecond, 5.0, "Should process at least 5 files per second")

			t.Logf("  Workers: %d, Duration: %s, Files/sec: %.2f",
				tc.maxWorkers, duration, filesPerSecond)
		})
	}
}

func TestMultiLanguageCacheEffectiveness(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	javaFile := filepath.Join(tmpDir, "CacheTest.java")
	javaContent := `/**
 * @ServiceSpec
 * operationId: "cachedOperation"
 * description: "Operation for cache testing"
 */
public class CacheTest {}`

	err := os.WriteFile(javaFile, []byte(javaContent), 0644)
	require.NoError(t, err)

	tsFile := filepath.Join(tmpDir, "cacheTest.ts")
	tsContent := `/**
 * @ServiceSpec
 * operationId: "cachedTsOperation"
 * description: "TypeScript operation for cache testing"
 */
export function cachedTsOperation() {}`

	err = os.WriteFile(tsFile, []byte(tsContent), 0644)
	require.NoError(t, err)

	config := DefaultParserConfig()
	config.EnableCache = true
	parser := NewSpecParserWithConfig(config)

	// First parse (cache miss)
	start1 := time.Now()
	result1, err := parser.ParseFromSource(tmpDir)
	duration1 := time.Since(start1)

	require.NoError(t, err)
	assert.Len(t, result1.Specs, 2)
	assert.Equal(t, 0, result1.Metrics["cache_hits"])
	assert.Equal(t, 2, result1.Metrics["cache_misses"])

	// Second parse (should use cache)
	start2 := time.Now()
	result2, err := parser.ParseFromSource(tmpDir)
	duration2 := time.Since(start2)

	require.NoError(t, err)
	assert.Len(t, result2.Specs, 2)
	assert.Equal(t, 2, result2.Metrics["cache_hits"])
	assert.Equal(t, 0, result2.Metrics["cache_misses"])

	// Cache should make second parse significantly faster
	assert.Less(t, duration2, duration1, "Cached parse should be faster")
	assert.Greater(t, parser.GetCacheSize(), 0, "Cache should contain entries")

	t.Logf("Cache effectiveness test:")
	t.Logf("  First parse (cache miss): %s", duration1)
	t.Logf("  Second parse (cache hit): %s", duration2)
	t.Logf("  Cache size: %d", parser.GetCacheSize())
	t.Logf("  Speed improvement: %.2fx", float64(duration1)/float64(duration2))
}

func TestMultiLanguageResourceLimits(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files of different sizes
	smallFile := filepath.Join(tmpDir, "Small.java")
	smallContent := `/**
 * @ServiceSpec
 * operationId: "smallOperation"
 * description: "Small operation"
 */
public class Small {}`

	err := os.WriteFile(smallFile, []byte(smallContent), 0644)
	require.NoError(t, err)

	// Create a large file that should be skipped (11MB)
	largeFile := filepath.Join(tmpDir, "Large.java")
	largeContent := make([]byte, 11*1024*1024) // 11MB of data

	// Fill with valid Java content
	header := `/**
 * @ServiceSpec
 * operationId: "largeOperation"
 * description: "Large operation"
 */
public class Large {
`
	copy(largeContent, header)

	// Fill the rest with comment characters
	for i := len(header); i < len(largeContent)-2; i++ {
		largeContent[i] = '/'
	}
	largeContent[len(largeContent)-2] = '}'
	largeContent[len(largeContent)-1] = '\n'

	err = os.WriteFile(largeFile, []byte(largeContent), 0644)
	require.NoError(t, err)

	// Verify the large file exceeds the limit
	info, err := os.Stat(largeFile)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(10*1024*1024), "Large file should exceed 10MB limit")

	parser := NewSpecParser()
	result, err := parser.ParseFromSource(tmpDir)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Should only find the small file's spec
	assert.Len(t, result.Specs, 1, "Should only process files within size limit")
	assert.Equal(t, "smallOperation", result.Specs[0].OperationID)
	assert.Empty(t, result.Errors, "Large files should be skipped, not cause errors")

	// Metrics should reflect only processed files
	assert.Equal(t, 1, result.Metrics["total_files"])
	assert.Equal(t, 1, result.Metrics["processed_files"])
}