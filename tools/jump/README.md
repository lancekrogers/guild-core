# Jump Tool

A frecency-based directory jumping tool for AI agents in the Guild framework. Similar to `z` or `autojump`, but designed specifically for agent use via the Tool interface.

## Features

- **Frecency-based ranking**: Combines frequency (how often visited) with recency (how recently visited)
- **Fuzzy matching**: Find directories with partial names
- **Pure Go implementation**: Uses SQLite (modernc.org/sqlite) with no CGO dependencies
- **Concurrent safe**: Uses WAL mode for better concurrent access
- **Automatic cleanup**: Removes non-existent directories from the database

## Usage

The jump tool is registered automatically with the Guild framework and can be used by agents through the standard Tool interface.

### Jump to a directory

```json
{"query": "docs"}
```

Returns: `"/absolute/path/to/documents"`

### Track a directory visit

```json
{"query": "/path/to/project", "track": true}
```

Returns: `"ok"`

### Get recent directories

```json
{"recent": 5}
```

Returns: `["/dir1", "/dir2", "/dir3", "/dir4", "/dir5"]`

## How it works

1. **Tracking**: When you visit a directory with `track: true`, the tool records:
   - The directory path
   - Increments the visit frequency
   - Updates the last visit timestamp (millisecond precision)

2. **Finding**: When searching with a query:
   - Retrieves all tracked directories from the database
   - Performs fuzzy matching on directory base names
   - Calculates frecency scores: `score = frequency / (1 + hours_since_last_visit)`
   - Returns the best match combining fuzzy match score and frecency

3. **Storage**: Data is stored in `~/.guild/jump.db` using SQLite

## Database Schema

```sql
CREATE TABLE visits (
  dir  TEXT PRIMARY KEY,    -- Absolute directory path
  freq INTEGER NOT NULL,    -- Visit count
  last INTEGER NOT NULL     -- Unix timestamp (milliseconds)
);
```

## Performance

The tool is designed for high performance:

- Tracking: ≥1,000 tracks/second
- Finding: ≥5,000 finds/second
- Database uses WAL mode for concurrent access
- Connections are created on-demand and closed immediately

## Examples

### Agent workflow

```go
// Jump to frequently used project
result, _ := tools.Execute(ctx, "jump", `{"query":"guild-framework"}`)
// result = "/home/user/projects/guild-framework"

// Track current directory after work
cwd, _ := os.Getwd()
tools.Execute(ctx, "jump", fmt.Sprintf(`{"query":%q,"track":true}`, cwd))
```

### Common queries

- `{"query": "proj"}` - Matches directories like "projects", "my-project"
- `{"query": "doc"}` - Matches "documents", "docs", "documentation"
- `{"query": "down"}` - Matches "downloads", "downloaded-files"

## Testing

The tool includes comprehensive tests covering:

- Basic tracking and finding
- Frecency calculations
- Recent directory listing
- Edge cases (empty queries, non-existent directories)
- Concurrent access
- Tool interface integration

Run tests with:

```bash
go test ./tools/jump/...
```
