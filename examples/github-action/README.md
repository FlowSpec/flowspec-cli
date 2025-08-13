# GitHub Action Integration Example

This directory contains examples of how to integrate FlowSpec CLI with GitHub Actions for automated contract validation in CI/CD pipelines.

## Overview

FlowSpec provides a GitHub Action that makes it easy to validate service contracts as part of your CI/CD workflow. This ensures that your services continue to meet their specifications with every code change.

## Files

- `.github/workflows/flowspec-validation.yml` - Complete workflow example
- `.github/workflows/contract-generation.yml` - Traffic exploration workflow
- `.github/workflows/multi-service.yml` - Multi-service validation
- `contracts/` - Sample contract files
- `traces/` - Sample trace files for testing

## Quick Start

### Basic Validation Workflow

Create `.github/workflows/flowspec-validation.yml`:

```yaml
name: FlowSpec Contract Validation
on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v3
        
      - name: FlowSpec Verification
        uses: flowspec/flowspec-action@v1
        with:
          path: ./contracts/service-spec.yaml
          trace: ./traces/integration-test.json
          ci: true
          
      - name: Upload FlowSpec Reports
        uses: actions/upload-artifact@v3
        if: always()
        with:
          name: flowspec-reports
          path: artifacts/
```

### Advanced Multi-Service Workflow

```yaml
name: Multi-Service Contract Validation
on: [push, pull_request]

jobs:
  validate-contracts:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        service:
          - name: user-service
            contract: contracts/user-service.yaml
            trace: traces/user-service-integration.json
          - name: order-service
            contract: contracts/order-service.yaml
            trace: traces/order-service-integration.json
          - name: payment-service
            contract: contracts/payment-service.yaml
            trace: traces/payment-service-integration.json
    
    steps:
      - name: Checkout code
        uses: actions/checkout@v3
        
      - name: Validate ${{ matrix.service.name }}
        uses: flowspec/flowspec-action@v1
        with:
          path: ${{ matrix.service.contract }}
          trace: ${{ matrix.service.trace }}
          ci: true
          version: latest
          
      - name: Upload ${{ matrix.service.name }} Reports
        uses: actions/upload-artifact@v3
        if: always()
        with:
          name: flowspec-reports-${{ matrix.service.name }}
          path: artifacts/
```

## Action Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `path` | Path to source code or YAML contract file | Yes | - |
| `trace` | Path to trace file | Yes | - |
| `version` | FlowSpec CLI version to use | No | `latest` |
| `ci` | Enable CI mode for concise output | No | `true` |
| `status-aggregation` | Status code aggregation strategy | No | `auto` |
| `required-threshold` | Required field threshold (0.0-1.0) | No | `0.95` |

## Action Outputs

| Output | Description |
|--------|-------------|
| `exit-code` | Exit code from FlowSpec CLI (0=success, 1=validation failed, etc.) |
| `report-path` | Path to generated JSON report |
| `artifacts-path` | Path to artifacts directory |

## Examples

### 1. Basic Contract Validation

```yaml
name: Contract Validation
on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Validate Service Contract
        uses: flowspec/flowspec-action@v1
        with:
          path: ./service-spec.yaml
          trace: ./traces/integration-test.json
```

### 2. Source Code Annotation Validation

```yaml
name: ServiceSpec Validation
on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Validate ServiceSpec Annotations
        uses: flowspec/flowspec-action@v1
        with:
          path: ./src
          trace: ./traces/e2e-test.json
          ci: true
```

### 3. Contract Generation from Traffic

