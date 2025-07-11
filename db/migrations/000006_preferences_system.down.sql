-- Drop preference inheritance table and indexes
DROP INDEX IF EXISTS idx_preference_inheritance_parent;
DROP INDEX IF EXISTS idx_preference_inheritance_child;
DROP TABLE IF EXISTS preference_inheritance;

-- Drop preferences table triggers and indexes
DROP TRIGGER IF EXISTS update_preferences_timestamp;
DROP INDEX IF EXISTS idx_preferences_updated_at;
DROP INDEX IF EXISTS idx_preferences_key;
DROP INDEX IF EXISTS idx_preferences_scope_id;
DROP INDEX IF EXISTS idx_preferences_scope;
DROP TABLE IF EXISTS preferences;