# TUI Testing Best Practices with teatest

## Problem: Terminal Corruption

When running Bubble Tea tests with teatest, the terminal can become corrupted if:
- Tests are interrupted with Ctrl+C
- Tests fail to properly clean up
- Tests exit abnormally

This causes:
- Box drawing characters to break in the Makefile output
- Ctrl+C to stop working properly
- Terminal to remain in alternate screen mode

## Solution 1: Use the TeaTestHelper

We've created a helper that ensures proper cleanup:

```go
import (
    "testing"
    "github.com/guild-ventures/guild-core/internal/testutil"
)

func TestMyTUIComponent(t *testing.T) {
    helper := testutil.NewTeaTestHelper(t)
    
    model := NewMyModel()
    tm := helper.RunModel(model)
    
    // Your test code here
    tm.Send("hello")
    
    // Always use safe quit
    testutil.SendQuit(t, tm)
}
```

## Solution 2: Use make test-teatest

Run TUI tests with proper terminal protection:

```bash
make test-teatest
```

This target:
- Sets up a trap to restore terminal on exit
- Filters output to show only relevant test results
- Ensures terminal is reset after tests complete

## Solution 3: Fix Corrupted Terminal

If your terminal is already corrupted:

```bash
make fix-terminal
```

Or manually:

```bash
# Quick fix
./scripts/fix-terminal.sh

# Manual fix
printf '\033c'  # Full reset
reset           # Reset terminal
stty sane       # Restore terminal settings
```

## Best Practices

1. **Always use cleanup helpers** in your tests
2. **Never call os.Exit()** directly in tests
3. **Use tea.Quit** instead of tea.Kill
4. **Set timeouts** to prevent hanging tests
5. **Test in CI** with NO_COLOR=1 to avoid ANSI issues

## Example: Safe TUI Test

```go
func TestCommandPalette(t *testing.T) {
    helper := testutil.NewTeaTestHelper(t)
    
    // Create your model
    commands := []Command{
        {Name: "test", Description: "Test command"},
    }
    model := NewCommandPalette(commands)
    
    // Run with helper
    tm := helper.RunModel(model)
    
    // Test interactions
    tm.Send("test")
    tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
    
    // Safe cleanup
    testutil.SendQuit(t, tm)
}
```

## UTF-8 Box Drawing

The Makefile now uses UTF-8 box drawing characters that are more resilient:
- `┌─┐` instead of ASCII boxes
- Single-width glyphs that align properly
- Work with most modern terminals

## Terminal Reset in Build Tool

The build tool now:
- Resets terminal before starting (`fmt.Print("\033c")`)
- Restores terminal on exit (defer)
- Uses UTF-8 box drawing for better compatibility

## Dashboard Feature

View project status without risk of corruption:

```bash
make dashboard
```

This shows:
- Build status
- Test results
- Quick commands
- All with proper terminal handling