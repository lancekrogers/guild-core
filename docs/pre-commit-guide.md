# Pre-commit Hooks Guide for Guild-Core

This guide explains how to use pre-commit hooks to maintain code quality in the Guild-Core project.

## 🚀 Quick Start

### Installation

```bash
# Option 1: Use the setup script (recommended)
./scripts/setup-pre-commit.sh

# Option 2: Use Makefile
make pre-commit-install

# Option 3: Manual installation
brew install pre-commit  # macOS
pip install pre-commit   # Python

# Then install the hooks
pre-commit install
```

## 🛡️ What Gets Checked

Pre-commit runs the following checks before each commit:

### Go Code Quality
- **go fmt** - Ensures consistent code formatting
- **go vet** - Catches common Go errors
- **golangci-lint** - Comprehensive linting with auto-fixes
- **goimports-reviser** - Organizes imports properly

### General Code Quality
- **Trailing whitespace** - Removes unnecessary whitespace
- **End of file fixer** - Ensures files end with newline
- **Check YAML** - Validates YAML syntax
- **Check large files** - Prevents accidental large file commits
- **Detect private keys** - Security check for credentials
- **Mixed line endings** - Enforces consistent line endings (LF)

### Guild-Specific Checks
- **No development artifacts** - Prevents .disabled, .old, .wip files
- **Build verification** - Ensures code compiles
- **Short tests** - Runs quick unit tests

### Security
- **detect-secrets** - Scans for potential secrets or API keys

### Documentation
- **markdownlint** - Ensures consistent markdown formatting

## 📋 Usage

### Running Pre-commit

Pre-commit runs automatically on `git commit`, but you can also run it manually:

```bash
# Run on all files
make pre-commit

# Or directly
pre-commit run --all-files

# Run on specific files
pre-commit run --files path/to/file.go

# Run specific hooks
pre-commit run go-fmt --all-files
pre-commit run golangci-lint --all-files
```

### Skipping Pre-commit (Emergency Only)

If you need to commit without running pre-commit:

```bash
git commit --no-verify -m "Emergency fix"
```

⚠️ **Warning**: Only use this in emergencies. Always follow up with proper fixes.

### Updating Pre-commit

Keep pre-commit hooks up to date:

```bash
# Update hook versions
make pre-commit-update

# Or directly
pre-commit autoupdate
```

## 🔧 Configuration

Pre-commit is configured in `.pre-commit-config.yaml`. Key settings:

- **golangci-lint**: Configured in `.golangci.yml`
- **markdownlint**: Configured in `.markdownlint.json`
- **secrets**: Baseline in `.secrets.baseline`

## 🐛 Troubleshooting

### Common Issues

1. **"pre-commit not found"**
   ```bash
   make pre-commit-install
   ```

2. **"golangci-lint not found"**
   ```bash
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   ```

3. **"Build failed" in pre-commit**
   - Run `go mod tidy`
   - Check for compilation errors: `go build ./...`

4. **Markdown lint failures**
   - Most rules are relaxed in `.markdownlint.json`
   - Fix remaining issues or update config if too strict

5. **Secret detection false positives**
   - Run `detect-secrets scan --baseline .secrets.baseline`
   - Review and update baseline if legitimate

### Disabling Specific Checks

If a check is consistently problematic:

1. **For a specific file**: Add to `.pre-commit-config.yaml` exclude patterns
2. **For a specific line**: Use inline comments (e.g., `//nolint` for Go)
3. **Globally**: Modify the hook configuration

## 📊 CI Integration

Pre-commit checks are also run in CI to ensure no commits bypass local hooks. The same checks run in:

- Pull request validation
- Main branch protection
- Nightly quality checks

## 🎯 Benefits

1. **Consistent Code Style**: Automatic formatting ensures uniform code
2. **Early Error Detection**: Catch issues before code review
3. **Security**: Prevent accidental credential commits
4. **Time Saving**: No manual formatting or basic error fixing
5. **Code Quality**: Maintain high standards automatically

## 📚 Additional Resources

- [Pre-commit documentation](https://pre-commit.com/)
- [golangci-lint configuration](https://golangci-lint.run/usage/configuration/)
- [Conventional Commits](https://www.conventionalcommits.org/) (optional)
