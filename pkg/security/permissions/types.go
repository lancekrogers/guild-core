// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package permissions

import (
	"fmt"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// Effect represents whether a permission allows or denies access
type Effect int

const (
	EffectAllow Effect = iota
	EffectDeny
)

func (e Effect) String() string {
	switch e {
	case EffectAllow:
		return "allow"
	case EffectDeny:
		return "deny"
	default:
		return "unknown"
	}
}

// Permission defines a specific access control rule
type Permission struct {
	ID         string      `json:"id"`
	Resource   string      `json:"resource"`   // e.g., "file:*", "git:commit", "shell:*"
	Action     string      `json:"action"`     // e.g., "read", "write", "execute", "*"
	Effect     Effect      `json:"effect"`     // Allow or Deny
	Conditions []Condition `json:"conditions"` // Additional constraints
	Priority   int         `json:"priority"`   // Higher numbers take precedence
}

// Validate ensures the permission is well-formed
func (p Permission) Validate() error {
	if p.Resource == "" {
		return gerror.New(gerror.ErrCodeValidation, "permission resource cannot be empty", nil).
			WithComponent("Permission").
			WithOperation("Validate")
	}

	if p.Action == "" {
		return gerror.New(gerror.ErrCodeValidation, "permission action cannot be empty", nil).
			WithComponent("Permission").
			WithOperation("Validate")
	}

	if p.Effect != EffectAllow && p.Effect != EffectDeny {
		return gerror.New(gerror.ErrCodeValidation, "permission effect must be allow or deny", nil).
			WithComponent("Permission").
			WithOperation("Validate")
	}

	return nil
}

// Role defines a collection of permissions that can be assigned to agents
type Role struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Permissions []Permission `json:"permissions"`
	Inherits    []string     `json:"inherits"` // Role IDs this role inherits from
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// Validate ensures the role is well-formed
func (r Role) Validate() error {
	if r.ID == "" {
		return gerror.New(gerror.ErrCodeValidation, "role ID cannot be empty", nil).
			WithComponent("Role").
			WithOperation("Validate")
	}

	if r.Name == "" {
		return gerror.New(gerror.ErrCodeValidation, "role name cannot be empty", nil).
			WithComponent("Role").
			WithOperation("Validate")
	}

	// Validate all permissions
	for i, perm := range r.Permissions {
		if err := perm.Validate(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeValidation,
				fmt.Sprintf("invalid permission at index %d", i)).
				WithComponent("Role").
				WithOperation("Validate")
		}
	}

	return nil
}

// Policy defines conditional permissions that can be applied based on context
type Policy struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Conditions  []Condition  `json:"conditions"`  // When this policy applies
	Permissions []Permission `json:"permissions"` // Permissions granted/denied when policy applies
	Priority    int          `json:"priority"`    // Higher numbers take precedence
	Enabled     bool         `json:"enabled"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// AppliesTo checks if this policy should be applied to the given agent
func (p Policy) AppliesTo(agentID string) bool {
	if !p.Enabled {
		return false
	}

	ctx := EvaluationContext{
		Agent: agentID,
		Time:  time.Now(),
	}

	for _, condition := range p.Conditions {
		if !condition.Evaluate(ctx) {
			return false
		}
	}

	return true
}

// Validate ensures the policy is well-formed
func (p Policy) Validate() error {
	if p.ID == "" {
		return gerror.New(gerror.ErrCodeValidation, "policy ID cannot be empty", nil).
			WithComponent("Policy").
			WithOperation("Validate")
	}

	if p.Name == "" {
		return gerror.New(gerror.ErrCodeValidation, "policy name cannot be empty", nil).
			WithComponent("Policy").
			WithOperation("Validate")
	}

	// Validate all permissions
	for i, perm := range p.Permissions {
		if err := perm.Validate(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeValidation,
				fmt.Sprintf("invalid permission at index %d", i)).
				WithComponent("Policy").
				WithOperation("Validate")
		}
	}

	return nil
}

// Decision represents the result of a permission check
type Decision struct {
	Allowed            bool          `json:"allowed"`
	Reason             string        `json:"reason"`
	AppliedPermissions []Permission  `json:"applied_permissions,omitempty"`
	AppliedPolicies    []string      `json:"applied_policies,omitempty"`
	EvaluationTime     time.Duration `json:"evaluation_time"`
	Timestamp          time.Time     `json:"timestamp"`
}

// EvaluationContext provides information for permission and condition evaluation
type EvaluationContext struct {
	Agent     string                 `json:"agent"`
	User      string                 `json:"user,omitempty"`
	Resource  string                 `json:"resource"`
	Action    string                 `json:"action"`
	Time      time.Time              `json:"time"`
	IPAddress string                 `json:"ip_address,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Condition defines an interface for evaluating contextual constraints
type Condition interface {
	// Evaluate returns true if the condition is satisfied
	Evaluate(ctx EvaluationContext) bool

	// Name returns a human-readable name for this condition
	Name() string

	// Description returns a detailed description of what this condition checks
	Description() string
}

// Assignment represents the association between an agent and roles
type Assignment struct {
	AgentID   string     `json:"agent_id"`
	RoleIDs   []string   `json:"role_ids"`
	GrantedBy string     `json:"granted_by"`
	GrantedAt time.Time  `json:"granted_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// IsExpired checks if this assignment has expired
func (a Assignment) IsExpired() bool {
	return a.ExpiresAt != nil && time.Now().After(*a.ExpiresAt)
}

// Validate ensures the assignment is well-formed
func (a Assignment) Validate() error {
	if a.AgentID == "" {
		return gerror.New(gerror.ErrCodeValidation, "assignment agent ID cannot be empty", nil).
			WithComponent("Assignment").
			WithOperation("Validate")
	}

	if len(a.RoleIDs) == 0 {
		return gerror.New(gerror.ErrCodeValidation, "assignment must have at least one role", nil).
			WithComponent("Assignment").
			WithOperation("Validate")
	}

	return nil
}
