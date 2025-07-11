# Terminal Package

The `terminal` package provides comprehensive cross-platform terminal capability detection and rendering strategies for the Guild framework. It ensures optimal user experience across different terminal environments through intelligent feature detection and graceful degradation.

## Features

- **Cross-Platform Support**: Works seamlessly on Windows, macOS, and Linux
- **Capability Detection**: Automatically detects terminal features like color support, Unicode, mouse tracking, and more
- **Profile Management**: Pre-configured profiles for popular terminals (iTerm2, Windows Terminal, VS Code, etc.)
- **Graceful Degradation**: Automatically falls back to simpler rendering when features aren't available
- **Performance Optimized**: Caches detection results to avoid repeated system calls
- **Testing Support**: Includes a full terminal emulator for comprehensive testing

## Quick Start

```go
import "github.com/lancekrogers/guild/pkg/terminal"

// Detect current terminal capabilities
detector := terminal.NewDetector()
caps := detector.Detect()

// Select appropriate renderer based on capabilities
renderer := terminal.SelectRenderer(caps)

// Render content based on terminal capabilities
if caps.Unicode {
    renderer.DrawBox("✨ Unicode Supported! ✨")
} else {
    renderer.DrawBox("Unicode Supported!")
}
```

## Core Components

### 1. Detector (`detector.go`)

The detector identifies terminal capabilities through environment analysis:

```go
detector := terminal.NewDetector()
caps := detector.Detect()

// Check specific capabilities
if caps.SupportsColor() {
    // Use colored output
}

if caps.SupportsMouse() {
    // Enable mouse tracking
}
```

### 2. Capabilities (`capabilities.go`)

Capabilities represent what a terminal can do:

```go
type Capabilities struct {
    Colors          ColorSupport  // NoColor, Basic16, ANSI256, TrueColor24Bit
    Unicode         bool          // Unicode character support
    Mouse           bool          // Mouse tracking support
    Size            bool          // Terminal size detection
    TrueColor       bool          // 24-bit color support
    Hyperlinks      bool          // Clickable links
    Images          bool          // Image rendering (iTerm2, Kitty)
    CursorShape     bool          // Cursor customization
    AlternateScreen bool          // Alternate screen buffer
    // ... and more
}
```

### 3. Renderers (`renderers.go`)

Three rendering strategies based on capabilities:

- **RichRenderer**: Full features for modern terminals
- **StandardRenderer**: ANSI colors and basic Unicode
- **FallbackRenderer**: ASCII-only for limited environments

```go
renderer := terminal.SelectRenderer(caps)

// Automatic selection based on capabilities
// Rich terminals get RichRenderer
// Basic terminals get StandardRenderer  
// Minimal terminals get FallbackRenderer
```

### 4. Profiles (`profiles.go`)

Pre-configured terminal profiles for optimal experience:

```go
pd := terminal.NewProfileDetector()

// Auto-detect terminal
profile, _ := pd.Detect(context.Background())

// Or apply specific profile
pd.ApplyProfile("iterm2")

// List available profiles
profiles := pd.ListProfiles()
```

Supported profiles include:
- iTerm2 (macOS)
- Windows Terminal
- VS Code Integrated Terminal
- Kitty
- Alacritty
- GNOME Terminal
- Konsole
- tmux/screen
- SSH sessions
- CI environments
- And more...

### 5. Fallbacks (`fallbacks.go`)

Intelligent fallback strategies for graceful degradation:

```go
fp := terminal.NewFallbackProvider()

// Register custom fallbacks
fp.RegisterStrategy(terminal.FallbackStrategy{
    Name:        "custom-icon",
    From:        "🚀",
    To:          "[>]",
    When:        func(caps Capabilities) bool { return !caps.Unicode },
    Priority:    100,
})

// Apply fallbacks to content
content := fp.Apply("✓ Success with 🚀 rocket!", caps)
// Returns: "[OK] Success with [>] rocket!" on ASCII-only terminals
```

### 6. Terminal Emulator (`emulator.go`)

Full terminal emulator for testing:

```go
em := terminal.NewEmulator(80, 24)

// Process ANSI sequences
em.ProcessInput("\x1b[31mRed Text\x1b[0m")

// Verify output
cell := em.GetCell(0, 0)
if cell.FG != terminal.Red {
    t.Error("Expected red foreground")
}
```

## Feature Detection

The package detects various terminal features:

### Color Support
- No color (dumb terminals)
- Basic 16 colors
- 256 colors (ANSI)
- True color (24-bit RGB)

