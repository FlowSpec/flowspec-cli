# YAML Contract Examples

This directory contains examples of standalone YAML service contracts and how to use them with FlowSpec CLI.

## Overview

YAML contracts allow you to define service specifications independently of source code, providing:
- Version-controlled service contracts
- Language-agnostic specifications
- Easy integration with CI/CD pipelines
- Clear separation of concerns

## Files

- `user-service.yaml` - Complete user service contract example
- `order-service.yaml` - Order service contract with complex operations
- `minimal-service.yaml` - Minimal contract example
- `legacy-format.yaml` - Example of legacy format (still supported)
- `test-traces/` - Sample trace files for verification
- `verify-contracts.sh` - Script to verify all contracts

## Quick Start

```bash
# Verify a single contract
flowspec-cli verify --path user-service.yaml --trace test-traces/user-service-trace.json

# Verify all contracts
./verify-contracts.sh

# Use in CI mode
flowspec-cli verify --path user-service.yaml --trace test-traces/user-service-trace.json --ci
```

## Contract Structure

### Basic Structure

```yaml
apiVersion: flowspec/v1alpha1
kind: ServiceSpec
metadata:
  name: service-name
  version: v1.0.0
spec:
  endpoints:
    - path: /api/resource/{id}
      operations:
        - method: GET
          responses:
            statusCodes: [200, 404]
          required:
            headers: ["authorization"]
            query: []
          optional:
            headers: ["accept-language"]
            query: ["include"]
```

### Response Specifications

#### Exact Status Codes
```yaml
responses:
  statusCodes: [200, 400, 404, 500]
  aggregation: "exact"
```

#### Status Ranges
```yaml
responses:
  statusRanges: ["2xx", "4xx", "5xx"]
  aggregation: "range"
```

#### Auto Aggregation
```yaml
responses:
  statusCodes: [200, 201, 400, 404]
  aggregation: "auto"  # Will use ranges for 2xx, exact for others
```

### Field Requirements

#### Required Fields
Fields that must be present in requests:
```yaml
required:
  headers: ["authorization", "content-type"]
  query: ["user_id"]
```

#### Optional Fields
Fields that may be present:
```yaml
optional:
  headers: ["accept-language", "user-agent"]
  query: ["include", "format"]
```

### Statistics (Optional)
Track endpoint usage statistics:
```yaml
stats:
  supportCount: 150
  firstSeen: "2025-08-01T10:00:00Z"
  lastSeen: "2025-08-10T15:30:00Z"
```

## Examples

### User Service Contract

Complete example with authentication, CRUD operations, and query parameters:

```yaml
apiVersion: flowspec/v1alpha1
kind: ServiceSpec
metadata:
  name: user-service
  version: v2.1.0
spec:
  endpoints:
    - path: /api/users/{id}
      operations:
        - method: GET
          responses:
            statusRanges: ["2xx", "4xx"]
            aggregation: "range"
          required:
            headers: ["authorization"]
          optional:
            headers: ["accept-language"]
            query: ["include"]
        - method: PUT
          responses:
            statusCodes: [200, 400, 404]
            aggregation: "exact"
          required:
            headers: ["authorization", "content-type"]
        - method: DELETE
          responses:
            statusCodes: [204, 404]
            aggregation: "exact"
          required:
            headers: ["authorization"]
    - path: /api/users
      operations:
        - method: GET
          responses:
            statusCodes: [200]
            aggregation: "exact"
          optional:
            query: ["page", "limit", "sort"]
        - method: POST
          responses:
            statusCodes: [201, 400]
            aggregation: "exact"
          required:
            headers: ["content-type"]
```

### Order Service Contract

Example with complex business logic and multiple status codes:

```yaml
apiVersion: flowspec/v1alpha1
kind: ServiceSpec
metadata:
  name: order-service
  version: v1.5.0
spec:
  endpoints:
    - path: /api/orders/{id}
      operations:
        - method: GET
          responses:
            statusRanges: ["2xx", "4xx"]
          required:
            headers: ["authorization"]
          stats:
            supportCount: 1250
            firstSeen: "2025-07-15T09:00:00Z"
            lastSeen: "2025-08-10T16:45:00Z"
    - path: /api/orders/{id}/status
      operations:
        - method: PUT
          responses:
            statusCodes: [200, 400, 404, 409]
            aggregation: "exact"
          required:
            headers: ["authorization", "content-type"]
    - path: /api/orders
      operations:
        - method: GET
          responses:
            statusCodes: [200]
          required:
            headers: ["authorization"]
          optional:
            query: ["status", "customer_id", "date_from", "date_to"]
        - method: POST
          responses:
            statusCodes: [201, 400, 422]
            aggregation: "exact"
          required:
            headers: ["authorization", "content-type"]
```

