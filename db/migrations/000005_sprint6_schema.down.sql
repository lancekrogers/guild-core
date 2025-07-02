-- Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
-- SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

-- Migration 005 Rollback: Remove performance optimization Schema Extensions

-- Drop triggers first
DROP TRIGGER IF EXISTS cleanup_old_alerts;
DROP TRIGGER IF EXISTS cleanup_old_performance_data;
DROP TRIGGER IF EXISTS cleanup_old_session_data;

-- Drop views
DROP VIEW IF EXISTS performance_summary;
DROP VIEW IF EXISTS session_summary;

-- Drop system metrics tables
DROP TABLE IF EXISTS system_metrics;

-- Drop monitoring tables
DROP TABLE IF EXISTS slo_violations;
DROP TABLE IF EXISTS monitoring_alerts;

-- Drop cache tables
DROP TABLE IF EXISTS cache_metrics;

-- Drop performance tables
DROP TABLE IF EXISTS performance_optimizations;
DROP TABLE IF EXISTS performance_hotspots;
DROP TABLE IF EXISTS performance_profiles;

-- Drop session analytics tables
DROP TABLE IF EXISTS session_metrics;
DROP TABLE IF EXISTS session_interactions;

-- Drop session management tables
DROP TABLE IF EXISTS session_ui_state;
DROP TABLE IF EXISTS session_messages;
DROP TABLE IF EXISTS session_data;