```yaml
name: Contract Generation
on:
  schedule:
    - cron: '0 2 * * *'  # Daily at 2 AM
  workflow_dispatch:

jobs:
  generate-contracts:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Download Production Logs
        run: |
          # Download logs from your logging system
          aws s3 cp s3://my-logs/nginx-access.log ./logs/
          
      - name: Generate Contracts
        run: |
          flowspec-cli explore --traffic ./logs/nginx-access.log --out ./contracts/generated-contract.yaml
          
      - name: Validate Generated Contract
        uses: flowspec/flowspec-action@v1
        with:
          path: ./contracts/generated-contract.yaml
          trace: ./traces/validation-trace.json
          
      - name: Create Pull Request
        uses: peter-evans/create-pull-request@v5
        with:
          title: 'Update generated service contracts'
          body: 'Automated contract update based on production traffic analysis'
          branch: update-contracts
```

### 4. Multi-Environment Validation

```yaml
name: Multi-Environment Validation
on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        environment: [dev, staging, prod]
        
    steps:
      - uses: actions/checkout@v3
      
      - name: Download ${{ matrix.environment }} traces
        run: |
          # Download environment-specific traces
          curl -o traces/${{ matrix.environment }}-trace.json \
            "https://api.example.com/traces/${{ matrix.environment }}/latest"
            
      - name: Validate against ${{ matrix.environment }}
        uses: flowspec/flowspec-action@v1
        with:
          path: ./contracts/service-spec.yaml
          trace: ./traces/${{ matrix.environment }}-trace.json
          ci: true
          
      - name: Upload ${{ matrix.environment }} Report
        uses: actions/upload-artifact@v3
        if: always()
        with:
          name: flowspec-report-${{ matrix.environment }}
          path: artifacts/
```

### 5. Conditional Validation

```yaml
name: Conditional Contract Validation
on: [push, pull_request]

jobs:
  check-changes:
    runs-on: ubuntu-latest
    outputs:
      contracts-changed: ${{ steps.changes.outputs.contracts }}
      traces-changed: ${{ steps.changes.outputs.traces }}
    steps:
      - uses: actions/checkout@v3
      - uses: dorny/paths-filter@v2
        id: changes
        with:
          filters: |
            contracts:
              - 'contracts/**'
            traces:
              - 'traces/**'
              
  validate:
    needs: check-changes
    if: needs.check-changes.outputs.contracts-changed == 'true' || needs.check-changes.outputs.traces-changed == 'true'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Validate Contracts
        uses: flowspec/flowspec-action@v1
        with:
          path: ./contracts/
          trace: ./traces/integration-test.json
          ci: true
```

### 6. Performance Monitoring

```yaml
name: Contract Validation with Performance Monitoring
on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Validate Contracts
        uses: flowspec/flowspec-action@v1
        id: validation
        with:
          path: ./contracts/service-spec.yaml
          trace: ./traces/performance-test.json
          ci: true
          
      - name: Parse Performance Metrics
        run: |
          # Extract performance metrics from JSON report
          jq '.performanceInfo' artifacts/flowspec-summary.json > performance-metrics.json
          
      - name: Comment Performance Results
        uses: actions/github-script@v6
        if: github.event_name == 'pull_request'
        with:
          script: |
            const fs = require('fs');
            const metrics = JSON.parse(fs.readFileSync('performance-metrics.json', 'utf8'));
            
            const comment = `## FlowSpec Performance Report
            
            - **Memory Usage**: ${metrics.memoryUsageMB} MB
            - **Execution Time**: ${metrics.executionTimeMs} ms
            - **Validation Speed**: ${metrics.validationsPerSecond} validations/sec
            
            ${metrics.memoryUsageMB > 100 ? '⚠️ High memory usage detected' : '✅ Memory usage within normal range'}
            `;
            
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: comment
            });
```

## Error Handling

### Exit Codes

The FlowSpec Action respects the CLI exit codes:

- `0`: Success - all validations passed
- `1`: Validation failed - service behavior doesn't match specifications
- `2`: Contract format error - invalid YAML or specification format
- `3`: Parse error - unable to parse input files
- `4`: System error - technical issues during execution
- `64`: Usage error - invalid command line arguments

### Handling Failures

