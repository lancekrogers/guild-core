// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package permissions

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
)

// PermissionModel implements a hierarchical permission system with role inheritance
type PermissionModel struct {
	roles       map[string]*Role
	policies    map[string]*Policy
	assignments map[string]*Assignment // agent ID -> assignment
	mu          sync.RWMutex
	logger      observability.Logger
}

// NewPermissionModel creates a new permission model with default roles
func NewPermissionModel(ctx context.Context) *PermissionModel {
	logger := observability.GetLogger(ctx).
		WithComponent("PermissionModel")

	pm := &PermissionModel{
		roles:       make(map[string]*Role),
		policies:    make(map[string]*Policy),
		assignments: make(map[string]*Assignment),
		logger:      logger,
	}

	// Add predefined roles
	pm.addPredefinedRoles()

	logger.Info("Permission model initialized with predefined roles")
	return pm
}

// addPredefinedRoles adds the default system roles
func (pm *PermissionModel) addPredefinedRoles() {
	now := time.Now()

	// Read-only role
	readOnlyRole := &Role{
		ID:          "read-only",
		Name:        "Read Only",
		Description: "Can read files and view git status",
		Permissions: []Permission{
			{
				ID:       "read-only-file-read",
				Resource: "file:*",
				Action:   "read",
				Effect:   EffectAllow,
				Priority: 100,
			},
			{
				ID:       "read-only-git-log",
				Resource: "git:*",
				Action:   "log",
				Effect:   EffectAllow,
				Priority: 100,
			},
			{
				ID:       "read-only-git-status",
				Resource: "git:*",
				Action:   "status",
				Effect:   EffectAllow,
				Priority: 100,
			},
			{
				ID:       "read-only-git-show",
				Resource: "git:*",
				Action:   "show",
				Effect:   EffectAllow,
				Priority: 100,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	pm.roles[readOnlyRole.ID] = readOnlyRole

	// Developer role
	developerRole := &Role{
		ID:          "developer",
		Name:        "Developer",
		Description: "Can read/write files, commit code, and run safe shell commands",
		Inherits:    []string{"read-only"},
		Permissions: []Permission{
			{
				ID:       "developer-file-write",
				Resource: "file:*",
				Action:   "write",
				Effect:   EffectAllow,
				Priority: 200,
			},
			{
				ID:       "developer-git-commit",
				Resource: "git:*",
				Action:   "commit",
				Effect:   EffectAllow,
				Priority: 200,
			},
			{
				ID:       "developer-git-branch",
				Resource: "git:*",
				Action:   "branch",
				Effect:   EffectAllow,
				Priority: 200,
			},
			{
				ID:         "developer-shell-safe",
				Resource:   "shell:*",
				Action:     "execute",
				Effect:     EffectAllow,
				Conditions: []Condition{NewSafeCommandCondition()},
				Priority:   200,
			},
			{
				ID:       "developer-file-execute",
				Resource: "file:*",
				Action:   "execute",
				Effect:   EffectAllow,
				Priority: 200,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	pm.roles[developerRole.ID] = developerRole

	// Architect role
	architectRole := &Role{
		ID:          "architect",
		Name:        "Architect",
		Description: "Full development access including file deletion, git push, and package management",
		Inherits:    []string{"developer"},
		Permissions: []Permission{
			{
				ID:       "architect-file-delete",
				Resource: "file:*",
				Action:   "delete",
				Effect:   EffectAllow,
				Priority: 300,
			},
			{
				ID:       "architect-git-push",
				Resource: "git:*",
				Action:   "push",
				Effect:   EffectAllow,
				Priority: 300,
			},
			{
				ID:       "architect-package-install",
				Resource: "package:*",
				Action:   "install",
				Effect:   EffectAllow,
				Priority: 300,
			},
			{
				ID:       "architect-database-all",
				Resource: "database:*",
				Action:   "*",
				Effect:   EffectAllow,
				Priority: 300,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	pm.roles[architectRole.ID] = architectRole

	// Admin role
	adminRole := &Role{
		ID:          "admin",
		Name:        "Administrator",
		Description: "Full system access with no restrictions",
		Inherits:    []string{"architect"},
		Permissions: []Permission{
			{
				ID:       "admin-all-access",
				Resource: "*",
				Action:   "*",
				Effect:   EffectAllow,
				Priority: 1000,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	pm.roles[adminRole.ID] = adminRole
}

// CheckPermission evaluates whether an agent has permission to perform an action on a resource
func (pm *PermissionModel) CheckPermission(ctx context.Context, agentID, resource, action string) Decision {
	startTime := time.Now()
	logger := pm.logger.WithOperation("CheckPermission")

	if err := ctx.Err(); err != nil {
		return Decision{
			Allowed:        false,
			Reason:         "context cancelled",
			EvaluationTime: time.Since(startTime),
			Timestamp:      time.Now(),
		}
	}

	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Get agent's assignment
	assignment, exists := pm.assignments[agentID]
	if !exists || assignment.IsExpired() {
		logger.Debug("No valid role assignment found for agent", "agent_id", agentID)
		return Decision{
			Allowed:        false,
			Reason:         "no roles assigned or assignment expired",
			EvaluationTime: time.Since(startTime),
			Timestamp:      time.Now(),
		}
	}

	// Collect all permissions (with inheritance)
	permissions := pm.collectAllPermissions(assignment.RoleIDs)
	logger.Debug("Collected permissions", "count", len(permissions), "agent_id", agentID)

	// Apply policies
	policyPermissions := pm.collectPolicyPermissions(agentID)
	permissions = append(permissions, policyPermissions...)

	// Create evaluation context
	evalCtx := EvaluationContext{
		Agent:     agentID,
		Resource:  resource,
		Action:    action,
		Time:      time.Now(),
		RequestID: getRequestID(ctx),
		Metadata:  make(map[string]interface{}),
	}

	// Add agent roles to context for role-based conditions
	evalCtx.Metadata["agent_roles"] = assignment.RoleIDs

	// Evaluate permissions (explicit deny takes precedence)
	decision := pm.evaluatePermissions(permissions, evalCtx)
	decision.EvaluationTime = time.Since(startTime)
	decision.Timestamp = time.Now()

	logger.Debug("Permission check completed",
		"agent_id", agentID,
		"resource", resource,
		"action", action,
		"allowed", decision.Allowed,
		"reason", decision.Reason,
		"evaluation_time", decision.EvaluationTime,
	)

	return decision
}

// collectAllPermissions gathers permissions from roles with inheritance
func (pm *PermissionModel) collectAllPermissions(roleIDs []string) []Permission {
	var permissions []Permission
	visited := make(map[string]bool)

	for _, roleID := range roleIDs {
		pm.collectRolePermissions(roleID, &permissions, visited)
	}

	return permissions
}

// collectRolePermissions recursively collects permissions from a role and its inherited roles
func (pm *PermissionModel) collectRolePermissions(roleID string, permissions *[]Permission, visited map[string]bool) {
	if visited[roleID] {
		return // Prevent infinite recursion
	}
	visited[roleID] = true

	role, exists := pm.roles[roleID]
	if !exists {
		pm.logger.Warn("Role not found during permission collection", "role_id", roleID)
		return
	}

	// First collect from inherited roles (lower priority)
	for _, inheritedRoleID := range role.Inherits {
		pm.collectRolePermissions(inheritedRoleID, permissions, visited)
	}

	// Then add this role's permissions (higher priority)
	*permissions = append(*permissions, role.Permissions...)
}

// collectPolicyPermissions gathers permissions from applicable policies
func (pm *PermissionModel) collectPolicyPermissions(agentID string) []Permission {
	var permissions []Permission

	for _, policy := range pm.policies {
		if policy.AppliesTo(agentID) {
			permissions = append(permissions, policy.Permissions...)
		}
	}

	return permissions
}

// evaluatePermissions applies permission rules to make a decision
func (pm *PermissionModel) evaluatePermissions(permissions []Permission, ctx EvaluationContext) Decision {
	// Sort permissions by priority (higher priority first)
	sort.Slice(permissions, func(i, j int) bool {
		return permissions[i].Priority > permissions[j].Priority
	})

	var allowingPermissions []Permission
	var appliedPolicies []string

	for _, perm := range permissions {
		if !pm.matches(perm, ctx.Resource, ctx.Action) {
			continue
		}

		// Check conditions
		if !pm.evaluateConditions(perm.Conditions, ctx) {
			continue
		}

		// Explicit deny takes precedence
		if perm.Effect == EffectDeny {
			return Decision{
				Allowed: false,
				Reason:  fmt.Sprintf("explicitly denied by permission %s (resource: %s, action: %s)", perm.ID, perm.Resource, perm.Action),
			}
		}

		// Collect allowing permissions
		if perm.Effect == EffectAllow {
			allowingPermissions = append(allowingPermissions, perm)
		}
	}

	if len(allowingPermissions) > 0 {
		return Decision{
			Allowed:            true,
			Reason:             "permission granted",
			AppliedPermissions: allowingPermissions,
			AppliedPolicies:    appliedPolicies,
		}
	}

	return Decision{
		Allowed: false,
		Reason:  "no matching permission found",
	}
}

// matches checks if a permission applies to the given resource and action
func (pm *PermissionModel) matches(perm Permission, resource, action string) bool {
	// Action matching
	if perm.Action != "*" && perm.Action != action {
		return false
	}

	// Resource matching with wildcards
	return pm.matchesResource(perm.Resource, resource)
}

// matchesResource performs resource pattern matching supporting wildcards
func (pm *PermissionModel) matchesResource(pattern, resource string) bool {
	// Exact match
	if pattern == resource {
		return true
	}

	// Universal wildcard
	if pattern == "*" {
		return true
	}

	// Empty pattern should not match anything except empty resource
	if len(pattern) == 0 {
		return len(resource) == 0
	}

	// Suffix wildcard (e.g., "file:*" matches "file:anything")
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(resource) >= len(prefix) && resource[:len(prefix)] == prefix
	}

	// Prefix wildcard (e.g., "*:read" matches "anything:read")
	if len(pattern) > 0 && pattern[0] == '*' {
		suffix := pattern[1:]
		return len(resource) >= len(suffix) && resource[len(resource)-len(suffix):] == suffix
	}

	// Middle wildcard (e.g., "file:*/main.go" matches "file:anything/main.go")
	if idx := strings.Index(pattern, "*"); idx != -1 {
		before := pattern[:idx]
		after := pattern[idx+1:]

		if len(resource) < len(before)+len(after) {
			return false
		}

		return resource[:len(before)] == before &&
			resource[len(resource)-len(after):] == after
	}

	return false
}

// evaluateConditions checks if all conditions are satisfied
func (pm *PermissionModel) evaluateConditions(conditions []Condition, ctx EvaluationContext) bool {
	for _, condition := range conditions {
		if !condition.Evaluate(ctx) {
			return false
		}
	}
	return true
}

// AddRole adds a new role to the permission model
func (pm *PermissionModel) AddRole(ctx context.Context, role Role) error {
	logger := pm.logger.WithOperation("AddRole")

	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("PermissionModel").
			WithOperation("AddRole")
	}

	if err := role.Validate(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "role validation failed").
			WithComponent("PermissionModel").
			WithOperation("AddRole")
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if role already exists
	if _, exists := pm.roles[role.ID]; exists {
		return gerror.New(gerror.ErrCodeAlreadyExists, "role already exists", nil).
			WithComponent("PermissionModel").
			WithOperation("AddRole").
			WithDetails("role_id", role.ID)
	}

	// Validate inheritance chain
	if err := pm.validateInheritanceChain(role.ID, role.Inherits); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid inheritance chain").
			WithComponent("PermissionModel").
			WithOperation("AddRole")
	}

	role.CreatedAt = time.Now()
	role.UpdatedAt = time.Now()
	pm.roles[role.ID] = &role

	logger.Info("Role added successfully", "role_id", role.ID, "role_name", role.Name)
	return nil
}

// UpdateRole modifies an existing role
func (pm *PermissionModel) UpdateRole(ctx context.Context, role Role) error {
	logger := pm.logger.WithOperation("UpdateRole")

	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("PermissionModel").
			WithOperation("UpdateRole")
	}

	if err := role.Validate(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "role validation failed").
			WithComponent("PermissionModel").
			WithOperation("UpdateRole")
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if role exists
	existing, exists := pm.roles[role.ID]
	if !exists {
		return gerror.New(gerror.ErrCodeNotFound, "role not found", nil).
			WithComponent("PermissionModel").
			WithOperation("UpdateRole").
			WithDetails("role_id", role.ID)
	}

	// Validate inheritance chain
	if err := pm.validateInheritanceChain(role.ID, role.Inherits); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid inheritance chain").
			WithComponent("PermissionModel").
			WithOperation("UpdateRole")
	}

	role.CreatedAt = existing.CreatedAt
	role.UpdatedAt = time.Now()
	pm.roles[role.ID] = &role

	logger.Info("Role updated successfully", "role_id", role.ID, "role_name", role.Name)
	return nil
}

// DeleteRole removes a role from the permission model
func (pm *PermissionModel) DeleteRole(ctx context.Context, roleID string) error {
	logger := pm.logger.WithOperation("DeleteRole")

	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("PermissionModel").
			WithOperation("DeleteRole")
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if role exists
	if _, exists := pm.roles[roleID]; !exists {
		return gerror.New(gerror.ErrCodeNotFound, "role not found", nil).
			WithComponent("PermissionModel").
			WithOperation("DeleteRole").
			WithDetails("role_id", roleID)
	}

	// Check if role is being used
	for agentID, assignment := range pm.assignments {
		for _, assignedRoleID := range assignment.RoleIDs {
			if assignedRoleID == roleID {
				return gerror.New(gerror.ErrCodeAlreadyExists, "role is assigned to agents", nil).
					WithComponent("PermissionModel").
					WithOperation("DeleteRole").
					WithDetails("role_id", roleID).
					WithDetails("agent_id", agentID)
			}
		}
	}

	// Check if role is inherited by other roles
	for _, role := range pm.roles {
		for _, inheritedRoleID := range role.Inherits {
			if inheritedRoleID == roleID {
				return gerror.New(gerror.ErrCodeAlreadyExists, "role is inherited by other roles", nil).
					WithComponent("PermissionModel").
					WithOperation("DeleteRole").
					WithDetails("role_id", roleID).
					WithDetails("inheriting_role", role.ID)
			}
		}
	}

	delete(pm.roles, roleID)

	logger.Info("Role deleted successfully", "role_id", roleID)
	return nil
}

// validateInheritanceChain ensures there are no circular dependencies
func (pm *PermissionModel) validateInheritanceChain(roleID string, inherits []string) error {
	visited := make(map[string]bool)
	return pm.checkCircularInheritance(roleID, inherits, visited)
}

// checkCircularInheritance recursively checks for circular inheritance
func (pm *PermissionModel) checkCircularInheritance(currentRole string, inherits []string, visited map[string]bool) error {
	if visited[currentRole] {
		return gerror.New(gerror.ErrCodeValidation, "circular inheritance detected", nil).
			WithComponent("PermissionModel").
			WithOperation("checkCircularInheritance").
			WithDetails("role_id", currentRole)
	}

	visited[currentRole] = true

	for _, inheritedRole := range inherits {
		if inheritedRole == currentRole {
			return gerror.New(gerror.ErrCodeValidation, "role cannot inherit from itself", nil).
				WithComponent("PermissionModel").
				WithOperation("checkCircularInheritance").
				WithDetails("role_id", currentRole)
		}

		role, exists := pm.roles[inheritedRole]
		if !exists {
			return gerror.New(gerror.ErrCodeValidation, "inherited role does not exist", nil).
				WithComponent("PermissionModel").
				WithOperation("checkCircularInheritance").
				WithDetails("missing_role", inheritedRole)
		}

		if err := pm.checkCircularInheritance(inheritedRole, role.Inherits, visited); err != nil {
			return err
		}
	}

	delete(visited, currentRole)
	return nil
}

// AssignRole assigns a role to an agent
func (pm *PermissionModel) AssignRole(ctx context.Context, agentID, roleID string, grantedBy string) error {
	logger := pm.logger.WithOperation("AssignRole")

	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("PermissionModel").
			WithOperation("AssignRole")
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Validate role exists
	if _, exists := pm.roles[roleID]; !exists {
		return gerror.New(gerror.ErrCodeNotFound, "role not found", nil).
			WithComponent("PermissionModel").
			WithOperation("AssignRole").
			WithDetails("role_id", roleID)
	}

	// Get or create assignment
	assignment, exists := pm.assignments[agentID]
	if !exists {
		assignment = &Assignment{
			AgentID:   agentID,
			RoleIDs:   []string{},
			GrantedBy: grantedBy,
			GrantedAt: time.Now(),
		}
	}

	// Check if role already assigned
	for _, existingRoleID := range assignment.RoleIDs {
		if existingRoleID == roleID {
			return gerror.New(gerror.ErrCodeAlreadyExists, "role already assigned", nil).
				WithComponent("PermissionModel").
				WithOperation("AssignRole").
				WithDetails("agent_id", agentID).
				WithDetails("role_id", roleID)
		}
	}

	// Add role to assignment
	assignment.RoleIDs = append(assignment.RoleIDs, roleID)
	pm.assignments[agentID] = assignment

	logger.Info("Role assigned successfully",
		"agent_id", agentID,
		"role_id", roleID,
		"granted_by", grantedBy,
	)

	return nil
}

// RevokeRole removes a role from an agent
func (pm *PermissionModel) RevokeRole(ctx context.Context, agentID, roleID string) error {
	logger := pm.logger.WithOperation("RevokeRole")

	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("PermissionModel").
			WithOperation("RevokeRole")
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	assignment, exists := pm.assignments[agentID]
	if !exists {
		return gerror.New(gerror.ErrCodeNotFound, "no role assignment found", nil).
			WithComponent("PermissionModel").
			WithOperation("RevokeRole").
			WithDetails("agent_id", agentID)
	}

	// Find and remove role
	for i, assignedRoleID := range assignment.RoleIDs {
		if assignedRoleID == roleID {
			// Remove role from slice
			assignment.RoleIDs = append(assignment.RoleIDs[:i], assignment.RoleIDs[i+1:]...)

			// If no roles left, remove assignment
			if len(assignment.RoleIDs) == 0 {
				delete(pm.assignments, agentID)
			}

			logger.Info("Role revoked successfully", "agent_id", agentID, "role_id", roleID)
			return nil
		}
	}

	return gerror.New(gerror.ErrCodeNotFound, "role not assigned to agent", nil).
		WithComponent("PermissionModel").
		WithOperation("RevokeRole").
		WithDetails("agent_id", agentID).
		WithDetails("role_id", roleID)
}

// GetAgentRoles returns the roles assigned to an agent
func (pm *PermissionModel) GetAgentRoles(agentID string) []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	assignment, exists := pm.assignments[agentID]
	if !exists || assignment.IsExpired() {
		return []string{}
	}

	// Make a copy to avoid external modification
	roles := make([]string, len(assignment.RoleIDs))
	copy(roles, assignment.RoleIDs)
	return roles
}

// GetRole returns a role by ID
func (pm *PermissionModel) GetRole(roleID string) (*Role, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	role, exists := pm.roles[roleID]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "role not found", nil).
			WithComponent("PermissionModel").
			WithOperation("GetRole").
			WithDetails("role_id", roleID)
	}

	// Return a copy to avoid external modification
	roleCopy := *role
	return &roleCopy, nil
}

// ListRoles returns all available roles
func (pm *PermissionModel) ListRoles() []*Role {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	roles := make([]*Role, 0, len(pm.roles))
	for _, role := range pm.roles {
		roleCopy := *role
		roles = append(roles, &roleCopy)
	}

	return roles
}

// getRequestID extracts the request ID from context
func getRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}
	return ""
}
