# Guild Demo Troubleshooting Guide

This guide helps resolve common issues during Guild framework demonstrations and ensures smooth demo experiences.

## 🚨 Emergency Quick Fixes

### Demo Day Checklist (30 seconds)
```bash
# Quick pre-demo validation
./guild demo-check --verbose --api-keys --performance

# If issues found, try these quick fixes:
export COLORTERM=truecolor
./guild init --quiet --name demo-guild
rm -rf .guild && ./guild init
```

## 📋 Common Issues and Solutions

### 1. gRPC Connection Failed

**Symptom**: `Failed to connect to localhost:50051` or similar connection errors

**Immediate Solutions**:
```bash
# Check if port is in use
lsof -i :50051

# Kill conflicting processes
sudo lsof -ti:50051 | xargs kill -9

# Start server manually with different port
./guild daemon --grpc-address localhost:50052 &

# Use alternative port in chat
./guild chat --grpc-address localhost:50052
```

**Root Causes**:
- Previous guild process still running
- Port blocked by firewall
- Network configuration issues

**Prevention**:
- Always run `./guild daemon stop` before demos
- Use `--grpc-address` flag for custom ports
- Check firewall settings in advance

### 2. Agents Not Responding

**Symptom**: No response after sending `@agent` messages, or agents appear offline

**Immediate Solutions**:
```bash
# Verify agents are configured
./guild agents list

# Check API keys are set
echo $OPENAI_API_KEY
echo $ANTHROPIC_API_KEY

# Try with mock provider for demo
./guild chat --provider mock

# Restart with verbose logging
./guild chat --verbose --debug
```

**Root Causes**:
- Missing or invalid API keys
- Network connectivity issues
- Agent configuration errors
- Provider service outages

**Prevention**:
- Set backup API keys for multiple providers
- Test agent responses before demos
- Have mock provider ready as fallback
- Monitor provider service status

### 3. Visual Rendering Issues

**Symptom**: No colors, broken formatting, missing unicode characters

**Immediate Solutions**:
```bash
# Check terminal capabilities
echo $COLORTERM $TERM $TERM_PROGRAM

# Force color support
export COLORTERM=truecolor
export TERM=xterm-256color

# Try different terminal
# macOS: Use iTerm2 instead of Terminal.app
# Windows: Use Windows Terminal instead of CMD
# Linux: Use modern terminal emulator

# Disable colors as fallback
export NO_COLOR=1
./guild chat --no-color
```

**Terminal-Specific Fixes**:

| Terminal | Color Fix | Unicode Fix |
|----------|-----------|-------------|
| iTerm2 | Set "Report Terminal Type" to xterm-256color | Enable Unicode normalization |
| Terminal.app | Preferences → Profiles → Advanced → "Declare terminal as" VT-100 | Use UTF-8 encoding |
| Windows Terminal | Settings → Profile → Command line add `--colorterm truecolor` | Ensure font supports Unicode |
| VS Code | Set `"terminal.integrated.gpuAcceleration": "on"` | Install Nerd Fonts |

**Prevention**:
- Test on target presentation setup
- Have screenshots ready as backup
- Use widely-supported terminal emulators

### 4. Performance Issues

**Symptom**: Slow responses, UI lag, high CPU usage

**Immediate Solutions**:
```bash
# Close other applications
killall Chrome Safari Slack Docker

# Increase process priority (macOS)
sudo nice -n -10 ./guild chat

# Clear memory caches
./guild cache clear

# Use smaller demo dataset
./guild chat --max-history 50

# Monitor resource usage
top -pid `pgrep guild`
```

**Optimization Settings**:
```bash
# Reduce visual effects
export GUILD_ANIMATIONS=false
export GUILD_REDUCED_MOTION=true

# Limit concurrent operations
export GUILD_MAX_AGENTS=3
export GUILD_BATCH_SIZE=1

# Use faster providers
./guild chat --provider deepseek  # Usually faster than OpenAI
```

**Prevention**:
- Run performance tests before demos
- Close unnecessary applications
- Use SSD storage for better I/O
- Monitor system resources during practice

### 5. Recording Problems

**Symptom**: asciinema or agg fails, poor quality recordings

**Immediate Solutions**:
```bash
# Update recording tools
brew upgrade asciinema agg

# Check permissions
chmod 755 scripts/demo-recording/
ls -la scripts/demo-recording/

# Use alternative recording
# macOS: QuickTime Player → New Screen Recording
# Windows: Windows Game Bar (Win+G)
# Linux: SimpleScreenRecorder or OBS

# Manual fallback
script -a demo-transcript.txt  # Record terminal session
```

**Recording Best Practices**:
```bash
# Optimal settings for different purposes
# For GIFs (social media):
agg --speed 2.0 --theme dracula --font-size 16

# For documentation:
agg --speed 1.0 --theme github-light --font-size 14

# For presentations:
agg --speed 1.5 --theme monokai --font-size 18
```

