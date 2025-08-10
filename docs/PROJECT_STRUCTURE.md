# FlowSpec CLI Project Structure

This document describes the organization and structure of the FlowSpec CLI project.

## Root Directory Structure

```
flowspec-cli/
├── .github/                    # GitHub workflows and templates
├── .kiro/                      # Kiro IDE configuration
├── build/                      # Build artifacts (gitignored)
├── cmd/                        # Command-line applications
│   └── flowspec-cli/           # Main CLI application
├── coverage/                   # Test coverage reports (gitignored)
├── docs/                       # Documentation
├── examples/                   # Example projects and usage
├── internal/                   # Private application code
├── npm/                        # NPM package configuration
├── scripts/                    # Build and utility scripts
├── testdata/                   # Test data files
├── .gitignore                  # Git ignore rules
├── .golangci.yml              # Linter configuration
├── CHANGELOG.md               # Version history
├── CODE_OF_CONDUCT.md         # Community guidelines
├── CONTRIBUTING.md            # Contribution guidelines
├── LICENSE                    # Apache 2.0 license
├── Makefile                   # Build automation
├── README.md                  # Project overview
├── go.mod                     # Go module definition
└── go.sum                     # Go module checksums
```

## Directory Details

### `/cmd/flowspec-cli/`
Contains the main CLI application entry point.

- `main.go` - CLI application with Cobra commands
- Version information is managed via build-time ldflags

### `/internal/`
Private application packages following Go conventions.

```
internal/
├── engine/         # Core validation engine
├── i18n/          # Internationalization support
├── ingestor/      # Trace data ingestion
├── models/        # Data models and structures
├── monitor/       # Performance monitoring
├── parser/        # ServiceSpec annotation parsing
└── renderer/      # Report rendering (human/JSON)
```

### `/docs/`
All project documentation organized by topic.

```
docs/
├── en/                           # English documentation
├── zh/                           # Chinese documentation
├── project-management/           # Project management docs
├── I18N_IMPLEMENTATION_SUMMARY.md  # i18n implementation details
├── LANGUAGE_CONFIGURATION.md    # Language setup guide
├── MIGRATION_GUIDE.md           # Version migration guide
├── NPM_WORKFLOW.md              # NPM publishing workflow
├── PROJECT_STRUCTURE.md         # This file
└── RELEASE_NOTES_v0.2.0.md      # Release notes
```

### `/examples/`
Example projects demonstrating FlowSpec CLI usage.

### `/scripts/`
Build, test, and utility scripts.

- `coverage.sh` - Test coverage reporting
- `integration-test-scenarios.sh` - Integration testing
- `performance-test.sh` - Performance benchmarking

### `/testdata/`
Test data files used by unit and integration tests.

## Build Artifacts

The following directories contain build artifacts and are gitignored:

- `build/` - Compiled binaries and packages
- `coverage/` - Test coverage reports
- `performance_reports/` - Performance test results

## Configuration Files

- `.gitignore` - Defines ignored files and directories
- `.golangci.yml` - Linter configuration
- `Makefile` - Build automation and common tasks
- `go.mod` / `go.sum` - Go module dependencies

## Documentation Organization

Documentation is organized by:

1. **Language**: English (`en/`) and Chinese (`zh/`) subdirectories
2. **Topic**: Specific documentation files in the root `docs/` directory
3. **Audience**: User guides, developer docs, and project management

## Best Practices

### File Naming
- Use kebab-case for documentation files: `MIGRATION_GUIDE.md`
- Use snake_case for Go files: `alignment_report.go`
- Use descriptive names that indicate purpose

### Directory Structure
- Follow Go project layout conventions
- Keep private code in `internal/`
- Separate concerns into focused packages
- Use clear, descriptive package names

### Documentation
- Keep README.md concise and focused on getting started
- Place detailed documentation in `docs/`
- Maintain both English and Chinese versions for key docs
- Update this structure document when making significant changes

## Version Management

Version information is managed through:

1. **Git Tags**: Semantic versioning (e.g., `v0.2.0`)
2. **Build-time Variables**: Set via Makefile ldflags
3. **CHANGELOG.md**: Detailed version history
4. **Release Notes**: Specific release documentation in `docs/`

## Maintenance

This structure should be maintained by:

1. Keeping build artifacts out of version control
2. Organizing documentation logically
3. Following Go project conventions
4. Updating this document when structure changes
5. Regular cleanup of temporary files and outdated documentation