# Multi-Instance Daemon Support

Guild Framework supports running multiple daemon instances simultaneously, enabling seamless work across multiple projects and campaigns. Each campaign can have its own dedicated daemon with multiple chat sessions.

## Overview

The multi-instance architecture uses Unix domain sockets to provide:

- **Campaign Isolation**: Each campaign runs in its own daemon instance
- **Multiple Sessions**: Up to 10 concurrent chat sessions per campaign
- **Zero Port Conflicts**: Unix sockets eliminate port management
- **Automatic Management**: Daemons start automatically and shut down when idle

## Architecture

```
~/.guild/
├── run/                        # Runtime sockets
│   ├── abc123def456.sock      # Campaign "shop" primary session
│   ├── abc123def456-1.sock    # Campaign "shop" session 1
│   ├── 789xyz012345.sock      # Campaign "blog" primary session
│   └── ...
└── campaigns/
    ├── shop/
    │   ├── memory.db          # Shared campaign data
    │   ├── daemon.log         # Primary daemon log
    │   └── daemon-1.log       # Session 1 log
    └── blog/
        └── ...
```

## Basic Usage

### Starting Chat Sessions

Guild automatically starts daemons when you run `guild chat`:

```bash
# Start chat for current campaign (auto-detected)
guild chat

# Start chat for specific campaign
guild chat --campaign shop

# Multiple sessions for same campaign
# Terminal 1:
guild chat --campaign shop  # Gets primary session (0)

# Terminal 2:
guild chat --campaign shop  # Automatically gets session 1
```

### Checking Status

View running daemons and their sessions:

```bash
# Status of current campaign
guild status

# Status of all running daemons
guild status --all
```

Example output:

```
● All Guild Daemon Instances

🏰 Campaign: shop
  Session 0:
    Status: running
    Socket: /Users/you/.guild/run/abc123def456.sock
    Transport: Unix Socket

  Session 1:
    Status: running
    Socket: /Users/you/.guild/run/abc123def456-1.sock
    Transport: Unix Socket

🏰 Campaign: blog
  Session 0:
    Status: running
    Socket: /Users/you/.guild/run/789xyz012345.sock
    Transport: Unix Socket

📊 Total instances: 3
```

### Stopping Daemons

Stop daemons when done:

```bash
# Stop current campaign's daemon
guild stop

# Stop specific campaign
guild stop --campaign shop

# Stop specific session
guild stop --campaign shop --session 1

# Stop all daemons
guild stop --all

# Force stop with custom timeout
guild stop --all --force --timeout 10s
```

## Advanced Features

### Session Management

Each campaign supports up to 10 concurrent sessions (0-9):

- **Session 0**: Primary session (default)
- **Sessions 1-9**: Additional sessions

Sessions are automatically allocated when you start multiple chat instances for the same campaign.

### Resource Management

Daemons run with controlled resources:

- **CPU Priority**: Nice level +5 (lower priority)
- **Memory Limits**: Configurable via environment
- **Idle Timeout**: 15 minutes by default

Configure resource limits:

```bash
# Set memory limit (in bytes)
export GUILD_MEMORY_LIMIT=536870912  # 512MB

# Set custom idle timeout
export GUILD_IDLE_TIMEOUT=30m
```

### Automatic Cleanup

Guild automatically:

- Detects and cleans stale socket files
- Recovers from crashed daemons
- Shuts down idle daemons
- Removes socket files on shutdown

### Campaign Detection

Guild automatically detects the campaign from your current directory:

1. Looks for `.guild/guild.yaml` in current directory
2. Searches parent directories up to home
3. Uses campaign name from configuration

## Troubleshooting

### "Maximum sessions reached"

Each campaign is limited to 10 concurrent sessions. To free up a session:

```bash
# List sessions for campaign
guild status --all

# Stop specific session
guild stop --campaign shop --session 5
```

### "Socket connection failed"

If you see socket connection errors:

1. Check if daemon is running: `guild status`
2. Clean stale sockets: `guild stop --all`
3. Try starting fresh: `guild chat --campaign <name>`

### Permission Errors

Socket files require proper permissions (0600). If you see permission errors:

```bash
# Check socket permissions
ls -la ~/.guild/run/

# Clean and restart
guild stop --all
rm -rf ~/.guild/run/*.sock
guild chat
```

### Finding Socket Paths

Campaign names are hashed for socket filenames to ensure compatibility:

```bash
# View socket paths for running daemons
guild status --all

# Socket naming convention:
# Primary: <campaign-hash>.sock
# Additional: <campaign-hash>-<session>.sock
```

## Best Practices

1. **Let Guild Manage Daemons**: Use `guild chat` to auto-start daemons
2. **Clean Shutdown**: Use `guild stop` instead of killing processes
3. **Monitor Resources**: Check `guild status --all` periodically
4. **Use Campaigns**: Organize work by campaign for better isolation

## Platform Notes

### macOS

- Socket path limit: 104 characters (handled by hashing)
- Sockets stored in user home directory
- Full multi-instance support

### Linux

- Socket path limit: 108 characters (handled by hashing)
- Sockets stored in user home directory
- Full multi-instance support

### Windows

- Currently uses TCP mode (single instance)
- Unix socket support planned for WSL2
- Use `--tcp` flag for compatibility

## Environment Variables

Control daemon behavior with environment variables:

- `GUILD_IDLE_TIMEOUT`: Idle timeout duration (default: 15m)
- `GUILD_MEMORY_LIMIT`: Memory limit in bytes
- `GUILD_SOCKET_DIR`: Override socket directory (default: ~/.guild/run)
- `GUILD_LOG_LEVEL`: Logging verbosity (debug, info, warn, error)

## Security Considerations

- Socket files are created with 0600 permissions (owner only)
- Socket directory has 0700 permissions
- Each campaign's data is isolated
- No network exposure (Unix sockets are local only)

## Integration with Tools

The multi-instance architecture integrates seamlessly with Guild tools:

- Each daemon maintains its own tool state
- File operations are scoped to campaign workspace
- Memory and context isolated per campaign
- Concurrent tool execution across sessions

This multi-instance support enables efficient workflows across multiple projects while maintaining isolation and resource efficiency.
