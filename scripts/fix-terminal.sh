#!/bin/bash
# Fix terminal after teatest corruption

echo "🔧 Fixing terminal state..."

# Method 1: Reset terminal
reset

# Method 2: Use tput
tput cnorm  # Show cursor
tput rmcup  # Exit alternate screen
tput sgr0   # Reset all attributes

# Method 3: stty sane
stty sane

# Method 4: Direct ANSI escape sequences
printf '\033[?1049l'  # Exit alternate buffer
printf '\033[?25h'    # Show cursor
printf '\033[0m'      # Reset colors

echo "✅ Terminal should be restored!"
echo ""
echo "If still having issues, try:"
echo "  1. Press Ctrl+L to clear screen"
echo "  2. Type 'reset' and press Enter"
echo "  3. Close and reopen terminal tab"