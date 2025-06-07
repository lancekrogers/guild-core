# Installation Guide

## Prerequisites

- Go 1.21 or later
- Git
- Make (optional but recommended)

## Installing Guild

### From Source

```bash
# Clone the repository
git clone https://github.com/guild-ventures/guild-core.git
cd guild-core

# Install dependencies
go mod download

# Build the CLI
make build
# or
go build -o guild cmd/guild/main.go

# Add to PATH (optional)
sudo mv guild /usr/local/bin/
```

### Using Go Install

```bash
go install github.com/guild-ventures/guild-core/cmd/guild@latest
```

## Verify Installation

```bash
guild --version
guild --help
```

## Next Steps

- [Quick Start Tutorial](./quickstart.md)
- [Configuration Guide](./configuration.md)
- [Creating Your First Agent](./first-agent.md)
