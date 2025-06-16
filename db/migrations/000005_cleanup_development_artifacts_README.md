# Database Cleanup Migration (000005)

## Purpose
This migration removes development artifacts and test data from the production database to ensure a clean, professional deployment.

## What Gets Cleaned

### 1. Test Data Patterns
The migration removes entries with these prefixes/patterns:
- `test-`, `test ` 
- `demo-`, `demo `
- `example-`, `example `
- `sample-`, `sample `
- `temp-`, `temp `, `tmp-`, `tmp `
- `mock-`, `mock `, `fake-`, `fake `
- `debug-`, `debug `
- `dev-`, `dev `, `development-`, `development `

### 2. Specific Entities Cleaned
- **Campaigns**: Test/demo campaigns by name
- **Commissions**: Test/demo commissions by title
- **Agents**: Test/demo agents by name or ID, including specific demo agents from `cost_demo`:
  - `tools-agent`
  - `quick-coder`
  - `balanced-dev`
  - `senior-architect`
  - `expert-advisor`
  - `ai-specialist`
- **Chat Sessions**: Test/demo chat sessions by name
- **Prompt Chains**: Test/demo prompt chains by name

### 3. Orphaned Data Cleanup
The migration also removes orphaned records:
- Commissions without campaigns
- Boards without commissions
- Tasks without valid commission or board
- Task events without tasks
- Chat messages without sessions
- Session bookmarks without valid session or message

## Important Notes

1. **This migration is NOT reversible** - The down migration cannot restore deleted data
2. **Run with caution in production** - Ensure you have backups before running
3. **Case-insensitive matching** - Uses `LOWER()` to catch all variations
4. **Cascading deletes** - Foreign key constraints will automatically clean up related records

## Testing the Migration

Before running in production:
1. Test on a copy of your production database
2. Verify no legitimate data matches the cleanup patterns
3. Check the count of records that will be affected:

```sql
-- Check campaigns that will be deleted
SELECT COUNT(*) FROM campaigns 
WHERE LOWER(name) LIKE 'test-%' OR LOWER(name) LIKE 'demo-%' -- etc...

-- Check other tables similarly
```

## Running the Migration

```bash
# Using migrate CLI
migrate -path db/migrations -database "sqlite3://.guild/memory.db" up

# Or using Guild's migration command (if available)
guild db migrate up
```