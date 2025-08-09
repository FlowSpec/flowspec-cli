# NPM Package Test Suite

This directory contains a comprehensive test suite for the FlowSpec CLI NPM package wrapper. The test suite ensures >80% code coverage and validates all functionality across different scenarios.

## Test Structure

### Unit Tests
- **`platform.test.js`** - Tests platform detection utilities
- **`download.test.js`** - Tests download and checksum verification functionality  
- **`binary.test.js`** - Tests binary management and execution scenarios

### Integration Tests
- **`download.integration.test.js`** - Tests complete download, extraction, and setup flow
- **`install.integration.test.js`** - Tests the main installation script and process

### End-to-End Tests
- **`e2e.test.js`** - Tests actual npm install and npx execution scenarios
- **`cli-wrapper.test.js`** - Tests the CLI wrapper script functionality

### Mock Server Tests
- **`mock-server.test.js`** - Tests download scenarios using local HTTP servers without external dependencies

### Error Scenario Tests
- **`error-scenarios.test.js`** - Comprehensive error handling, retry logic, and edge case testing

### Test Utilities
- **`setup.js`** - Jest setup file with common test utilities and configurations

## Running Tests

### All Tests
```bash
npm test
```

### Test Categories
```bash
# Unit tests only
npm run test:unit

# Integration tests only
npm run test:integration

# End-to-end tests only
npm run test:e2e

# Mock server tests only
npm run test:mock-server

# Error scenario tests only
npm run test:error-scenarios

# CLI wrapper tests only
npm run test:cli-wrapper
```

### Coverage and CI
```bash
# Generate coverage report
npm run test:coverage

# CI-friendly test run
npm run test:ci

# Verbose output
npm run test:verbose
```

### Watch Mode
```bash
npm run test:watch
```

## Test Coverage Requirements

The test suite maintains >80% coverage across:
- **Branches**: 80%
- **Functions**: 80% 
- **Lines**: 80%
- **Statements**: 80%

Coverage reports are generated in the `coverage/` directory.

## Test Categories Explained

### 1. Unit Tests
Test individual modules in isolation with mocked dependencies:

- **Platform Detection**: All supported/unsupported platform combinations
- **Download Manager**: HTTP requests, checksum verification, file operations
- **Binary Manager**: Binary execution, version reading, path resolution

### 2. Integration Tests
Test complete workflows with real file system operations:

- **Download Integration**: Full download → extract → verify → setup flow
- **Installation Integration**: Complete installation process simulation

### 3. End-to-End Tests
Test real-world usage scenarios:

- **NPM Installation**: Actual `npm install` from package tarball
- **NPX Execution**: Real `npx flowspec-cli` command execution
- **Package.json Scripts**: Integration with npm scripts
- **Cross-platform Compatibility**: Platform-specific behavior validation

### 4. Mock Server Tests
Test network scenarios without external dependencies:

- **Successful Downloads**: Various file sizes and response types
- **Error Scenarios**: 404, 500, timeouts, corrupted responses
- **Retry Logic**: Network failures and recovery
- **Performance**: Large files, concurrent downloads

### 5. Error Scenario Tests
Comprehensive error handling validation:

- **Platform Errors**: Unsupported platforms, corrupted process objects
- **Network Errors**: DNS failures, connection issues, SSL problems
- **File System Errors**: Permission denied, disk full, file corruption
- **Process Errors**: Binary execution failures, signal handling
- **Recovery Scenarios**: Cleanup failures, partial state recovery

## Test Utilities

### Global Test Utilities (`testUtils`)
- `suppressConsole()` / `restoreConsole()` - Control console output during tests
- `createTempDir()` / `cleanupTempDir()` - Temporary directory management
- `sleep(ms)` - Async delay utility
- `createMockServer()` - HTTP server creation for testing
- `randomString()` - Random string generation
- `generateMockChecksum()` - Mock checksum generation

### Global Constants (`TEST_CONSTANTS`)
- `SUPPORTED_PLATFORMS` - Array of supported platform configurations
- `UNSUPPORTED_PLATFORMS` - Array of unsupported platform configurations
- `MOCK_PACKAGE_JSON` - Standard mock package.json content
- `MOCK_CHECKSUMS` - Mock checksum values for testing

### Custom Jest Matchers
- `toBeNetworkError()` - Validates network-related errors
- `toBeFileSystemError()` - Validates filesystem-related errors
- `toBeValidChecksum()` - Validates SHA256 checksum format

## Test Data and Fixtures

### Mock Binary Content
Tests use generated tar.gz archives with mock binary content that responds to `--version` flags appropriately for each platform.

### Checksum Validation
All download tests include proper SHA256 checksum verification to ensure data integrity.

### Platform Simulation
Tests simulate all supported platform combinations using `Object.defineProperty()` to override `process.platform` and `process.arch`.

## Continuous Integration

The test suite is designed for CI environments:

- **Deterministic**: No external network dependencies in core tests
- **Fast**: Unit tests complete in seconds, integration tests in minutes
- **Reliable**: Comprehensive error handling and cleanup
- **Isolated**: Each test cleans up after itself

### CI Configuration
```bash
npm run test:ci
```

This command:
- Runs all tests once (no watch mode)
- Generates coverage reports
- Uses CI-optimized Jest settings
- Exits with appropriate codes for CI systems

## Debugging Tests

### Verbose Output
```bash
npm run test:verbose
```

### Individual Test Files
```bash
npx jest platform.test.js --verbose
```

### Debug Mode
```bash
node --inspect-brk node_modules/.bin/jest --runInBand platform.test.js
```

## Test Performance

### Timeouts
- Unit tests: 5 seconds default
- Integration tests: 30 seconds
- E2E tests: 60-120 seconds
- Mock server tests: 10 seconds

### Parallelization
Jest runs tests in parallel by default. Use `--runInBand` for sequential execution when debugging.

## Contributing to Tests

When adding new functionality:

1. **Add unit tests** for the core logic
2. **Add integration tests** for complete workflows
3. **Add error scenario tests** for failure cases
4. **Update coverage thresholds** if needed
5. **Document new test utilities** in this README

### Test Naming Conventions
- Test files: `*.test.js`
- Test descriptions: Use "should" statements
- Test groups: Use `describe()` blocks for logical grouping
- Mock files: Prefix with `mock-` or `.mock-`

### Mock Strategy
- **Unit tests**: Mock all external dependencies
- **Integration tests**: Mock only network/external services
- **E2E tests**: Minimal mocking, real operations where safe

## Troubleshooting

### Common Issues

1. **Tests timing out**: Increase timeout in individual tests or globally
2. **File system permissions**: Ensure test directories are writable
3. **Network tests failing**: Check if mock servers are properly set up
4. **Coverage not meeting threshold**: Add tests for uncovered branches

### Debug Commands
```bash
# Run specific test with debug output
DEBUG=* npm test -- --testNamePattern="should download binary"

# Run tests with Node.js debugging
node --inspect-brk node_modules/.bin/jest --runInBand

# Check test coverage for specific files
npx jest --coverage --collectCoverageFrom="lib/platform.js"
```

## Test Metrics

The test suite includes:
- **~200+ test cases** across all categories
- **>80% code coverage** on all core modules
- **Cross-platform validation** for 5 supported platforms
- **Error scenario coverage** for 50+ error conditions
- **Performance testing** for large files and concurrent operations
- **Real-world simulation** through E2E tests