**Prevention**:
- Test recording setup beforehand
- Have backup recording methods ready
- Record in smaller segments
- Keep original .cast files for re-conversion

## 🆘 Emergency Fallbacks

### If Live Demo Completely Fails

**Option 1: Pre-recorded Video**
- Have MP4 backup ready on desktop
- Practice smooth transition to video
- Narrate over video to maintain engagement

**Option 2: Static Screenshots with Narration**
```
Prepare these screenshots in advance:
1. Guild initialization screen
2. Rich markdown rendering example
3. Multi-agent status panel
4. Code generation with syntax highlighting
5. Final architecture diagram
```

**Option 3: Code Walkthrough**
- Open code editor with Guild source
- Walk through key architectural components
- Show tests passing as proof of functionality
- Demonstrate configuration files

### Backup Demo Flow (No Live System)

1. **Introduction (30 seconds)**
   - "Let me show you Guild through some prepared examples..."
   - Show commission document (examples/commissions/task-management-api.md)

2. **Architecture Overview (60 seconds)**
   - Draw on whiteboard or show architecture diagram
   - Explain agent roles and coordination
   - Highlight competitive advantages

3. **Code Examples (90 seconds)**
   - Show agent configuration (guild.yaml)
   - Walk through key source files
   - Demonstrate test suite passing

4. **Results (30 seconds)**
   - Show generated architecture documents
   - Display file structure created by agents
   - Highlight time savings and quality

## 🔧 Advanced Troubleshooting

### Debug Mode Commands
```bash
# Enable comprehensive debugging
export GUILD_DEBUG=true
export GUILD_LOG_LEVEL=debug
export GUILD_TRACE_REQUESTS=true

# Run with full logging
./guild chat --debug --verbose --log-file debug.log

# Monitor in real-time
tail -f debug.log | grep -E "(ERROR|WARN|DEBUG)"
```

### Memory and Resource Issues
```bash
# Monitor memory usage
while true; do
    ps aux | grep guild | grep -v grep
    sleep 2
done

# Check file descriptor limits
ulimit -n
lsof | grep guild | wc -l

# Clear system caches (macOS)
sudo purge

# Clear system caches (Linux)
sudo sync && sudo sysctl vm.drop_caches=3
```

### Network and Connectivity
```bash
# Test DNS resolution
nslookup api.openai.com
nslookup api.anthropic.com

# Test HTTP connectivity
curl -I https://api.openai.com/v1/models
curl -I https://api.anthropic.com/v1/messages

# Check proxy settings
echo $HTTP_PROXY $HTTPS_PROXY $NO_PROXY

# Test local network
nc -zv localhost 50051
nc -zv localhost 50052
```

## 📞 Support Contacts

### During Demo Day
- **Emergency hotline**: Keep developer phone ready
- **Backup presenter**: Have colleague ready to take over
- **Technical support**: Remote screen sharing setup

### For Practice Sessions
- Test with full setup 24 hours before
- Record practice session for review
- Get feedback from test audience
- Prepare Q&A for common questions

## 📚 Preparation Checklist

### 24 Hours Before Demo
- [ ] Full system test on target hardware
- [ ] Record backup video
- [ ] Test all scenarios end-to-end
- [ ] Verify API keys have sufficient credits
- [ ] Update all dependencies
- [ ] Prepare offline fallback materials

### 1 Hour Before Demo
- [ ] Close all unnecessary applications
- [ ] Disable notifications and updates
- [ ] Run `./guild demo-check --verbose`
- [ ] Test microphone and screen sharing
- [ ] Have backup device ready
- [ ] Clear browser cache and restart

### 5 Minutes Before Demo
- [ ] Final `./guild demo-check`
- [ ] Start recording (if needed)
- [ ] Open backup materials
- [ ] Take deep breath and smile 😊

## 💡 Pro Tips for Smooth Demos

### Presentation Techniques
1. **Narrate your actions**: "Now I'm going to ask the manager agent to..."
2. **Highlight unique features**: "Notice how Guild shows real-time agent status..."
3. **Handle delays gracefully**: "While this processes, let me explain..."
4. **Engage the audience**: "What would you like to see the agents build next?"

### Technical Tricks
1. **Pre-type complex commands**: Copy-paste for speed and accuracy
2. **Use tab completion**: Shows interactive features naturally
3. **Leverage command history**: Demonstrates usability
4. **Show error recovery**: Demonstrates robustness

### Risk Mitigation
1. **Multiple backup plans**: Video, screenshots, code walkthrough
2. **Practice transitions**: Smooth switches between backup methods
3. **Time buffers**: Finish early to allow for Q&A
4. **Graceful degradation**: "In the interest of time, let me show you..."

Remember: The goal is to showcase Guild's capabilities, not to have a perfect technical demo. Audience understanding and engagement matter more than flawless execution.
