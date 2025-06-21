# TeaTest Terminal Cleanup Guide

## Problem

When running teatest tests, the terminal can become corrupted if:
- Tests are interrupted with Ctrl-C
- Tests fail to call `WaitFinished` properly
- Tests don't properly quit the Bubble Tea program
- Terminal state isn't restored after alternate screen mode

This results in:
- Box drawing characters appearing broken (└ becomes â)
- Ctrl-C not working properly in the terminal
- Need to open new terminal tabs

## Solution

Use the provided test helpers that ensure proper terminal cleanup.

## Migration Guide

### Step 1: Import the test helpers

```go
import (
    "github.com/guild-ventures/guild-core/internal/testing"
)
```

### Step 2: Replace direct teatest usage

**Before:**
```go
func TestMyComponent(t *testing.T) {
    model := createTestModel()
    
    tm := teatest.NewTestModel(
        t,
        model,
        teatest.WithInitialTermSize(80, 24),
    )
    
    // Test code...
    
    tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
    tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}
```

**After:**
```go
func TestMyComponent(t *testing.T) {
    helper := testing.NewTeaTestHelper(t)
    model := createTestModel()
    
    helper.RunTeaTest(model, func(tm *teatest.TestModel) {
        // Test code...
        
        // No need to manually send quit or call WaitFinished
        // Cleanup happens automatically
    })
}
```

### Step 3: Update test patterns

1. **Always ensure models can quit gracefully:**
   ```go
   func (m *MyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
       switch msg := msg.(type) {
       case tea.KeyMsg:
           switch msg.String() {
           case "ctrl+c", "q", "esc":
               return m, tea.Quit
           }
       }
       // ... rest of update logic
   }
   ```

2. **Use helper functions for common operations:**
   ```go
   // Wait for specific output
   testing.WaitForOutput(t, tm, func(b []byte) bool {
       return bytes.Contains(b, []byte("Ready"))
   }, 2*time.Second)
   
   // Send input with processing time
   testing.SendAndWait(tm, tea.KeyMsg{Type: tea.KeyEnter}, 100*time.Millisecond)
   ```

3. **For tests that need to verify quit behavior:**
   ```go
   helper.RunTeaTest(model, func(tm *teatest.TestModel) {
       // Do test actions...
       
       // Send quit command
       tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
       
       // The helper will ensure proper cleanup even if quit fails
   })
   ```

## Files to Update

The following files need to be updated to use the new test helpers:

1. `internal/chat/guild_selector_test.go`
   - Update all integration tests starting at line 928
   - Replace direct teatest usage with helper

2. `internal/chat/commands/palette_test.go`
   - Update all integration tests starting at line 259
   - Ensure proper quit handling in test model

3. `internal/ui/init/init_tui_test.go`
   - Update all integration tests starting at line 437
   - Use helper for cleanup

## Manual Terminal Recovery

If your terminal is already corrupted:

1. **Quick fix:**
   ```bash
   reset
   ```

2. **Alternative fix:**
   ```bash
   stty sane
   tput reset
   ```

3. **Clear screen and reset:**
   ```bash
   printf '\033[?1049l\033[?25h\033[0m\033[H\033[2J'
   ```

## Best Practices

1. **Always use the test helper** for teatest-based tests
2. **Ensure all models can quit** via Ctrl-C, Esc, or 'q'
3. **Don't rely on defer alone** - signal interrupts can bypass defers
4. **Test your tests** - run them and interrupt with Ctrl-C to ensure cleanup works
5. **Add timeout protection** - don't let tests hang indefinitely

## Example: Complete Test File

```go
package mypackage_test

import (
    "bytes"
    "testing"
    "time"
    
    tea "github.com/charmbracelet/bubbletea"
    "github.com/guild-ventures/guild-core/internal/testing"
)

func TestMyFeature(t *testing.T) {
    helper := testing.NewTeaTestHelper(t)
    
    t.Run("basic functionality", func(t *testing.T) {
        model := NewMyModel()
        
        helper.RunTeaTest(model, func(tm *teatest.TestModel) {
            // Wait for initial render
            testing.WaitForOutput(t, tm, func(b []byte) bool {
                return bytes.Contains(b, []byte("Welcome"))
            }, 2*time.Second)
            
            // Test navigation
            testing.SendAndWait(tm, tea.KeyMsg{Type: tea.KeyDown}, 50*time.Millisecond)
            testing.SendAndWait(tm, tea.KeyMsg{Type: tea.KeyEnter}, 50*time.Millisecond)
            
            // Verify result
            testing.WaitForOutput(t, tm, func(b []byte) bool {
                return bytes.Contains(b, []byte("Success"))
            }, time.Second)
        })
    })
}
```

## Debugging Terminal Issues

If you're still experiencing issues:

1. **Check for goroutine leaks:**
   ```go
   defer goleak.VerifyNone(t)
   ```

2. **Add debug logging:**
   ```go
   t.Logf("Terminal state before test: %v", os.Getenv("TERM"))
   ```

3. **Use verbose test output:**
   ```bash
   go test -v ./internal/chat/... 2>&1 | tee test.log
   ```

4. **Check for panic recovery:**
   Ensure tests don't swallow panics that prevent cleanup

## Prevention

1. **Run tests in CI** with terminal emulation to catch issues early
2. **Use timeouts** on all teatest operations
3. **Document quit keys** in test models
4. **Review test output** for escape sequences or corruption