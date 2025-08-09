package main

/**
 * @ServiceSpec
 * operationId: "testOperation"
 * description: "Test operation for i18n"
 * preconditions:
 *   - {"==": [{"var": "span.status"}, "OK"]}
 * postconditions:
 *   - {"==": [{"var": "span.name"}, "test-operation"]}
 */
func TestFunction() {
    // Test function
}