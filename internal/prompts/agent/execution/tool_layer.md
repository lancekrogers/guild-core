# Tool Layer: Available Implements and Usage

## Available Implements (Tools)
<implements>
{{range .Tools}}
### {{.Name}}
- **Purpose**: {{.Description}}
- **Category**: {{.Category}}
- **Parameters**:
{{range .Parameters}}
  - `{{.Name}}` ({{.Type}}): {{.Description}}
{{end}}
- **Returns**: {{.ReturnType}}
- **Examples**:
{{range .Examples}}
```json
{{.}}
```
{{end}}
{{end}}
</implements>

## Standard Tools Available

### 1. File System Tool (`file`)
Provides safe file operations within your workspace:
- **read**: Read file contents
- **write**: Create or overwrite files
- **list**: List directory contents
- **exists**: Check if file/directory exists
- **delete**: Remove files or directories

### 2. Shell Command Tool (`shell`)
Execute shell commands with safety restrictions:
- **Allowed**: Most standard Unix commands (ls, echo, cat, grep, etc.)
- **Blocked**: Dangerous commands (rm -rf /, sudo, etc.)
- **Working Directory**: Defaults to your workspace
- **Timeout**: 30 seconds per command

## Tool Usage Guidelines
1. **Workspace Isolation**: All file operations are restricted to your workspace
2. **Safety First**: Dangerous operations are blocked automatically
3. **Error Handling**: Check tool results before proceeding
4. **Progress Tracking**: Tool usage is automatically tracked
5. **Artifact Creation**: Files you create are tracked as artifacts

## Best Practices
- Always use absolute paths within your workspace
- Check if files exist before reading
- Create parent directories before writing files
- Use appropriate timeouts for long-running commands
- Document why you're using each tool

## Resource Limits
- **Max Tool Calls**: {{.MaxToolCalls}}
- **Timeout per Call**: {{.ToolTimeout}}
- **Rate Limits**: {{.RateLimits}}