# FlowSpec Trace Format Guide

This guide explains the correct trace formats for FlowSpec CLI verification.

## Supported Trace Formats

FlowSpec CLI supports two main trace formats:

1. **OTLP JSON Format** (OpenTelemetry Protocol JSON)
2. **FlowSpec Trace JSON Format** (Custom format)

## OTLP JSON Format

This is the recommended format for OpenTelemetry traces. Here's the structure:

```json
{
  "resourceSpans": [{
    "resource": {
      "attributes": [
        {"key": "service.name", "value": {"stringValue": "your-service"}},
        {"key": "service.version", "value": {"stringValue": "1.0.0"}}
      ]
    },
    "scopeSpans": [{
      "scope": {
        "name": "your-tracer",
        "version": "1.0.0"
      },
      "spans": [
        {
          "traceId": "1234567890abcdef1234567890abcdef",
          "spanId": "abcdef1234567890",
          "name": "operationName",
          "kind": "SPAN_KIND_SERVER",
          "startTimeUnixNano": "1640995200000000000",
          "endTimeUnixNano": "1640995201000000000",
          "status": {
            "code": "STATUS_CODE_OK"
          },
          "attributes": [
            {"key": "http.method", "value": {"stringValue": "POST"}},
            {"key": "http.url", "value": {"stringValue": "/api/users"}},
            {"key": "http.status_code", "value": {"intValue": 201}},
            {"key": "request.body.email", "value": {"stringValue": "user@example.com"}},
            {"key": "response.body.userId", "value": {"stringValue": "12345"}}
          ]
        }
      ]
    }]
  }]
}
```

## Key Requirements

### 1. Span Names
- For **source code annotations**: Use operation names (e.g., "createUser", "getUser")
- For **YAML contracts**: Can use HTTP paths (e.g., "POST /api/users") or operation names

### 2. Required Attributes
- `http.method`: HTTP method (GET, POST, PUT, DELETE)
- `http.url`: Full URL path
- `http.status_code`: HTTP status code (as intValue)

### 3. Optional Attributes (for assertions)
- `request.body.*`: Request body fields
- `request.params.*`: URL parameters
- `request.headers.*`: Request headers
- `response.body.*`: Response body fields
- `response.headers.*`: Response headers

## Examples

### Working Examples
- `test-traces/user-service-trace.json` - Complete OTLP format with all required attributes
- `../simple-user-service/traces/success-scenario.json` - Reference implementation

### Common Issues Fixed

1. **Wrong Format**: Old Jaeger-style format → OTLP JSON format
2. **Missing Attributes**: Added request/response body attributes for assertions
3. **Incorrect Span Names**: HTTP paths → Operation names for source code matching

## Verification Commands

```bash
# Verify with YAML contract (path-based matching)
flowspec-cli verify --path user-service.yaml --trace test-traces/user-service-trace.json

# Verify with source code (span name-based matching)
flowspec-cli verify --path ../simple-user-service/src --trace test-traces/user-service-trace.json
```

## Troubleshooting

### "Unsupported trace format" Error
- Ensure you're using OTLP JSON format
- Check that the JSON structure matches the examples above

### "No matching spans found"
- For source code: Ensure span names match operation names in annotations
- For YAML contracts: Ensure HTTP paths and methods match
- Verify that required attributes are present

### "Assertion failed" Errors
- Check that span attributes match the expected variable names
- Ensure attribute values have correct types (stringValue vs intValue)
- Verify that all required request/response attributes are included