# Contribution Guide

Thank you for your interest in the FlowSpec CLI project! We welcome all forms of contributions, including but not limited to:

- 🐛 Reporting Bugs
- 💡 Suggesting New Features
- 📝 Improving Documentation
- 🔧 Submitting Code Fixes or New Features
- 🧪 Writing Test Cases
- 📖 Translating Documents

## Development Environment Setup

### Prerequisites

- **Go**: 1.21 or higher
- **Git**: For version control
- **Make**: For build scripts (optional)
- **golangci-lint**: For code quality checks (recommended)

### Install golangci-lint

```bash
# macOS
brew install golangci-lint

# Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.54.2

# Windows
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.54.2
```

### Clone and Set Up the Project

```bash
# 1. Fork the project to your GitHub account
# 2. Clone your fork
git clone https://github.com/YOUR_USERNAME/flowspec-cli.git
cd flowspec-cli

# 3. Add the upstream repository
git remote add upstream https://github.com/FlowSpec/flowspec-cli.git

# 4. Install dependencies
make deps

# 5. Validate environment setup
make ci-dev
```

## Development Workflow

### 1. Create a Feature Branch

```bash
# Create a new branch from the latest main branch
git checkout main
git pull upstream main
git checkout -b feature/your-feature-name

# Or for a bug fix
git checkout -b fix/issue-number-description
```

### 2. Develop and Test

```bash
# Format code
make fmt

# Run code checks
make vet
make lint

# Run tests
make test

# Generate test coverage report
make coverage

# Build the project
make build

# Run full CI checks
make ci-dev
```

### 3. Commit Code

We use the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```bash
# Commit format
git commit -m "type(scope): description"

# Examples
git commit -m "feat(parser): add support for Python ServiceSpec annotations"
git commit -m "fix(engine): resolve JSONLogic evaluation context issue"
git commit -m "docs(readme): update installation instructions"
git commit -m "test(ingestor): add unit tests for large file processing"
```

#### Commit Types

- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Changes that do not affect the meaning of the code (white-space, formatting, missing semi-colons, etc)
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `test`: Adding missing tests or correcting existing tests
- `chore`: Changes to the build process or auxiliary tools and libraries such as documentation generation
- `perf`: A code change that improves performance
- `ci`: Changes to our CI configuration files and scripts

### 4. Push and Create a Pull Request

```bash
# Push to your fork
git push origin feature/your-feature-name

# Create a Pull Request on GitHub
```

## Coding Standards

### Go Code Style

We follow standard Go code style:

