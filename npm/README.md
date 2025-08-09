# @flowspec/cli

[![npm version](https://badge.fury.io/js/%40flowspec%2Fcli.svg)](https://badge.fury.io/js/%40flowspec%2Fcli)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Node.js Version](https://img.shields.io/badge/Node.js-14%2B-brightgreen.svg)](https://nodejs.org/)

FlowSpec CLI is a powerful command-line tool for parsing ServiceSpec annotations from source code, ingesting OpenTelemetry traces, and performing alignment validation between specifications and actual execution traces. This NPM package provides a convenient wrapper for Node.js projects.

## Installation

### As a Development Dependency (Recommended)

Install the FlowSpec CLI as a development dependency in your Node.js project:

```bash
npm install @flowspec/cli --save-dev
```

### Global Installation

Install globally to use across multiple projects:

```bash
npm install -g @flowspec/cli
```

### Using with npx (No Installation Required)

Use directly with npx for one-off commands:

```bash
npx @flowspec/cli --help
```

## Usage

### Basic Command Line Interface

Once installed, you can use the `flowspec-cli` command:

```bash
# Show help and available commands
flowspec-cli --help

# Check version
flowspec-cli --version

# Perform alignment validation
flowspec-cli align --path=./src --trace=./traces/run-1.json --output=human

# JSON format output for CI/CD integration
flowspec-cli align --path=./src --trace=./traces/run-1.json --output=json

# Verbose output for debugging
flowspec-cli align --path=./src --trace=./traces/run-1.json --output=human --verbose
```

### Integration with package.json Scripts

Add FlowSpec CLI to your package.json scripts for seamless integration:

```json
{
  "scripts": {
    "validate:specs": "flowspec-cli align --path=./src --trace=./traces/integration.json --output=json",
    "test:integration": "flowspec-cli align --path=./services --trace=./traces/e2e.json --verbose",
    "ci:validate": "flowspec-cli align --path=. --trace=./traces/ci-run.json --output=json > validation-report.json",
    "pretest": "flowspec-cli align --path=./src --trace=./traces/unit-tests.json",
    "postbuild": "flowspec-cli align --path=./dist --trace=./traces/build-validation.json"
  }
}
```

Then run with npm:

```bash
npm run validate:specs
npm run test:integration
npm run ci:validate
```

### Advanced Usage Examples

#### Multi-Service Validation

```bash
# Validate multiple services
flowspec-cli align --path=./services/user-service --trace=./traces/user-service.json --output=json
flowspec-cli align --path=./services/order-service --trace=./traces/order-service.json --output=json
```

#### CI/CD Pipeline Integration

```json
{
  "scripts": {
    "test:e2e": "npm run test:e2e:run && npm run validate:e2e",
    "test:e2e:run": "jest --config=jest.e2e.config.js",
    "validate:e2e": "flowspec-cli align --path=./src --trace=./traces/e2e-results.json --output=json --log-level=info"
  }
}
```

#### Development Workflow

```json
{
  "scripts": {
    "dev": "concurrently \"npm run dev:server\" \"npm run dev:validate\"",
    "dev:server": "nodemon src/server.js",
    "dev:validate": "chokidar \"traces/*.json\" -c \"flowspec-cli align --path=./src --trace={path} --output=human\""
  }
}
```

### Using with npx

Perfect for one-off validations or trying FlowSpec CLI without installation:

```bash
# Quick validation
npx @flowspec/cli align --path=./my-service --trace=./traces/test-run.json --output=human

# Generate validation report
npx @flowspec/cli align --path=./src --trace=./traces/integration.json --output=json > report.json

# Verbose debugging
npx @flowspec/cli align --path=./services --trace=./traces/debug.json --verbose --log-level=debug
```

## Command Options

- `--path, -p`: Source code directory path (default: ".")
- `--trace, -t`: OpenTelemetry trace file path (required)
- `--output, -o`: Output format (human|json, default: "human")
- `--verbose, -v`: Enable verbose output
- `--log-level`: Set log level (debug, info, warn, error)

## ServiceSpec Annotation Examples

FlowSpec CLI parses ServiceSpec annotations from your source code. Here are examples in different languages:

### TypeScript/JavaScript

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
export async function createUser(request: CreateUserRequest): Promise<User> {
  // Implementation
}
```

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
public User createUser(CreateUserRequest request) {
  // Implementation
}
```

## Platform Support

This package automatically downloads the appropriate binary for your platform during installation:

- **Linux**: x64, ARM64
- **macOS**: x64 (Intel), ARM64 (Apple Silicon)  
- **Windows**: x64

The binary is downloaded from the official [FlowSpec CLI releases](https://github.com/flowspec/flowspec-cli/releases) and verified with SHA256 checksums for security.

## Requirements

- **Node.js**: 14.0.0 or higher
- **Operating System**: Linux, macOS, or Windows
- **Architecture**: x64 or ARM64 (platform dependent)
- **Network**: Internet connection required for initial installation

## Troubleshooting

### Installation Issues

#### Binary Download Failures

If the installation fails during binary download:

1. **Check network connectivity**:
   ```bash
   # Test GitHub access
   curl -I https://api.github.com/repos/flowspec/flowspec-cli/releases/latest
   ```

2. **Clear npm cache and retry**:
   ```bash
   npm cache clean --force
   npm install @flowspec/cli --save-dev
   ```

3. **Check proxy settings** (if behind corporate firewall):
   ```bash
   npm config set proxy http://proxy.company.com:8080
   npm config set https-proxy http://proxy.company.com:8080
   ```

4. **Verify platform support**:
   ```bash
   node -e "console.log(process.platform, process.arch)"
   ```
   Supported combinations: linux-x64, linux-arm64, darwin-x64, darwin-arm64, win32-x64

#### Checksum Verification Failures

If checksum verification fails:

1. **Retry installation** (may be a temporary download issue):
   ```bash
   npm uninstall @flowspec/cli
   npm install @flowspec/cli --save-dev
   ```

2. **Check for corrupted download**:
   ```bash
   # Remove any cached binaries and reinstall
   rm -rf node_modules/@flowspec/cli
   npm install @flowspec/cli --save-dev
   ```

#### Permission Issues (Unix/Linux/macOS)

If you encounter permission errors during installation:

1. **Fix binary permissions**:
   ```bash
   chmod +x node_modules/.bin/flowspec-cli
   ```

2. **For global installations**, use sudo or fix npm permissions:
   ```bash
   # Option 1: Use sudo (not recommended)
   sudo npm install -g @flowspec/cli
   
   # Option 2: Fix npm permissions (recommended)
   mkdir ~/.npm-global
   npm config set prefix '~/.npm-global'
   echo 'export PATH=~/.npm-global/bin:$PATH' >> ~/.bashrc
   source ~/.bashrc
   npm install -g @flowspec/cli
   ```

### Runtime Issues

#### Command Not Found

If `flowspec-cli` command is not found:

1. **Verify installation**:
   ```bash
   npm list @flowspec/cli
   ```

2. **Check binary location**:
   ```bash
   ls -la node_modules/.bin/flowspec-cli
   ```

3. **Use npx as alternative**:
   ```bash
   npx @flowspec/cli --help
   ```

4. **Reinstall package**:
   ```bash
   npm uninstall @flowspec/cli
   npm install @flowspec/cli --save-dev
   ```

#### Binary Execution Failures

If the binary fails to execute:

1. **Check binary integrity**:
   ```bash
   node_modules/.bin/flowspec-cli --version
   ```

2. **Verify binary permissions** (Unix/Linux/macOS):
   ```bash
   ls -la node_modules/@flowspec/cli/bin/
   chmod +x node_modules/@flowspec/cli/bin/flowspec-cli-*
   ```

3. **Check system compatibility**:
   ```bash
   # On Linux, check for missing dependencies
   ldd node_modules/@flowspec/cli/bin/flowspec-cli-linux-* 2>/dev/null || echo "Static binary"
   ```

4. **Force reinstallation**:
   ```bash
   rm -rf node_modules/@flowspec/cli
   npm install @flowspec/cli --save-dev
   ```

### Network and Proxy Issues

#### Corporate Firewall/Proxy

If installation fails behind a corporate firewall:

1. **Configure npm proxy settings**:
   ```bash
   npm config set proxy http://proxy.company.com:8080
   npm config set https-proxy http://proxy.company.com:8080
   npm config set registry https://registry.npmjs.org/
   ```

2. **Bypass SSL verification** (if necessary, not recommended for production):
   ```bash
   npm config set strict-ssl false
   ```

3. **Use alternative registry**:
   ```bash
   npm install @flowspec/cli --registry https://registry.npmjs.org/
   ```

#### GitHub API Rate Limiting

If you encounter rate limiting issues:

1. **Wait and retry** (rate limits reset hourly)
2. **Use authenticated requests** (if you have a GitHub token):
   ```bash
   export GITHUB_TOKEN=your_token_here
   npm install @flowspec/cli --save-dev
   ```

### Platform-Specific Issues

#### Windows

1. **PowerShell execution policy**:
   ```powershell
   Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
   ```

2. **Windows Defender/Antivirus**:
   - Add `node_modules/@flowspec/cli/bin/` to antivirus exclusions
   - Temporarily disable real-time protection during installation

#### macOS

1. **Gatekeeper issues**:
   ```bash
   # If macOS blocks the binary
   xattr -d com.apple.quarantine node_modules/@flowspec/cli/bin/flowspec-cli-darwin-*
   ```

2. **Apple Silicon compatibility**:
   ```bash
   # Verify correct ARM64 binary is downloaded
   file node_modules/@flowspec/cli/bin/flowspec-cli-darwin-arm64
   ```

#### Linux

1. **Missing system dependencies**:
   ```bash
   # Install required system libraries (if needed)
   sudo apt-get update && sudo apt-get install -y ca-certificates
   ```

2. **SELinux issues**:
   ```bash
   # If SELinux blocks execution
   sudo setsebool -P allow_execstack 1
   ```

### Getting Help

If you continue to experience issues:

1. **Check existing issues**: [GitHub Issues](https://github.com/flowspec/flowspec-cli/issues)
2. **Create a new issue** with:
   - Your operating system and architecture (`node -e "console.log(process.platform, process.arch)"`)
   - Node.js version (`node --version`)
   - npm version (`npm --version`)
   - Complete error message
   - Steps to reproduce

3. **Enable debug logging**:
   ```bash
   DEBUG=* npm install @flowspec/cli --save-dev
   ```

## Migration from Manual Installation

If you're currently using FlowSpec CLI with manual installation or `go install`, see our [Migration Guide](https://github.com/flowspec/flowspec-cli/blob/main/docs/MIGRATION_GUIDE.md) for step-by-step instructions.

### Quick Migration for Node.js Projects

1. Install the NPM package: `npm install @flowspec/cli --save-dev`
2. Update your scripts in `package.json`
3. Remove the manually installed binary (optional)

## Development

This package is a wrapper around the native FlowSpec CLI binary. The source code for the CLI tool is available at [flowspec/flowspec-cli](https://github.com/flowspec/flowspec-cli).

### How It Works

1. **Installation**: The `postinstall` script automatically downloads the correct binary for your platform
2. **Platform Detection**: Detects your OS and architecture to select the appropriate binary
3. **Binary Management**: Downloads, verifies, and sets up the binary in the correct location
4. **CLI Wrapper**: Provides a transparent wrapper that forwards all commands to the native binary

### Security

- All binaries are downloaded from official GitHub releases
- SHA256 checksums are verified for every download
- HTTPS is used for all network communications
- No external dependencies beyond Node.js standard library

## Related Projects

- [FlowSpec CLI](https://github.com/flowspec/flowspec-cli) - The main CLI tool
- [FlowSpec Documentation](https://github.com/flowspec/flowspec-cli/tree/main/docs) - Comprehensive documentation
- [FlowSpec Examples](https://github.com/flowspec/flowspec-cli/tree/main/examples) - Example projects

## License

Apache-2.0 - see the [LICENSE](https://github.com/flowspec/flowspec-cli/blob/main/LICENSE) file for details.

## Support

If you encounter issues or have questions:

1. 📚 Check the [NPM Package Documentation](https://github.com/flowspec/flowspec-cli/blob/main/npm/README.md)
2. 📖 Read the [Main Documentation](https://github.com/flowspec/flowspec-cli#readme)
3. 🔧 Try the [Troubleshooting Guide](#troubleshooting)
4. 🔍 Search [GitHub Issues](https://github.com/flowspec/flowspec-cli/issues)
5. 🐛 [Create a New Issue](https://github.com/flowspec/flowspec-cli/issues/new)

### Community

- 💬 [GitHub Discussions](https://github.com/flowspec/flowspec-cli/discussions) - Questions and discussions
- 🐛 [GitHub Issues](https://github.com/flowspec/flowspec-cli/issues) - Bug reports and feature requests
- 📧 [Mailing List](mailto:youming@flowspec.org) - Project announcements

---

**Note**: This NPM package is automatically published when new FlowSpec CLI releases are created. The version number matches the CLI tool version exactly.