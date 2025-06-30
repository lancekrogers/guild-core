// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package permissions

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCraftPermissionModel_PredefinedRoles(t *testing.T) {
	ctx := context.Background()
	pm := NewPermissionModel(ctx)

	// Test read-only role exists
	readOnlyRole, err := pm.GetRole("read-only")
	require.NoError(t, err)
	assert.Equal(t, "read-only", readOnlyRole.ID)
	assert.Equal(t, "Read Only", readOnlyRole.Name)
	assert.Len(t, readOnlyRole.Permissions, 4) // file:read, git:log, git:status, git:show

	// Test developer role exists and inherits
	developerRole, err := pm.GetRole("developer")
	require.NoError(t, err)
	assert.Equal(t, "developer", developerRole.ID)
	assert.Contains(t, developerRole.Inherits, "read-only")
	assert.Len(t, developerRole.Permissions, 5) // file:write, git:commit, git:branch, shell:safe, file:execute

	// Test architect role exists
	architectRole, err := pm.GetRole("architect")
	require.NoError(t, err)
	assert.Equal(t, "architect", architectRole.ID)
	assert.Contains(t, architectRole.Inherits, "developer")

	// Test admin role exists
	adminRole, err := pm.GetRole("admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", adminRole.ID)
	assert.Contains(t, adminRole.Inherits, "architect")
}

func TestGuildPermissionModel_BasicPermissionCheck(t *testing.T) {
	ctx := context.Background()
	pm := NewPermissionModel(ctx)

	// Assign developer role to agent
	err := pm.AssignRole(ctx, "agent1", "developer", "test")
	require.NoError(t, err)

	// Test file read permission (inherited from read-only)
	decision := pm.CheckPermission(ctx, "agent1", "file:/project/main.go", "read")
	assert.True(t, decision.Allowed)
	assert.Equal(t, "permission granted", decision.Reason)

	// Test file write permission (developer role)
	decision = pm.CheckPermission(ctx, "agent1", "file:/project/main.go", "write")
	assert.True(t, decision.Allowed)

	// Test file delete permission (not allowed for developer)
	decision = pm.CheckPermission(ctx, "agent1", "file:/project/main.go", "delete")
	assert.False(t, decision.Allowed)
	assert.Equal(t, "no matching permission found", decision.Reason)
}

func TestJourneymanPermissionModel_RoleInheritance(t *testing.T) {
	ctx := context.Background()
	pm := NewPermissionModel(ctx)

	// Assign architect role to agent
	err := pm.AssignRole(ctx, "agent1", "architect", "test")
	require.NoError(t, err)

	// Should inherit read-only permissions
	decision := pm.CheckPermission(ctx, "agent1", "file:/project/main.go", "read")
	assert.True(t, decision.Allowed)

	// Should inherit developer permissions
	decision = pm.CheckPermission(ctx, "agent1", "file:/project/main.go", "write")
	assert.True(t, decision.Allowed)

	decision = pm.CheckPermission(ctx, "agent1", "git:main", "commit")
	assert.True(t, decision.Allowed)

	// Should have architect permissions
	decision = pm.CheckPermission(ctx, "agent1", "file:/project/main.go", "delete")
	assert.True(t, decision.Allowed)

	decision = pm.CheckPermission(ctx, "agent1", "git:origin", "push")
	assert.True(t, decision.Allowed)
}

func TestCraftPermissionModel_ExplicitDeny(t *testing.T) {
	ctx := context.Background()
	pm := NewPermissionModel(ctx)

	// Create a role with explicit deny
	restrictedRole := Role{
		ID:   "restricted",
		Name: "Restricted Developer",
		Permissions: []Permission{
			{
				ID:       "allow-file-read",
				Resource: "file:*",
				Action:   "read",
				Effect:   EffectAllow,
				Priority: 100,
			},
			{
				ID:       "deny-config-write",
				Resource: "file:config/*",
				Action:   "write",
				Effect:   EffectDeny,
				Priority: 200, // Higher priority than allow
			},
		},
	}

	err := pm.AddRole(ctx, restrictedRole)
	require.NoError(t, err)

	err = pm.AssignRole(ctx, "agent1", "restricted", "test")
	require.NoError(t, err)

	// Should allow general file read
	decision := pm.CheckPermission(ctx, "agent1", "file:/project/main.go", "read")
	assert.True(t, decision.Allowed)

	// Should deny config file write (explicit deny takes precedence)
	decision = pm.CheckPermission(ctx, "agent1", "file:config/app.yaml", "write")
	assert.False(t, decision.Allowed)
	assert.Contains(t, decision.Reason, "explicitly denied")
}

