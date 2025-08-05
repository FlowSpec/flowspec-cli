# FlowSpec CLI Example Projects

This directory contains example projects for using the FlowSpec CLI, demonstrating how to use ServiceSpec annotations in different languages and scenarios.

## Example List

This directory contains an example project for using the FlowSpec CLI.

### 1. [Simple User Service](simple-user-service/)
- **Language**: Java
- **Scenario**: Basic CRUD operations
- **Features**: Demonstrates basic preconditions and postconditions for success and failure scenarios.

## Quick Start

### Running the Example

```bash
# Navigate to the example directory
cd examples/simple-user-service

# Run FlowSpec validation against a success scenario
flowspec-cli align \
  --path=./src \
  --trace=./traces/success-scenario.json \
  --output=human

# Run validation against a precondition failure scenario
flowspec-cli align \
  --path=./src \
  --trace=./traces/precondition-failure.json \
  --output=human

# View the report in JSON format
flowspec-cli align \
  --path=./src \
  --trace=./traces/success-scenario.json \
  --output=json
```


### Generating Trace Data

Each example project includes scripts to generate trace data:

```bash
# Run the application and generate traces
./scripts/generate-traces.sh

# View the generated trace files
ls -la traces/
```

## Example Scenarios

### Success Scenarios
- All ServiceSpec assertions pass
- Demonstrates validation of normal business flows

### Failure Scenarios
- Precondition failures
- Postcondition failures
- Mixed scenarios (partially successful, partially failed)

### Edge Cases
- Missing trace data
- Malformed annotations
- Performance stress tests

## Learning Path

This example demonstrates:
- Basic format of ServiceSpec annotations in Java.
- Simple assertion expressions for preconditions and postconditions.
- How success and failure scenarios are reported.

## Best Practices

### Writing ServiceSpec Annotations
- Use meaningful `operationId`s.
- Write clear `description`s.
- Keep assertion expressions concise and clear.
- Consider edge cases and error handling.

### Generating Trace Data
- Ensure Span names match the `operationId`.
- Include sufficient attribute information for assertions.
- Record complete request and response data.
- Maintain the chronological order of trace data.

### Project Integration
- Integrate FlowSpec validation into your CI/CD pipeline.
- Regularly update trace data.
- Monitor validation result trends.
- Establish a process for handling validation failures.

## Troubleshooting

### Common Issues

1.  **ServiceSpec Not Found**
    -   Check file paths and extensions.
    -   Verify the annotation format is correct.

2.  **Trace Matching Failed**
    -   Ensure the `operationId` matches the Span name.
    -   Check the integrity of the trace data.

3.  **Assertion Evaluation Error**
    -   Validate the JSONLogic expression syntax.
    -   Check if the variable paths are correct.

### Debugging Tips

```bash
# Enable verbose output
flowspec-cli align --path=./src --trace=./trace.json --verbose

# Use debug log level
flowspec-cli align --path=./src --trace=./trace.json --log-level=debug

# Check parsing results
flowspec-cli align --path=./src --trace=./trace.json --output=json | jq .
```

## Contributing Examples

We welcome contributions of new example projects! Please follow these guidelines:

### Example Project Structure
```
example-name/
├── README.md           # Example description
├── src/               # Source code
├── traces/            # Trace data files
├── scripts/           # Helper scripts
└── expected-results/  # Expected validation results
```

### Submission Requirements
- Include a complete README.md.
- Provide trace data for various scenarios.
- Include expected validation results.
- Add necessary comments and documentation.

## Feedback and Suggestions

If you have any suggestions for the examples or find any issues, please:

1.  Report issues in [GitHub Issues](https://github.com/FlowSpec/flowspec-cli/issues).
2.  Discuss improvements in [GitHub Discussions](https://github.com/FlowSpec/flowspec-cli/discussions).
3.  Submit a Pull Request to contribute a new example.

---

**Tip**: These examples will be continuously updated with the development of the FlowSpec CLI. It is recommended to check the latest version regularly.
