// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package access

import (
	"context"
	"crypto/md5"
	"fmt"
	"sync"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/observability"
	"github.com/lancekrogers/guild-core/pkg/security/permissions"
	"github.com/lancekrogers/guild-core/pkg/tools"
)

// AccessController manages runtime access control for tool execution
type AccessController struct {
	permissions *permissions.PermissionModel
	cache       *PermissionCache
	auditor     AuditLogger
	eventBus    EventBus
	logger      observability.Logger
}

// NewAccessController creates a new access controller
func NewAccessController(ctx context.Context, permissionModel *permissions.PermissionModel, auditor AuditLogger, eventBus EventBus) *AccessController {
	logger := observability.GetLogger(ctx).
		WithComponent("AccessController")

	return &AccessController{
		permissions: permissionModel,
		cache:       NewPermissionCache(ctx, 1000, 5*time.Minute),
		auditor:     auditor,
		eventBus:    eventBus,
		logger:      logger,
	}
}

// CheckAccess validates if an agent can use a specific tool with given parameters
func (ac *AccessController) CheckAccess(ctx context.Context, req AccessRequest) (*AccessDecision, error) {
	logger := ac.logger.WithOperation("CheckAccess")

	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AccessController").
			WithOperation("CheckAccess")
	}

	startTime := time.Now()

	// Build resource and action from request
	resource := ac.buildResourceIdentifier(req)
	action := req.Action

	// Check cache first
	cacheKey := ac.generateCacheKey(req.AgentID, resource, action)
	if cached := ac.cache.Get(cacheKey); cached != nil {
		logger.Debug("Permission check served from cache",
			"agent_id", req.AgentID,
			"resource", resource,
			"action", action,
			"allowed", cached.Allowed,
		)

		// Update statistics
		cached.CacheHit = true
		cached.CheckTime = time.Since(startTime)
		return cached, nil
	}

	// Perform permission check
	decision := ac.permissions.CheckPermission(ctx, req.AgentID, resource, action)

	// Convert to access decision
	accessDecision := &AccessDecision{
		Allowed:    decision.Allowed,
		Reason:     decision.Reason,
		Resource:   resource,
		Action:     action,
		AgentID:    req.AgentID,
		CheckTime:  time.Since(startTime),
		CacheHit:   false,
		Timestamp:  time.Now(),
		Conditions: req.Conditions,
	}

	// Cache the decision
	ac.cache.Set(cacheKey, accessDecision, 5*time.Minute)

	// Log the decision
	if decision.Allowed {
		logger.Debug("Access granted",
			"agent_id", req.AgentID,
			"resource", resource,
			"action", action,
			"check_time", accessDecision.CheckTime,
		)

		// Audit allowed access
		if ac.auditor != nil {
			ac.auditor.LogAllowed(ctx, AuditEntry{
				Timestamp: time.Now(),
				AgentID:   req.AgentID,
				Resource:  resource,
				Action:    action,
				Result:    "allowed",
				Reason:    decision.Reason,
				Duration:  accessDecision.CheckTime,
				RequestID: getRequestID(ctx),
				Metadata:  req.Metadata,
			})
		}
	} else {
		logger.Warn("Access denied",
			"agent_id", req.AgentID,
			"resource", resource,
			"action", action,
			"reason", decision.Reason,
			"check_time", accessDecision.CheckTime,
		)

		// Audit denied access
		if ac.auditor != nil {
			ac.auditor.LogDenied(ctx, AuditEntry{
				Timestamp: time.Now(),
				AgentID:   req.AgentID,
				Resource:  resource,
				Action:    action,
				Result:    "denied",
				Reason:    decision.Reason,
				Duration:  accessDecision.CheckTime,
				RequestID: getRequestID(ctx),
				Metadata:  req.Metadata,
			})
		}

		// Publish access denied event
		if ac.eventBus != nil {
			event := AccessEvent{
				Type:      "access.denied",
				Timestamp: time.Now(),
				AgentID:   req.AgentID,
				Resource:  resource,
				Action:    action,
				Reason:    decision.Reason,
			}
			ac.eventBus.PublishAccessEvent(ctx, event)
		}
	}

	return accessDecision, nil
}

