-- Cleanup development artifacts migration
-- This migration removes test data, demo entries, and development artifacts from production database

-- Delete test campaigns (names starting with test-, demo-, example-, sample-, temp-, tmp-, mock-, fake-)
DELETE FROM campaigns 
WHERE LOWER(name) LIKE 'test-%' 
   OR LOWER(name) LIKE 'test %'
   OR LOWER(name) LIKE 'demo-%'
   OR LOWER(name) LIKE 'demo %' 
   OR LOWER(name) LIKE 'example-%'
   OR LOWER(name) LIKE 'example %'
   OR LOWER(name) LIKE 'sample-%'
   OR LOWER(name) LIKE 'sample %'
   OR LOWER(name) LIKE 'temp-%'
   OR LOWER(name) LIKE 'temp %'
   OR LOWER(name) LIKE 'tmp-%'
   OR LOWER(name) LIKE 'tmp %'
   OR LOWER(name) LIKE 'mock-%'
   OR LOWER(name) LIKE 'mock %'
   OR LOWER(name) LIKE 'fake-%'
   OR LOWER(name) LIKE 'fake %'
   OR LOWER(name) LIKE 'debug-%'
   OR LOWER(name) LIKE 'debug %'
   OR LOWER(name) LIKE 'dev-%'
   OR LOWER(name) LIKE 'dev %'
   OR LOWER(name) LIKE 'development-%'
   OR LOWER(name) LIKE 'development %'
   OR LOWER(name) = 'test'
   OR LOWER(name) = 'demo'
   OR LOWER(name) = 'example'
   OR LOWER(name) = 'sample';

-- Delete test commissions (cascade will handle related tasks, boards, etc.)
DELETE FROM commissions 
WHERE LOWER(title) LIKE 'test-%'
   OR LOWER(title) LIKE 'test %'
   OR LOWER(title) LIKE 'demo-%'
   OR LOWER(title) LIKE 'demo %'
   OR LOWER(title) LIKE 'example-%'
   OR LOWER(title) LIKE 'example %'
   OR LOWER(title) LIKE 'sample-%'
   OR LOWER(title) LIKE 'sample %'
   OR LOWER(title) LIKE 'temp-%'
   OR LOWER(title) LIKE 'temp %'
   OR LOWER(title) LIKE 'tmp-%'
   OR LOWER(title) LIKE 'tmp %'
   OR LOWER(title) LIKE 'mock-%'
   OR LOWER(title) LIKE 'mock %'
   OR LOWER(title) LIKE 'fake-%'
   OR LOWER(title) LIKE 'fake %'
   OR LOWER(title) LIKE 'debug-%'
   OR LOWER(title) LIKE 'debug %'
   OR LOWER(title) = 'test'
   OR LOWER(title) = 'demo'
   OR LOWER(title) = 'example'
   OR LOWER(title) = 'sample';

-- Delete test/demo agents
DELETE FROM agents 
WHERE LOWER(name) LIKE 'test %'
   OR LOWER(name) LIKE 'demo %'
   OR LOWER(name) LIKE 'example %'
   OR LOWER(name) LIKE 'sample %'
   OR LOWER(name) LIKE 'mock %'
   OR LOWER(name) LIKE 'fake %'
   OR LOWER(name) LIKE 'temp %'
   OR LOWER(name) LIKE 'tmp %'
   OR LOWER(name) LIKE 'debug %'
   OR LOWER(id) LIKE 'test-%'
   OR LOWER(id) LIKE 'demo-%'
   OR LOWER(id) LIKE 'example-%'
   OR LOWER(id) LIKE 'sample-%'
   OR LOWER(id) LIKE 'mock-%'
   OR LOWER(id) LIKE 'fake-%'
   OR LOWER(id) LIKE 'temp-%'
   OR LOWER(id) LIKE 'tmp-%'
   OR LOWER(id) LIKE 'debug-%'
   -- Specific demo agents from cost_demo
   OR id IN ('tools-agent', 'quick-coder', 'balanced-dev', 'senior-architect', 'expert-advisor', 'ai-specialist')
   -- Common test agent names
   OR LOWER(name) IN ('test manager', 'test developer', 'test worker', 'test agent');

-- Delete test chat sessions
DELETE FROM chat_sessions 
WHERE LOWER(name) LIKE 'test%'
   OR LOWER(name) LIKE 'demo%'
   OR LOWER(name) LIKE 'example%'
   OR LOWER(name) LIKE 'sample%'
   OR LOWER(name) LIKE 'temp%'
   OR LOWER(name) LIKE 'tmp%'
   OR LOWER(name) LIKE 'mock%'
   OR LOWER(name) LIKE 'fake%'
   OR LOWER(name) LIKE 'debug%'
   OR LOWER(name) LIKE 'dev%'
   OR name = 'Test Session'
   OR name = 'Demo Session';

-- Delete orphaned data (commissions without campaigns)
DELETE FROM commissions 
WHERE campaign_id NOT IN (SELECT id FROM campaigns);

-- Delete orphaned boards (boards without commissions)
DELETE FROM boards 
WHERE commission_id NOT IN (SELECT id FROM commissions);

-- Delete orphaned tasks (tasks without valid commission or board)
DELETE FROM tasks 
WHERE commission_id NOT IN (SELECT id FROM commissions)
   OR (board_id IS NOT NULL AND board_id NOT IN (SELECT id FROM boards));

-- Delete orphaned task events (events without tasks)
DELETE FROM task_events 
WHERE task_id NOT IN (SELECT id FROM tasks);

-- Delete orphaned chat messages (messages without sessions)
DELETE FROM chat_messages 
WHERE session_id NOT IN (SELECT id FROM chat_sessions);

-- Delete orphaned session bookmarks
DELETE FROM session_bookmarks 
WHERE session_id NOT IN (SELECT id FROM chat_sessions)
   OR message_id NOT IN (SELECT id FROM chat_messages);

-- Delete any prompt chains with test/demo names (if they exist)
DELETE FROM prompt_chains 
WHERE LOWER(name) LIKE 'test%'
   OR LOWER(name) LIKE 'demo%'
   OR LOWER(name) LIKE 'example%'
   OR LOWER(name) LIKE 'sample%'
   OR LOWER(name) LIKE 'temp%'
   OR LOWER(name) LIKE 'tmp%'
   OR LOWER(name) LIKE 'mock%'
   OR LOWER(name) LIKE 'fake%'
   OR LOWER(name) LIKE 'debug%';

-- Add a timestamp comment to track when cleanup was performed
-- (This is just a comment, not executed)
-- Cleanup performed on: [Migration will add timestamp automatically]