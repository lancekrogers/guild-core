# Kanban Demo Commission

This commission demonstrates Guild's real-time kanban capabilities by building a task tracking API. Watch the kanban board as agents collaborate in real-time, with tasks flowing through columns as work progresses.

## Commission: Task Tracking API

Build a comprehensive task tracking REST API with real-time capabilities to showcase Guild's kanban board in action.

### Project Overview

Create a production-ready task management system that demonstrates:

- Multi-agent collaboration via kanban board
- Real-time task state transitions
- Blocking and dependency management
- Performance with high task volumes

### Requirements

#### Core Functionality

- **REST API** with full CRUD operations for tasks
- **Real-time Updates** via WebSocket connections
- **SQLite Database** with proper schema and migrations
- **Authentication System** using JWT tokens
- **Task Dependencies** and blocking resolution
- **Comprehensive Testing** with >80% coverage

#### API Endpoints

```
GET    /api/tasks              # List all tasks
POST   /api/tasks              # Create new task
GET    /api/tasks/{id}         # Get task details
PUT    /api/tasks/{id}         # Update task
DELETE /api/tasks/{id}         # Delete task
POST   /api/tasks/{id}/block   # Block task with reason
POST   /api/tasks/{id}/unblock # Unblock task
GET    /api/health             # Health check endpoint
```

#### Task Schema

```json
{
  "id": "uuid",
  "title": "string",
  "description": "string", 
  "status": "enum(todo|in_progress|blocked|ready_for_review|done)",
  "priority": "enum(low|medium|high)",
  "assignee": "string",
  "tags": ["string"],
  "dependencies": ["task_id"],
  "blockers": ["blocker_id"],
  "created_at": "timestamp",
  "updated_at": "timestamp",
  "due_date": "timestamp?"
}
```

### Expected Kanban Flow

Watch the Guild kanban board as agents work through these phases:

#### Phase 1: Planning and Architecture (Elena - Guild Master)

**Tasks appear in BACKLOG/TODO columns:**

- API Design and Architecture
- Database Schema Design  
- Authentication Strategy
- Testing Framework Setup
- Documentation Structure

*Elena assigns tasks to appropriate specialists based on their expertise.*

#### Phase 2: Foundation Development (Marcus - Developer)

**Tasks move to IN PROGRESS:**

- Database Setup and Migrations
- Basic API Framework (Express/FastAPI/Gin)
- Authentication Middleware
- Request/Response Models
- Error Handling Framework

*Watch tasks transition from TODO → IN PROGRESS as Marcus starts work.*

#### Phase 3: Core Implementation (Marcus continues)

**Parallel tasks in IN PROGRESS:**

- Task CRUD Operations
- WebSocket Real-time Updates
- Dependency Management Logic
- Task Blocking/Unblocking
- Database Queries and Optimization

*Multiple tasks progress simultaneously, showing parallel development.*

#### Phase 4: Testing and Quality (Vera - Tester)

**Tasks move to IN PROGRESS/READY FOR REVIEW:**

- Unit Test Suite
- Integration Tests
- API Endpoint Testing
- WebSocket Connection Tests
- Performance Testing
- Security Testing

*Tasks may move to BLOCKED if issues are discovered.*

#### Phase 5: Blocking Scenarios (Demonstrates Resolution)

**Watch blocking and unblocking in action:**

1. **API Authentication Task** → BLOCKED
   - Reason: "Need to choose between JWT vs. OAuth2"
   - Review file created in `.guild/kanban/review/`
   - Elena resolves by specifying JWT approach
   - Task automatically unblocks and resumes

2. **Database Performance** → BLOCKED  
   - Reason: "Query timeouts with large datasets"
   - Marcus investigates and adds indexes
   - Task unblocks and moves to completion

3. **WebSocket Implementation** → BLOCKED
   - Reason: "CORS issues in browser testing"
   - Vera provides browser testing environment
   - Task unblocks and completes

#### Phase 6: Integration and Deployment (All Agents)

**Final tasks in READY FOR REVIEW/DONE:**

- API Documentation (OpenAPI/Swagger)
- Docker Containerization
- Environment Configuration
- Deployment Scripts
- Performance Benchmarks
- Final Integration Testing

### Live Demo Instructions

#### Setup

```bash
# Terminal 1: Start Guild daemon
guild serve

# Terminal 2: Initialize the demo project
guild init kanban-demo
cd kanban-demo

# Copy this commission to the project
cp examples/kanban-demo-commission.md .

# Terminal 3: Start the kanban board
guild kanban view
```

#### Running the Demo

```bash
# Terminal 1: Start the commission
guild chat
> @elena please implement the task tracking API from kanban-demo-commission.md

# Watch Terminal 3 (kanban board) for real-time updates
```

