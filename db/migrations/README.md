# Database Migrations

## Current State (MVP)

The Guild Framework uses a single, authoritative migration (`000001_complete_schema.up.sql`) that contains the complete database schema for the MVP. This approach was adopted to provide a clean slate after resolving migration issues during development.

### Schema Overview

The database schema includes the following tables:

1. **campaigns** - Top-level project containers
2. **commissions** - Objectives/goals within campaigns
3. **boards** - Kanban boards for organizing tasks (one per commission)
4. **agents** - AI agent configurations
5. **tasks** - Work items assigned to agents
6. **task_events** - History of task changes
7. **prompt_chains** - Agent conversation history
8. **prompt_chain_messages** - Individual messages in agent conversations
9. **chat_sessions** - Persistent user chat sessions
10. **chat_messages** - Messages within chat sessions
11. **session_bookmarks** - Bookmarked important messages
12. **memory_store** - General key-value storage

### Migration Tool

The project uses [golang-migrate](https://github.com/golang-migrate/migrate) for database migrations. The migration is automatically run when the database is initialized.

### Test Schema

For tests, the schema is duplicated in `pkg/storage/init.go` in the `createTestSchema` function. This must be kept in sync with the migration file.

### Future Migrations

When adding new migrations:

1. Create new numbered migration files (e.g., `000002_add_feature.up.sql`)
2. Always include both `.up.sql` and `.down.sql` files
3. Test migrations thoroughly before committing
4. Update the test schema in `pkg/storage/init.go` if needed

### Migration History

- **000001_complete_schema** - Initial complete schema for MVP (combines all previous migrations)
