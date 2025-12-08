// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package preferences

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/storage"
)

// Service provides high-level preference management with caching and validation
type Service struct {
	repo  storage.PreferencesRepository
	cache *PreferenceCache
	mu    sync.RWMutex
}

// NewService creates a new preference service
func NewService(repo storage.PreferencesRepository) *Service {
	return &Service{
		repo:  repo,
		cache: NewPreferenceCache(5*time.Minute, 10*time.Minute),
	}
}

// GetSystemPreference retrieves a system-wide preference
func (s *Service) GetSystemPreference(ctx context.Context, key string) (interface{}, error) {
	return s.getPreference(ctx, "system", nil, key)
}

// SetSystemPreference sets a system-wide preference
func (s *Service) SetSystemPreference(ctx context.Context, key string, value interface{}) error {
	return s.setPreference(ctx, "system", nil, key, value)
}

// GetUserPreference retrieves a user preference
func (s *Service) GetUserPreference(ctx context.Context, userID, key string) (interface{}, error) {
	return s.getPreference(ctx, "user", &userID, key)
}

// SetUserPreference sets a user preference
func (s *Service) SetUserPreference(ctx context.Context, userID, key string, value interface{}) error {
	return s.setPreference(ctx, "user", &userID, key, value)
}

// GetCampaignPreference retrieves a campaign preference
func (s *Service) GetCampaignPreference(ctx context.Context, campaignID, key string) (interface{}, error) {
	return s.getPreference(ctx, "campaign", &campaignID, key)
}

// SetCampaignPreference sets a campaign preference
func (s *Service) SetCampaignPreference(ctx context.Context, campaignID, key string, value interface{}) error {
	return s.setPreference(ctx, "campaign", &campaignID, key, value)
}

// GetGuildPreference retrieves a guild preference
func (s *Service) GetGuildPreference(ctx context.Context, guildID, key string) (interface{}, error) {
	return s.getPreference(ctx, "guild", &guildID, key)
}

// SetGuildPreference sets a guild preference
func (s *Service) SetGuildPreference(ctx context.Context, guildID, key string, value interface{}) error {
	return s.setPreference(ctx, "guild", &guildID, key, value)
}

// GetAgentPreference retrieves an agent preference
func (s *Service) GetAgentPreference(ctx context.Context, agentID, key string) (interface{}, error) {
	return s.getPreference(ctx, "agent", &agentID, key)
}

// SetAgentPreference sets an agent preference
func (s *Service) SetAgentPreference(ctx context.Context, agentID, key string, value interface{}) error {
	return s.setPreference(ctx, "agent", &agentID, key, value)
}

// ResolvePreference resolves a preference through the inheritance hierarchy
func (s *Service) ResolvePreference(ctx context.Context, key string, agentID, guildID, campaignID, userID *string) (interface{}, error) {
	// Build scope chain
	scopes := []storage.PreferenceScope{}

	// Add scopes in priority order (most specific first)
	if agentID != nil {
		scopes = append(scopes, storage.PreferenceScope{Scope: "agent", ScopeID: agentID})
	}
	if guildID != nil {
		scopes = append(scopes, storage.PreferenceScope{Scope: "guild", ScopeID: guildID})
	}
	if campaignID != nil {
		scopes = append(scopes, storage.PreferenceScope{Scope: "campaign", ScopeID: campaignID})
	}
	if userID != nil {
		scopes = append(scopes, storage.PreferenceScope{Scope: "user", ScopeID: userID})
	}
	// System scope is always included (added by repository)

	// Check cache for resolved preference
	cacheKey := s.buildCacheKey("resolved", key, scopes...)
	if cached, found := s.cache.Get(cacheKey); found {
		return cached, nil
	}

	// Resolve through repository
	pref, err := s.repo.ResolvePreference(ctx, key, scopes)
	if err != nil {
		return nil, err
	}

	// Cache the result
	s.cache.Set(cacheKey, pref.Value)

	return pref.Value, nil
}

// GetPreferences retrieves multiple preferences for a scope
func (s *Service) GetPreferences(ctx context.Context, scope string, scopeID *string, keys []string) (map[string]interface{}, error) {
	prefs, err := s.repo.GetPreferencesByKeys(ctx, scope, scopeID, keys)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	for _, pref := range prefs {
		result[pref.Key] = pref.Value
	}

	// Add default values for missing keys
	for _, key := range keys {
		if _, ok := result[key]; !ok {
			if defaultVal, exists := DefaultPreferences[key]; exists {
				result[key] = defaultVal
			}
		}
	}

	return result, nil
}

// SetPreferences sets multiple preferences at once
func (s *Service) SetPreferences(ctx context.Context, scope string, scopeID *string, prefs map[string]interface{}) error {
	// Validate preferences
	for key, value := range prefs {
		if err := s.validatePreference(key, value); err != nil {
			return err
		}
	}

	// Clear cache for affected scope
	s.cache.Clear()

	return s.repo.SetPreferences(ctx, scope, scopeID, prefs)
}