### Input Features
- Mouse tracking (X11, SGR protocols)
- Bracketed paste mode
- Keyboard protocols

### Display Features  
- Unicode rendering
- Box drawing characters
- Emoji support
- Hyperlink support
- Image protocols (Sixel, iTerm2, Kitty)

### Advanced Features
- Alternate screen buffer
- Cursor shape control
- Window title setting
- Notifications

## Environment Variables

The package respects standard environment variables:

- `TERM`: Terminal type
- `COLORTERM`: Color terminal indicator
- `TERM_PROGRAM`: Terminal application name
- `NO_COLOR`: Disable color output
- `FORCE_COLOR`: Force color output
- `CI`: Continuous integration environment
- `GUILD_TERMINAL_PROFILE`: Force specific profile

## Usage Examples

### Basic Terminal UI

```go
// Create a terminal UI component
func CreateUI() {
    detector := terminal.NewDetector()
    caps := detector.Detect()
    renderer := terminal.SelectRenderer(caps)
    
    // Draw a box with appropriate characters
    renderer.DrawBox("Welcome to Guild")
    
    // Use colors if available
    if caps.SupportsColor() {
        renderer.SetColor(terminal.Green)
        renderer.Print("✓ Success")
        renderer.Reset()
    }
}
```

### Progress Bar with Fallbacks

```go
func ShowProgress(percent int) {
    caps := terminal.DefaultDetector.Detect()
    
    if caps.Unicode {
        // Rich progress bar
        fmt.Printf("Progress: [%s%s] %d%%\r",
            strings.Repeat("█", percent/5),
            strings.Repeat("░", 20-percent/5),
            percent)
    } else {
        // ASCII fallback
        fmt.Printf("Progress: [%s%s] %d%%\r",
            strings.Repeat("#", percent/5),
            strings.Repeat("-", 20-percent/5),
            percent)
    }
}
```

### Terminal-Aware Logging

```go
logger := logging.NewLogger(logging.Config{
    // Adapt output based on terminal
    Formatter: func(entry logging.Entry) string {
        caps := terminal.DefaultDetector.Detect()
        
        if caps.SupportsRichUI() {
            // Rich formatting with colors and icons
            return formatRich(entry)
        } else if caps.SupportsColor() {
            // Basic colors only
            return formatColor(entry)
        } else {
            // Plain text
            return formatPlain(entry)
        }
    },
})
```

## Testing

The package includes comprehensive tests for all components:

```bash
go test ./pkg/terminal/...
```

Test coverage includes:
- Platform-specific detection
- Profile matching
- Renderer selection
- Fallback strategies
- Terminal emulation
- ANSI sequence parsing

## Best Practices

1. **Cache Detection Results**: Terminal capabilities don't change during execution
   ```go
   var caps = terminal.DefaultDetector.Detect() // Global cache
   ```

2. **Always Provide Fallbacks**: Never assume capabilities
   ```go
   if caps.Unicode {
       return "✓"
   }
   return "[OK]"
   ```

3. **Test Multiple Environments**: Use the emulator for testing
   ```go
   em := terminal.NewEmulator(80, 24)
   em.SetCapabilities(terminal.Capabilities{Colors: terminal.NoColor})
   // Test rendering in limited environment
   ```

4. **Respect User Preferences**: Honor NO_COLOR and FORCE_COLOR
   ```go
   if os.Getenv("NO_COLOR") != "" {
       // Disable all color output
   }
   ```

## Architecture

The package follows a clean architecture with clear separation of concerns:

```
terminal/
├── detector.go      # Capability detection
├── capabilities.go  # Capability definitions
├── renderers.go     # Rendering strategies
├── profiles.go      # Terminal profiles
├── fallbacks.go     # Fallback strategies
├── emulator.go      # Testing support
├── features/        # Feature-specific detection
│   ├── colors.go
│   ├── unicode.go
│   ├── mouse.go
│   ├── size.go
│   └── input.go
└── profiles/        # Platform-specific profiles
    ├── windows.go
    ├── macos.go
    ├── linux.go
    └── minimal.go
```

## Performance Considerations

- Detection results are cached using `sync.Once`
- No repeated environment variable lookups
- Minimal allocations in hot paths
- Profile matching uses early exit strategies

## Contributing

When adding new features:

1. Update the `Capabilities` struct
2. Add detection logic to appropriate feature module
3. Update relevant terminal profiles
4. Add fallback strategies if needed
5. Include comprehensive tests
6. Document the feature in this README

## License

This package is part of the Guild framework and follows the same license terms.