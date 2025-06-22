#!/bin/bash

# Test script to verify Elena welcome message in chat

echo "Testing Elena Welcome Message Integration..."

# Create test directory
TEST_DIR="/tmp/guild-elena-test-$(date +%s)"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

echo "1. Initializing guild project with campaign..."
../../../bin/guild init --quick

echo -e "\n2. Starting chat with campaign (this should show Elena's welcome)..."
echo "Note: You should see Elena's personalized welcome message instead of the generic one."
echo "Press Ctrl+C to exit chat after viewing the welcome message."
echo -e "\nStarting in 3 seconds...\n"
sleep 3

# Start chat - this should show Elena's welcome
../../../bin/guild chat --campaign "test-campaign"