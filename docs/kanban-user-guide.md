# Guild Kanban User Guide

## Real-time Task Tracking with Kanban

Guild includes a built-in kanban board that shows your project's progress in real-time. As agents work on tasks, you can watch them move across the board automatically through the daemon's event streaming system.

## Getting Started

### Prerequisites

1. Ensure the Guild daemon is running:

   ```bash
   guild serve
   ```

2. Initialize a campaign in your project directory:

   ```bash
   guild init my-project
   cd my-project
   ```

### Starting the Kanban View

Open the kanban board alongside your chat session:

```bash
# Terminal 1: Chat with agents
guild chat

# Terminal 2: Watch progress in real-time  
guild kanban view
```

The kanban board will automatically connect to the daemon's event stream for real-time updates.

## Kanban Board Layout

```
┌─ Guild Kanban Board ──────────────────────────────────────────────────────┐
│ Campaign: my-project                                   Connected ● Event Stream │
│                                                                              │
│  TODO            IN PROGRESS      BLOCKED        READY FOR REVIEW    DONE  │
│  ┌────────────┐   ┌────────────┐   ┌────────┐    ┌────────────┐    ┌────┐ │
│  │ API-001    │   │ API-003    │   │ API-005│    │ API-007    │    │API-│ ││
│  │ Auth setup │   │ User CRUD  │   │ Tests  │    │ Code review│    │002 │ ││  
│  │ @elena     │   │ @marcus    │   │ Blocked│    │ @vera      │    │✓   │ ││
│  │ ⭐High     │   │ ◐ Working  │   │ 🚫 API │    │ 📝 Ready   │    │    │ ││
│  └────────────┘   └────────────┘   └────────┘    └────────────┘    └────┘ │
│  ┌────────────┐                                   ┌────────────┐           │
│  │ API-004    │                                   │ API-008    │           │
│  │ Docs       │                                   │ Deploy     │           │
│  │ @elena     │                                   │ @marcus    │           │
│  │ ●Medium    │                                   │ 📋 Review  │           │
│  └────────────┘                                   └────────────┘           │
│                                                                              │
│ [h/l] Move columns [j/k] Scroll [/] Search [?] Help [r] Refresh [q] Quit   │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Column Structure

Tasks flow through 5 standardized columns:

1. **TODO**: Tasks ready to be worked on (kanban.StatusTodo)
2. **IN PROGRESS**: Tasks currently being worked on (kanban.StatusInProgress)  
3. **BLOCKED**: Tasks blocked by dependencies (kanban.StatusBlocked)
4. **READY FOR REVIEW**: Tasks completed and awaiting review (kanban.StatusReadyForReview)
5. **DONE**: Tasks that are complete (kanban.StatusDone)

## Keyboard Navigation

### Column Navigation

- **h** or **←**: Move to previous column
- **l** or **→**: Move to next column  
- **1-5**: Jump directly to column (1=TODO, 2=IN PROGRESS, etc.)

### Task Navigation

- **j** or **↓**: Scroll down in current column
- **k** or **↑**: Scroll up in current column
- **J**: Page down (scroll multiple rows)
- **K**: Page up (scroll multiple rows)

### Task Interaction

- **Enter**: View task details
- **Space**: Toggle task expansion (future feature)

### Search and Filter

- **/**: Enter search mode
- **Esc**: Exit search mode
- Type while in search mode to filter tasks by title, description, or assignee

### Controls

- **?**: Toggle help display
- **r** or **R**: Force refresh from server
- **q** or **Ctrl+C**: Quit kanban view

## Understanding Task States

### Automatic State Transitions

Tasks automatically move through states as agents work:

1. **Planning Phase**: New tasks appear in TODO
2. **Active Work**: Tasks move to IN PROGRESS when agents start
3. **Obstacles**: Tasks move to BLOCKED when dependencies arise
4. **Completion**: Tasks move to READY FOR REVIEW when work finishes
5. **Final State**: Tasks move to DONE after review approval

### Task Card Information

Each task card displays:

- **Task ID**: Unique identifier (e.g., API-001)
- **Title**: Brief description of the work
- **Assignee**: Agent responsible (@elena, @marcus, @vera)
- **Priority**: Visual indicator (⭐High, ●Medium, ○Low)
- **Status**: Work indicator (◐ Working, 🚫 Blocked, 📝 Ready, ✓ Done)

## Real-time Updates

The kanban board updates automatically via the daemon's event streaming:

### Event Types Monitored

- **task.created**: New tasks appear immediately
- **task.moved**: Tasks move between columns instantly  
- **task.updated**: Task details refresh in real-time
- **task.completed**: Completion status updates live
- **task.assigned**: Assignment changes show immediately
- **task.blocked/unblocked**: Blocking status updates instantly

### Connection Indicators

- **🟢 Connected**: Event stream active, real-time updates enabled
- **🔴 Disconnected**: Event stream offline, polling fallback active
- **⚠️ No Daemon**: Running without event stream (basic mode)

### Performance Characteristics

- **Update Latency**: < 200ms from event to UI display
- **Throughput**: Handles 200+ tasks with smooth 30 FPS rendering
- **Auto-reconnect**: Reconnects to event stream after disconnections

## Managing Kanban Boards

### Listing Available Boards

```bash
guild kanban list
```

Output example:

```
📋 Found 3 kanban board(s):