// ExportPreferences exports all preferences for a scope
func (s *Service) ExportPreferences(ctx context.Context, scope string, scopeID *string) ([]byte, error) {
	prefs, err := s.repo.ExportPreferences(ctx, scope, scopeID)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(prefs, "", "  ")
}

// ImportPreferences imports preferences from JSON
func (s *Service) ImportPreferences(ctx context.Context, scope string, scopeID *string, data []byte) error {
	var prefs map[string]interface{}
	if err := json.Unmarshal(data, &prefs); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid JSON data").
			WithComponent("PreferenceService").
			WithOperation("ImportPreferences")
	}

	// Clear cache before import
	s.cache.Clear()

	return s.repo.ImportPreferences(ctx, scope, scopeID, prefs)
}

// DeletePreferencesByScope deletes all preferences for a scope
func (s *Service) DeletePreferencesByScope(ctx context.Context, scope string, scopeID *string) error {
	// Clear cache
	s.cache.Clear()

	return s.repo.DeletePreferencesByScope(ctx, scope, scopeID)
}

// Internal helper methods

func (s *Service) getPreference(ctx context.Context, scope string, scopeID *string, key string) (interface{}, error) {
	// Check cache first
	cacheKey := s.buildCacheKey(scope, key, storage.PreferenceScope{Scope: scope, ScopeID: scopeID})
	if cached, found := s.cache.Get(cacheKey); found {
		return cached, nil
	}

	// Get from repository
	pref, err := s.repo.GetPreferenceByKey(ctx, scope, scopeID, key)
	if err != nil {
		// If not found, check for default value
		if gerr, ok := err.(*gerror.GuildError); ok && gerr.Code == gerror.ErrCodeNotFound {
			if defaultVal, exists := DefaultPreferences[key]; exists {
				return defaultVal, nil
			}
		}
		return nil, err
	}

	// Cache the result
	s.cache.Set(cacheKey, pref.Value)

	return pref.Value, nil
}

func (s *Service) setPreference(ctx context.Context, scope string, scopeID *string, key string, value interface{}) error {
	// Validate preference
	if err := s.validatePreference(key, value); err != nil {
		return err
	}

	// Clear cache for this key
	cacheKey := s.buildCacheKey(scope, key, storage.PreferenceScope{Scope: scope, ScopeID: scopeID})
	s.cache.Delete(cacheKey)

	// Check if preference exists
	existing, err := s.repo.GetPreferenceByKey(ctx, scope, scopeID, key)
	if err != nil && !isNotFound(err) {
		return err
	}

	if existing != nil {
		// Update existing
		existing.Value = value
		return s.repo.UpdatePreference(ctx, existing)
	}

	// Create new
	pref := &storage.Preference{
		Scope:   scope,
		ScopeID: scopeID,
		Key:     key,
		Value:   value,
		Version: 1,
	}

	return s.repo.CreatePreference(ctx, pref)
}

func (s *Service) validatePreference(key string, value interface{}) error {
	// Check if key has validation rules
	if validator, exists := PreferenceValidators[key]; exists {
		if err := validator(value); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInvalidInput, "preference validation failed").
				WithComponent("PreferenceService").
				WithOperation("validatePreference").
				WithDetails("key", key)
		}
	}

	// Basic type validation for known preferences
	if expectedType, exists := PreferenceTypes[key]; exists {
		if !isValidType(value, expectedType) {
			return gerror.New(gerror.ErrCodeInvalidInput, "invalid preference type", nil).
				WithComponent("PreferenceService").
				WithOperation("validatePreference").
				WithDetails("key", key).
				WithDetails("expected_type", expectedType).
				WithDetails("actual_type", fmt.Sprintf("%T", value))
		}
	}

	return nil
}

func (s *Service) buildCacheKey(scope, key string, scopes ...storage.PreferenceScope) string {
	if len(scopes) == 0 {
		return fmt.Sprintf("%s:%s", scope, key)
	}

	// For resolved preferences, include all scopes in cache key
	cacheKey := fmt.Sprintf("resolved:%s", key)
	for _, s := range scopes {
		if s.ScopeID != nil {
			cacheKey += fmt.Sprintf(":%s:%s", s.Scope, *s.ScopeID)
		} else {
			cacheKey += fmt.Sprintf(":%s:_", s.Scope)
		}
	}
	return cacheKey
}

func isNotFound(err error) bool {
	if gerr, ok := err.(*gerror.GuildError); ok {
		return gerr.Code == gerror.ErrCodeNotFound
	}
	return false
}

func isValidType(value interface{}, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "int":
		switch value.(type) {
		case int, int32, int64, float64:
			return true
		}
		return false
	case "bool":
		_, ok := value.(bool)
		return ok
	case "float":
		switch value.(type) {
		case float32, float64, int, int32, int64:
			return true
		}
		return false
	case "[]string":
		_, ok := value.([]string)
		if !ok {
			// Also accept []interface{} with string elements
			if arr, ok := value.([]interface{}); ok {
				for _, v := range arr {
					if _, ok := v.(string); !ok {
						return false
					}
				}
				return true
			}
		}
		return ok
	case "map":
		_, ok := value.(map[string]interface{})
		return ok
	default:
		return true // Unknown types are allowed
	}
}