```yaml
- name: Validate Contracts
  uses: flowspec/flowspec-action@v1
  id: validation
  continue-on-error: true
  with:
    path: ./contracts/service-spec.yaml
    trace: ./traces/test.json
    
- name: Handle Validation Failure
  if: steps.validation.outcome == 'failure'
  run: |
    echo "Contract validation failed with exit code: ${{ steps.validation.outputs.exit-code }}"
    
    case "${{ steps.validation.outputs.exit-code }}" in
      1)
        echo "::warning::Service behavior doesn't match contract specifications"
        ;;
      2)
        echo "::error::Contract format error - please check YAML syntax"
        exit 1
        ;;
      3)
        echo "::error::Parse error - please check trace file format"
        exit 1
        ;;
      *)
        echo "::error::Unexpected error occurred"
        exit 1
        ;;
    esac
```

## Artifacts

The action automatically generates artifacts in the `artifacts/` directory:

- `flowspec-summary.json` - JSON summary report
- `flowspec-report.xml` - JUnit XML report for test result integration

### Using Artifacts

```yaml
- name: Upload FlowSpec Reports
  uses: actions/upload-artifact@v3
  if: always()
  with:
    name: flowspec-reports
    path: artifacts/
    
- name: Publish Test Results
  uses: dorny/test-reporter@v1
  if: always()
  with:
    name: FlowSpec Contract Tests
    path: artifacts/flowspec-report.xml
    reporter: java-junit
```

## Security Considerations

### Secrets Management

If your traces contain sensitive data, use GitHub Secrets:

```yaml
- name: Download Secure Traces
  env:
    TRACE_API_KEY: ${{ secrets.TRACE_API_KEY }}
  run: |
    curl -H "Authorization: Bearer $TRACE_API_KEY" \
      -o traces/secure-trace.json \
      "https://api.example.com/traces/latest"
```

### Private Repositories

For private repositories, ensure the action has appropriate permissions:

```yaml
permissions:
  contents: read
  actions: read
  checks: write  # For publishing test results
```

## Troubleshooting

### Common Issues

1. **Action not found**: Ensure you're using the correct action name and version
2. **Permission denied**: Check repository permissions and secrets
3. **File not found**: Verify paths are relative to repository root
4. **Large trace files**: Consider using compressed traces or sampling

### Debug Mode

Enable debug logging:

```yaml
- name: Validate with Debug
  uses: flowspec/flowspec-action@v1
  with:
    path: ./contracts/service-spec.yaml
    trace: ./traces/test.json
  env:
    ACTIONS_STEP_DEBUG: true
```

### Manual CLI Usage

For debugging, you can also run the CLI directly:

```yaml
- name: Setup Node.js
  uses: actions/setup-node@v3
  with:
    node-version: '18'
    
- name: Install FlowSpec CLI
  run: npm install -g @flowspec/cli
  
- name: Manual Validation
  run: |
    flowspec-cli verify --path ./contracts/service-spec.yaml --trace ./traces/test.json --debug
```

## Best Practices

1. **Use matrix builds** for multiple services
2. **Cache dependencies** when possible
3. **Upload artifacts** for debugging failed runs
4. **Use conditional execution** to avoid unnecessary runs
5. **Monitor performance** metrics over time
6. **Secure sensitive data** with GitHub Secrets
7. **Use appropriate timeouts** for large trace files

## Integration with Other Tools

### Slack Notifications

```yaml
- name: Notify Slack on Failure
  if: failure()
  uses: 8398a7/action-slack@v3
  with:
    status: failure
    text: 'FlowSpec contract validation failed'
  env:
    SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK }}
```

### Teams Integration

```yaml
- name: Notify Teams
  if: always()
  uses: skitionek/notify-microsoft-teams@master
  with:
    webhook_url: ${{ secrets.TEAMS_WEBHOOK }}
    title: FlowSpec Validation Results
    summary: Contract validation completed with ${{ job.status }}
```