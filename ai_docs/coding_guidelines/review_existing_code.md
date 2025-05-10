## Review Existing Code

@context

Use this command to review existing code before implementing new features. This helps avoid duplication and ensures consistency with the current architecture.

### Process

1. **Search for relevant files**
2. **Analyze their structure and dependencies**
3. **Understand current implementation patterns**
4. **Identify extension points**
5. **Determine what's missing for the new feature**

### Search Commands

```bash
# Find files related to a specific feature
find . -type f -name "*.go" | xargs grep -l "search_term" | sort

# Look at Go files in a specific package
find ./pkg/package_name -type f -name "*.go" | sort

# Find interface definitions
find . -type f -name "*.go" | xargs grep -l "type.*interface" | sort

# Look for specific function implementations
find . -type f -name "*.go" | xargs grep -l "func.*FunctionName" | sort
```

### Analysis Questions

For each file you review, answer:

1. What functionality does this implement?
2. What interfaces does it define or implement?
3. What dependencies does it have?
4. How does it integrate with other components?
5. What patterns does it follow (e.g., factory, singleton)?
6. What extension points exist?
7. What would need to be modified for the new feature?

### Specific Packages to Review for Objective System

```bash
# Check objective package
find ./pkg/objective -type f -name "*.go" | sort
cat $(find ./pkg/objective -type f -name "*.go" | sort)

# Check existing generators if any
find ./pkg -type f -name "*.go" | xargs grep -l "generator\|Generator" | grep -v "_test.go" | sort

# Check existing prompt management
find ./internal -type f -name "*.go" | xargs grep -l "prompt\|Prompt" | grep -v "_test.go" | sort

# Check UI components
find ./pkg -type f -name "*.go" | xargs grep -l "bubble\|tea\|model\|view\|update" | grep -v "_test.go" | sort

# Check CLI commands
find ./cmd -type f -name "*.go" | xargs grep -l "cobra\|Command" | grep -v "_test.go" | sort
```

### Documentation

For understanding the existing code patterns, check:

```bash
# View README files
find . -name "README.md" | xargs cat

# Look for design docs
find ./docs -name "*.md" | xargs cat

# Check Go reference documentation
go doc ./pkg/package_name
```

### Output Format

Provide a structured analysis of what you find:

```
## Existing Code Analysis

### Discovered Components
- List of relevant files and their primary purpose

### Current Architecture
- How components are currently structured
- Existing interfaces and implementations
- Integration patterns

### Extension Points
- Where new code can be added
- What interfaces need implementation
- What needs to be modified

### Implementation Plan
- How to implement the new feature while maintaining consistency
- Steps to extend existing code
```