🏗️  Main Workshop Board (main-board)
    Central board for tracking all guild work
    📊 Total: 15 | TODO: 5 | In Progress: 3 | Blocked: 1 | Review: 2 | Done: 4
    🕒 Created: 2025-01-15 09:30

🏗️  Backend API Project (backend-tasks)  
    API development tracking board
    📊 Total: 8 | TODO: 2 | In Progress: 2 | Blocked: 0 | Review: 1 | Done: 3
    🕒 Created: 2025-01-14 14:22

🏗️  Frontend Components (frontend-ui)
    UI component development board  
    📊 Total: 12 | TODO: 4 | In Progress: 1 | Blocked: 2 | Review: 3 | Done: 2
    🕒 Created: 2025-01-13 11:15

Use 'guild kanban view <board-id>' to open a board in the interactive UI.
```

### Creating New Boards

```bash
# Create with name only
guild kanban create "My Project"

# Create with name and description
guild kanban create "Backend Tasks" "API development tracking board"
```

### Viewing Specific Boards

```bash
# View default main board
guild kanban view

# View specific board by ID  
guild kanban view backend-tasks

# View board without starting daemon
guild kanban view --no-daemon my-board
```

## Handling Blocked Tasks

When a task enters BLOCKED state:

### Visual Indicators

1. Task card shows red "🚫 Blocked" indicator
2. Blocking reason displayed in task details
3. Task appears in BLOCKED column

### Resolution Workflow  

1. A review file is created in `.guild/kanban/review/`
2. Edit the file to provide resolution details
3. Task automatically resumes when unblocked
4. Task moves back to appropriate column

### Example Blocking Scenarios

- **Missing Dependencies**: "Waiting for API key from client"
- **External Blockers**: "Blocked by database migration"  
- **Review Required**: "Needs architecture review before proceeding"
- **Resource Constraints**: "Waiting for test environment availability"

## Search and Filtering

### Search Functionality

Enter search mode with `/` key:

```
Search: api auth
```

Search looks through:

- Task titles
- Task descriptions  
- Assignee names
- Task IDs

### Search Examples

- `api` - Find all API-related tasks
- `@elena` - Find tasks assigned to Elena
- `blocked` - Find blocked or blocking tasks
- `high` - Find high-priority tasks

## Troubleshooting

### Connection Issues

**Problem**: "Guild server is not reachable"

```bash
# Check daemon status
guild status

# Start daemon manually
guild serve

# Check logs for errors
tail -f ~/.guild/logs/daemon.log
```

**Problem**: Event stream disconnected

- Kanban automatically reconnects after 5 seconds
- Manual refresh with `r` key forces immediate reconnection
- Check daemon logs for networking issues

### Performance Issues

**Problem**: Slow rendering with many tasks

- Board is optimized for 200+ tasks
- Large datasets use virtualized scrolling
- Low-quality mode activates automatically under load

**Problem**: High memory usage

- Task cache automatically purges old data
- Viewport-based rendering limits memory footprint
- Restart kanban view if memory issues persist

### Data Sync Issues

**Problem**: Tasks not updating in real-time

1. Check event stream connection status
2. Verify daemon is running: `guild status`
3. Force refresh with `r` key
4. Restart kanban view if issues persist

**Problem**: Missing tasks

1. Tasks may be filtered by current search
2. Clear search with `Esc` key  
3. Check if tasks are in different columns
4. Refresh data with `r` key

## Advanced Features

### Multi-Board Workflow

For complex projects, use multiple boards:

```bash
# Development board
guild kanban view dev-board

# Testing board  
guild kanban view test-board

# Production board
guild kanban view prod-board
```

### Integration with Chat

The kanban board integrates seamlessly with Guild chat:

1. **Create tasks**: Ask agents to create tasks via chat
2. **Watch progress**: Use kanban to monitor real-time progress  
3. **Resolve blocks**: Chat with agents about blocked tasks
4. **Review completion**: Use kanban to see completed work

### Keyboard Shortcuts Summary

| Key | Action |
|-----|--------|
| `h`/`l` | Navigate columns |
| `j`/`k` | Scroll tasks |  
| `J`/`K` | Page scroll |
| `1-5` | Jump to column |
| `/` | Search mode |
| `Esc` | Exit search |
| `Enter` | Task details |
| `?` | Toggle help |
| `r` | Refresh |
| `q` | Quit |

## Best Practices

### Effective Task Management

1. **Keep task titles concise** - Fits better in card view
2. **Use descriptive assignees** - Easy visual identification  
3. **Monitor blocked tasks** - Address blockers quickly
4. **Regular reviews** - Check READY FOR REVIEW column frequently

### Performance Optimization

1. **Use search filtering** - Reduce visual clutter with many tasks
2. **Multiple boards** - Split large projects into focused boards
3. **Regular cleanup** - Archive completed tasks periodically

### Team Collaboration  

1. **Shared viewing** - Multiple team members can view same board
2. **Real-time coordination** - See live updates as team works
3. **Block communication** - Use chat to resolve blocked tasks quickly

---

*For more information, see the [Guild Documentation](../README.md) or run `guild --help`.*
