// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package corpus

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// ActivityLog represents a collection of user view records
type ActivityLog struct {
	// ViewLogs is a map from user IDs to their view logs
	ViewLogs map[string][]ViewLog `json:"view_logs"`
}

// NewActivityLog creates a new activity log
func NewActivityLog() *ActivityLog {
	return &ActivityLog{
		ViewLogs: make(map[string][]ViewLog),
	}
}

// TrackUserView records a document view by a user
func TrackUserView(ctx context.Context, user, docPath string, cfg Config) error {
	if user == "" || docPath == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "corpus", nil).WithComponent("track_user_view").WithOperation("user and document path are required")
	}

	// Ensure the activities directory exists
	activitiesDir := cfg.ActivitiesPath
	if activitiesDir == "" {
		activitiesDir = filepath.Join(cfg.CorpusPath, ViewLogDirName)
	}

	if err := os.MkdirAll(activitiesDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("track_user_view").WithOperation("failed to create activities directory")
	}

	// Create or load the activity log
	activityLogPath := filepath.Join(activitiesDir, "user_activity.json")
	var activityLog *ActivityLog

	// Check if file exists
	if _, err := os.Stat(activityLogPath); os.IsNotExist(err) {
		// Create new activity log
		activityLog = NewActivityLog()
	} else {
		// Load existing activity log
		data, err := os.ReadFile(activityLogPath)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("track_user_view").WithOperation("failed to read activity log")
		}

		activityLog = &ActivityLog{
			ViewLogs: make(map[string][]ViewLog),
		}
		if err := json.Unmarshal(data, activityLog); err != nil {
			// If parsing fails, create a new log
			activityLog = NewActivityLog()
		}
	}

	// Create a new view log entry
	viewLog := ViewLog{
		User:      user,
		DocPath:   docPath,
		Timestamp: time.Now(),
	}

	// Add the entry to the user's view logs
	if activityLog.ViewLogs[user] == nil {
		activityLog.ViewLogs[user] = []ViewLog{}
	}
	activityLog.ViewLogs[user] = append(activityLog.ViewLogs[user], viewLog)

	// Limit the number of entries per user (keep last 100)
	maxEntries := 100
	if len(activityLog.ViewLogs[user]) > maxEntries {
		activityLog.ViewLogs[user] = activityLog.ViewLogs[user][len(activityLog.ViewLogs[user])-maxEntries:]
	}

	// Save the updated activity log
	data, err := json.MarshalIndent(activityLog, "", "  ")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("track_user_view").WithOperation("failed to serialize activity log")
	}

	if err := os.WriteFile(activityLogPath, data, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("track_user_view").WithOperation("failed to save activity log")
	}

	// Also save user-specific log file for backward compatibility
	userLogPath := filepath.Join(activitiesDir, user+".json")

	// Get user logs from activity log
	userLogs := activityLog.ViewLogs[user]

	// Save user-specific log
	userData, err := json.MarshalIndent(userLogs, "", "  ")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("track_user_view").WithOperation("failed to marshal user log")
	}

	if err := os.WriteFile(userLogPath, userData, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("track_user_view").WithOperation("failed to save user log")
	}

	return nil
}

// GetUserActivity retrieves a user's document viewing history
func GetUserActivity(ctx context.Context, user string, cfg Config) ([]ViewLog, error) {
	if user == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "corpus", nil).WithComponent("get_user_activity").WithOperation("user is required")
	}

	// Check for the activity log file
	activitiesDir := cfg.ActivitiesPath
	if activitiesDir == "" {
		activitiesDir = filepath.Join(cfg.CorpusPath, ViewLogDirName)
	}
	activityLogPath := filepath.Join(activitiesDir, "user_activity.json")

	if _, err := os.Stat(activityLogPath); os.IsNotExist(err) {
		return []ViewLog{}, nil // No activity log yet
	}

	// Load the activity log
	data, err := os.ReadFile(activityLogPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("get_user_activity").WithOperation("failed to read activity log")
	}

	var activityLog ActivityLog
	if err := json.Unmarshal(data, &activityLog); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("get_user_activity").WithOperation("failed to parse activity log")
	}

	// Return the user's view logs (or empty slice if none)
	viewLogs, ok := activityLog.ViewLogs[user]
	if !ok {
		return []ViewLog{}, nil
	}

	return viewLogs, nil
}

