#!/bin/bash
# Fix git submodule configuration for guild-core
# This script fixes the "fatal: this operation must be run in a work tree" error
# that occurs when pre-commit hooks reset the git configuration

set -e

CONFIG_FILE="../.git/modules/guild-core/config"

echo "Fixing git submodule configuration..."

# Check if we're in the guild-core directory
if [ ! -f ".git" ]; then
    echo "Error: This script must be run from the guild-core directory"
    exit 1
fi

# Check if config file exists
if [ ! -f "$CONFIG_FILE" ]; then
    echo "Error: Git config file not found at $CONFIG_FILE"
    exit 1
fi

# Set core.bare to false
echo "Setting core.bare = false..."
git config --file="$CONFIG_FILE" core.bare false

# Set worktree path
echo "Setting core.worktree = ../../../guild-core..."
git config --file="$CONFIG_FILE" core.worktree ../../../guild-core

# Verify configuration
echo "Verifying git status works..."
if git status --porcelain > /dev/null 2>&1; then
    echo "✅ Git configuration fixed successfully!"
    echo "Git status is now working properly."
else
    echo "❌ Git status still not working. Manual intervention may be required."
    exit 1
fi