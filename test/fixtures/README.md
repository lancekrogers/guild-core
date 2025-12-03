# Test Fixtures

This directory contains test fixtures for Guild Framework testing.

## Directory Structure

```
fixtures/
├── README.md              # This file
├── configs/               # Sample configuration files
│   ├── agents/           # Agent configuration examples
│   ├── guilds/           # Guild configuration examples  
│   └── campaigns/        # Campaign configuration examples
├── responses/            # Mock agent responses
│   ├── elena/           # Elena agent responses
│   ├── marcus/          # Marcus agent responses
│   └── vera/            # Vera agent responses
└── projects/            # Sample project structures
    ├── go-project/      # Go project example
    ├── js-project/      # JavaScript project example
    └── python-project/  # Python project example
```

## Usage

Test fixtures are used by integration tests, end-to-end tests, and benchmarks to provide consistent test data.

### Loading Fixtures

```go
import "github.com/guild-ventures/guild-core/test/fixtures"

// Load agent config
agentConfig := fixtures.LoadAgentConfig("elena-guild-master.yaml")

// Load mock response
response := fixtures.LoadMockResponse("elena", "greeting")
```

### Adding New Fixtures

1. Add files to appropriate subdirectory
2. Follow naming conventions (kebab-case)
3. Include documentation comments in YAML files
4. Update this README if adding new categories