// buildResourceIdentifier creates a standardized resource identifier from the request
func (ac *AccessController) buildResourceIdentifier(req AccessRequest) string {
	switch req.ToolName {
	case "file":
		if path, ok := req.Parameters["path"].(string); ok {
			return fmt.Sprintf("file:%s", path)
		}
		return "file:*"

	case "git":
		if command, ok := req.Parameters["command"].(string); ok {
			return fmt.Sprintf("git:%s", command)
		}
		return "git:*"

	case "shell":
		if command, ok := req.Parameters["command"].(string); ok {
			return fmt.Sprintf("shell:%s", command)
		}
		return "shell:*"

	case "database":
		if query, ok := req.Parameters["query"].(string); ok {
			return fmt.Sprintf("database:%s", query)
		}
		return "database:*"

	case "api":
		if url, ok := req.Parameters["url"].(string); ok {
			return fmt.Sprintf("api:%s", url)
		}
		return "api:*"

	case "package":
		if packageName, ok := req.Parameters["package"].(string); ok {
			return fmt.Sprintf("package:%s", packageName)
		}
		return "package:*"

	default:
		return fmt.Sprintf("%s:*", req.ToolName)
	}
}

// generateCacheKey creates a unique cache key for permission decisions
func (ac *AccessController) generateCacheKey(agentID, resource, action string) string {
	key := fmt.Sprintf("%s:%s:%s", agentID, resource, action)
	hash := fmt.Sprintf("%x", md5.Sum([]byte(key)))
	return hash
}

// InvalidateCache clears cached permissions for an agent
func (ac *AccessController) InvalidateCache(agentID string) {
	ac.cache.InvalidateAgent(agentID)
	ac.logger.Debug("Cache invalidated for agent", "agent_id", agentID)
}

// InvalidateAllCache clears all cached permissions
func (ac *AccessController) InvalidateAllCache() {
	ac.cache.Clear()
	ac.logger.Debug("All permission cache cleared")
}

// GetStats returns access control statistics
func (ac *AccessController) GetStats() AccessStats {
	return AccessStats{
		CacheStats:      ac.cache.GetStats(),
		ChecksPerformed: ac.cache.stats.totalChecks,
		CacheHitRate:    ac.cache.GetHitRate(),
	}
}

// PermissionCache provides fast access to permission decisions
type PermissionCache struct {
	cache      map[string]*cacheEntry
	mu         sync.RWMutex
	maxSize    int
	defaultTTL time.Duration
	stats      *cacheStats
}

type cacheEntry struct {
	decision  *AccessDecision
	expiresAt time.Time
}

type cacheStats struct {
	hits        int64
	misses      int64
	totalChecks int64
	evictions   int64
}

// NewPermissionCache creates a new permission cache
func NewPermissionCache(ctx context.Context, maxSize int, defaultTTL time.Duration) *PermissionCache {
	cache := &PermissionCache{
		cache:      make(map[string]*cacheEntry),
		maxSize:    maxSize,
		defaultTTL: defaultTTL,
		stats:      &cacheStats{},
	}

	// Start cleanup goroutine
	go cache.cleanupExpired(ctx)

	return cache
}

// Get retrieves a cached permission decision
func (pc *PermissionCache) Get(key string) *AccessDecision {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	pc.stats.totalChecks++

	entry, exists := pc.cache[key]
	if !exists || time.Now().After(entry.expiresAt) {
		pc.stats.misses++
		return nil
	}

	pc.stats.hits++
	return entry.decision
}

// Set stores a permission decision in cache
func (pc *PermissionCache) Set(key string, decision *AccessDecision, ttl time.Duration) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	// Evict if cache is full
	if len(pc.cache) >= pc.maxSize {
		pc.evictOldest()
	}

	pc.cache[key] = &cacheEntry{
		decision:  decision,
		expiresAt: time.Now().Add(ttl),
	}
}

// InvalidateAgent removes all cache entries for a specific agent
func (pc *PermissionCache) InvalidateAgent(agentID string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	for key, entry := range pc.cache {
		if entry.decision != nil && entry.decision.AgentID == agentID {
			delete(pc.cache, key)
		}
	}
}

// Clear removes all cache entries
func (pc *PermissionCache) Clear() {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.cache = make(map[string]*cacheEntry)
}

// GetStats returns cache statistics
func (pc *PermissionCache) GetStats() CacheStats {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	return CacheStats{
		Size:      len(pc.cache),
		MaxSize:   pc.maxSize,
		Hits:      pc.stats.hits,
		Misses:    pc.stats.misses,
		Evictions: pc.stats.evictions,
	}
}

// GetHitRate calculates the cache hit rate
func (pc *PermissionCache) GetHitRate() float64 {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	if pc.stats.totalChecks == 0 {
		return 0.0
	}

	return float64(pc.stats.hits) / float64(pc.stats.totalChecks)
}

