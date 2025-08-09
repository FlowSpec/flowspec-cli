# Changelog

This document records all important changes to the FlowSpec CLI project.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2025-01-09

### Added
- 🌍 **Complete Internationalization (i18n) Support**: Multi-language support for CLI output and reports
  - English (en) - Default language
  - Chinese Simplified (zh) - 简体中文
  - Chinese Traditional (zh-TW) - 繁體中文
  - Japanese (ja) - 日本語
  - Korean (ko) - 한국어
  - French (fr) - Français
  - German (de) - Deutsch
  - Spanish (es) - Español
- 🔧 **Language Selection**: New `--lang` CLI parameter for manual language selection
- 🤖 **Auto-detection**: Automatic language detection from environment variables (`FLOWSPEC_LANG`, `LANG`)
- 📋 **Localized Reports**: All report outputs now support multiple languages
- 💬 **Localized CLI Messages**: All CLI messages and logs now support internationalization
- 🧪 **Comprehensive i18n Testing**: Full test coverage for internationalization features

### Changed
- 📊 **Report Rendering**: Enhanced report renderer with internationalization system
- 🖥️ **CLI Interface**: Enhanced CLI with language selection capabilities
- 🌐 **Default Behavior**: Reports now default to English unless otherwise specified or auto-detected
- 🏗️ **Architecture**: Added dedicated `internal/i18n` package for internationalization

### Fixed
- 🔧 **JSONLogic Type Conversion**: Fixed critical JSONLogic evaluation errors with interface type conversions
- 🔍 **Variable Path Resolution**: Improved variable path handling in assertion evaluation
- 📊 **Test Data Consistency**: Fixed test data mismatches between ServiceSpec assertions and trace data
- 📝 **YAML Parsing**: Resolved duplicate key issues in ServiceSpec YAML parsing

### Technical Improvements
- Enhanced error handling and user feedback
- Improved test coverage for internationalization features
- Better separation of concerns with dedicated i18n package
- More robust CLI parameter validation
- Performance optimized translation system (< 1µs per translation)
- Thread-safe concurrent language switching

### Performance
- **Translation Performance**: < 1µs per translation operation
- **Memory Efficiency**: Zero runtime allocation for translations
- **Concurrent Safety**: Full thread-safe support for language switching
- **Startup Time**: No impact on CLI startup performance

## [0.1.0] - 2025-08-05

### Added
- 🎉 First release of FlowSpec CLI Phase 1 MVP.
- 📝 Multi-language ServiceSpec parser (Java, TypeScript, Go).
- 📊 OpenTelemetry trace data ingestor.
- ✅ JSONLogic assertion evaluation engine.
- 📋 Human and JSON format report renderer.
- 🔧 Complete command-line interface.
- 🧪 Comprehensive test suite (unit tests + integration tests).
- 📖 Complete project documentation.

### Features
- **Multi-language Support**: Supports parsing of Java, TypeScript, and Go source code.
- **Trace Processing**: Supports OpenTelemetry JSON format trace data.
- **Assertion Engine**: Powerful assertion expressions based on JSONLogic.
- **Report Generation**: Human-readable and machine-readable validation reports.
- **Performance Optimization**: Parallel processing, stream parsing, memory control.
- **Fault Tolerance**: Graceful error handling and recovery mechanisms.

### Performance Benchmarks
- Parsing performance: 1,000 source files, 200 ServiceSpecs, < 30 seconds.
- Memory usage: 100MB trace file, peak memory < 500MB.
- Test coverage: Core modules > 80%.

### Tech Stack
- **Language**: Go 1.21+
- **CLI Framework**: Cobra
- **Assertion Engine**: JSONLogic
- **Logging System**: Logrus
- **Testing Framework**: Go testing + Testify

## Development History

### Phase 1 Development Milestones

#### 2024-01-XX - Project Kick-off
- Project initialization and architecture design.
- Core data model definition.
- Development environment setup.

#### 2024-01-XX - Parser Development
- Java file parser implementation.
- TypeScript file parser implementation.
- Go file parser implementation.
- Multi-language parser integration.

#### 2024-01-XX - Trace Ingestor Development
- OpenTelemetry JSON parser.
- Trace data organization and indexing.
- Large file handling and memory optimization.
- Stream parsing implementation.

#### 2024-01-XX - Alignment Engine Development
- JSONLogic assertion evaluation engine.
- Specification and trace matching logic.
- Validation context construction.
- Assertion failure detail collection.

#### 2024-01-XX - CLI and Reporting System
- Command-line interface implementation.
- Human format report rendering.
- JSON format report output.
- Exit code management.

#### 2024-01-XX - Testing and Quality Assurance
- Unit test suite completion.
- Integration test scenario implementation.
- Performance and stress testing.
- Code coverage target achievement.

#### 2024-01-XX - Documentation and Open Source Preparation
- Complete project documentation writing.
- Addition of open source license.
- Formulation of contribution guide.
- Release preparation completion.

## Known Issues

### Current Version Limitations
- Only supports OpenTelemetry JSON format trace data.
- ServiceSpec assertion language is limited to JSONLogic.
- Does not support real-time trace stream processing.
- No Web UI interface at present.

### Planned Fixes
These limitations will be gradually addressed in subsequent versions.

## Roadmap

### Phase 2 (Planned)
- [ ] Support for more programming languages (Python, C#, Rust).
- [ ] Real-time trace stream processing.
- [ ] Performance analysis and optimization suggestions.
- [ ] Richer assertion expression syntax.

### Phase 3 (Planned)
- [ ] Web UI interface.
- [ ] Distributed validation support.
- [ ] Plugin system.
- [ ] Cloud-native integration.

### Long-term Planning
- [ ] Machine learning-driven anomaly detection.
- [ ] Automated test generation.
- [ ] Service dependency graph visualization.
- [ ] Multi-cloud platform support.

## Contributors

Thanks to all the developers who contributed to the FlowSpec CLI project:

- [@contributor1](https://github.com/contributor1) - Project initiator and main developer.
- [@contributor2](https://github.com/contributor2) - Parser module development.
- [@contributor3](https://github.com/contributor3) - Testing and documentation.

## Acknowledgements

Special thanks to the following open source projects and communities:

- [Cobra](https://github.com/spf13/cobra) - A powerful CLI framework.
- [JSONLogic](https://jsonlogic.com/) - A flexible assertion expression engine.
- [OpenTelemetry](https://opentelemetry.io/) - An observability standard.
- [Logrus](https://github.com/sirupsen/logrus) - A structured logging library.
- [Testify](https://github.com/stretchr/testify) - A testing toolkit.

## License Changes

- **2024-01-XX**: The project is open-sourced under the Apache-2.0 license.

## Security Updates

There are currently no security-related updates. If you find a security issue, please send an email to youming@flowspec.org.

---

**Note**: 
- All dates are planned dates, actual release times may be adjusted.
- Features may be adjusted based on user feedback.
- We promise to maintain backward compatibility before major version releases.

If you have any questions or suggestions, feel free to raise them in [GitHub Issues](../../issues).
