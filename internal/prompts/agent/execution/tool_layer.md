# Tool Layer: Available Implements and Usage

## Available Implements (Tools)
<implements>
{{range .Tools}}
### {{.Name}}
- **Purpose**: {{.Description}}
- **Usage**: `{{.Usage}}`
- **Parameters**:
{{range .Parameters}}
  - `{{.Name}}` ({{.Type}}): {{.Description}}
{{end}}
- **Returns**: {{.ReturnType}}
- **Example**:
```
{{.Example}}
```
{{end}}
</implements>

## Tool Usage Guidelines
1. **Efficiency**: Use the most appropriate implement for each task
2. **Validation**: Always validate implement outputs before proceeding
3. **Error Handling**: Handle implement failures gracefully
4. **Chaining**: Combine implements effectively to achieve complex goals
5. **Documentation**: Document significant implement usage in your output

## Resource Limits
- **Max Tool Calls**: {{.MaxToolCalls}}
- **Timeout per Call**: {{.ToolTimeout}}
- **Rate Limits**: {{.RateLimits}}