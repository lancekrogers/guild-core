# Changelog

All notable changes to the Guild Tool Parser will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-01-08

### Added

#### Core Features

- **Robust parsing architecture** with proper JSON/XML parsing instead of regex
- **Multi-provider support** for OpenAI and Anthropic formats
- **Automatic format detection** with confidence scoring (0.0-1.0)
- **Context support** for cancellation and timeouts
- **Streaming parser** for handling large inputs efficiently
- **Extensible architecture** for adding custom formats

#### Observability

- **Prometheus metrics** for monitoring parser performance
- **OpenTelemetry tracing** for distributed tracing support
- **Health check system** with multiple check types
- **Alerting framework** with customizable conditions
- **Real-time dashboard** for monitoring parser status

#### Configuration

- `WithMaxInputSize()` - Set maximum input size
- `WithTimeout()` - Set parsing timeout
- `WithStrictValidation()` - Enable strict validation mode
- `WithEnableFuzzyMatch()` - Control fuzzy matching
- `WithCustomDetector()` - Add custom format detectors
- `WithCustomParser()` - Add custom format parsers

#### Testing

- Comprehensive unit tests for all components
- Integration tests for real-world scenarios
- Performance benchmarks
- Fuzz tests for security
- Property-based tests
- Test helpers and utilities

### Changed

- **Parsing approach**: Replaced regex with proper JSON/XML parsing
- **Error handling**: Returns empty slice instead of error for no tool calls
- **Package structure**: Consolidated into main parser package with subpackages
- **Import cycles**: Resolved through proper architecture with types package

### Fixed

- Fragile regex-based format detection
- Import cycles between packages
- Poor error handling and recovery
- Missing context support
- Lack of observability

### Security

- Input size limits to prevent DoS
- Timeout support to prevent hanging
- Safe handling of malformed inputs
- Protection against XML bombs
- Validated through fuzz testing

## [0.1.0] - 2024-12-01 (Previous Version)

### Initial Implementation

- Basic regex-based parser
- Support for OpenAI format only
- Limited error handling
- No observability features