func TestGuildPermissionModel_SafeCommandCondition(t *testing.T) {
	ctx := context.Background()
	pm := NewPermissionModel(ctx)

	// Assign developer role (has safe shell permission)
	err := pm.AssignRole(ctx, "agent1", "developer", "test")
	require.NoError(t, err)

	// Safe command should be allowed
	decision := pm.CheckPermission(ctx, "agent1", "shell:ls -la", "execute")
	assert.True(t, decision.Allowed)

	// Dangerous command should be denied
	decision = pm.CheckPermission(ctx, "agent1", "shell:rm -rf /", "execute")
	assert.False(t, decision.Allowed)

	decision = pm.CheckPermission(ctx, "agent1", "shell:sudo apt-get install", "execute")
	assert.False(t, decision.Allowed)
}

func TestJourneymanPermissionModel_MultipleRoles(t *testing.T) {
	ctx := context.Background()
	pm := NewPermissionModel(ctx)

	// Create a test role
	testRole := Role{
		ID:   "tester",
		Name: "Tester",
		Permissions: []Permission{
			{
				ID:       "test-run",
				Resource: "test:*",
				Action:   "*",
				Effect:   EffectAllow,
				Priority: 200,
			},
		},
	}

	err := pm.AddRole(ctx, testRole)
	require.NoError(t, err)

	// Assign multiple roles
	err = pm.AssignRole(ctx, "agent1", "developer", "test")
	require.NoError(t, err)

	err = pm.AssignRole(ctx, "agent1", "tester", "test")
	require.NoError(t, err)

	// Should have developer permissions
	decision := pm.CheckPermission(ctx, "agent1", "file:/project/main.go", "write")
	assert.True(t, decision.Allowed)

	// Should have tester permissions
	decision = pm.CheckPermission(ctx, "agent1", "test:unit", "run")
	assert.True(t, decision.Allowed)

	// Check agent roles
	roles := pm.GetAgentRoles("agent1")
	assert.Contains(t, roles, "developer")
	assert.Contains(t, roles, "tester")
}

func TestCraftPermissionModel_CustomRole(t *testing.T) {
	ctx := context.Background()
	pm := NewPermissionModel(ctx)

	// Create custom role
	customRole := Role{
		ID:          "custom",
		Name:        "Custom Role",
		Description: "Custom permissions for testing",
		Permissions: []Permission{
			{
				ID:       "custom-api-access",
				Resource: "api:github.com",
				Action:   "request",
				Effect:   EffectAllow,
				Priority: 100,
			},
		},
	}

	err := pm.AddRole(ctx, customRole)
	require.NoError(t, err)

	err = pm.AssignRole(ctx, "agent1", "custom", "test")
	require.NoError(t, err)

	decision := pm.CheckPermission(ctx, "agent1", "api:github.com", "request")
	assert.True(t, decision.Allowed)

	decision = pm.CheckPermission(ctx, "agent1", "api:other.com", "request")
	assert.False(t, decision.Allowed)
}

func TestGuildPermissionModel_RoleValidation(t *testing.T) {
	ctx := context.Background()
	pm := NewPermissionModel(ctx)

	// Test invalid role (empty ID)
	invalidRole := Role{
		Name: "Invalid Role",
	}

	err := pm.AddRole(ctx, invalidRole)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "role ID cannot be empty")

	// Test role with invalid permission
	roleWithInvalidPerm := Role{
		ID:   "invalid-perm",
		Name: "Role with Invalid Permission",
		Permissions: []Permission{
			{
				Resource: "", // Invalid empty resource
				Action:   "read",
				Effect:   EffectAllow,
			},
		},
	}

	err = pm.AddRole(ctx, roleWithInvalidPerm)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission resource cannot be empty")
}

func TestJourneymanPermissionModel_CircularInheritance(t *testing.T) {
	ctx := context.Background()
	pm := NewPermissionModel(ctx)

	// Create role A that inherits from B
	roleA := Role{
		ID:       "role-a",
		Name:     "Role A",
		Inherits: []string{"role-b"},
	}

	// Create role B that inherits from A (circular)
	roleB := Role{
		ID:       "role-b",
		Name:     "Role B",
		Inherits: []string{"role-a"},
	}

	// Add role A first
	err := pm.AddRole(ctx, roleA)
	assert.Error(t, err) // Should fail because role-b doesn't exist

	// Add role B first
	err = pm.AddRole(ctx, roleB)
	assert.Error(t, err) // Should fail because role-a doesn't exist

	// Create valid roles first, then try to create circular dependency
	roleC := Role{
		ID:   "role-c",
		Name: "Role C",
	}

	roleD := Role{
		ID:   "role-d",
		Name: "Role D",
	}

	err = pm.AddRole(ctx, roleC)
	require.NoError(t, err)

	err = pm.AddRole(ctx, roleD)
	require.NoError(t, err)

	// Now try to create circular inheritance by updating
	roleC.Inherits = []string{"role-d"}
	err = pm.UpdateRole(ctx, roleC)
	require.NoError(t, err)

	roleD.Inherits = []string{"role-c"}
	err = pm.UpdateRole(ctx, roleD)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular inheritance")
}

