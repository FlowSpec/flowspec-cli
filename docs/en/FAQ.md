# Frequently Asked Questions (FAQ)

This document answers common questions about using the FlowSpec CLI.

## Table of Contents

- [Installation and Configuration](#installation-and-configuration)
- [Usage Issues](#usage-issues)
- [ServiceSpec Annotations](#servicespec-annotations)
- [Trace Data](#trace-data)
- [Performance and Optimization](#performance-and-optimization)
- [Troubleshooting](#troubleshooting)

## Installation and Configuration

### Q: Which Go versions are supported?

A: The FlowSpec CLI requires Go 1.21 or higher. We recommend using the latest stable version.

```bash
# Check Go version
go version

# If the version is too low, please upgrade Go
```

### Q: How can I use the FlowSpec CLI in CI/CD?

A: You can integrate it into your CI/CD pipeline as follows:

```yaml
# GitHub Actions Example
- name: Install FlowSpec CLI
  run: go install github.com/FlowSpec/flowspec-cli/cmd/flowspec-cli@latest

- name: Run FlowSpec Validation
  run: |
    flowspec-cli align \
      --path=./src \
      --trace=./traces/integration-test.json \
      --output=json > flowspec-report.json

- name: Upload Report
  uses: actions/upload-artifact@v3
  with:
    name: flowspec-report
    path: flowspec-report.json
```

### Q: How do I configure the log level?

A: Use the `--log-level` parameter:

```bash
# Debug mode
flowspec-cli align --path=./src --trace=./trace.json --log-level=debug

# Silent mode
flowspec-cli align --path=./src --trace=./trace.json --log-level=error
```

## Usage Issues

### Q: Why are no ServiceSpecs found?

A: Possible reasons:

1.  **Incorrect file path**: Ensure `--path` points to the correct source code directory.
2.  **Incorrect annotation format**: Check the syntax of your ServiceSpec annotations.
3.  **Unsupported file type**: Ensure file extensions are `.java`, `.ts`, or `.go`.

```bash
# Use verbose output to see the parsing process
flowspec-cli align --path=./src --trace=./trace.json --verbose
```

### Q: How to handle large projects?

A: For large projects, we recommend:

1.  **Batch processing**: Divide the project into subdirectories and validate them separately.
2.  **Parallel processing**: The FlowSpec CLI automatically uses multiple cores for parallel processing.
3.  **Memory monitoring**: Use `--verbose` to check memory usage.

```bash
# Batch processing example
flowspec-cli align --path=./src/user-service --trace=./traces/user-service.json
flowspec-cli align --path=./src/order-service --trace=./traces/order-service.json
```

### Q: What do the exit codes mean?

A: The FlowSpec CLI uses the following exit codes:

- `0`: Validation successful, all assertions passed.
- `1`: Validation failed, some assertions did not pass.
- `2`: System error, such as file not found, parsing error, etc.

```bash
# Check the exit code in a script
flowspec-cli align --path=./src --trace=./trace.json
if [ $? -eq 0 ]; then
    echo "Validation successful"
elif [ $? -eq 1 ]; then
    echo "Validation failed"
else
    echo "System error"
fi
```

## ServiceSpec Annotations

### Q: What is the basic format of a ServiceSpec annotation?

A: The basic format is as follows:

```java
/**
 * @ServiceSpec
 * operationId: "uniqueOperationId"
 * description: "Operation description"
 * preconditions: {
 *   "condition_name": JSONLogic_Expression
 * }
 * postconditions: {
 *   "condition_name": JSONLogic_Expression
 * }
 */
```

### Q: How do I write complex assertion expressions?

A: Use JSONLogic syntax to write complex expressions:

```json
{
  "preconditions": {
    "request_validation": {
      "and": [
        {"!=": [{"var": "span.attributes.http.method"}, null]},
        {"in": [{"var": "span.attributes.http.method"}, ["POST", "PUT"]}},
        {=": [{"var": "span.attributes.request.body.password.length"}, 8]}
      ]
    }
  },
  "postconditions": {
    "response_validation": {
      "or": [
        {"==": [{"var": "span.attributes.http.status_code"}, 200]},
        {"==": [{"var": "span.attributes.http.status_code"}, 201]}
      ]
    }
  }
}
```

### Q: Which variables can be accessed in assertions?

A: You can access the following in assertion expressions:

**In preconditions**:
- `span.attributes.*`: Span attributes
- `span.startTime`: Start time
- `span.name`: Span name

**In postconditions**:
- All variables from preconditions
- `endTime`: End time
- `status.code`: Status code
- `status.message`: Status message
- `events[*].*`: Array of events

### Q: How to handle optional fields?

A: Use conditional logic to handle optional fields:

```json
{
  "postconditions": {
    "optional_field_check": {
      "if": [
        {"!=": [{"var": "span.attributes.response.body.optionalField"}, null]},
        {=": [{"var": "span.attributes.response.body.optionalField"}, 0]},
        true
      ]
    }
  }
}
```

## Trace Data

### Q: Which trace data formats are supported?

A: Currently, we support the OpenTelemetry JSON format.

### Q: How do I generate OpenTelemetry trace data?

A: You can use various OpenTelemetry SDKs:

```java
// Java Example
import io.opentelemetry.api.OpenTelemetry;
import io.opentelemetry.api.trace.Tracer;

Tracer tracer = OpenTelemetry.getGlobalTracer("my-service");
Span span = tracer.spanBuilder("createUser").startSpan();
// ... business logic
span.end();
```

### Q: What if the trace file is too large?

A: The FlowSpec CLI supports large file handling:

1.  **Stream processing**: Automatically uses stream parsing to avoid memory overflow.
2.  **Memory limit**: Default peak memory usage is limited to 500MB.
3.  **Sharding**: You can split large trace files into smaller ones.

```bash
# Monitor memory usage
flowspec-cli align --path=./src --trace=./large-trace.json --verbose
```

### Q: How are ServiceSpecs and Spans matched?

A: Matching is based on `operationId`:

1.  The `operationId` field in the ServiceSpec.
2.  The `name` or `operation.name` attribute in the Span.
3.  Fuzzy matching and regular expression matching are supported.

## Performance and Optimization

### Q: How to improve parsing performance?

A: Performance optimization suggestions:

1.  **Parallel processing**: The FlowSpec CLI automatically uses multiple cores.
2.  **File filtering**: Only scan directories containing ServiceSpecs.
3.  **Caching mechanism**: Enable parsing result caching.

```bash
# Use performance monitoring
flowspec-cli align --path=./src --trace=./trace.json --verbose
```

### Q: What if memory usage is too high?

A: Memory optimization strategies:

1.  **Batch processing**: Process large projects in smaller batches.
2.  **Adjust limits**: Adjust memory limits based on system configuration.
3.  **Clear cache**: Periodically clear the parsing cache.

### Q: How to perform performance benchmark tests?

A: Use the built-in performance tests:

```bash
# Run performance tests
make performance-test

# Run benchmark tests
make benchmark

# Run stress tests
make stress-test
```

## Troubleshooting

### Q: How to debug parsing errors?

A: Debugging steps:

1.  **Enable verbose output**: Use the `--verbose` parameter.
2.  **Check logs**: View detailed error messages.
3.  **Validate syntax**: Ensure the ServiceSpec annotation syntax is correct.

```bash
# Debug mode
flowspec-cli align --path=./src --trace=./trace.json --verbose --log-level=debug
```

### Q: What to do about JSONLogic expression errors?

A: Common issues and solutions:

1.  **Syntax error**: Check if the JSON format is correct.
2.  **Variable reference error**: Ensure the variable path is correct.
3.  **Type mismatch**: Check if the data types match.

```bash
# Use an online JSONLogic testing tool to validate expressions
# https://jsonlogic.com/play.html
```

### Q: What if trace data parsing fails?

A: Check the following:

1.  **File format**: Ensure it is a valid JSON format.
2.  **Data structure**: Check if it conforms to the OpenTelemetry specification.
3.  **File permissions**: Ensure you have read permissions for the file.

```bash
# Validate JSON format
cat trace.json | jq . > /dev/null && echo "JSON format is correct" || echo "JSON format is incorrect"
```

### Q: How to report a bug?

A: When reporting a bug, please provide:

1.  **Version information**: `flowspec-cli --version`
2.  **Command-line arguments**: The full command-line invocation.
3.  **Error message**: The complete error output.
4.  **Environment information**: OS, Go version, etc.
5.  **Steps to reproduce**: Detailed steps to reproduce the issue.

```bash
# Collect environment information
flowspec-cli --version
go version
uname -a
```

## More Help

If you can't find the answer to your question here, please:

1.  Check the [API Documentation](./API.md)
2.  Check the [Architecture Document](./ARCHITECTURE.md)
3.  Search in [GitHub Issues](https://github.com/FlowSpec/flowspec-cli/issues)
4.  Ask in [GitHub Discussions](https://github.com/FlowSpec/flowspec-cli/discussions)
5.  Email us at youming@flowspec.org

---

**Tip**: This FAQ is continuously updated. If you have new questions or suggestions, feel free to submit an Issue or Pull Request.