- Use `go fmt` to format code
- Follow the [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use meaningful variable and function names
- Add documentation comments for public functions and types
- Keep functions concise and with a single responsibility

### Code Organization

```
flowspec-cli/
├── cmd/flowspec-cli/     # CLI entry point
│   ├── main.go          # Main function
│   └── *_test.go        # CLI tests
├── internal/            # Internal packages (not exposed)
│   ├── parser/          # ServiceSpec parser
│   ├── ingestor/        # OpenTelemetry trace ingestor
│   ├── engine/          # Alignment validation engine
│   ├── renderer/        # Report renderer
│   └── models/          # Data models
├── testdata/            # Test data files
├── scripts/             # Build and test scripts
└── docs/                # Project documentation
```

### Testing Requirements

- **Unit Tests**: All new features must include unit tests.
- **Test Coverage**: Core modules must achieve over 80% coverage.
- **Integration Tests**: Important features must include integration tests.
- **Test Naming**: Use the format `TestFunctionName_Scenario_ExpectedResult`.

```go
func TestSpecParser_ParseJavaFile_ValidAnnotation_ReturnsServiceSpec(t *testing.T) {
    // Test implementation
}

func TestAlignmentEngine_Align_PreconditionFails_ReturnsFailedStatus(t *testing.T) {
    // Test implementation
}
```

### Error Handling

- Use Go's standard error handling patterns.
- Provide sufficient context for errors.
- Use `fmt.Errorf` to wrap errors.
- Use custom error types where appropriate.

```go
// Good error handling example
func (p *SpecParser) parseFile(filepath string) (*ServiceSpec, error) {
    content, err := os.ReadFile(filepath)
    if err != nil {
        return nil, fmt.Errorf("failed to read file %s: %w", filepath, err)
    }
    
    spec, err := p.parseContent(content)
    if err != nil {
        return nil, fmt.Errorf("failed to parse ServiceSpec in %s: %w", filepath, err)
    }
    
    return spec, nil
}
```

## Testing Guide

### Running Tests

```bash
# Run all tests
make test

# Run tests for a specific package
go test ./internal/parser/

# Run a specific test
go test -run TestSpecParser_ParseJavaFile ./internal/parser/

# Run tests and show coverage
make coverage

# Run performance tests
make performance-tests-only

# Run stress tests
make stress-test
```

### Writing Tests

#### Unit Test Example

```go
func TestSpecParser_ParseJavaFile_ValidAnnotation_ReturnsServiceSpec(t *testing.T) {
    // Arrange
    parser := NewSpecParser()
    testFile := "testdata/valid_java_annotation.java"
    
    // Act
    result, err := parser.ParseFile(testFile)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, "createUser", result.OperationID)
    assert.NotEmpty(t, result.Preconditions)
    assert.NotEmpty(t, result.Postconditions)
}
```

#### Integration Test Example

```go
func TestCLI_AlignCommand_EndToEnd_Success(t *testing.T) {
    // Prepare test data
    tempDir := t.TempDir()
    setupTestProject(t, tempDir)
    
    // Execute CLI command
    cmd := exec.Command("./build/flowspec-cli", 
        "align", 
        "--path", tempDir,
        "--trace", "testdata/success-trace.json",
        "--output", "json")
    
    output, err := cmd.CombinedOutput()
    
    // Assert results
    assert.NoError(t, err)
    
    var report AlignmentReport
    err = json.Unmarshal(output, &report)
    assert.NoError(t, err)
    assert.Equal(t, 3, report.Summary.Total)
    assert.Equal(t, 3, report.Summary.Success)
}
```

## Performance Requirements

### Performance Benchmarks

- **Parsing Performance**: 1,000 source files, 200 ServiceSpecs, completed in under 30 seconds.
- **Memory Usage**: 100MB trace file, peak memory usage not to exceed 500MB.
- **Concurrency Safety**: Safe for operation in a multi-threaded environment.

### Performance Testing

```bash
# Run performance benchmarks
make benchmark

# Run large-scale tests
make performance-tests-only

# Run memory usage tests
go test -run TestMemoryUsage ./cmd/flowspec-cli/ -timeout 30m
```

## Documentation Contribution

### Document Types

- **README.md**: Project introduction and basic usage instructions.
- **API Documentation**: API docs generated using `godoc`.
- **Technical Documentation**: Architecture design, implementation details, etc.
- **User Guides**: Detailed tutorials and examples.

### Documentation Standards

- Use clear and concise language.
- Provide practical code examples.
- Keep documentation in sync with code updates.
- Maintain parity between English and Chinese documentation.

## Pull Request Guide

### PR Title Format

```
type(scope): description

# Example
feat(parser): add Python ServiceSpec annotation support
fix(engine): resolve JSONLogic context variable issue
docs(contributing): update development setup instructions
```

### PR Description Template

```markdown
## Change Type
- [ ] Bug fix
- [ ] New feature
- [ ] Documentation update
- [ ] Performance optimization
- [ ] Code refactoring
- [ ] Test improvement

## Description of Change
Briefly describe the changes and purpose of this PR.

## Related Issue
Fixes #123
Closes #456

## Testing
- [ ] Added new unit tests
- [ ] Added integration tests
- [ ] All existing tests pass
- [ ] Manual testing passed

## Checklist
- [ ] My code follows the project's style guidelines
- [ ] I have added necessary documentation
- [ ] Test coverage meets requirements
- [ ] All CI checks pass
```

### Code Review

All PRs require a code review:

1.  **Automated Checks**: The CI/CD pipeline will automatically run tests and code checks.
2.  **Manual Review**: At least one maintainer's approval is required.
3.  **Feedback Handling**: Respond to review comments and make changes promptly.

## Release Process

### Workflow Overview

- **Daily Development**: Contributors should use `make` targets (`make quality`, `make test`, etc.) for daily development and testing.
- **Release Preparation**: Creating a new release is a standardized process managed by a dedicated script.

### Creating a Release (for Maintainers)

Maintainers should use the `prepare-release.sh` script to create a new release. This script automates all the necessary steps to ensure a consistent and reliable release.

```bash
# Usage: ./scripts/prepare-release.sh [VERSION]
# Example for version 1.0.0
./scripts/prepare-release.sh 1.0.0
```

The script will:
1.  Run prerequisite checks (clean git status, correct branch, etc.).
2.  Update the version in `version.go` and `CHANGELOG.md`.
3.  Run all CI checks (`make ci`) to ensure quality.
4.  Build and package all release binaries (`make package`).
5.  Create a final git commit and tag for the release.
6.  Provide instructions for publishing the release on GitHub.

This approach ensures that every release is built and tagged in a uniform way, minimizing human error.

## Versioning Specification

We use [Semantic Versioning](https://semver.org/):

- `MAJOR.MINOR.PATCH` (e.g., 1.2.3)
- `MAJOR`: Incompatible API changes
- `MINOR`: Backward-compatible functionality additions
- `PATCH`: Backward-compatible bug fixes

## Community Code of Conduct

### Our Pledge

In the interest of fostering an open and welcoming environment, we as contributors and maintainers pledge to making participation in our project and our community a harassment-free experience for everyone, regardless of age, body size, disability, ethnicity, sex characteristics, gender identity and expression, level of experience, education, socio-economic status, nationality, personal appearance, race, religion, or sexual identity and orientation.

### Unacceptable Behavior

- The use of sexualized language or imagery and unwelcome sexual attention or advances
- Trolling, insulting/derogatory comments, and personal or political attacks
- Public or private harassment
- Publishing others' private information, such as a physical or electronic address, without explicit permission
- Other conduct which could reasonably be considered inappropriate in a professional setting

## Getting Help

If you encounter problems during contribution, you can get help in the following ways:

- 📧 **Email**: Send an email to [youming@flowspec.org](mailto:youming@flowspec.org)
- 💬 **Discussions**: Ask questions in GitHub Discussions
- 🐛 **Issues**: Create an Issue to describe the problem
- 📖 **Documentation**: Check the project documentation and Wiki

## Acknowledgements

Thanks to all the developers who have contributed to the FlowSpec CLI project! Your contributions make this project better.

---

**Note**: This is an actively developed project, and the contribution guide may be updated as the project evolves. Please check the latest version regularly.
