-- Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
-- SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

-- Corpus documents table
CREATE TABLE IF NOT EXISTS corpus_documents (
    id TEXT PRIMARY KEY,
    path TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL,
    title TEXT,
    description TEXT,
    checksum TEXT NOT NULL,
    last_modified TIMESTAMP NOT NULL,
    last_indexed TIMESTAMP,
    metadata JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for corpus_documents
CREATE INDEX IF NOT EXISTS idx_corpus_documents_path ON corpus_documents(path);
CREATE INDEX IF NOT EXISTS idx_corpus_documents_type ON corpus_documents(type);
CREATE INDEX IF NOT EXISTS idx_corpus_documents_modified ON corpus_documents(last_modified);

-- Corpus chunks table
CREATE TABLE IF NOT EXISTS corpus_chunks (
    id TEXT PRIMARY KEY,
    document_id TEXT NOT NULL,
    chunk_index INTEGER NOT NULL,
    start_offset INTEGER NOT NULL,
    end_offset INTEGER NOT NULL,
    content_hash TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (document_id) REFERENCES corpus_documents(id) ON DELETE CASCADE,
    UNIQUE(document_id, chunk_index)
);

-- Index for finding chunks by document
CREATE INDEX IF NOT EXISTS idx_corpus_chunks_document ON corpus_chunks(document_id);

-- Corpus knowledge table
CREATE TABLE IF NOT EXISTS corpus_knowledge (
    id TEXT PRIMARY KEY,
    source_type TEXT NOT NULL, -- 'chat', 'code_analysis', 'manual'
    source_id TEXT,
    content TEXT NOT NULL,
    confidence REAL DEFAULT 1.0,
    tags TEXT, -- JSON array
    metadata JSON,
    extracted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    validated_at TIMESTAMP,
    validation_status TEXT DEFAULT 'pending' -- 'pending', 'validated', 'rejected'
);

-- Indexes for corpus_knowledge
CREATE INDEX IF NOT EXISTS idx_corpus_knowledge_source ON corpus_knowledge(source_type, source_id);
CREATE INDEX IF NOT EXISTS idx_corpus_knowledge_status ON corpus_knowledge(validation_status);

-- Sync states table
CREATE TABLE IF NOT EXISTS corpus_sync_states (
    document_id TEXT PRIMARY KEY,
    file_path TEXT NOT NULL,
    file_checksum TEXT,
    file_modified TIMESTAMP,
    db_checksum TEXT,
    db_modified TIMESTAMP,
    last_synced_at TIMESTAMP,
    status TEXT DEFAULT 'pending',
    conflict_details TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Document tags table (normalized)
CREATE TABLE IF NOT EXISTS corpus_document_tags (
    document_id TEXT NOT NULL,
    tag TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (document_id, tag),
    FOREIGN KEY (document_id) REFERENCES corpus_documents(id) ON DELETE CASCADE
);

-- Index on tags for filtering
CREATE INDEX IF NOT EXISTS idx_corpus_document_tags_tag ON corpus_document_tags(tag);