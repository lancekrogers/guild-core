-- Agent performance tracking for manager intelligence
-- Stores historical performance data for intelligent task assignment

-- Agent performance metrics
CREATE TABLE agent_performance (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id),
    tasks_completed INTEGER DEFAULT 0,
    average_time_hours REAL DEFAULT 0.0,
    success_rate REAL DEFAULT 0.0,
    complexity_handle REAL DEFAULT 1.0, -- avg complexity handled
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Agent specialty proficiency scores
CREATE TABLE agent_specialties (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id),
    specialty TEXT NOT NULL, -- backend, frontend, testing, etc.
    proficiency REAL NOT NULL DEFAULT 0.5, -- 0.0-1.0 proficiency score
    tasks_completed INTEGER DEFAULT 0,
    success_rate REAL DEFAULT 0.0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(agent_id, specialty)
);

-- Agent capability scores
CREATE TABLE agent_capabilities (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id),
    capability TEXT NOT NULL, -- coding, testing, documentation, etc.
    proficiency REAL NOT NULL DEFAULT 0.5, -- 0.0-1.0 proficiency score
    tasks_completed INTEGER DEFAULT 0,
    success_rate REAL DEFAULT 0.0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(agent_id, capability)
);

-- Agent availability tracking
CREATE TABLE agent_availability (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id),
    current_load REAL DEFAULT 0.0, -- 0.0-1.0 current workload
    active_tasks INTEGER DEFAULT 0,
    max_concurrent_tasks INTEGER DEFAULT 3,
    last_assignment TIMESTAMP,
    status TEXT DEFAULT 'available' CHECK (status IN ('available', 'busy', 'offline')),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(agent_id)
);

-- Task assignment history for learning
CREATE TABLE task_assignments (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(id),
    agent_id TEXT NOT NULL REFERENCES agents(id),
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    success BOOLEAN,
    complexity_rating INTEGER DEFAULT 1,
    performance_score REAL DEFAULT 0.5,
    notes TEXT
);

-- Performance indexes
CREATE INDEX idx_agent_performance_agent ON agent_performance(agent_id);
CREATE INDEX idx_agent_specialties_agent ON agent_specialties(agent_id);
CREATE INDEX idx_agent_specialties_specialty ON agent_specialties(specialty);
CREATE INDEX idx_agent_capabilities_agent ON agent_capabilities(agent_id);
CREATE INDEX idx_agent_capabilities_capability ON agent_capabilities(capability);
CREATE INDEX idx_agent_availability_agent ON agent_availability(agent_id);
CREATE INDEX idx_agent_availability_status ON agent_availability(status);
CREATE INDEX idx_task_assignments_task ON task_assignments(task_id);
CREATE INDEX idx_task_assignments_agent ON task_assignments(agent_id);
CREATE INDEX idx_task_assignments_completed ON task_assignments(completed_at);

-- Trigger to update agent_performance.updated_at
CREATE TRIGGER update_agent_performance_timestamp
AFTER UPDATE ON agent_performance
BEGIN
    UPDATE agent_performance 
    SET updated_at = CURRENT_TIMESTAMP 
    WHERE id = NEW.id;
END;

-- Trigger to update agent_specialties.updated_at
CREATE TRIGGER update_agent_specialties_timestamp
AFTER UPDATE ON agent_specialties
BEGIN
    UPDATE agent_specialties 
    SET updated_at = CURRENT_TIMESTAMP 
    WHERE id = NEW.id;
END;

-- Trigger to update agent_capabilities.updated_at
CREATE TRIGGER update_agent_capabilities_timestamp
AFTER UPDATE ON agent_capabilities
BEGIN
    UPDATE agent_capabilities 
    SET updated_at = CURRENT_TIMESTAMP 
    WHERE id = NEW.id;
END;

-- Trigger to update agent_availability.updated_at
CREATE TRIGGER update_agent_availability_timestamp
AFTER UPDATE ON agent_availability
BEGIN
    UPDATE agent_availability 
    SET updated_at = CURRENT_TIMESTAMP 
    WHERE id = NEW.id;
END;