package corpus

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTrackUserView(t *testing.T) {
	ctx := context.Background()
	
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "corpus-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test configuration
	cfg := Config{
		CorpusPath:     tempDir,
		ActivitiesPath: filepath.Join(tempDir, ".activities"),
		MaxSizeBytes:   1024 * 1024,
	}

	// Create activities directory
	err = os.MkdirAll(cfg.ActivitiesPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create activities directory: %v", err)
	}

	// Create a test document
	doc := CorpusDoc{
		Title:     "Test Document",
		Body:      "Test content",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save the document
	err = Save(ctx, &doc, cfg)
	if err != nil {
		t.Fatalf("Failed to save document: %v", err)
	}

	// Track a user view
	user := "test-user"
	err = TrackUserView(ctx, user, doc.FilePath, cfg)
	if err != nil {
		t.Fatalf("Failed to track user view: %v", err)
	}

	// Verify the activity log was created
	userLogPath := filepath.Join(cfg.ActivitiesPath, user+".json")
	if _, err := os.Stat(userLogPath); os.IsNotExist(err) {
		t.Fatalf("Activity log file was not created")
	}

	// Read the activity log
	logData, err := os.ReadFile(userLogPath)
	if err != nil {
		t.Fatalf("Failed to read activity log: %v", err)
	}

	// Parse the activity log
	var logs []ViewLog
	err = json.Unmarshal(logData, &logs)
	if err != nil {
		t.Fatalf("Failed to parse activity log: %v", err)
	}

	// Verify the log entry
	if len(logs) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(logs))
	}

	log := logs[0]
	if log.User != user {
		t.Errorf("Expected user %s, got %s", user, log.User)
	}

	if log.DocPath != doc.FilePath {
		t.Errorf("Expected document path %s, got %s", doc.FilePath, log.DocPath)
	}

	if log.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}

	// Track another view
	time.Sleep(10 * time.Millisecond) // Ensure timestamp is different
	err = TrackUserView(ctx, user, doc.FilePath, cfg)
	if err != nil {
		t.Fatalf("Failed to track second user view: %v", err)
	}

	// Read the updated activity log
	logData, err = os.ReadFile(userLogPath)
	if err != nil {
		t.Fatalf("Failed to read updated activity log: %v", err)
	}

	// Parse the updated activity log
	err = json.Unmarshal(logData, &logs)
	if err != nil {
		t.Fatalf("Failed to parse updated activity log: %v", err)
	}

	// Verify multiple log entries
	if len(logs) != 2 {
		t.Fatalf("Expected 2 log entries, got %d", len(logs))
	}

	// Test GetUserActivities
	activities, err := GetUserActivities(ctx, user, cfg)
	if err != nil {
		t.Fatalf("Failed to get user activities: %v", err)
	}

	if len(activities) != 2 {
		t.Fatalf("Expected 2 activity entries, got %d", len(activities))
	}

	// Test GetPopularDocuments
	popular, err := GetPopularDocuments(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to get popular documents: %v", err)
	}

	if len(popular) != 1 {
		t.Fatalf("Expected 1 popular document, got %d", len(popular))
	}

	viewCount, ok := popular[doc.FilePath]
	if !ok {
		t.Errorf("Expected document path %s in popular documents", doc.FilePath)
	}

	if viewCount != 2 {
		t.Errorf("Expected view count 2, got %d", viewCount)
	}
}