## Migration from Legacy Format

### Legacy Format (Deprecated)
```yaml
spec:
  endpoints:
    - path: /api/users/{id}
      methods: ["GET", "PUT", "DELETE"]
      statusCodes: [200, 400, 404]
      requiredHeaders: ["authorization"]
```

### New Format (Recommended)
```yaml
spec:
  endpoints:
    - path: /api/users/{id}
      operations:
        - method: GET
          responses:
            statusCodes: [200, 404]
          required:
            headers: ["authorization"]
        - method: PUT
          responses:
            statusCodes: [200, 400, 404]
          required:
            headers: ["authorization", "content-type"]
        - method: DELETE
          responses:
            statusCodes: [204, 404]
          required:
            headers: ["authorization"]
```

## Best Practices

### 1. Use Semantic Versioning
```yaml
metadata:
  name: user-service
  version: v2.1.0  # Major.Minor.Patch
```

### 2. Group Related Operations
Keep operations for the same resource path together:
```yaml
- path: /api/users/{id}
  operations:
    - method: GET
      # ... GET operation spec
    - method: PUT
      # ... PUT operation spec
    - method: DELETE
      # ... DELETE operation spec
```

### 3. Use Appropriate Aggregation
- `exact`: When you need precise status code matching
- `range`: For flexible status code ranges (2xx, 4xx, 5xx)
- `auto`: Let FlowSpec decide based on the status codes

### 4. Document Required vs Optional Fields
Be explicit about what fields are required vs optional:
```yaml
required:
  headers: ["authorization"]  # Must be present
  query: []                   # No required query params
optional:
  headers: ["accept-language"] # May be present
  query: ["include", "format"] # Optional query params
```

### 5. Include Statistics for Generated Contracts
When generating from traffic logs, include statistics:
```yaml
stats:
  supportCount: 150           # Number of requests seen
  firstSeen: "2025-08-01T10:00:00Z"
  lastSeen: "2025-08-10T15:30:00Z"
```

## Validation

### Schema Validation
FlowSpec automatically validates YAML contracts against the schema:
- Required fields: `apiVersion`, `kind`, `metadata`, `spec`
- Valid `apiVersion`: `flowspec/v1alpha1`
- Valid `kind`: `ServiceSpec`

### Common Validation Errors

1. **Missing required fields**:
   ```
   Error: missing required field 'apiVersion'
   ```

2. **Invalid status aggregation**:
   ```
   Error: invalid aggregation 'invalid', must be one of: exact, range, auto
   ```

3. **Malformed path parameters**:
   ```
   Error: path parameter must be in format {paramName}
   ```

## Integration Examples

### GitHub Actions
```yaml
name: Contract Validation
on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Validate User Service Contract
        run: |
          flowspec-cli verify \
            --path contracts/user-service.yaml \
            --trace traces/user-service-integration.json \
            --ci
            
      - name: Validate Order Service Contract
        run: |
          flowspec-cli verify \
            --path contracts/order-service.yaml \
            --trace traces/order-service-integration.json \
            --ci
```

### NPM Scripts
```json
{
  "scripts": {
    "validate:contracts": "flowspec-cli verify --path contracts/ --trace traces/",
    "validate:user-service": "flowspec-cli verify --path contracts/user-service.yaml --trace traces/user-service.json",
    "validate:order-service": "flowspec-cli verify --path contracts/order-service.yaml --trace traces/order-service.json"
  }
}
```

### Makefile
```makefile
.PHONY: validate-contracts
validate-contracts:
	@echo "Validating service contracts..."
	@flowspec-cli verify --path contracts/user-service.yaml --trace traces/user-service.json --ci
	@flowspec-cli verify --path contracts/order-service.yaml --trace traces/order-service.json --ci
	@echo "All contracts validated successfully!"

.PHONY: validate-user-service
validate-user-service:
	flowspec-cli verify --path contracts/user-service.yaml --trace traces/user-service.json --output human
```

## Troubleshooting

### Contract Not Found
```bash
# Ensure the path is correct
flowspec-cli verify --path ./contracts/service.yaml --trace ./traces/test.json

# Check file permissions
ls -la contracts/service.yaml
```

### Schema Validation Errors
```bash
# Use debug mode for detailed error information
flowspec-cli verify --path service.yaml --trace test.json --debug
```

### Trace Format Issues
```bash
# Check trace file format
file traces/test.json

# Validate JSON syntax
jq . traces/test.json
```