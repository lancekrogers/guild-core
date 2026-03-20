# Guild Workspace Architecture

## Critical Directory Structure

Guild uses a **two-level directory architecture** that MUST be followed:

### 1. Global User Directory: `~/.guild/`

Located in the user's home directory, this contains:

- Global configuration
- Daemon process files  
- Provider credentials
- User-level settings

```
~/.guild/
├── config.yaml          # Global user configuration
├── daemon.sock          # Unix socket for daemon communication
├── daemon.log           # Daemon process logs
├── daemon.pid           # Daemon process ID
└── providers/           # Provider API keys and credentials
    ├── openai.yaml
    ├── anthropic.yaml
    └── ollama.yaml
```

### 2. Workspace Directory: `.campaign/`

Located at the root of each workspace, this contains:

- Workspace-specific configuration
- Local agent memory
- Task management
- Workspace-specific prompts

```
my-workspace/
└── .campaign/                  # WORKSPACE CONFIG DIRECTORY
    ├── campaign.yaml           # ← THIS IS THE MAIN CONFIG FILE
    ├── memory.db              # Workspace-specific memory
    ├── objectives/            # Commission definitions
    │   ├── commission-1.md
    │   └── commission-2.md
    ├── kanban/                # Task tracking
    │   ├── todo/
    │   ├── in-progress/
    │   └── done/
    ├── archives/              # Agent conversation history
    └── prompts/               # Custom prompts for this workspace
```

## ⚠️ CRITICAL: Common Mistakes to Avoid

### ❌ NEVER Create `.guild/` at Workspace Level

**WRONG:**

```
my-workspace/
├── .guild/           # ❌ NEVER CREATE THIS
│   └── guild.yaml    # ❌ WRONG - should be campaign.yaml in .campaign/
└── .campaign/        # Created but not used
```

**CORRECT:**

```
my-workspace/
└── .campaign/        # ✅ ONLY workspace directory
    └── campaign.yaml # ✅ Workspace config goes here
```

### Why This Matters

1. **`.guild/` at workspace level conflicts with the global daemon**
2. **Chat tool looks for `campaign.yaml` in `.campaign/`, not `guild.yaml` in `.guild/`**
3. **Mixing the two patterns breaks workspace isolation**

## Terminology Clarification

### Campaign = Workspace

A **campaign** is Guild's term for a workspace that can contain:

- Multiple related projects
- Shared agent memory
- Common objectives
- Unified task management

Example workspace structure:

```
ai-startup-campaign/           # The campaign/workspace
├── .campaign/                 # Workspace config
│   └── campaign.yaml
├── backend-api/              # Sub-project 1
├── frontend-app/             # Sub-project 2
├── ml-models/                # Sub-project 3
└── shared-docs/              # Shared resources
```

## Code Implementation Guidelines

### When Creating Directories

```go
// CORRECT: Check for .campaign directory
campaignDir := filepath.Join(workspaceRoot, ".campaign")
configPath := filepath.Join(campaignDir, "campaign.yaml")

// WRONG: Creating .guild at workspace level
// guildDir := filepath.Join(workspaceRoot, ".guild")  // ❌ NEVER DO THIS
// configPath := filepath.Join(guildDir, "guild.yaml") // ❌ WRONG
```

### When Looking for Configuration

```go
// CORRECT: Look for campaign.yaml in .campaign/
func findWorkspaceConfig(dir string) (string, error) {
    campaignPath := filepath.Join(dir, ".campaign", "campaign.yaml")
    if _, err := os.Stat(campaignPath); err == nil {
        return campaignPath, nil
    }
    // ... traverse up directories
}

// WRONG: Looking for guild.yaml in .guild/
// This should ONLY happen for the global ~/.guild/ directory
```

## Testing Implications

Integration tests should:

1. Create `.campaign/` directories for workspace tests
2. Use `~/.guild/` only for global config tests
3. Never mix the two patterns

## Migration Path

If you encounter a workspace with `.guild/`:

1. Move `guild.yaml` to `.campaign/campaign.yaml`
2. Move all workspace data to `.campaign/`
3. Remove the `.guild/` directory
4. Update any references in code

## Summary

- **Global**: `~/.guild/` - User-level configuration
- **Workspace**: `.campaign/` - Workspace-level configuration  
- **Config File**: `campaign.yaml` in `.campaign/`, NOT `guild.yaml` in `.guild/`
- **Campaign**: Guild's term for a workspace containing related projects
