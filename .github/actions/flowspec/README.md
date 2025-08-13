# FlowSpec GitHub Action

This GitHub Action provides automated verification of service specifications against trace data using FlowSpec CLI.

## Features

- 🚀 **Cross-platform support**: Linux, macOS, and Windows
- 📦 **Automatic CLI installation**: Downloads and installs the appropriate FlowSpec CLI version
- 🔒 **Security**: Verifies checksums for downloaded binaries
- 📊 **Rich reporting**: Generates human-readable and machine-readable reports
- 🎯 **CI/CD optimized**: Simplified output for CI environments
- 📁 **Artifact management**: Automatically uploads reports and logs

## Usage

### Basic Usage

```yaml
- name: Verify FlowSpec
  uses: ./.github/actions/flowspec
  with:
    path: './src'
    trace: './traces/integration-test.json'
```

### Advanced Usage

```yaml
- name: Verify FlowSpec
  uses: ./.github/actions/flowspec
  with:
    path: './service-spec.yaml'
    trace: './traces/e2e-test.json'
    version: 'v1.2.0'
    ci: 'true'
    status-aggregation: 'range'
    required-threshold: '0.90'
    min-samples: '10'
    output-format: 'json'
    lang: 'en'
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `path` | Path to source code directory or YAML contract file | ✅ | - |
| `trace` | Path to trace file (OpenTelemetry, HAR, or flowspec-trace.json) | ✅ | - |
| `version` | FlowSpec CLI version (e.g., "v1.0.0" or "latest") | ❌ | `latest` |
| `ci` | Enable CI mode for simplified output | ❌ | `true` |
| `status-aggregation` | Status code aggregation strategy (range\|exact\|auto) | ❌ | `auto` |
| `required-threshold` | Required field threshold (0.0-1.0) | ❌ | `0.95` |
| `min-samples` | Minimum samples required for endpoint inclusion | ❌ | `5` |
| `output-format` | Output format (human\|json) | ❌ | `human` |
| `lang` | Language for output (en\|zh\|ja\|ko\|fr\|de\|es) | ❌ | `en` |

## Outputs

| Output | Description |
|--------|-------------|
| `result` | Verification result (`success` or `failure`) |
| `exit-code` | Exit code from FlowSpec CLI |
| `summary-json` | Path to generated summary JSON file |
| `junit-xml` | Path to generated JUnit XML file |

## Exit Codes

The action respects FlowSpec CLI's exit code conventions:

- `0`: Success - All verifications passed
- `1`: Validation failed - Specifications do not match traces
- `2`: Contract format error - Invalid YAML or specification format
- `3`: Parse error - Unable to parse trace or log files
- `4`: System error - Runtime or environment issue
- `64`: Usage error - Invalid command line arguments

## Examples

### Complete Workflow Example

```yaml
name: FlowSpec Verification

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  verify-specs:
    runs-on: ubuntu-latest
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4
      
    - name: Verify service specifications
      id: flowspec
      uses: ./.github/actions/flowspec
      with:
        path: './specs/service-spec.yaml'
        trace: './test-data/integration-traces.json'
        version: 'latest'
        ci: 'true'
        
    - name: Comment PR with results
      if: github.event_name == 'pull_request' && failure()
      uses: actions/github-script@v7
      with:
        script: |
          const fs = require('fs');
          let comment = '## FlowSpec Verification Failed\\n\\n';
          
          if (fs.existsSync('artifacts/flowspec-summary.json')) {
            const summary = JSON.parse(fs.readFileSync('artifacts/flowspec-summary.json', 'utf8'));
            comment += `- **Checks**: ${summary.checks}\\n`;
            comment += `- **Passed**: ${summary.passed}\\n`;
            comment += `- **Failed**: ${summary.failed}\\n`;
            comment += `- **Duration**: ${summary.duration}\\n\\n`;
          }
          
          comment += 'Please check the [workflow logs](${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}) for details.';
          
          github.rest.issues.createComment({
            issue_number: context.issue.number,
            owner: context.repo.owner,
            repo: context.repo.repo,
            body: comment
          });
```

### Multi-Platform Testing

```yaml
name: Cross-Platform Verification

on: [push, pull_request]

jobs:
  verify:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
        
    runs-on: ${{ matrix.os }}
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Verify FlowSpec
      uses: ./.github/actions/flowspec
      with:
        path: './src'
        trace: './traces/test-run.json'
        version: 'latest'
```

### Using with Different Trace Formats

```yaml
jobs:
  verify-opentelemetry:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - name: Verify with OpenTelemetry traces
      uses: ./.github/actions/flowspec
      with:
        path: './specs'
        trace: './traces/otel-traces.json'
        
  verify-har:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - name: Verify with HAR files
      uses: ./.github/actions/flowspec
      with:
        path: './service-spec.yaml'
        trace: './traces/browser-session.har'
```

## Artifacts

The action automatically uploads the following artifacts:

- **Verification logs**: Complete output from FlowSpec CLI
- **Summary JSON**: Machine-readable summary (if generated)
- **JUnit XML**: Test results in JUnit format (if generated)

Artifacts are retained for 30 days and can be downloaded from the workflow run page.

## Troubleshooting

### Common Issues

1. **Binary not found after installation**
   - Check if the platform is supported
   - Verify network connectivity to GitHub releases
   - Check if the specified version exists

2. **Checksum verification failed**
   - This usually indicates a corrupted download
   - The action will retry automatically
   - Check network stability

3. **Permission denied errors**
   - On self-hosted runners, ensure the runner has appropriate permissions
   - For Linux/macOS, sudo access may be required

4. **Unsupported platform**
   - Currently supports Linux (amd64), macOS (amd64), and Windows (amd64)
   - ARM64 support may be added in future versions

### Debug Mode

To enable debug logging, add the following to your workflow:

```yaml
env:
  ACTIONS_STEP_DEBUG: true
```

### Manual Installation

If the automatic installation fails, you can manually install FlowSpec CLI:

```yaml
- name: Manual FlowSpec CLI installation
  run: |
    curl -fsSL https://raw.githubusercontent.com/FlowSpec/flowspec-cli/main/.github/actions/flowspec/install.sh | bash -s -- latest
```

## Security

- All downloads are verified using SHA256 checksums
- Binaries are downloaded from official GitHub releases only
- No external dependencies or third-party registries are used
- The action runs with minimal required permissions

## Contributing

To contribute to this action:

1. Fork the repository
2. Make your changes
3. Test with different platforms and scenarios
4. Submit a pull request

## License

This action is licensed under the Apache-2.0 License. See the [LICENSE](../../../LICENSE) file for details.