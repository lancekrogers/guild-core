# Pre-commit Setup for Guild-Core Submodule

Since `guild-core` is a git submodule, pre-commit hooks need to be installed in the parent repository.

## Installation Steps

1. **From the parent repository** (guild-framework):
   ```bash
   cd /Users/lancerogers/Dev/AI/guild-framework
   pre-commit install
   ```

2. **Verify the hooks are installed**:
   ```bash
   ls -la .git/hooks/pre-commit
   ```

3. **Test the hooks** by making a commit:
   ```bash
   git add .
   git commit -m "test: pre-commit hooks"
   ```

## Alternative: Standalone Repository Setup

If you want pre-commit to work directly in guild-core:

1. **Initialize guild-core as a standalone repository**:
   ```bash
   cd guild-core
   rm .git  # Remove submodule pointer
   git init
   git add .
   git commit -m "Initial commit"
   pre-commit install
   ```

2. **Or clone guild-core separately** (not as a submodule):
   ```bash
   git clone <guild-core-repo-url> guild-core-standalone
   cd guild-core-standalone
   pre-commit install
   ```

## Manual Pre-commit Runs

Even without hooks installed, you can still run pre-commit manually:

```bash
# Run all checks
pre-commit run --all-files

# Run specific hooks
pre-commit run go-fmt --all-files
pre-commit run golangci-lint --all-files
```

## Troubleshooting

If pre-commit doesn't run on commit:
1. Check you're committing from the parent repository (where .git is a directory)
2. Verify hooks are installed: `ls -la .git/hooks/`
3. Make sure the pre-commit config file exists in the repository
4. Try reinstalling: `pre-commit uninstall && pre-commit install`