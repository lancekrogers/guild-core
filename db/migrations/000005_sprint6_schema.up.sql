-- Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
-- SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

-- Migration 005: performance optimization Schema Extensions
-- Adds tables for session management, performance monitoring, and analytics

-- Session management tables
CREATE TABLE IF NOT EXISTS session_data (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    campaign_id TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    encrypted_data BLOB, -- AES-256-GCM encrypted session state
    compression_type TEXT DEFAULT 'gzip',
    metadata JSON,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_session_user_id ON session_data(user_id);
CREATE INDEX IF NOT EXISTS idx_session_campaign_id ON session_data(campaign_id);
CREATE INDEX IF NOT EXISTS idx_session_expires_at ON session_data(expires_at);

CREATE TABLE IF NOT EXISTS session_messages (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    agent_id TEXT,
    user_id TEXT,
    content TEXT NOT NULL,
    message_type TEXT NOT NULL, -- 'user', 'agent', 'system', 'error'
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata JSON,
    FOREIGN KEY (session_id) REFERENCES session_data(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_session_messages_session_id ON session_messages(session_id);
CREATE INDEX IF NOT EXISTS idx_session_messages_timestamp ON session_messages(timestamp);
CREATE INDEX IF NOT EXISTS idx_session_messages_agent_id ON session_messages(agent_id);

CREATE TABLE IF NOT EXISTS session_ui_state (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    component_name TEXT NOT NULL, -- 'chat', 'kanban', 'editor'
    state_data JSON NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES session_data(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_session_ui_state_session_id ON session_ui_state(session_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_session_ui_state_unique ON session_ui_state(session_id, component_name);

-- Session analytics tables
CREATE TABLE IF NOT EXISTS session_interactions (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    interaction_type TEXT NOT NULL, -- 'message_sent', 'agent_mentioned', 'command_executed'
    agent_id TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    duration_ms INTEGER,
    metadata JSON,
    FOREIGN KEY (session_id) REFERENCES session_data(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_session_interactions_session_id ON session_interactions(session_id);
CREATE INDEX IF NOT EXISTS idx_session_interactions_type ON session_interactions(interaction_type);
CREATE INDEX IF NOT EXISTS idx_session_interactions_timestamp ON session_interactions(timestamp);

CREATE TABLE IF NOT EXISTS session_metrics (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    metric_name TEXT NOT NULL, -- 'message_count', 'duration', 'agent_switches'
    metric_value REAL NOT NULL,
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata JSON,
    FOREIGN KEY (session_id) REFERENCES session_data(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_session_metrics_session_id ON session_metrics(session_id);
CREATE INDEX IF NOT EXISTS idx_session_metrics_name ON session_metrics(metric_name);
CREATE INDEX IF NOT EXISTS idx_session_metrics_recorded_at ON session_metrics(recorded_at);

-- Performance monitoring tables
CREATE TABLE IF NOT EXISTS performance_profiles (
    id TEXT PRIMARY KEY,
    session_id TEXT,
    agent_id TEXT,
    profile_type TEXT NOT NULL, -- 'cpu', 'memory', 'trace'
    duration_ms INTEGER NOT NULL,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    status TEXT DEFAULT 'running', -- 'running', 'completed', 'failed'
    metadata JSON,
    FOREIGN KEY (session_id) REFERENCES session_data(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_performance_profiles_session_id ON performance_profiles(session_id);
CREATE INDEX IF NOT EXISTS idx_performance_profiles_type ON performance_profiles(profile_type);
CREATE INDEX IF NOT EXISTS idx_performance_profiles_started_at ON performance_profiles(started_at);

CREATE TABLE IF NOT EXISTS performance_hotspots (
    id TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL,
    function_name TEXT NOT NULL,
    file_path TEXT,
    line_number INTEGER,
    cpu_time_ms REAL,
    cpu_percentage REAL,
    call_count INTEGER,
    avg_duration_ms REAL,
    severity TEXT, -- 'low', 'medium', 'high', 'critical'
    FOREIGN KEY (profile_id) REFERENCES performance_profiles(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_performance_hotspots_profile_id ON performance_hotspots(profile_id);
CREATE INDEX IF NOT EXISTS idx_performance_hotspots_severity ON performance_hotspots(severity);
CREATE INDEX IF NOT EXISTS idx_performance_hotspots_cpu_percentage ON performance_hotspots(cpu_percentage);

CREATE TABLE IF NOT EXISTS performance_optimizations (
    id TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL,
    optimization_type TEXT NOT NULL, -- 'inlining', 'caching', 'memory_pool'
    description TEXT NOT NULL,
    impact_level TEXT, -- 'low', 'medium', 'high', 'critical'
    difficulty_level TEXT, -- 'low', 'medium', 'high', 'expert'
    confidence REAL, -- 0.0 to 1.0
    estimated_gain TEXT,
    code_sample TEXT,
    FOREIGN KEY (profile_id) REFERENCES performance_profiles(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_performance_optimizations_profile_id ON performance_optimizations(profile_id);
CREATE INDEX IF NOT EXISTS idx_performance_optimizations_impact ON performance_optimizations(impact_level);
CREATE INDEX IF NOT EXISTS idx_performance_optimizations_confidence ON performance_optimizations(confidence);

-- Cache performance tables
CREATE TABLE IF NOT EXISTS cache_metrics (
    id TEXT PRIMARY KEY,
    cache_name TEXT NOT NULL,
    cache_level TEXT NOT NULL, -- 'l1', 'l2', 'distributed'
    hit_count INTEGER DEFAULT 0,
    miss_count INTEGER DEFAULT 0,
    hit_rate REAL GENERATED ALWAYS AS (
        CASE 
            WHEN (hit_count + miss_count) > 0 
            THEN CAST(hit_count AS REAL) / (hit_count + miss_count)
            ELSE 0.0 
        END
    ) STORED,
    eviction_count INTEGER DEFAULT 0,
    memory_usage_bytes INTEGER DEFAULT 0,
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cache_metrics_name ON cache_metrics(cache_name);
CREATE INDEX IF NOT EXISTS idx_cache_metrics_recorded_at ON cache_metrics(recorded_at);
CREATE INDEX IF NOT EXISTS idx_cache_metrics_hit_rate ON cache_metrics(hit_rate);

-- Monitoring and alerting tables
CREATE TABLE IF NOT EXISTS monitoring_alerts (
    id TEXT PRIMARY KEY,
    alert_type TEXT NOT NULL, -- 'slo_violation', 'performance_degradation', 'error_rate'
    severity TEXT NOT NULL, -- 'low', 'medium', 'high', 'critical'
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    source_component TEXT, -- 'performance-profiler', 'cache-manager', 'session-manager'
    triggered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    status TEXT DEFAULT 'active', -- 'active', 'acknowledged', 'resolved'
    metadata JSON
);

CREATE INDEX IF NOT EXISTS idx_monitoring_alerts_type ON monitoring_alerts(alert_type);
CREATE INDEX IF NOT EXISTS idx_monitoring_alerts_severity ON monitoring_alerts(severity);
CREATE INDEX IF NOT EXISTS idx_monitoring_alerts_triggered_at ON monitoring_alerts(triggered_at);
CREATE INDEX IF NOT EXISTS idx_monitoring_alerts_status ON monitoring_alerts(status);

CREATE TABLE IF NOT EXISTS slo_violations (
    id TEXT PRIMARY KEY,
    slo_name TEXT NOT NULL,
    target_value REAL NOT NULL,
    actual_value REAL NOT NULL,
    error_budget REAL,
    window_duration_minutes INTEGER,
    detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    metadata JSON
);

CREATE INDEX IF NOT EXISTS idx_slo_violations_name ON slo_violations(slo_name);
CREATE INDEX IF NOT EXISTS idx_slo_violations_detected_at ON slo_violations(detected_at);
CREATE INDEX IF NOT EXISTS idx_slo_violations_resolved_at ON slo_violations(resolved_at);

-- System performance metrics
CREATE TABLE IF NOT EXISTS system_metrics (
    id TEXT PRIMARY KEY,
    metric_name TEXT NOT NULL, -- 'cpu_usage', 'memory_usage', 'response_time_p95'
    metric_value REAL NOT NULL,
    metric_unit TEXT, -- 'percent', 'bytes', 'milliseconds'
    component TEXT, -- 'chat-service', 'agent-manager', 'session-service'
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata JSON
);

CREATE INDEX IF NOT EXISTS idx_system_metrics_name ON system_metrics(metric_name);
CREATE INDEX IF NOT EXISTS idx_system_metrics_component ON system_metrics(component);
CREATE INDEX IF NOT EXISTS idx_system_metrics_recorded_at ON system_metrics(recorded_at);

-- Views for common queries
CREATE VIEW IF NOT EXISTS session_summary AS
SELECT 
    s.id,
    s.user_id,
    s.campaign_id,
    s.created_at,
    s.updated_at,
    COUNT(sm.id) as message_count,
    MAX(sm.timestamp) as last_message_at,
    COUNT(DISTINCT sm.agent_id) as agent_count
FROM session_data s
LEFT JOIN session_messages sm ON s.id = sm.session_id
GROUP BY s.id, s.user_id, s.campaign_id, s.created_at, s.updated_at;

CREATE VIEW IF NOT EXISTS performance_summary AS
SELECT 
    pp.id,
    pp.session_id,
    pp.agent_id,
    pp.profile_type,
    pp.duration_ms,
    pp.started_at,
    pp.completed_at,
    COUNT(ph.id) as hotspot_count,
    COUNT(po.id) as optimization_count,
    AVG(ph.cpu_percentage) as avg_cpu_percentage
FROM performance_profiles pp
LEFT JOIN performance_hotspots ph ON pp.id = ph.profile_id
LEFT JOIN performance_optimizations po ON pp.id = po.profile_id
WHERE pp.status = 'completed'
GROUP BY pp.id, pp.session_id, pp.agent_id, pp.profile_type, pp.duration_ms, pp.started_at, pp.completed_at;

-- Cleanup old data triggers
CREATE TRIGGER IF NOT EXISTS cleanup_old_session_data
    AFTER INSERT ON session_data
    WHEN NEW.expires_at IS NOT NULL
    BEGIN
        DELETE FROM session_data WHERE expires_at < datetime('now') AND id != NEW.id;
    END;

-- Performance data retention (keep 30 days)
CREATE TRIGGER IF NOT EXISTS cleanup_old_performance_data
    AFTER INSERT ON performance_profiles
    BEGIN
        DELETE FROM performance_profiles WHERE started_at < datetime('now', '-30 days') AND id != NEW.id;
    END;

-- Alert data retention (keep 90 days)
CREATE TRIGGER IF NOT EXISTS cleanup_old_alerts
    AFTER INSERT ON monitoring_alerts
    BEGIN
        DELETE FROM monitoring_alerts WHERE triggered_at < datetime('now', '-90 days') AND id != NEW.id;
    END;