## Implement Guild Objective System

@context

This command will guide you through implementing the Guild objective system. First, check if components already exist to avoid duplication.

### 1. Check Existing Implementation

```bash
# Check for existing objective-related files
find . -type f -name "*.go" | xargs grep -l "objective" | grep -v "_test.go"

# Check for existing prompt-related files
find . -type f -name "*.go" | xargs grep -l "prompt" | grep -v "_test.go"

# Check for existing UI-related files
find . -type f -name "*.go" | xargs grep -l "bubbletea\|bubble tea\|bubbles" | grep -v "_test.go"
```

### 2. Review Specification Documents

```
cat specs/features/objectives/objectives.md
cat specs/features/objectives/objective_ui.md
```

### 3. Implementation Process

Follow this process to implement the objective system:

#### A. Internal Prompt System

1. **Check** if `internal/prompts` exists, if not create it
2. **Implement** prompt management:
   - Create `internal/prompts/manager.go` if needed
   - Implement `internal/prompts/objective/loader.go`
   - Add prompt markdown files in `internal/prompts/objective/markdown/`

#### B. Generator Package

1. **Check** if `pkg/generator` exists, if not create it
2. **Implement** generator interfaces:
   - Create/update `pkg/generator/interface.go`
   - Implement objective generator in `pkg/generator/objective/generator.go`
   - Add AI docs generator and specs generator

#### C. Core Objective Package

1. **Check** existing code in `pkg/objective/`
2. **Extend** or implement the core functionality:
   - Update `pkg/objective/models.go` if needed
   - Enhance parser in `pkg/objective/parser.go`
   - Add status tracking in `pkg/objective/status.go`
   - Implement lifecycle in `pkg/objective/lifecycle.go`

#### D. Bubble Tea UI

1. **Check** for existing UI components
2. **Implement** Bubble Tea UI:
   - Create model in `pkg/ui/objective/model.go`
   - Add view in `pkg/ui/objective/view.go`
   - Implement update in `pkg/ui/objective/update.go`
   - Add dashboard in `pkg/ui/objective/dashboard.go`

#### E. CLI Commands

1. **Check** existing commands in `cmd/guild/`
2. **Implement** CLI commands:
   - Add `cmd/guild/objective_cmd.go` for managing individual objectives
   - Create `cmd/guild/objectives_cmd.go` for the dashboard

### 4. Testing Strategy

For each component, create appropriate tests:

1. **Prompt System**:

   - Test prompt loading
   - Test template rendering
   - Use test fixtures in `internal/prompts/objective/testdata/`

2. **Generator**:

   - Test with mock LLM client
   - Test document generation
   - Test error handling

3. **Objective Package**:

   - Test parsing
   - Test lifecycle transitions
   - Test serialization/deserialization

4. **UI Components**:

   - Test model updates
   - Test view rendering
   - Use mock sessions

5. **CLI Commands**:
   - Test command execution
   - Test flag parsing
   - Test output formatting

### 5. Integration

Ensure all components work together:

1. CLI commands use the UI components
2. UI components use the objectiv[48;76;141;2736;2538te package
3. Generators use the prompt system and LLM providers
4. Objective package handles file I/O and status tracking

### Expected Output

The final implementation should allow users to:

1. Create objectives from scratch or based on natural language descriptions
2. Refine objectives with additional context
3. Generate AI docs and specs based on objectives
4. View a dashboard of all objectives and their status
5. Track the lifecycle of objectives from creation to implementation