func TestCraftPermissionModel_NoRoleAssignment(t *testing.T) {
	ctx := context.Background()
	pm := NewPermissionModel(ctx)

	// Check permission for agent with no roles
	decision := pm.CheckPermission(ctx, "unassigned-agent", "file:/project/main.go", "read")
	assert.False(t, decision.Allowed)
	assert.Contains(t, decision.Reason, "no roles assigned")
}

func TestGuildPermissionModel_RoleRevocation(t *testing.T) {
	ctx := context.Background()
	pm := NewPermissionModel(ctx)

	// Assign role
	err := pm.AssignRole(ctx, "agent1", "developer", "test")
	require.NoError(t, err)

	// Verify permission exists
	decision := pm.CheckPermission(ctx, "agent1", "file:/project/main.go", "write")
	assert.True(t, decision.Allowed)

	// Revoke role
	err = pm.RevokeRole(ctx, "agent1", "developer")
	require.NoError(t, err)

	// Verify permission is gone
	decision = pm.CheckPermission(ctx, "agent1", "file:/project/main.go", "write")
	assert.False(t, decision.Allowed)

	// Verify no roles assigned
	roles := pm.GetAgentRoles("agent1")
	assert.Empty(t, roles)
}

func TestScribePermissionModel_RoleDeletion(t *testing.T) {
	ctx := context.Background()
	pm := NewPermissionModel(ctx)

	// Create custom role
	customRole := Role{
		ID:   "deletable",
		Name: "Deletable Role",
	}

	err := pm.AddRole(ctx, customRole)
	require.NoError(t, err)

	// Try to delete role that's assigned
	err = pm.AssignRole(ctx, "agent1", "deletable", "test")
	require.NoError(t, err)

	err = pm.DeleteRole(ctx, "deletable")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "role is assigned to agents")

	// Revoke assignment and try again
	err = pm.RevokeRole(ctx, "agent1", "deletable")
	require.NoError(t, err)

	err = pm.DeleteRole(ctx, "deletable")
	require.NoError(t, err)

	// Verify role is gone
	_, err = pm.GetRole("deletable")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "role not found")
}

func TestJourneymanPermissionModel_Performance(t *testing.T) {
	ctx := context.Background()
	pm := NewPermissionModel(ctx)

	// Assign roles
	err := pm.AssignRole(ctx, "agent1", "architect", "test")
	require.NoError(t, err)

	// Measure permission check performance
	start := time.Now()
	for i := 0; i < 1000; i++ {
		decision := pm.CheckPermission(ctx, "agent1", "file:/project/main.go", "write")
		assert.True(t, decision.Allowed)
	}
	duration := time.Since(start)

	// Should be fast (less than 1ms per check on average)
	avgDuration := duration / 1000
	assert.Less(t, avgDuration, time.Millisecond, "Permission checks should be fast")

	t.Logf("Average permission check time: %v", avgDuration)
}

func TestCraftPermissionModel_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	pm := NewPermissionModel(ctx)

	// Assign role
	err := pm.AssignRole(ctx, "agent1", "developer", "test")
	require.NoError(t, err)

	// Run concurrent permission checks
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				decision := pm.CheckPermission(ctx, "agent1", "file:/project/main.go", "write")
				assert.True(t, decision.Allowed)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestGuildPermissionModel_ContextCancellation(t *testing.T) {
	ctx := context.Background()
	pm := NewPermissionModel(ctx)

	// Create cancelled context
	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()

	// Permission check should handle cancellation gracefully
	decision := pm.CheckPermission(cancelledCtx, "agent1", "file:/project/main.go", "read")
	assert.False(t, decision.Allowed)
	assert.Contains(t, decision.Reason, "context cancelled")
}

// Benchmark permission checks
func BenchmarkCraftPermissionCheck_Developer(b *testing.B) {
	ctx := context.Background()
	pm := NewPermissionModel(ctx)

	err := pm.AssignRole(ctx, "agent1", "developer", "test")
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pm.CheckPermission(ctx, "agent1", "file:/project/main.go", "write")
	}
}

func BenchmarkJourneymanPermissionCheck_Architect(b *testing.B) {
	ctx := context.Background()
	pm := NewPermissionModel(ctx)

	err := pm.AssignRole(ctx, "agent1", "architect", "test")
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pm.CheckPermission(ctx, "agent1", "file:/project/main.go", "delete")
	}
}
