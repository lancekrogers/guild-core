-- Rollback template system migration
-- Remove all template-related tables and indexes

-- Drop triggers first
DROP TRIGGER IF EXISTS increment_template_usage;
DROP TRIGGER IF EXISTS update_template_timestamp;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS template_usage;
DROP TABLE IF EXISTS template_categories;
DROP TABLE IF EXISTS template_variables;
DROP TABLE IF EXISTS templates;