#!/bin/bash

# Test script for Guild Chat keybindings

echo "Guild Chat Keybinding Test"
echo "========================="
echo ""
echo "This will test the chat interface. Make sure the guild daemon is running:"
echo "  ./bin/guild daemon &"
echo ""
echo "Key combinations to test:"
echo "1. Shift+Enter - Should insert a newline (multiline input)"
echo "2. Ctrl+Q, Esc, or Ctrl+D - Should exit chat"
echo "3. /exit, /quit, or /q - Should exit chat"
echo "4. Ctrl+Shift+C - Copy text"
echo "5. Ctrl+Shift+V - Paste text"
echo "6. Ctrl+Alt+V - Toggle vim mode"
echo "7. Ctrl+H - Show help"
echo ""
echo "Press Enter to start the chat interface..."
read

# Start the chat
./bin/guild chat --campaign "test"