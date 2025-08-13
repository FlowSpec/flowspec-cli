# FlowSpec CLI

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)](#)
[![Coverage](https://img.shields.io/badge/Coverage-80%25+-brightgreen.svg)](#)

FlowSpec CLI is a powerful command-line tool for parsing ServiceSpec annotations from source code, ingesting OpenTelemetry traces, and performing alignment validation between specifications and actual execution traces. It helps developers discover service integration issues early in the development cycle, ensuring the reliability of microservice architectures.

## Project Status

🚧 **In Development** - This is the implementation of FlowSpec Phase 1 MVP and is currently under active development.

## Core Value

- 🔍 **Early Problem Detection**: Discover service integration issues during the development phase.
- 📝 **Code as Documentation**: ServiceSpec annotations are embedded directly in the source code, keeping them in sync.
- 🌐 **Multi-Language Support**: Supports mainstream languages like Java, TypeScript, and Go.
- 🚀 **CI/CD Integration**: Easily integrates into continuous integration workflows.
- 📊 **Detailed Reports**: Provides human-readable and machine-readable validation reports.

## Features

- 🌍 **Internationalization**: Full multi-language support (English, Chinese, Japanese, Korean, French, German, Spanish)
- 📝 **Multi-Language Parsing**: Parse ServiceSpec annotations from Java, TypeScript, and Go source code
- 📊 **OpenTelemetry Integration**: Ingest and process OpenTelemetry trace data
- ✅ **Alignment Validation**: Perform validation between specifications and actual traces
- 🔍 **Traffic Exploration**: Automatically discover service patterns from traffic logs and generate YAML contracts
- 📄 **YAML Contract Support**: Use standalone YAML files as service contracts independent of source code
- 📋 **Localized Reports**: Generate detailed validation reports in multiple languages (Human and JSON formats)
- 🔧 **CLI Integration**: Command-line interface with language selection for easy CI/CD integration
- 🤖 **Auto Language Detection**: Automatically detects preferred language from environment variables
- 🚀 **High Performance**: Optimized for speed with concurrent processing and memory efficiency
- 🎯 **CI/CD Optimized**: Enhanced CI mode with concise output and GitHub Action support

## Quick Start

### Installation

#### Using NPM (Recommended for Node.js projects)

Install as a development dependency in your Node.js project:

```bash
npm install @flowspec/cli --save-dev
```

Or install globally:

```bash
npm install -g @flowspec/cli
```

You can also use it directly with npx without installation:

```bash
npx @flowspec/cli --help
```

#### Using go install

```bash
go install github.com/FlowSpec/flowspec-cli/cmd/flowspec-cli@latest
```

#### Build from Source

```bash
# Clone the repository
git clone https://github.com/FlowSpec/flowspec-cli.git
cd flowspec-cli

# Install dependencies
make deps

# Build
make build

# Install to GOPATH
make install
```

#### Download Pre-compiled Binaries

Visit the [Releases](https://github.com/FlowSpec/flowspec-cli/releases) page to download pre-compiled binaries for your platform.

### Verify Installation

```bash
flowspec-cli --version
flowspec-cli --help
```

## Usage

### Basic Usage

#### Contract Validation

```bash
# Perform alignment validation with source code (traditional approach)
flowspec-cli align --path=./my-project --trace=./traces/run-1.json --output=human

# Verify using YAML contract file (recommended)
flowspec-cli verify --path=./service-spec.yaml --trace=./traces/run-1.json

# Verify using directory (prefers YAML, falls back to source code)
flowspec-cli verify --path=./my-project --trace=./traces/run-1.json --output=json

# Specify language for output
flowspec-cli verify --path=./my-project --trace=./traces/run-1.json --lang=zh

# CI mode with concise output
flowspec-cli verify --path=./my-project --trace=./traces/run-1.json --ci
```

#### Traffic Exploration and Contract Generation

```bash
# Basic traffic exploration with Nginx access logs
flowspec-cli explore --traffic ./logs/access.log --out ./service-spec.yaml

# Explore directory of log files
flowspec-cli explore --traffic ./logs/ --out ./contract.yaml --log-format combined

# With time filtering and custom thresholds
flowspec-cli explore --traffic ./logs/ --out ./contract.yaml \
  --since "2025-08-01T00:00:00Z" \
  --until "2025-08-10T23:59:59Z" \
  --required-threshold 0.9 \
  --min-samples 10

# Custom service metadata
flowspec-cli explore --traffic ./logs/ --out ./contract.yaml \
  --service-name "user-service" \
  --service-version "v2.1.0"
```

### Language Support

FlowSpec CLI supports multiple languages for output and reports:

```bash
# English (default)
flowspec-cli verify --path=./src --trace=./trace.json --lang=en

# Chinese Simplified
flowspec-cli verify --path=./src --trace=./trace.json --lang=zh

# Chinese Traditional
flowspec-cli verify --path=./src --trace=./trace.json --lang=zh-TW

# Japanese
flowspec-cli verify --path=./src --trace=./trace.json --lang=ja

# Korean
flowspec-cli verify --path=./src --trace=./trace.json --lang=ko

# French
flowspec-cli verify --path=./src --trace=./trace.json --lang=fr

# German
flowspec-cli verify --path=./src --trace=./trace.json --lang=de

# Spanish
flowspec-cli verify --path=./src --trace=./trace.json --lang=es
```

**Auto-detection**: If no language is specified, FlowSpec CLI will automatically detect your preferred language from environment variables (`FLOWSPEC_LANG` or `LANG`).

**Language Priority**: Command line `--lang` flag > `FLOWSPEC_LANG` environment variable > `LANG` environment variable > English (default)

### Command Options

#### align / verify Commands

- `--path, -p`: Source code directory path or YAML contract file (default: ".")
- `--trace, -t`: OpenTelemetry trace file path (required)
- `--output, -o`: Output format (human|json, default: "human")
- `--lang`: Language for output (en, zh, zh-TW, ja, ko, fr, de, es). Auto-detected if not specified
- `--ci`: Enable CI mode with concise output
- `--strict`: Enable strict validation mode
- `--debug`: Enable debug mode with detailed logging
- `--timeout`: Timeout for single ServiceSpec alignment (default: 30s)
- `--max-workers`: Maximum number of concurrent workers (default: 4)
- `--verbose, -v`: Enable verbose output
- `--log-level`: Set log level (debug, info, warn, error)

#### explore Command

- `--traffic`: Path to traffic log files or directory (required)
- `--out`: Output path for generated YAML contract (required)
- `--log-format`: Log format (combined, common, or custom, default: "combined")
- `--regex`: Custom regex pattern for log parsing
- `--since`: Start time filter (RFC3339 format)
- `--until`: End time filter (RFC3339 format)
- `--sample-rate`: Sampling rate (0.0-1.0, default: 1.0)
- `--status-aggregation`: Status code aggregation strategy (range, exact, auto, default: "auto")
- `--required-threshold`: Required field threshold (0.0-1.0, default: 0.95)
- `--min-samples`: Minimum samples required per endpoint (default: 5)
- `--path-clustering-threshold`: Path clustering threshold (0.0-1.0, default: 0.8)
- `--min-sample-size`: Minimum sample size for parameterization (default: 20)
- `--max-unique-values`: Maximum unique values to track per segment (default: 10000)
- `--service-name`: Service name for the contract (default: "generated-service")
- `--service-version`: Service version for the contract (default: "v1.0.0")

### Language Configuration

#### Manual Language Selection

```bash
# Set language via command line
flowspec-cli verify --path ./src --trace ./trace.json --lang zh
```

#### Environment Variables

```bash
# Set preferred language via environment variable
export FLOWSPEC_LANG=zh
flowspec-cli verify --path ./src --trace ./trace.json

# Or use system LANG variable
export LANG=zh_CN.UTF-8
flowspec-cli verify --path ./src --trace ./trace.json
```

#### Supported Languages

| Language | Code | Name | Status |
|----------|------|------|--------|
| English | `en` | English | ✅ Default |
| Chinese (Simplified) | `zh` | 简体中文 | ✅ Full Support |
| Chinese (Traditional) | `zh-TW` | 繁體中文 | ✅ Full Support |
| Japanese | `ja` | 日本語 | ✅ Full Support |
| Korean | `ko` | 한국어 | ✅ Full Support |
| French | `fr` | Français | ✅ Full Support |
| German | `de` | Deutsch | ✅ Full Support |
| Spanish | `es` | Español | ✅ Full Support |

**Note**: All languages support both human-readable reports and error messages. JSON output format is language-independent.

#### Language Features

- **Localized Reports**: Validation reports, error messages, and progress indicators
- **Localized Help**: Command help text and usage examples
- **Auto-detection**: Automatic language detection from system environment
- **Fallback**: Graceful fallback to English if unsupported language is specified
- **CI/CD Friendly**: Language settings work in automated environments

#### Examples by Language

```bash
# English (default)
flowspec-cli verify --path ./contract.yaml --trace ./trace.json --lang en
# Output: "✅ All 3 validations passed"

# Chinese Simplified
flowspec-cli verify --path ./contract.yaml --trace ./trace.json --lang zh
# Output: "✅ 所有 3 项验证通过"

# Japanese
flowspec-cli verify --path ./contract.yaml --trace ./trace.json --lang ja
# Output: "✅ 3つの検証がすべて成功しました"
```

### Using in Node.js Projects

If you installed FlowSpec CLI via NPM, you can integrate it into your Node.js development workflow:

#### In package.json Scripts

```json
{
  "scripts": {
    "validate:specs": "flowspec-cli verify --path=./src --trace=./traces/integration.json --output=json",
    "validate:contracts": "flowspec-cli verify --path=./contracts/ --trace=./traces/integration.json --ci",
    "generate:contracts": "flowspec-cli explore --traffic=./logs/ --out=./contracts/service-spec.yaml",
    "test:integration": "flowspec-cli verify --path=./services --trace=./traces/e2e.json --verbose",
    "ci:validate": "flowspec-cli verify --path=./contracts/service-spec.yaml --trace=./traces/ci-run.json --output=json > validation-report.json"
  }
}
```

Then run with npm:

```bash
npm run validate:specs
npm run test:integration
npm run ci:validate
```

#### With npx

Use directly with npx for one-off validation:

```bash
# Verify with source code annotations
npx @flowspec/cli verify --path=./my-service --trace=./traces/test-run.json --output=human

# Verify with YAML contract
npx @flowspec/cli verify --path=./service-spec.yaml --trace=./traces/test-run.json --ci

# Generate contract from traffic logs
npx @flowspec/cli explore --traffic=./logs/access.log --out=./service-spec.yaml
```

### GitHub Action Integration

FlowSpec provides a GitHub Action for easy CI/CD integration:

```yaml
name: FlowSpec Validation
on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: FlowSpec Verification
        uses: flowspec/flowspec-action@v1
        with:
          path: ./src
          trace: ./traces/integration-test.json
          ci: true
          
      - name: Upload FlowSpec Reports
        uses: actions/upload-artifact@v3
        if: always()
        with:
          name: flowspec-reports
          path: artifacts/
```

#### GitHub Action Inputs

- `path`: Path to source code or YAML contract file (required)
- `trace`: Path to trace file (required)
- `version`: FlowSpec CLI version (default: "latest")
- `ci`: Enable CI mode (default: "true")
- `status-aggregation`: Status code aggregation strategy (default: "auto")
- `required-threshold`: Required field threshold (default: "0.95")

## Contract Formats

FlowSpec supports two main contract formats: embedded ServiceSpec annotations in source code and standalone YAML contract files.

### YAML Contract Format

FlowSpec supports standalone YAML contract files that define service specifications independently of source code:

```yaml
apiVersion: flowspec/v1alpha1
kind: ServiceSpec
metadata:
  name: user-service
  version: v1.0.0
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
            query: []
          optional:
            headers: ["accept-language"]
            query: ["include"]
          stats:
            supportCount: 123
            firstSeen: "2025-08-01T12:00:00Z"
            lastSeen: "2025-08-10T12:00:00Z"
        - method: PUT
          responses:
            statusCodes: [200, 400, 500]
            aggregation: "exact"
          required:
            headers: ["authorization", "content-type"]
            query: []
    - path: /api/users
      operations:
        - method: POST
          responses:
            statusRanges: ["2xx", "4xx"]
          required:
            headers: ["authorization", "content-type"]
```

### ServiceSpec Annotation Format

FlowSpec also supports ServiceSpec annotations embedded in various programming languages:

### Java

```java
/**
 * @ServiceSpec
 * operationId: "createUser"
 * description: "Create a new user account"
 * preconditions: {
 *   "request.body.email": {"!=": null},
 *   "request.body.password": {">=": 8}
 * }
 * postconditions: {
 *   "response.status": {"==": 201},
 *   "response.body.userId": {"!=": null}
 * }
 */
public User createUser(CreateUserRequest request) { ... }
```

### TypeScript

```typescript
/**
 * @ServiceSpec
 * operationId: "createUser"
 * description: "Create a new user account"
 * preconditions: {
 *   "request.body.email": {"!=": null},
 *   "request.body.password": {">=": 8}
 * }
 * postconditions: {
 *   "response.status": {"==": 201},
 *   "response.body.userId": {"!=": null}
 * }
 */
function createUser(request: CreateUserRequest): Promise<User> { ... }
```

### Go

```go
// @ServiceSpec
// operationId: "createUser"
// description: "Create a new user account"
// preconditions: {
//   "request.body.email": {"!=": null},
//   "request.body.password": {">=": 8}
// }
// postconditions: {
//   "response.status": {"==": 201},
//   "response.body.userId": {"!=": null}
// }
func CreateUser(request CreateUserRequest) (*User, error) { ... }
```

## Development

### Prerequisites

- Go 1.21 or higher
- Make (optional, for build scripts)

### Build and Test

This project uses `make` to simplify common development tasks.

```bash
# Install or update dependencies
make deps

# Run all quality checks (formatting, vetting, linting)
make quality

# Run all unit tests
make test

# Run tests and generate a coverage report
make coverage

# Build the development binary
make build

# Remove all build artifacts and caches
make clean

# Run all CI checks locally (quality, tests, coverage, build)
make ci
```

### Project Structure

```
flowspec-cli/
├── cmd/flowspec-cli/     # CLI entry point
├── internal/             # Internal packages
│   ├── parser/          # ServiceSpec parser
│   ├── ingestor/        # OpenTelemetry trace ingestor
│   ├── engine/          # Alignment validation engine
│   └── renderer/        # Report renderer
├── testdata/            # Test data
├── build/               # Build output
└── Makefile            # Build scripts
```

## Migration Guide

### Migrating to YAML Contracts

If you're currently using embedded ServiceSpec annotations and want to migrate to standalone YAML contracts:

#### Step 1: Generate YAML from Traffic

```bash
# Generate initial YAML contract from your traffic logs
flowspec-cli explore --traffic ./logs/ --out ./service-spec.yaml --service-name "my-service"
```

#### Step 2: Review and Refine

Review the generated YAML contract and refine it based on your requirements:

```yaml
# Before (generated)
- path: /api/users/{var}
  operations:
    - method: GET
      responses:
        statusRanges: ["2xx", "4xx"]

# After (refined)
- path: /api/users/{id}
  operations:
    - method: GET
      responses:
        statusCodes: [200, 404]
        aggregation: "exact"
      required:
        headers: ["authorization"]
        query: []
```

#### Step 3: Update Validation Commands

```bash
# Before (source code annotations)
flowspec-cli align --path ./src --trace ./traces/test.json

# After (YAML contract)
flowspec-cli verify --path ./service-spec.yaml --trace ./traces/test.json
```

#### Step 4: Update CI/CD Pipelines

```yaml
# Before
- name: Validate ServiceSpecs
  run: flowspec-cli align --path ./src --trace ./traces/ci-test.json

# After
- name: Validate Contracts
  run: flowspec-cli verify --path ./service-spec.yaml --trace ./traces/ci-test.json --ci
```

### Format Compatibility

FlowSpec maintains backward compatibility:

- **Old format** (methods array): Still supported for existing contracts
- **New format** (operations array): Recommended for new contracts
- **Mixed usage**: You can use both formats in the same project

#### Old Format (Deprecated but Supported)
```yaml
spec:
  endpoints:
    - path: /api/users/{id}
      methods: ["GET", "PUT"]
      statusCodes: [200, 404]
```

#### New Format (Recommended)
```yaml
spec:
  endpoints:
    - path: /api/users/{id}
      operations:
        - method: GET
          responses:
            statusCodes: [200, 404]
        - method: PUT
          responses:
            statusCodes: [200, 400]
```

### Migrating from Manual Installation to NPM

If you're currently using FlowSpec CLI with manual installation or `go install`, you can easily migrate to the NPM package:

#### For Node.js Projects

1. **Install the NPM package**:
   ```bash
   npm install @flowspec/cli --save-dev
   ```

2. **Update your scripts** in `package.json`:
   ```json
   {
     "scripts": {
       "validate": "flowspec-cli align --path=./src --trace=./traces/test.json --output=json"
     }
   }
   ```

3. **Remove manual binary** (optional):
   ```bash
   # Remove from PATH or delete manually installed binary
   rm /usr/local/bin/flowspec-cli  # or wherever you installed it
   ```

#### For CI/CD Pipelines

Replace manual installation steps:

**Before (manual installation)**:
```yaml
- name: Install FlowSpec CLI
  run: |
    curl -L https://github.com/flowspec/flowspec-cli/releases/latest/download/flowspec-cli-linux-amd64.tar.gz | tar xz
    sudo mv flowspec-cli /usr/local/bin/
```

**After (NPM installation)**:
```yaml
- name: Setup Node.js
  uses: actions/setup-node@v3
  with:
    node-version: '18'
    
- name: Install FlowSpec CLI
  run: npm install -g @flowspec/cli
```

#### Benefits of NPM Installation

- ✅ **Automatic platform detection** - No need to specify architecture
- ✅ **Version management** - Easy to pin specific versions
- ✅ **Integrated workflow** - Works seamlessly with Node.js projects
- ✅ **Dependency management** - Managed alongside other dev dependencies
- ✅ **Security** - Automatic checksum verification

## Examples

### Traffic Exploration Example

Discover service patterns from Nginx access logs:

```bash
# Generate contract from access logs
flowspec-cli explore --traffic ./logs/access.log --out ./service-spec.yaml

# Example generated YAML contract
cat service-spec.yaml
```

Output:
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
          required:
            headers: ["authorization"]
          stats:
            supportCount: 45
            firstSeen: "2025-08-01T10:30:00Z"
            lastSeen: "2025-08-10T15:45:00Z"
```

### Contract Verification Example

Verify the generated contract against trace data:

```bash
# Verify using the generated YAML contract
flowspec-cli verify --path ./service-spec.yaml --trace ./traces/test-run.json --ci
```

Output:
```
✅ All 3 checks passed in 1.2s
```

### Complete Workflow Example

```bash
# Step 1: Explore traffic patterns and generate contract
flowspec-cli explore \
  --traffic ./logs/nginx-access.log \
  --out ./contracts/user-service.yaml \
  --service-name "user-service" \
  --service-version "v1.2.0" \
  --required-threshold 0.9

# Step 2: Verify contract against integration test traces
flowspec-cli verify \
  --path ./contracts/user-service.yaml \
  --trace ./traces/integration-test.json \
  --output json > validation-report.json

# Step 3: Check results
echo "Validation completed with exit code: $?"
```

## Example Projects

Check out the example projects in the [examples](examples/) directory to learn how to use FlowSpec CLI in a real project.

## Documentation

- 📖 [API Documentation](docs/en/API.md) - Detailed API interface documentation
- 🏗️ [Architecture Document](docs/en/ARCHITECTURE.md) - Technical architecture and design decisions
- 🔄 [Migration Guide v2](docs/MIGRATION_GUIDE_v2.md) - Complete migration guide from v1 to v2
- 📋 [Version Compatibility](docs/VERSION_COMPATIBILITY.md) - Version compatibility and upgrade paths
- 🤝 [Contribution Guide](CONTRIBUTING.md) - How to participate in project development
- 📋 [Changelog](CHANGELOG.md) - Version update history

## Performance Benchmarks

- **Parsing Performance**: 1,000 source files, 200 ServiceSpecs, < 30 seconds
- **Memory Usage**: 100MB trace file, peak memory < 500MB
- **Test Coverage**: Core modules > 80%

## Roadmap

### Completed ✅
- [x] Traffic exploration and contract generation
- [x] YAML contract support
- [x] CI/CD optimizations and GitHub Action
- [x] Enhanced error handling and exit codes
- [x] Multi-format trace support

### Planned 🚧
- [ ] Support for more programming languages (Python, C#, Rust)
- [ ] Real-time trace stream processing
- [ ] Web UI interface
- [ ] Performance analysis and optimization suggestions
- [ ] Integration test automation
- [ ] HAR file support for traffic exploration
- [ ] Advanced path clustering algorithms

## Contribution

We welcome contributions of all forms! Please check out [CONTRIBUTING.md](CONTRIBUTING.md) to learn how to get involved.

### Contributors

Thank you to all the developers who have contributed to the FlowSpec CLI!

## License

This project is licensed under the Apache-2.0 License. See the [LICENSE](LICENSE) file for details.

## Support

If you encounter problems or have questions, please:

1. 📚 Check the [Documentation](https://github.com/FlowSpec/flowspec-cli/tree/main/docs/en) and [FAQ](https://github.com/FlowSpec/flowspec-cli/blob/main/docs/en/FAQ.md)
2. 🔍 Search existing [GitHub Issues](https://github.com/FlowSpec/flowspec-cli/issues)
3. 💬 Participate in [GitHub Discussions](https://github.com/FlowSpec/flowspec-cli/discussions)
4. 🐛 [Create a new Issue](https://github.com/FlowSpec/flowspec-cli/issues/new/choose) to describe your problem

## Community

- 💬 [GitHub Discussions](https://github.com/FlowSpec/flowspec-cli/discussions) - Discussions and Q&A
- 🐛 [GitHub Issues](https://github.com/FlowSpec/flowspec-cli/issues) - Bug reports and feature requests
- 📧 [Mailing List](mailto:youming@flowspec.org) - Project announcements
- 💬 [Discord Community](https://discord.gg/8zD56fYN) - Real-time communication

---

**Note**: This is a project under active development, and APIs and features may change. We will maintain backward compatibility before major version releases.

⭐ If you find this project helpful, please give us a Star!

---
**Disclaimer**: This project is supported and maintained by FlowSpec.