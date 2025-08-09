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

- 📝 Parse ServiceSpec annotations from multi-language source code (Java, TypeScript, Go).
- 📊 Ingest and process OpenTelemetry trace data.
- ✅ Perform alignment validation between specifications and actual traces.
- 📋 Generate detailed validation reports (Human and JSON formats).
- 🔧 Supports a command-line interface for easy integration into CI/CD pipelines.

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

```bash
# Perform alignment validation
flowspec-cli align --path=./my-project --trace=./traces/run-1.json --output=human

# JSON format output
flowspec-cli align --path=./my-project --trace=./traces/run-1.json --output=json

# Verbose output
flowspec-cli align --path=./my-project --trace=./traces/run-1.json --output=human --verbose
```

### Command Options

- `--path, -p`: Source code directory path (default: ".")
- `--trace, -t`: OpenTelemetry trace file path (required)
- `--output, -o`: Output format (human|json, default: "human")
- `--verbose, -v`: Enable verbose output
- `--log-level`: Set log level (debug, info, warn, error)

### Using in Node.js Projects

If you installed FlowSpec CLI via NPM, you can integrate it into your Node.js development workflow:

#### In package.json Scripts

```json
{
  "scripts": {
    "validate:specs": "flowspec-cli align --path=./src --trace=./traces/integration.json --output=json",
    "test:integration": "flowspec-cli align --path=./services --trace=./traces/e2e.json --verbose",
    "ci:validate": "flowspec-cli align --path=. --trace=./traces/ci-run.json --output=json > validation-report.json"
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
npx @flowspec/cli align --path=./my-service --trace=./traces/test-run.json --output=human
```

## ServiceSpec Annotation Format

FlowSpec supports ServiceSpec annotations in various programming languages:

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

## Example Projects

Check out the example projects in the [examples](examples/) directory to learn how to use FlowSpec CLI in a real project.

## Documentation

- 📖 [API Documentation](docs/en/API.md) - Detailed API interface documentation
- 🏗️ [Architecture Document](docs/en/ARCHITECTURE.md) - Technical architecture and design decisions
- 🔄 [Migration Guide](docs/MIGRATION_GUIDE.md) - Migrate from manual installation to NPM
- 🤝 [Contribution Guide](CONTRIBUTING.md) - How to participate in project development
- 📋 [Changelog](CHANGELOG.md) - Version update history

## Performance Benchmarks

- **Parsing Performance**: 1,000 source files, 200 ServiceSpecs, < 30 seconds
- **Memory Usage**: 100MB trace file, peak memory < 500MB
- **Test Coverage**: Core modules > 80%

## Roadmap

- [ ] Support for more programming languages (Python, C#, Rust)
- [ ] Real-time trace stream processing
- [ ] Web UI interface
- [ ] Performance analysis and optimization suggestions
- [ ] Integration test automation

## Contribution

We welcome contributions of all forms! Please check out [CONTRIBUTING.md](CONTRIBUTING.md) to learn how to get involved.

### Contributors

Thank you to all the developers who have contributed to the FlowSpec CLI!

## License

This project is licensed under the Apache-2.0 License. See the [LICENSE](LICENSE) file for details.

## Support

If you encounter problems or have questions, please:

1. 📚 Check the [Documentation](https://github.com/FlowSpec/flowspec_cli/tree/main/docs/en) and [FAQ](https://github.com/FlowSpec/flowspec_cli/blob/main/docs/en/FAQ.md)
2. 🔍 Search existing [GitHub Issues](https://github.com/FlowSpec/flowspec_cli/issues)
3. 💬 Participate in [GitHub Discussions](https://github.com/FlowSpec/flowspec_cli/discussions)
4. 🐛 [Create a new Issue](https://github.com/FlowSpec/flowspec_cli/issues/new/choose) to describe your problem

## Community

- 💬 [GitHub Discussions](https://github.com/FlowSpec/flowspec_cli/discussions) - Discussions and Q&A
- 🐛 [GitHub Issues](https://github.com/FlowSpec/flowspec_cli/issues) - Bug reports and feature requests
- 📧 [Mailing List](mailto:youming@flowspec.org) - Project announcements
- 💬 [Discord Community](https://discord.gg/8zD56fYN) - Real-time communication

---

**Note**: This is a project under active development, and APIs and features may change. We will maintain backward compatibility before major version releases.

⭐ If you find this project helpful, please give us a Star!

---
**Disclaimer**: This project is supported and maintained by FlowSpec.