### What to Observe

#### Real-time Task Flow

- **Task Creation**: New tasks appear instantly in TODO column
- **Status Transitions**: Watch tasks move between columns as work progresses
- **Assignment Changes**: Task cards update with assignee information
- **Parallel Work**: Multiple tasks in IN PROGRESS simultaneously

#### Blocking and Resolution

- **Blocking Events**: Tasks move to BLOCKED column with red indicators
- **Review Files**: `.guild/kanban/review/` files created for human intervention
- **Automatic Unblocking**: Tasks resume when blockers are resolved
- **Dependency Management**: Tasks wait for dependencies to complete

#### Performance Characteristics

- **Update Latency**: < 200ms from agent action to kanban display
- **Throughput**: Board handles 50+ tasks smoothly
- **Search Performance**: Filter tasks by assignee/title in real-time
- **Memory Efficiency**: Virtualized scrolling for large task lists

### Expected Outcomes

#### Successful Completion Metrics

- ✅ **All tasks in DONE column** (typically 25-35 tasks)
- ✅ **Zero tasks remaining blocked**
- ✅ **API fully functional** with all endpoints working
- ✅ **Tests passing** with >80% coverage
- ✅ **Documentation complete** and accessible

#### Kanban Board Statistics

- **Total Tasks**: ~30 tasks across all phases
- **Task Distribution**:
  - DONE: 25-30 tasks
  - READY FOR REVIEW: 0-2 tasks  
  - IN PROGRESS: 0-1 tasks
  - BLOCKED: 0 tasks
  - TODO: 0 tasks

#### Performance Demonstration

- **Event Latency**: All updates < 200ms
- **UI Responsiveness**: Smooth scrolling and navigation
- **Search Functionality**: Instant filtering and results
- **Memory Usage**: Efficient with large task counts

### Technical Implementation Notes

#### Agent Coordination Patterns

```
Elena (Planning) → Creates and assigns tasks
     ↓
Marcus (Development) → Implements core functionality  
     ↓
Vera (Testing) → Validates and tests implementation
     ↓
All Agents → Integration and deployment
```

#### Blocking Resolution Workflow

```
Task encounters issue → Moves to BLOCKED
     ↓
Review file created → Human or agent intervention
     ↓  
Resolution provided → Task automatically unblocks
     ↓
Work resumes → Task progresses to completion
```

#### Real-time Event Flow

```
Agent performs action → Event published to daemon
     ↓
Daemon broadcasts event → Kanban UI receives update
     ↓
UI updates display → Visual change within 200ms
```

### Troubleshooting

#### Common Issues

**Kanban board not updating:**

```bash
# Check daemon status
guild status

# Restart daemon if needed
guild serve --restart
```

**Tasks not appearing:**

```bash
# Verify event stream connection
guild kanban view --debug

# Check for board creation
guild kanban list
```

**Performance issues:**

```bash
# Enable performance monitoring
guild kanban view --profile

# Check memory usage
guild kanban view --stats
```

#### Debug Commands

```bash
# View all boards
guild kanban list

# Create test board manually
guild kanban create "Debug Board" "Manual testing board"

# Monitor daemon logs
tail -f ~/.guild/logs/daemon.log

# Check event stream
guild events watch --filter "task.*"
```

### Success Criteria Checklist

- [ ] Kanban board displays all created tasks
- [ ] Real-time updates work (< 200ms latency)
- [ ] Task blocking and unblocking demonstrated
- [ ] Multiple agents working in parallel visible
- [ ] Search and navigation functionality working
- [ ] Performance acceptable with 30+ tasks
- [ ] All tasks eventually reach DONE status
- [ ] Event streaming maintains stable connection
- [ ] Memory usage remains reasonable
- [ ] No UI rendering glitches or freezes

### Extension Ideas

For longer demonstrations, consider adding:

#### Advanced Features

- **Priority-based task ordering** with visual indicators
- **Task dependency visualization** with connection lines
- **Agent workload balancing** across team members
- **Sprint planning integration** with time estimates
- **Custom task statuses** beyond the default 5 columns

#### Performance Testing

- **Stress test with 200+ tasks** to show scalability
- **Rapid task creation/updates** to test event throughput
- **Multi-board scenarios** with different projects
- **Long-running demo** to test stability over time

#### Integration Scenarios  

- **CI/CD pipeline integration** with build status tasks
- **External tool connectivity** (GitHub, Jira, etc.)
- **Notification systems** for task state changes
- **Reporting and analytics** on task completion rates

---

*This commission is designed to showcase Guild's kanban capabilities in a realistic development scenario. The task tracking API provides a concrete deliverable while demonstrating real-time collaboration, event streaming, and visual task management.*
