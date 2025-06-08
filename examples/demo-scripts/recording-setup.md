# Demo Recording Setup Guide

**Goal**: Ensure smooth, professional demo recording that showcases Guild's capabilities with broadcast-quality visuals

## Environment Configuration

### Terminal Setup

```bash
# Clean, professional prompt
export PS1="$ "
export TERM=xterm-256color

# Optimal recording dimensions (16:9 aspect ratio)
resize -s 40 120  # 40 rows x 120 columns

# Ensure UTF-8 support for medieval characters
export LANG=en_US.UTF-8
export LC_ALL=en_US.UTF-8

# Set timezone for consistent timestamps
export TZ=UTC
```

### Theme Configuration

```bash
# Apply monokai theme with medieval purple accents
# Colors optimized for syntax highlighting visibility:
# - Purple accents: #6B46C1 (medieval theme)
# - Syntax highlighting: High contrast monokai palette
# - Background: #272822 (dark monokai)
# - Text: #F8F8F2 (light monokai)

# Font settings for recording
# - Family: JetBrains Mono or Fira Code (programming ligatures)
# - Size: 14-16pt (readable in recordings)
# - Weight: Medium (clear but not too bold)
```

### Guild Configuration Optimization

```yaml
# .guild/config/demo.yaml
demo:
  mode: true
  optimizations:
    # Speed up responses for smooth demos
    fast_init: true
    skip_health_checks: false  # Keep for reliability
    cache_responses: true

    # Visual enhancements
    rich_rendering: true
    syntax_highlighting: true
    medieval_theme: true

    # Performance tuning
    response_timeout: 30s
    max_concurrent_agents: 6

  # Demo-specific settings
  agent_status_indicators: true
  progress_animations: true
  typing_simulation: false  # Direct output for demos
```

## Pre-Recording Checklist

### Environment Verification

- [ ] Terminal size: 40x120 (verify with `tput lines cols`)
- [ ] Theme applied and visible
- [ ] Guild CLI compiled and accessible
- [ ] All demo commission files in place
- [ ] Agent configurations validated
- [ ] Network connectivity stable
- [ ] Recording software configured

### Content Preparation

- [ ] All demo scripts reviewed and timed
- [ ] Commission files contain rich markdown content
- [ ] Agent response samples validated
- [ ] Error recovery procedures practiced
- [ ] Demo data cleaned and consistent

### Technical Validation

```bash
# Verify Guild installation
guild version
guild agents list
guild config validate

# Test core functionality
guild init --dry-run
guild commission validate .guild/commissions/e-commerce-platform.md

# Verify rich content rendering
guild chat --test-mode --rich-content

# Check agent responsiveness
guild agents ping --all --timeout 10s
```

### Visual Quality Check

- [ ] Syntax highlighting visible and correct
- [ ] Markdown rendering professional
- [ ] Agent status indicators working
- [ ] No UI glitches or rendering errors
- [ ] Medieval theme consistent throughout
- [ ] Code blocks properly formatted
- [ ] Tables and lists render correctly

## Performance Optimization

### Response Time Optimization

```bash
# Pre-warm agent connections
guild agents warm-up --all

# Cache common responses for demo consistency
guild config cache enable
guild config cache-responses \
  --pattern="@service-architect.*API.*" \
  --pattern="@frontend-specialist.*React.*" \
  --pattern="@backend-specialist.*Go.*"

# Optimize LLM call performance
export ANTHROPIC_API_POOL_SIZE=3
export OPENAI_API_POOL_SIZE=3
```

### Demo Mode Configuration

```bash
# Enable demo optimizations
guild config demo-mode enable

# Fast initialization (skip non-essential checks)
guild config fast-init true

# Reliable connectivity settings
guild config retry-attempts 3
guild config timeout-increase 50%

# Memory optimization for smooth performance
guild config memory-limit 2GB
guild config gc-frequency reduced
```

## Recording Commands & Workflow

### Primary Recording Setup

```bash
# Start asciinema recording with optimal settings
asciinema rec \
  --title "Guild Framework Demo - $(date +%Y-%m-%d)" \
  --command /bin/bash \
  --overwrite \
  demo-guild-$(date +%Y%m%d-%H%M).cast

# Alternative: Use script for more control
script -t 2>demo-timing.txt demo-session.txt
```

### Post-Recording Processing

```bash
# Convert to optimized GIF with medieval theme
agg \
  --theme monokai \
  --font-size 14 \
  --line-height 1.2 \
  --cols 120 \
  --rows 40 \
  demo-guild-$(date +%Y%m%d).cast \
  demo-guild-$(date +%Y%m%d).gif

# Create multiple versions for different use cases
# Fast overview (1.5x speed)
agg --theme monokai --speed 1.5 \
  demo.cast demo-fast.gif

# Detailed walkthrough (0.8x speed)
agg --theme monokai --speed 0.8 \
  demo.cast demo-detailed.gif

# Social media version (square format)
agg --theme monokai --cols 80 --rows 80 \
  demo.cast demo-social.gif
```

### Video Export Options

```bash
# High-quality MP4 for presentations
svg-term --cast demo.cast --out demo.svg --window
# Then convert SVG to MP4 using external tools

# WebM for web embedding
# Use OBS Studio or similar for high-quality video recording
```

## Timing Guidelines & Best Practices

### Pacing Standards

