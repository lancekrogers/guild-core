-- Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
-- SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

-- Drop indexes first
DROP INDEX IF EXISTS idx_corpus_document_tags_tag;
DROP INDEX IF EXISTS idx_corpus_knowledge_status;
DROP INDEX IF EXISTS idx_corpus_knowledge_source;
DROP INDEX IF EXISTS idx_corpus_chunks_document;
DROP INDEX IF EXISTS idx_corpus_documents_modified;
DROP INDEX IF EXISTS idx_corpus_documents_type;
DROP INDEX IF EXISTS idx_corpus_documents_path;

-- Drop tables
DROP TABLE IF EXISTS corpus_document_tags;
DROP TABLE IF EXISTS corpus_sync_states;
DROP TABLE IF EXISTS corpus_knowledge;
DROP TABLE IF EXISTS corpus_chunks;
DROP TABLE IF EXISTS corpus_documents;