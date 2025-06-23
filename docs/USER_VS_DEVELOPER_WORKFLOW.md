# User vs Developer Workflow

Guild provides two distinct workflows optimized for different audiences:

## User Workflow (Fast Path - 30 Seconds to Productive)

The user workflow is designed to get you productive as quickly as possible:

```bash
# 1. Fast install (no go vet delays)
make install

# 2. Quick initialization with Elena agent
guild init my-project
cd my-project

# 3. Set API key and start chatting
export ANTHROPIC_API_KEY="your-key"
guild chat
```

### Key Features:
- **Fast Installation**: `make install` skips go vet for quick builds
- **Instant Setup**: `guild init` creates Elena agent automatically
- **Immediate Productivity**: Start chatting in 30 seconds
- **Optional Advanced Config**: `guild setup-wizard` for detailed control

### User Commands:
- `make install` - Fast installation without development checks
- `guild init` - Quick workspace setup with Elena
- `guild chat` - Start being productive
- `guild setup-wizard` - Advanced TUI configuration (optional)

## Developer Workflow (Full Validation)

The developer workflow ensures code quality and comprehensive validation:

```bash
# Full build with go vet validation
make build

# Run comprehensive test suite
make test

# Development helpers
make quick      # Fast build without visuals
make dashboard  # Project status
make clean      # Clean all artifacts
```

### Key Features:
- **Full Validation**: `make build` includes go vet checks
- **Comprehensive Testing**: Complete test suite with coverage
- **Visual Feedback**: Progress bars and build status
- **Quality Gates**: Ensures enterprise-ready code

### Developer Commands:
- `make build` - Full build with validation
- `make test` - Run all tests properly
- `make integration` - Integration test suite
- `make benchmark` - Performance benchmarks
- `make ci-*` - CI variants without colors

## Important Distinctions

### Installation Speed:
- **User**: `make install` - 30 seconds (no go vet)
- **Developer**: `make build` - 2+ minutes (full validation)

### Initial Setup:
- **User**: `guild init` - Creates Elena instantly
- **Developer**: Can customize agents and configurations

### Quality Checks:
- **User**: Skip validation for speed
- **Developer**: Full validation for code quality

### Testing:
- **User**: Not required
- **Developer**: Comprehensive test suite

## When to Use Which Workflow

### Use the User Workflow When:
- You want to start using Guild immediately
- You're evaluating Guild's capabilities
- You're an end user, not contributing code
- Speed is more important than validation

### Use the Developer Workflow When:
- You're contributing to Guild
- You need to ensure code quality
- You're debugging or testing changes
- You're preparing a release

## Migration Between Workflows

Users can always switch to developer mode:
```bash
# Switch from user to developer workflow
make clean
make build  # Full validation build
make test   # Run tests
```

Developers can use user commands for quick testing:
```bash
# Quick test without validation
make install
guild init test-project
```

## Summary

Guild's dual workflow system ensures both:
1. **Fast onboarding** for users who want immediate productivity
2. **Code quality** for developers contributing to the project

Choose the workflow that matches your needs!