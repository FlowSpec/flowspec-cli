# Simple User Service Example

This is a basic user management service example that demonstrates how to use the FlowSpec CLI to validate simple CRUD operations.

## Project Overview

This example implements a simple user management API with the following operations:
- Create User (`createUser`)
- Get User (`getUser`)
- Update User (`updateUser`)
- Delete User (`deleteUser`)

## File Structure

```
simple-user-service/
├── README.md
├── src/
│   └── UserService.java
├── traces/
│   ├── success-scenario.json
│   ├── precondition-failure.json
│   └── postcondition-failure.json
└── expected-results/
    └── success-report.json
```

## ServiceSpec Annotation Examples

### Create User

```java
/**
 * @ServiceSpec
 * operationId: "createUser"
 * description: "Create a new user account"
 * preconditions: {
 *   "email_required": {"!=": [{"var": "span.attributes.request.body.email"}, null]},
 *   "email_format": {"match": [{"var": "span.attributes.request.body.email"}, "^[\\w\\.-]+@[\\w\\.-]+\\.[a-zA-Z]{2,}$"]},
 *   "password_length": {">=": [{"var": "span.attributes.request.body.password.length"}, 8]}
 * }
 * postconditions: {
 *   "success_status": {"==": [{"var": "span.attributes.http.status_code"}, 201]},
 *   "user_id_generated": {"!=": [{"var": "span.attributes.response.body.userId"}, null]},
 *   "email_returned": {"==": [{"var": "span.attributes.response.body.email"}, {"var": "span.attributes.request.body.email"}]}
 * }
 */
public User createUser(CreateUserRequest request) {
    // Implementation logic
}
```

### Get User

```java
/**
 * @ServiceSpec
 * operationId: "getUser"
 * description: "Get user information by user ID"
 * preconditions: {
 *   "user_id_required": {"!=": [{"var": "span.attributes.request.params.userId"}, null]},
 *   "user_id_format": {"match": [{"var": "span.attributes.request.params.userId"}, "^[0-9]+$"]}
 * }
 * postconditions: {
 *   "success_or_not_found": {"in": [{"var": "span.attributes.http.status_code"}, [200, 404]]},
 *   "user_data_if_found": {
 *     "if": [
 *       {"==": [{"var": "span.attributes.http.status_code"}, 200]},
 *       {"and": [
 *         {"!=": [{"var": "span.attributes.response.body.userId"}, null]},
 *         {"!=": [{"var": "span.attributes.response.body.email"}, null]}
 *       ]},
 *       true
 *     ]
 *   }
 * }
 */
public User getUser(Long userId) {
    // Implementation logic
}
```

## Running the Example

### 1. Success Scenario Validation

```bash
# Run success scenario validation
flowspec-cli align \
  --path=./src \
  --trace=./traces/success-scenario.json \
  --output=human

# Expected output:
# ✅ All ServiceSpec validations passed
# 📊 Summary: 4 total, 4 success, 0 failed, 0 skipped
```

### 2. Precondition Failure Scenario

```bash
# Run precondition failure scenario
flowspec-cli align \
  --path=./src \
  --trace=./traces/precondition-failure.json \
  --output=human

# Expected output:
# ❌ createUser validation failed
# Precondition 'password_length' failed: Password length is less than 8
```

### 3. Postcondition Failure Scenario

```bash
# Run postcondition failure scenario
flowspec-cli align \
  --path=./src \
  --trace=./traces/postcondition-failure.json \
  --output=human

# Expected output:
# ❌ createUser validation failed
# Postcondition 'success_status' failed: Expected status code 201, but got 500
```

## Trace Data Description

### Success Scenario Trace

`traces/success-scenario.json` contains trace data for the successful execution of all operations:

```json
{
  "resourceSpans": [{
    "scopeSpans": [{
      "spans": [{
        "name": "createUser",
        "spanId": "abc123",
        "traceId": "trace001",
        "startTimeUnixNano": "1640995200000000000",
        "endTimeUnixNano": "1640995201000000000",
        "status": {"code": "STATUS_CODE_OK"},
        "attributes": [
          {"key": "http.method", "value": {"stringValue": "POST"}},
          {"key": "http.status_code", "value": {"intValue": 201}},
          {"key": "request.body.email", "value": {"stringValue": "user@example.com"}},
          {"key": "request.body.password.length", "value": {"intValue": 12}},
          {"key": "response.body.userId", "value": {"stringValue": "12345"}},
          {"key": "response.body.email", "value": {"stringValue": "user@example.com"}}
        ]
      }]
    }]
  }]
}
```

### Failure Scenario Traces

The trace data for failure scenarios intentionally contains data that does not satisfy the ServiceSpec assertions, used for testing the validation logic.

## Learning Points

### 1. Writing Assertion Expressions

- **Simple Comparison**: `{"==": [value1, value2]}`
- **Null Check**: `{"!=": [value, null]}`
- **Regex Match**: `{"match": [string, pattern]}`
- **Conditional Logic**: `{"if": [condition, then_value, else_value]}`

### 2. Variable Paths

- **Request Data**: `span.attributes.request.body.*`
- **Response Data**: `span.attributes.response.body.*`
- **HTTP Info**: `span.attributes.http.*`
- **Time Info**: `span.startTime`, `endTime`

### 3. Best Practices

- Use meaningful assertion names.
- Write clear error messages.
- Consider edge cases and exception handling.
- Keep assertion expressions concise.

## Extension Exercises

### 1. Add New ServiceSpecs

Try adding ServiceSpec annotations for the following operations:
- Bulk user creation
- User password reset
- User status update

### 2. Complex Assertion Expressions

Practice writing more complex assertions:
- Multi-condition combination validation
- Array data validation
- Time range validation

### 3. Error Scenario Testing

Create more error scenario traces:
- Network timeout
- Database connection failure
- Permission validation failure

## Troubleshooting

### Common Issues

1.  **ServiceSpec Not Found**
    -   Check the annotation format in the Java file.
    -   Ensure the annotation is above the method.

2.  **Trace Matching Failed**
    -   Verify that the Span name matches the `operationId`.
    -   Check if the trace file format is correct.

3.  **Assertion Evaluation Error**
    -   Use an online JSONLogic tool to validate the expression.
    -   Check if the variable paths are correct.

### Debugging Commands

```bash
# Verbose output mode
flowspec-cli align --path=./src --trace=./traces/success-scenario.json --verbose

# JSON output for analysis
flowspec-cli align --path=./src --trace=./traces/success-scenario.json --output=json | jq .

# Debug log level
flowspec-cli align --path=./src --trace=./traces/success-scenario.json --log-level=debug
```

---

This example provides you with the basic usage of the FlowSpec CLI.
