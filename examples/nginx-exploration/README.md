# Nginx Log Exploration Example

This example demonstrates how to use FlowSpec CLI to explore Nginx access logs and automatically generate service contracts.

## Overview

This example shows:
1. How to analyze Nginx access logs to discover service patterns
2. How to generate YAML service contracts from traffic data
3. How to verify the generated contracts against trace data
4. How to handle different log formats and configurations

## Files

- `access.log` - Sample Nginx access log with various endpoints
- `access-compressed.log.gz` - Compressed log file example
- `custom-format.log` - Log with custom format
- `expected-contract.yaml` - Expected generated contract
- `test-trace.json` - Sample trace data for verification
- `run-example.sh` - Script to run the complete example

## Quick Start

```bash
# Run the complete example
./run-example.sh

# Or run individual steps:

# Step 1: Generate contract from access logs
flowspec-cli explore --traffic access.log --out generated-contract.yaml

# Step 2: Verify contract against trace data
flowspec-cli verify --path generated-contract.yaml --trace test-trace.json
```

## Sample Nginx Access Log

The `access.log` file contains entries like:

```
192.168.1.100 - - [01/Aug/2025:10:30:15 +0000] "GET /api/users/123 HTTP/1.1" 200 1234 "-" "Mozilla/5.0"
192.168.1.101 - - [01/Aug/2025:10:30:16 +0000] "GET /api/users/456 HTTP/1.1" 200 1567 "-" "Mozilla/5.0"
192.168.1.102 - - [01/Aug/2025:10:30:17 +0000] "POST /api/users HTTP/1.1" 201 890 "-" "curl/7.68.0"
192.168.1.103 - - [01/Aug/2025:10:30:18 +0000] "PUT /api/users/789 HTTP/1.1" 200 1100 "-" "PostmanRuntime/7.28.0"
192.168.1.104 - - [01/Aug/2025:10:30:19 +0000] "GET /api/users/999 HTTP/1.1" 404 234 "-" "Mozilla/5.0"
```

## Expected Generated Contract

The exploration should generate a contract similar to:

```yaml
apiVersion: flowspec/v1alpha1
kind: ServiceSpec
metadata:
  name: generated-service
  version: v1.0.0
spec:
  endpoints:
    - path: /api/users/{var}
      operations:
        - method: GET
          responses:
            statusRanges: ["2xx", "4xx"]
            aggregation: "range"
          stats:
            supportCount: 3
            firstSeen: "2025-08-01T10:30:15Z"
            lastSeen: "2025-08-01T10:30:19Z"
        - method: PUT
          responses:
            statusCodes: [200]
            aggregation: "exact"
          stats:
            supportCount: 1
            firstSeen: "2025-08-01T10:30:18Z"
            lastSeen: "2025-08-01T10:30:18Z"
    - path: /api/users
      operations:
        - method: POST
          responses:
            statusCodes: [201]
            aggregation: "exact"
          stats:
            supportCount: 1
            firstSeen: "2025-08-01T10:30:17Z"
            lastSeen: "2025-08-01T10:30:17Z"
```

## Advanced Usage

### Custom Log Format

If you have a custom Nginx log format:

```bash
# Use custom regex pattern
flowspec-cli explore --traffic custom-format.log --out contract.yaml \
  --regex '^(?P<ip>\S+) - - \[(?P<time>[^\]]+)\] "(?P<method>\S+) (?P<path>\S+) HTTP/[^"]*" (?P<status>\d+) (?P<size>\d+)'
```

### Time Filtering

Filter logs by time range:

```bash
# Only analyze logs from specific time period
flowspec-cli explore --traffic access.log --out contract.yaml \
  --since "2025-08-01T10:00:00Z" \
  --until "2025-08-01T11:00:00Z"
```

### Custom Thresholds

Adjust clustering and field detection thresholds:

```bash
# More aggressive path clustering and stricter required field detection
flowspec-cli explore --traffic access.log --out contract.yaml \
  --path-clustering-threshold 0.7 \
  --required-threshold 0.99 \
  --min-samples 3
```

### Service Metadata

Specify service information:

```bash
# Generate contract with custom service metadata
flowspec-cli explore --traffic access.log --out contract.yaml \
  --service-name "user-api" \
  --service-version "v2.1.0"
```

## Compressed Files

FlowSpec automatically handles compressed log files:

```bash
# Works with .gz files
flowspec-cli explore --traffic access-compressed.log.gz --out contract.yaml

# Works with .zst files (if available)
flowspec-cli explore --traffic access.log.zst --out contract.yaml
```

## Error Handling

If you encounter parsing errors:

1. **Check log format**: Ensure your logs match the expected format (combined/common)
2. **Use custom regex**: For non-standard formats, provide a custom regex pattern
3. **Review error samples**: FlowSpec shows up to 10 failed parsing examples
4. **Check incomplete contracts**: High error rates result in incomplete contract warnings

Example with high error rate:

```bash
flowspec-cli explore --traffic malformed.log --out contract.yaml
# Output:
# WARN High error rate detected (25.0%). Please check your log format configuration.
# WARN Sample failed lines:
#   [1] invalid log line format
#   [2] another malformed line
# WARN Contract marked as incomplete due to high error rate
```

## Integration with CI/CD

Use in GitHub Actions:

```yaml
name: Contract Generation
on: [push]

jobs:
  generate-contract:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Generate Contract from Logs
        run: |
          flowspec-cli explore --traffic ./logs/access.log --out ./contracts/api-contract.yaml
          
      - name: Verify Contract
        run: |
          flowspec-cli verify --path ./contracts/api-contract.yaml --trace ./traces/test.json --ci
          
      - name: Upload Contract
        uses: actions/upload-artifact@v3
        with:
          name: generated-contracts
          path: contracts/
```

## Troubleshooting

### Common Issues

1. **No endpoints generated**: Check if log format is correct and contains valid HTTP requests
2. **Over-parameterization**: Adjust `--path-clustering-threshold` to be more conservative (higher value)
3. **Under-parameterization**: Lower the `--path-clustering-threshold` for more aggressive clustering
4. **Missing required fields**: Lower `--required-threshold` to detect more optional fields

### Debug Mode

Enable debug logging for detailed information:

```bash
flowspec-cli explore --traffic access.log --out contract.yaml --log-level debug
```

This will show:
- Detailed parsing information
- Path clustering decisions
- Field analysis results
- Performance metrics