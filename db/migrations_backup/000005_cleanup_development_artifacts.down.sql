-- Rollback for cleanup development artifacts migration
-- Note: This migration cannot be fully rolled back as deleted data cannot be restored
-- This is intentional as we're removing development artifacts that should not exist in production

-- Add a comment indicating this migration is not reversible
SELECT 'WARNING: Migration 000005_cleanup_development_artifacts is not reversible. Deleted test/development data cannot be restored.' AS warning;