// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// SQLitePreferencesRepository implements PreferencesRepository using SQLite
type SQLitePreferencesRepository struct {
	db *Database
}

// DefaultPreferencesRepositoryFactory creates a new SQLitePreferencesRepository
func DefaultPreferencesRepositoryFactory(database *Database) PreferencesRepository {
	return &SQLitePreferencesRepository{db: database}
}

// CreatePreference creates a new preference
func (r *SQLitePreferencesRepository) CreatePreference(ctx context.Context, pref *Preference) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("CreatePreference")
	}

	// Generate ID if not provided
	if pref.ID == "" {
		pref.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	pref.CreatedAt = now
	pref.UpdatedAt = now

	// Marshal value and metadata to JSON
	valueJSON, err := json.Marshal(pref.Value)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal preference value").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("CreatePreference").
			WithDetails("key", pref.Key)
	}

	var metadataJSON []byte
	if pref.Metadata != nil {
		metadataJSON, err = json.Marshal(pref.Metadata)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal preference metadata").
				WithComponent("SQLitePreferencesRepository").
				WithOperation("CreatePreference").
				WithDetails("key", pref.Key)
		}
	}

	query := `
		INSERT INTO preferences (id, scope, scope_id, key, value, version, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.DB().ExecContext(ctx, query,
		pref.ID, pref.Scope, pref.ScopeID, pref.Key, valueJSON, pref.Version, metadataJSON, pref.CreatedAt, pref.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return gerror.Wrap(err, gerror.ErrCodeAlreadyExists, "preference already exists").
				WithComponent("SQLitePreferencesRepository").
				WithOperation("CreatePreference").
				WithDetails("scope", pref.Scope).
				WithDetails("scope_id", pref.ScopeID).
				WithDetails("key", pref.Key)
		}
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create preference").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("CreatePreference")
	}

	return nil
}

// GetPreference retrieves a preference by ID
func (r *SQLitePreferencesRepository) GetPreference(ctx context.Context, id string) (*Preference, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("GetPreference")
	}

	query := `
		SELECT id, scope, scope_id, key, value, version, metadata, created_at, updated_at
		FROM preferences
		WHERE id = ?
	`

	pref := &Preference{}
	var valueJSON, metadataJSON []byte

	err := r.db.DB().QueryRowContext(ctx, query, id).Scan(
		&pref.ID, &pref.Scope, &pref.ScopeID, &pref.Key, &valueJSON, &pref.Version, &metadataJSON, &pref.CreatedAt, &pref.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, gerror.New(gerror.ErrCodeNotFound, "preference not found", err).
			WithComponent("SQLitePreferencesRepository").
			WithOperation("GetPreference").
			WithDetails("id", id)
	}
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get preference").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("GetPreference")
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal(valueJSON, &pref.Value); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal preference value").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("GetPreference")
	}

	if metadataJSON != nil {
		if err := json.Unmarshal(metadataJSON, &pref.Metadata); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal preference metadata").
				WithComponent("SQLitePreferencesRepository").
				WithOperation("GetPreference")
		}
	}

	return pref, nil
}

// UpdatePreference updates an existing preference with optimistic locking
func (r *SQLitePreferencesRepository) UpdatePreference(ctx context.Context, pref *Preference) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("UpdatePreference")
	}

	pref.UpdatedAt = time.Now()

	// Marshal value and metadata to JSON
	valueJSON, err := json.Marshal(pref.Value)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal preference value").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("UpdatePreference").
			WithDetails("key", pref.Key)
	}

	var metadataJSON []byte
	if pref.Metadata != nil {
		metadataJSON, err = json.Marshal(pref.Metadata)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal preference metadata").
				WithComponent("SQLitePreferencesRepository").
				WithOperation("UpdatePreference").
				WithDetails("key", pref.Key)
		}
	}

	query := `
		UPDATE preferences
		SET value = ?, version = version + 1, metadata = ?, updated_at = ?
		WHERE id = ? AND version = ?
	`

	result, err := r.db.DB().ExecContext(ctx, query, valueJSON, metadataJSON, pref.UpdatedAt, pref.ID, pref.Version)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to update preference").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("UpdatePreference")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get rows affected").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("UpdatePreference")
	}

	if rowsAffected == 0 {
		return gerror.New(gerror.ErrCodeConflict, "preference version conflict", nil).
			WithComponent("SQLitePreferencesRepository").
			WithOperation("UpdatePreference").
			WithDetails("id", pref.ID).
			WithDetails("version", pref.Version)
	}

	pref.Version++
	return nil
}

// DeletePreference deletes a preference by ID
func (r *SQLitePreferencesRepository) DeletePreference(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("DeletePreference")
	}

	query := `DELETE FROM preferences WHERE id = ?`
	_, err := r.db.DB().ExecContext(ctx, query, id)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to delete preference").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("DeletePreference")
	}

	return nil
}

// GetPreferenceByKey retrieves a preference by scope and key
func (r *SQLitePreferencesRepository) GetPreferenceByKey(ctx context.Context, scope string, scopeID *string, key string) (*Preference, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("GetPreferenceByKey")
	}

	query := `
		SELECT id, scope, scope_id, key, value, version, metadata, created_at, updated_at
		FROM preferences
		WHERE scope = ? AND (scope_id = ? OR (scope_id IS NULL AND ? IS NULL)) AND key = ?
	`

	pref := &Preference{}
	var valueJSON, metadataJSON []byte

	err := r.db.DB().QueryRowContext(ctx, query, scope, scopeID, scopeID, key).Scan(
		&pref.ID, &pref.Scope, &pref.ScopeID, &pref.Key, &valueJSON, &pref.Version, &metadataJSON, &pref.CreatedAt, &pref.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, gerror.New(gerror.ErrCodeNotFound, "preference not found", err).
			WithComponent("SQLitePreferencesRepository").
			WithOperation("GetPreferenceByKey").
			WithDetails("scope", scope).
			WithDetails("key", key)
	}
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get preference by key").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("GetPreferenceByKey")
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal(valueJSON, &pref.Value); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal preference value").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("GetPreferenceByKey")
	}

	if metadataJSON != nil {
		if err := json.Unmarshal(metadataJSON, &pref.Metadata); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal preference metadata").
				WithComponent("SQLitePreferencesRepository").
				WithOperation("GetPreferenceByKey")
		}
	}

	return pref, nil
}

// ListPreferencesByScope lists all preferences for a given scope
func (r *SQLitePreferencesRepository) ListPreferencesByScope(ctx context.Context, scope string, scopeID *string) ([]*Preference, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("ListPreferencesByScope")
	}

	query := `
		SELECT id, scope, scope_id, key, value, version, metadata, created_at, updated_at
		FROM preferences
		WHERE scope = ? AND (scope_id = ? OR (scope_id IS NULL AND ? IS NULL))
		ORDER BY key
	`

	rows, err := r.db.DB().QueryContext(ctx, query, scope, scopeID, scopeID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to list preferences by scope").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("ListPreferencesByScope")
	}
	defer rows.Close()

	var prefs []*Preference
	for rows.Next() {
		pref := &Preference{}
		var valueJSON, metadataJSON []byte

		err := rows.Scan(&pref.ID, &pref.Scope, &pref.ScopeID, &pref.Key, &valueJSON, &pref.Version, &metadataJSON, &pref.CreatedAt, &pref.UpdatedAt)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to scan preference row").
				WithComponent("SQLitePreferencesRepository").
				WithOperation("ListPreferencesByScope")
		}

		// Unmarshal JSON fields
		if err := json.Unmarshal(valueJSON, &pref.Value); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal preference value").
				WithComponent("SQLitePreferencesRepository").
				WithOperation("ListPreferencesByScope")
		}

		if metadataJSON != nil {
			if err := json.Unmarshal(metadataJSON, &pref.Metadata); err != nil {
				return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal preference metadata").
					WithComponent("SQLitePreferencesRepository").
					WithOperation("ListPreferencesByScope")
			}
		}

		prefs = append(prefs, pref)
	}

	return prefs, nil
}

// ListPreferencesByKey lists all preferences with a given key across all scopes
func (r *SQLitePreferencesRepository) ListPreferencesByKey(ctx context.Context, key string) ([]*Preference, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("ListPreferencesByKey")
	}

	query := `
		SELECT id, scope, scope_id, key, value, version, metadata, created_at, updated_at
		FROM preferences
		WHERE key = ?
		ORDER BY scope, scope_id
	`

	rows, err := r.db.DB().QueryContext(ctx, query, key)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to list preferences by key").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("ListPreferencesByKey")
	}
	defer rows.Close()

	var prefs []*Preference
	for rows.Next() {
		pref := &Preference{}
		var valueJSON, metadataJSON []byte

		err := rows.Scan(&pref.ID, &pref.Scope, &pref.ScopeID, &pref.Key, &valueJSON, &pref.Version, &metadataJSON, &pref.CreatedAt, &pref.UpdatedAt)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to scan preference row").
				WithComponent("SQLitePreferencesRepository").
				WithOperation("ListPreferencesByKey")
		}

		// Unmarshal JSON fields
		if err := json.Unmarshal(valueJSON, &pref.Value); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal preference value").
				WithComponent("SQLitePreferencesRepository").
				WithOperation("ListPreferencesByKey")
		}

		if metadataJSON != nil {
			if err := json.Unmarshal(metadataJSON, &pref.Metadata); err != nil {
				return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal preference metadata").
					WithComponent("SQLitePreferencesRepository").
					WithOperation("ListPreferencesByKey")
			}
		}

		prefs = append(prefs, pref)
	}

	return prefs, nil
}

// ResolvePreference resolves a preference through the inheritance hierarchy
func (r *SQLitePreferencesRepository) ResolvePreference(ctx context.Context, key string, scopes []PreferenceScope) (*Preference, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("ResolvePreference")
	}

	// Build the inheritance chain
	inheritanceChain := r.buildInheritanceChain(scopes)

	// Try each scope in the chain until we find a value
	for _, scope := range inheritanceChain {
		pref, err := r.GetPreferenceByKey(ctx, scope.Scope, scope.ScopeID, key)
		if err == nil {
			return pref, nil
		}
		// If error is not "not found", return it
		if !strings.Contains(err.Error(), "not found") {
			return nil, err
		}
	}

	return nil, gerror.New(gerror.ErrCodeNotFound, "preference not found in any scope", nil).
		WithComponent("SQLitePreferencesRepository").
		WithOperation("ResolvePreference").
		WithDetails("key", key)
}

// buildInheritanceChain builds the complete inheritance chain for resolution
func (r *SQLitePreferencesRepository) buildInheritanceChain(scopes []PreferenceScope) []PreferenceScope {
	// Default inheritance order: Agent -> Guild -> Campaign -> User -> System
	scopeOrder := map[string]int{
		"agent":    0,
		"guild":    1,
		"campaign": 2,
		"user":     3,
		"system":   4,
	}

	// Sort scopes by priority
	sorted := make([]PreferenceScope, len(scopes))
	copy(sorted, scopes)
	sort.Slice(sorted, func(i, j int) bool {
		return scopeOrder[sorted[i].Scope] < scopeOrder[sorted[j].Scope]
	})

	// Always append system scope at the end
	hasSystem := false
	for _, s := range sorted {
		if s.Scope == "system" {
			hasSystem = true
			break
		}
	}
	if !hasSystem {
		sorted = append(sorted, PreferenceScope{Scope: "system", ScopeID: nil})
	}

	return sorted
}

// GetPreferencesByKeys retrieves multiple preferences by keys
func (r *SQLitePreferencesRepository) GetPreferencesByKeys(ctx context.Context, scope string, scopeID *string, keys []string) ([]*Preference, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("GetPreferencesByKeys")
	}

	if len(keys) == 0 {
		return []*Preference{}, nil
	}

	// Build query with placeholders
	placeholders := make([]string, len(keys))
	args := make([]interface{}, 0, len(keys)+2)
	args = append(args, scope, scopeID, scopeID)
	for i, key := range keys {
		placeholders[i] = "?"
		args = append(args, key)
	}

	query := fmt.Sprintf(`
		SELECT id, scope, scope_id, key, value, version, metadata, created_at, updated_at
		FROM preferences
		WHERE scope = ? AND (scope_id = ? OR (scope_id IS NULL AND ? IS NULL)) AND key IN (%s)
		ORDER BY key
	`, strings.Join(placeholders, ","))

	rows, err := r.db.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get preferences by keys").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("GetPreferencesByKeys")
	}
	defer rows.Close()

	var prefs []*Preference
	for rows.Next() {
		pref := &Preference{}
		var valueJSON, metadataJSON []byte

		err := rows.Scan(&pref.ID, &pref.Scope, &pref.ScopeID, &pref.Key, &valueJSON, &pref.Version, &metadataJSON, &pref.CreatedAt, &pref.UpdatedAt)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to scan preference row").
				WithComponent("SQLitePreferencesRepository").
				WithOperation("GetPreferencesByKeys")
		}

		// Unmarshal JSON fields
		if err := json.Unmarshal(valueJSON, &pref.Value); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal preference value").
				WithComponent("SQLitePreferencesRepository").
				WithOperation("GetPreferencesByKeys")
		}

		if metadataJSON != nil {
			if err := json.Unmarshal(metadataJSON, &pref.Metadata); err != nil {
				return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal preference metadata").
					WithComponent("SQLitePreferencesRepository").
					WithOperation("GetPreferencesByKeys")
			}
		}

		prefs = append(prefs, pref)
	}

	return prefs, nil
}

// SetPreferences sets multiple preferences at once
func (r *SQLitePreferencesRepository) SetPreferences(ctx context.Context, scope string, scopeID *string, prefs map[string]interface{}) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("SetPreferences")
	}

	// Start transaction
	tx, err := r.db.DB().BeginTx(ctx, nil)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to begin transaction").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("SetPreferences")
	}
	defer tx.Rollback()

	for key, value := range prefs {
		// Check if preference exists within transaction
		existing, err := r.getPreferenceByKeyInTx(ctx, tx, scope, scopeID, key)
		if err != nil && !strings.Contains(err.Error(), "not found") {
			return err
		}

		if existing != nil {
			// Update existing
			existing.Value = value
			if err := r.updatePreferenceInTx(ctx, tx, existing); err != nil {
				return err
			}
		} else {
			// Create new
			pref := &Preference{
				ID:      uuid.New().String(),
				Scope:   scope,
				ScopeID: scopeID,
				Key:     key,
				Value:   value,
				Version: 1,
			}
			if err := r.createPreferenceInTx(ctx, tx, pref); err != nil {
				return err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to commit transaction").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("SetPreferences")
	}

	return nil
}

// getPreferenceByKeyInTx retrieves a preference by key within a transaction
func (r *SQLitePreferencesRepository) getPreferenceByKeyInTx(ctx context.Context, tx *sql.Tx, scope string, scopeID *string, key string) (*Preference, error) {
	query := `
		SELECT id, scope, scope_id, key, value, version, metadata, created_at, updated_at
		FROM preferences
		WHERE scope = ? AND (scope_id = ? OR (scope_id IS NULL AND ? IS NULL)) AND key = ?
	`

	pref := &Preference{}
	var valueJSON, metadataJSON []byte

	err := tx.QueryRowContext(ctx, query, scope, scopeID, scopeID, key).Scan(
		&pref.ID, &pref.Scope, &pref.ScopeID, &pref.Key, &valueJSON, &pref.Version, &metadataJSON, &pref.CreatedAt, &pref.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, gerror.New(gerror.ErrCodeNotFound, "preference not found", err).
			WithComponent("SQLitePreferencesRepository").
			WithOperation("getPreferenceByKeyInTx").
			WithDetails("scope", scope).
			WithDetails("key", key)
	}
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get preference by key in transaction").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("getPreferenceByKeyInTx")
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal(valueJSON, &pref.Value); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal preference value").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("getPreferenceByKeyInTx")
	}

	if metadataJSON != nil {
		if err := json.Unmarshal(metadataJSON, &pref.Metadata); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal preference metadata").
				WithComponent("SQLitePreferencesRepository").
				WithOperation("getPreferenceByKeyInTx")
		}
	}

	return pref, nil
}

// createPreferenceInTx creates a preference within a transaction
func (r *SQLitePreferencesRepository) createPreferenceInTx(ctx context.Context, tx *sql.Tx, pref *Preference) error {
	now := time.Now()
	pref.CreatedAt = now
	pref.UpdatedAt = now

	valueJSON, err := json.Marshal(pref.Value)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal preference value").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("createPreferenceInTx")
	}

	query := `
		INSERT INTO preferences (id, scope, scope_id, key, value, version, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = tx.ExecContext(ctx, query,
		pref.ID, pref.Scope, pref.ScopeID, pref.Key, valueJSON, pref.Version, nil, pref.CreatedAt, pref.UpdatedAt)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create preference in transaction").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("createPreferenceInTx")
	}

	return nil
}