// evictOldest removes the oldest cache entry (simple LRU approximation)
func (pc *PermissionCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range pc.cache {
		if oldestKey == "" || entry.expiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.expiresAt
		}
	}

	if oldestKey != "" {
		delete(pc.cache, oldestKey)
		pc.stats.evictions++
	}
}

// cleanupExpired removes expired entries periodically
func (pc *PermissionCache) cleanupExpired(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pc.removeExpired()
		}
	}
}

// removeExpired removes all expired cache entries
func (pc *PermissionCache) removeExpired() {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	now := time.Now()
	for key, entry := range pc.cache {
		if now.After(entry.expiresAt) {
			delete(pc.cache, key)
		}
	}
}

// ToolInterceptor wraps tool execution with permission checks
type ToolInterceptor struct {
	tool       tools.Tool
	controller *AccessController
	toolName   string
	logger     observability.Logger
}

// NewToolInterceptor creates a new tool interceptor
func NewToolInterceptor(ctx context.Context, tool tools.Tool, controller *AccessController, toolName string) *ToolInterceptor {
	logger := observability.GetLogger(ctx).
		WithComponent("ToolInterceptor")

	return &ToolInterceptor{
		tool:       tool,
		controller: controller,
		toolName:   toolName,
		logger:     logger,
	}
}

// Execute performs permission-checked tool execution
func (ti *ToolInterceptor) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	logger := ti.logger.WithOperation("Execute")

	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ToolInterceptor").
			WithOperation("Execute")
	}

	// Extract agent ID from context
	agentID, ok := ctx.Value("agent_id").(string)
	if !ok {
		return nil, gerror.New(gerror.ErrCodeValidation, "agent ID not found in context", nil).
			WithComponent("ToolInterceptor").
			WithOperation("Execute")
	}

	// Create access request
	req := AccessRequest{
		AgentID:    agentID,
		ToolName:   ti.toolName,
		Action:     "execute",
		Parameters: map[string]interface{}{"input": input},
		Timestamp:  time.Now(),
		RequestID:  getRequestID(ctx),
	}

	// Check access
	decision, err := ti.controller.CheckAccess(ctx, req)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "access check failed").
			WithComponent("ToolInterceptor").
			WithOperation("Execute")
	}

	if !decision.Allowed {
		logger.Warn("Tool execution denied",
			"agent_id", agentID,
			"tool", ti.toolName,
			"reason", decision.Reason,
		)

		return nil, gerror.New(gerror.ErrCodePermissionDenied, "permission denied", nil).
			WithComponent("ToolInterceptor").
			WithOperation("Execute").
			WithDetails("reason", decision.Reason).
			WithDetails("resource", decision.Resource)
	}

	// Execute tool with monitoring
	start := time.Now()
	result, err := ti.tool.Execute(ctx, input)
	duration := time.Since(start)

	// Log execution
	if err != nil {
		logger.WithError(err).Warn("Tool execution failed",
			"agent_id", agentID,
			"tool", ti.toolName,
			"duration", duration,
		)
	} else {
		logger.Debug("Tool execution successful",
			"agent_id", agentID,
			"tool", ti.toolName,
			"duration", duration,
		)
	}

	// Audit execution result
	if ti.controller.auditor != nil {
		ti.controller.auditor.LogExecution(ctx, AuditEntry{
			Timestamp: time.Now(),
			AgentID:   agentID,
			Resource:  fmt.Sprintf("tool:%s", ti.toolName),
			Action:    "execute",
			Result:    determineResult(err),
			Duration:  duration,
			RequestID: getRequestID(ctx),
			Metadata: map[string]interface{}{
				"tool_name": ti.toolName,
				"input":     input,
				"success":   err == nil,
			},
		})
	}

	return result, err
}

// Implement tool interface methods
func (ti *ToolInterceptor) Name() string {
	return ti.tool.Name()
}

func (ti *ToolInterceptor) Description() string {
	return ti.tool.Description()
}

func (ti *ToolInterceptor) Schema() map[string]interface{} {
	return ti.tool.Schema()
}

func (ti *ToolInterceptor) Examples() []string {
	return ti.tool.Examples()
}

func (ti *ToolInterceptor) Category() string {
	return ti.tool.Category()
}

func (ti *ToolInterceptor) RequiresAuth() bool {
	return ti.tool.RequiresAuth()
}

// Helper functions

func getRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}
	return ""
}

func determineResult(err error) string {
	if err == nil {
		return "success"
	}
	return "error"
}