- **Pause before typing**: 0.5-1 seconds (builds anticipation)
- **Typing speed**: 80-100 WPM (realistic but not slow)
- **Wait for responses**: Show agent "thinking" indicators
- **Transition delays**: 2-3 seconds between major sections
- **Reading time**: Allow 2 seconds per line for viewers

### Command Execution Flow

```bash
# Example timing pattern:
echo "Starting authentication demo..."  # 1s pause
sleep 1
guild chat --campaign e-commerce      # 2s for command
sleep 2
# Type in chat with natural pauses...
```

### Error Recovery Strategies

- **Agent timeout**: "Guild agents provide thoughtful responses - let's wait for quality"
- **Command error**: Have backup commands ready and tested
- **Network issues**: Pre-cached responses for critical demos
- **UI glitches**: Practice recovery commands and restart procedures

## Visual Quality Standards

### Code Presentation

- All code must have proper syntax highlighting
- Indentation must be consistent and visible
- Comments should be meaningful and professional
- Variable names should be realistic and clear

### Markdown Rendering

- Headers must have clear hierarchy (H1 > H2 > H3)
- Lists must be properly formatted and indented
- Tables must align correctly and be readable
- Emphasis (bold/italic) must be visible and consistent

### Professional Appearance

- No debug messages or error logs visible
- Clean terminal history (clear before recording)
- Consistent color scheme throughout
- Professional naming conventions
- No placeholder or "TODO" content

### Medieval Theme Integration

- Guild-specific terminology consistently used
- Purple accent color (#6B46C1) for highlights
- Medieval emojis and metaphors where appropriate
- Professional balance (not overly themed)

## Equipment & Software Requirements

### Recording Software Options

1. **asciinema** (Recommended for terminal)
   - Lightweight and high-quality
   - Easy post-processing with agg
   - Perfect text rendering

2. **OBS Studio** (For full-screen recording)
   - Professional video features
   - Multiple scene support
   - Advanced audio mixing

3. **QuickTime Player** (macOS simple option)
   - Built-in screen recording
   - Good for quick recordings
   - Limited editing capabilities

### Hardware Recommendations

- **Display**: 1920x1080 minimum resolution
- **Memory**: 8GB+ RAM for smooth performance
- **CPU**: Multi-core for parallel agent processing
- **Network**: Stable high-speed internet
- **Audio**: Quality microphone for narration

### Software Dependencies

```bash
# Install recording tools
brew install asciinema       # Terminal recording
npm install -g agg           # GIF conversion
brew install ffmpeg          # Video processing

# Verify installations
asciinema --version
agg --version
ffmpeg -version
```

## Demo-Specific Optimizations

### Quick Demo (2 minutes)

- Pre-load commission in memory
- Cache service-architect response
- Minimize typing delays
- Focus on visual impact

### Full Workflow (8 minutes)

- Warm up all agents beforehand
- Pre-stage kanban directory structure
- Cache complex technical responses
- Practice transitions between sections

### Multi-Agent Coordination (5 minutes)

- Test parallel agent responses
- Verify status indicators work
- Practice conflict resolution scenario
- Ensure cross-references display properly

## Quality Assurance Checklist

### Pre-Recording Final Check

- [ ] All scripts timed and practiced
- [ ] Demo environment clean and configured
- [ ] Recording software tested and working
- [ ] Backup plans for common issues prepared
- [ ] Rich content renders beautifully
- [ ] Medieval theme consistent but professional
- [ ] All agents responsive and cached
- [ ] Network connection stable
- [ ] Audio levels tested (if narrating)
- [ ] Lighting adequate (if showing screen)

### Post-Recording Validation

- [ ] Recording quality meets standards
- [ ] Audio clear and professional
- [ ] Visual elements render correctly
- [ ] No embarrassing errors or typos
- [ ] Timing appropriate for target audience
- [ ] File sizes reasonable for distribution
- [ ] Multiple formats created as needed

## Troubleshooting Common Issues

### Agent Response Delays

```bash
# If agents are slow:
guild agents restart --all
guild cache warm-up
guild config timeout increase

# Check API rate limits
guild status --detailed
```

### Rendering Problems

```bash
# If rich content doesn't render:
guild config rich-rendering verify
guild test markdown-rendering
guild test syntax-highlighting

# Reset terminal if needed
reset
source ~/.bashrc
```

### Recording Quality Issues

```bash
# If terminal looks wrong:
tput reset
resize -s 40 120
export TERM=xterm-256color

# If colors are wrong:
guild config theme reload
guild test color-output
```

## Distribution & Marketing

### File Organization

```
demo-recordings/
├── raw/                          # Original recordings
│   ├── demo-YYYYMMDD-HHMM.cast
│   └── demo-YYYYMMDD-HHMM.timing
├── processed/                    # Edited versions
│   ├── quick-demo.gif
│   ├── full-workflow.mp4
│   └── coordination-demo.webm
├── thumbnails/                   # Preview images
└── distribution/                 # Final versions
    ├── social-media/
    ├── presentations/
    └── documentation/
```

### Version Control

- Tag recordings with dates and versions
- Keep raw recordings for re-processing
- Document any post-processing applied
- Maintain script versions used for each recording

This comprehensive recording setup ensures professional, impressive demos that showcase Guild's capabilities effectively while maintaining broadcast quality standards.