// updatePreferenceInTx updates a preference within a transaction
func (r *SQLitePreferencesRepository) updatePreferenceInTx(ctx context.Context, tx *sql.Tx, pref *Preference) error {
	pref.UpdatedAt = time.Now()

	valueJSON, err := json.Marshal(pref.Value)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal preference value").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("updatePreferenceInTx")
	}

	query := `
		UPDATE preferences
		SET value = ?, version = version + 1, updated_at = ?
		WHERE id = ?
	`

	_, err = tx.ExecContext(ctx, query, valueJSON, pref.UpdatedAt, pref.ID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to update preference in transaction").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("updatePreferenceInTx")
	}

	return nil
}

// DeletePreferencesByScope deletes all preferences for a scope
func (r *SQLitePreferencesRepository) DeletePreferencesByScope(ctx context.Context, scope string, scopeID *string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("DeletePreferencesByScope")
	}

	query := `DELETE FROM preferences WHERE scope = ? AND (scope_id = ? OR (scope_id IS NULL AND ? IS NULL))`
	_, err := r.db.DB().ExecContext(ctx, query, scope, scopeID, scopeID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to delete preferences by scope").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("DeletePreferencesByScope")
	}

	return nil
}

// CreateInheritance creates an inheritance relationship
func (r *SQLitePreferencesRepository) CreateInheritance(ctx context.Context, inheritance *PreferenceInheritance) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("CreateInheritance")
	}

	if inheritance.ID == "" {
		inheritance.ID = uuid.New().String()
	}
	inheritance.CreatedAt = time.Now()

	query := `
		INSERT INTO preference_inheritance (id, child_scope, child_scope_id, parent_scope, parent_scope_id, priority, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.DB().ExecContext(ctx, query,
		inheritance.ID, inheritance.ChildScope, inheritance.ChildScopeID,
		inheritance.ParentScope, inheritance.ParentScopeID, inheritance.Priority, inheritance.CreatedAt)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create inheritance").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("CreateInheritance")
	}

	return nil
}

// GetInheritanceChain retrieves the inheritance chain for a scope
func (r *SQLitePreferencesRepository) GetInheritanceChain(ctx context.Context, scope string, scopeID *string) ([]*PreferenceInheritance, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("GetInheritanceChain")
	}

	query := `
		SELECT id, child_scope, child_scope_id, parent_scope, parent_scope_id, priority, created_at
		FROM preference_inheritance
		WHERE child_scope = ? AND (child_scope_id = ? OR (child_scope_id IS NULL AND ? IS NULL))
		ORDER BY priority DESC
	`

	rows, err := r.db.DB().QueryContext(ctx, query, scope, scopeID, scopeID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get inheritance chain").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("GetInheritanceChain")
	}
	defer rows.Close()

	var chain []*PreferenceInheritance
	for rows.Next() {
		inh := &PreferenceInheritance{}
		err := rows.Scan(&inh.ID, &inh.ChildScope, &inh.ChildScopeID,
			&inh.ParentScope, &inh.ParentScopeID, &inh.Priority, &inh.CreatedAt)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to scan inheritance row").
				WithComponent("SQLitePreferencesRepository").
				WithOperation("GetInheritanceChain")
		}
		chain = append(chain, inh)
	}

	return chain, nil
}

// DeleteInheritance deletes an inheritance relationship
func (r *SQLitePreferencesRepository) DeleteInheritance(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("DeleteInheritance")
	}

	query := `DELETE FROM preference_inheritance WHERE id = ?`
	_, err := r.db.DB().ExecContext(ctx, query, id)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to delete inheritance").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("DeleteInheritance")
	}

	return nil
}

// ExportPreferences exports all preferences for a scope as a map
func (r *SQLitePreferencesRepository) ExportPreferences(ctx context.Context, scope string, scopeID *string) (map[string]interface{}, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("SQLitePreferencesRepository").
			WithOperation("ExportPreferences")
	}

	prefs, err := r.ListPreferencesByScope(ctx, scope, scopeID)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	for _, pref := range prefs {
		result[pref.Key] = pref.Value
	}

	return result, nil
}

// ImportPreferences imports preferences from a map
func (r *SQLitePreferencesRepository) ImportPreferences(ctx context.Context, scope string, scopeID *string, prefs map[string]interface{}) error {
	return r.SetPreferences(ctx, scope, scopeID, prefs)
}
