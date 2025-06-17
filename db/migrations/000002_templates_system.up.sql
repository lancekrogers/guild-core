-- Template system for Guild Framework
-- Add template storage and management capabilities

-- Templates for reusable prompts and code patterns
CREATE TABLE templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    category TEXT NOT NULL DEFAULT 'general',
    content TEXT NOT NULL,
    language TEXT, -- Optional: for code templates (go, python, etc.)
    use_count INTEGER DEFAULT 0,
    is_built_in BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Template variables for dynamic content substitution
CREATE TABLE template_variables (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    default_value TEXT,
    required BOOLEAN DEFAULT FALSE,
    variable_type TEXT DEFAULT 'text' CHECK (variable_type IN ('text', 'code', 'multiline', 'select')),
    options JSON, -- For select type variables
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(template_id, name) -- Each template can have uniquely named variables
);

-- Template categories for organization
CREATE TABLE template_categories (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    icon TEXT, -- Unicode emoji or icon name
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- User template favorites and usage tracking
CREATE TABLE template_usage (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    campaign_id TEXT REFERENCES campaigns(id),
    used_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    variables_used JSON, -- Store the values used for this instance
    context TEXT -- Optional context about where/how it was used
);

-- Performance indexes
CREATE INDEX idx_templates_category ON templates(category);
CREATE INDEX idx_templates_name ON templates(name);
CREATE INDEX idx_templates_use_count ON templates(use_count DESC);
CREATE INDEX idx_templates_is_built_in ON templates(is_built_in);
CREATE INDEX idx_template_variables_template ON template_variables(template_id);
CREATE INDEX idx_template_usage_template ON template_usage(template_id);
CREATE INDEX idx_template_usage_campaign ON template_usage(campaign_id);
CREATE INDEX idx_template_usage_used_at ON template_usage(used_at DESC);

-- Insert default categories
INSERT INTO template_categories (id, name, description, icon, sort_order) VALUES
    ('code-review', 'Code Review', 'Templates for code review requests', '🔍', 1),
    ('debugging', 'Debugging', 'Templates for bug investigation and debugging', '🐛', 2),
    ('architecture', 'Architecture', 'Templates for system design and architecture', '🏗️', 3),
    ('documentation', 'Documentation', 'Templates for writing documentation', '📚', 4),
    ('testing', 'Testing', 'Templates for test planning and creation', '🧪', 5),
    ('prompting', 'AI Prompting', 'Templates for effective AI prompting', '🤖', 6),
    ('general', 'General', 'General purpose templates', '📝', 99);

-- Trigger to update templates.updated_at on modification
CREATE TRIGGER update_template_timestamp
AFTER UPDATE ON templates
BEGIN
    UPDATE templates 
    SET updated_at = CURRENT_TIMESTAMP 
    WHERE id = NEW.id;
END;

-- Trigger to increment use_count when template is used
CREATE TRIGGER increment_template_usage
AFTER INSERT ON template_usage
BEGIN
    UPDATE templates 
    SET use_count = use_count + 1 
    WHERE id = NEW.template_id;
END;