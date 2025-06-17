-- Template management queries for SQLC generation

-- name: CreateTemplate :one
INSERT INTO templates (
    id, name, description, category, content, language, is_built_in
) VALUES (
    ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: GetTemplate :one
SELECT * FROM templates 
WHERE id = ?;

-- name: GetTemplateByName :one
SELECT * FROM templates 
WHERE name = ?;

-- name: ListTemplates :many
SELECT * FROM templates 
ORDER BY use_count DESC, name ASC;

-- name: ListTemplatesByCategory :many
SELECT * FROM templates 
WHERE category = ?
ORDER BY use_count DESC, name ASC;

-- name: SearchTemplates :many
SELECT * FROM templates 
WHERE name LIKE '%' || ?1 || '%' 
   OR description LIKE '%' || ?1 || '%' 
   OR content LIKE '%' || ?1 || '%'
ORDER BY use_count DESC, name ASC;

-- name: UpdateTemplate :one
UPDATE templates 
SET name = ?, description = ?, category = ?, content = ?, language = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteTemplate :exec
DELETE FROM templates 
WHERE id = ?;

-- name: GetTemplateUsageCount :one
SELECT use_count FROM templates 
WHERE id = ?;

-- name: IncrementTemplateUsage :exec
UPDATE templates 
SET use_count = use_count + 1 
WHERE id = ?;

-- Template Variables queries

-- name: CreateTemplateVariable :one
INSERT INTO template_variables (
    id, template_id, name, description, default_value, required, variable_type, options
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: GetTemplateVariables :many
SELECT * FROM template_variables 
WHERE template_id = ?
ORDER BY name ASC;

-- name: UpdateTemplateVariable :one
UPDATE template_variables 
SET name = ?, description = ?, default_value = ?, required = ?, variable_type = ?, options = ?
WHERE id = ?
RETURNING *;

-- name: DeleteTemplateVariable :exec
DELETE FROM template_variables 
WHERE id = ?;

-- name: DeleteTemplateVariablesByTemplate :exec
DELETE FROM template_variables 
WHERE template_id = ?;

-- Template Categories queries

-- name: CreateTemplateCategory :one
INSERT INTO template_categories (
    id, name, description, icon, sort_order
) VALUES (
    ?, ?, ?, ?, ?
)
RETURNING *;

-- name: ListTemplateCategories :many
SELECT * FROM template_categories 
ORDER BY sort_order ASC, name ASC;

-- name: GetTemplateCategory :one
SELECT * FROM template_categories 
WHERE id = ?;

-- name: UpdateTemplateCategory :one
UPDATE template_categories 
SET name = ?, description = ?, icon = ?, sort_order = ?
WHERE id = ?
RETURNING *;

-- name: DeleteTemplateCategory :exec
DELETE FROM template_categories 
WHERE id = ?;

-- Template Usage tracking queries

-- name: CreateTemplateUsage :one
INSERT INTO template_usage (
    id, template_id, campaign_id, variables_used, context
) VALUES (
    ?, ?, ?, ?, ?
)
RETURNING *;

-- name: GetTemplateUsageHistory :many
SELECT tu.*, t.name as template_name 
FROM template_usage tu
JOIN templates t ON tu.template_id = t.id
WHERE tu.campaign_id = ?
ORDER BY tu.used_at DESC
LIMIT ?;

-- name: GetMostUsedTemplates :many
SELECT t.*, COUNT(tu.id) as recent_usage_count
FROM templates t
LEFT JOIN template_usage tu ON t.id = tu.template_id 
    AND tu.used_at > datetime('now', '-30 days')
GROUP BY t.id
ORDER BY recent_usage_count DESC, t.use_count DESC
LIMIT ?;

-- name: GetTemplateStats :one
SELECT 
    COUNT(*) as total_templates,
    COUNT(DISTINCT category) as total_categories,
    AVG(use_count) as avg_use_count,
    MAX(use_count) as max_use_count
FROM templates;

-- Combined queries for efficient data fetching

-- name: GetTemplateWithVariables :many
SELECT 
    t.id as template_id,
    t.name as template_name,
    t.description as template_description,
    t.category as template_category,
    t.content as template_content,
    t.language as template_language,
    t.use_count as template_use_count,
    t.is_built_in as template_is_built_in,
    t.created_at as template_created_at,
    t.updated_at as template_updated_at,
    tv.id as variable_id,
    tv.name as variable_name,
    tv.description as variable_description,
    tv.default_value as variable_default_value,
    tv.required as variable_required,
    tv.variable_type as variable_type,
    tv.options as variable_options
FROM templates t
LEFT JOIN template_variables tv ON t.id = tv.template_id
WHERE t.id = ?
ORDER BY tv.name ASC;