// GetMostViewedDocuments returns the most viewed documents across all users
func GetMostViewedDocuments(ctx context.Context, cfg Config, limit int) (map[string]int, error) {
	if limit <= 0 {
		limit = 10 // Default limit
	}

	// Check for the activity log file
	activitiesDir := cfg.ActivitiesPath
	if activitiesDir == "" {
		activitiesDir = filepath.Join(cfg.CorpusPath, ViewLogDirName)
	}
	activityLogPath := filepath.Join(activitiesDir, "user_activity.json")

	if _, err := os.Stat(activityLogPath); os.IsNotExist(err) {
		return map[string]int{}, nil // No activity log yet
	}

	// Load the activity log
	data, err := os.ReadFile(activityLogPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("get_activity").WithOperation("failed to read activity log")
	}

	var activityLog ActivityLog
	if err := json.Unmarshal(data, &activityLog); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("get_user_activity").WithOperation("failed to parse activity log")
	}

	// Count document views
	viewCounts := make(map[string]int)
	for _, userLogs := range activityLog.ViewLogs {
		for _, log := range userLogs {
			viewCounts[log.DocPath]++
		}
	}

	// Sort by view count (we'll do this by building a new map with the top entries)
	type docCount struct {
		path  string
		count int
	}

	var counts []docCount
	for path, count := range viewCounts {
		counts = append(counts, docCount{path, count})
	}

	// Sort in descending order
	// Note: In a real implementation, we'd use sort.Slice() here
	// This is a simplified implementation for brevity
	result := make(map[string]int)
	for i := 0; i < len(counts) && i < limit; i++ {
		// Find the max count
		maxIdx := 0
		for j := 1; j < len(counts); j++ {
			if counts[j].count > counts[maxIdx].count {
				maxIdx = j
			}
		}

		// Add to result
		result[counts[maxIdx].path] = counts[maxIdx].count

		// Remove from counts
		counts = append(counts[:maxIdx], counts[maxIdx+1:]...)
	}

	return result, nil
}

// GetUserActivities is an alias for GetUserActivity for backward compatibility
func GetUserActivities(ctx context.Context, user string, cfg Config) ([]ViewLog, error) {
	return GetUserActivity(ctx, user, cfg)
}

// GetPopularDocuments is an alias for GetMostViewedDocuments for backward compatibility
func GetPopularDocuments(ctx context.Context, cfg Config) (map[string]int, error) {
	return GetMostViewedDocuments(ctx, cfg, 10) // Use default limit of 10
}

// GetRecentActivity returns the most recent activity across all users
func GetRecentActivity(ctx context.Context, cfg Config, limit int) ([]ViewLog, error) {
	if limit <= 0 {
		limit = 20 // Default limit
	}

	// Check for the activity log file
	activitiesDir := cfg.ActivitiesPath
	if activitiesDir == "" {
		activitiesDir = filepath.Join(cfg.CorpusPath, ViewLogDirName)
	}
	activityLogPath := filepath.Join(activitiesDir, "user_activity.json")

	if _, err := os.Stat(activityLogPath); os.IsNotExist(err) {
		return []ViewLog{}, nil // No activity log yet
	}

	// Load the activity log
	data, err := os.ReadFile(activityLogPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("get_activity").WithOperation("failed to read activity log")
	}

	var activityLog ActivityLog
	if err := json.Unmarshal(data, &activityLog); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "corpus").WithComponent("get_user_activity").WithOperation("failed to parse activity log")
	}

	// Collect all view logs
	var allLogs []ViewLog
	for _, userLogs := range activityLog.ViewLogs {
		allLogs = append(allLogs, userLogs...)
	}

	// Sort by timestamp (most recent first)
	// Note: In a real implementation, we'd use sort.Slice() here
	// This is a simplified implementation for brevity
	var recentLogs []ViewLog
	for i := 0; i < limit && len(allLogs) > 0; i++ {
		// Find the most recent log
		mostRecentIdx := 0
		for j := 1; j < len(allLogs); j++ {
			if allLogs[j].Timestamp.After(allLogs[mostRecentIdx].Timestamp) {
				mostRecentIdx = j
			}
		}

		// Add to result
		recentLogs = append(recentLogs, allLogs[mostRecentIdx])

		// Remove from allLogs
		allLogs = append(allLogs[:mostRecentIdx], allLogs[mostRecentIdx+1:]...)
	}

	return recentLogs, nil
}
