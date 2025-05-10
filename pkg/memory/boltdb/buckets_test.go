package boltdb_test

import (
	"testing"

	"github.com/blockhead-consulting/Guild/pkg/memory/boltdb"
)

// TestAllBuckets tests that AllBuckets returns all expected bucket names
func TestAllBuckets(t *testing.T) {
	buckets := boltdb.AllBuckets()

	// Define the expected buckets
	expectedBuckets := []string{
		"prompt_chains",
		"prompt_chains_by_agent",
		"prompt_chains_by_task",
		"corpus_documents",
		"corpus_metadata",
		"tasks",
		"tasks_by_status",
		"tasks_by_agent",
		"config",
	}

	// Check that the number of buckets matches
	if len(buckets) != len(expectedBuckets) {
		t.Errorf("Expected %d buckets, got %d", len(expectedBuckets), len(buckets))
	}

	// Check that all expected buckets are present
	for _, expected := range expectedBuckets {
		found := false
		for _, actual := range buckets {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected bucket '%s' not found", expected)
		}
	}
}

// TestBucketConstants tests that bucket constants match expected values
func TestBucketConstants(t *testing.T) {
	// Test all bucket constants
	testCases := []struct {
		constant string
		expected string
	}{
		{boltdb.BucketPromptChains, "prompt_chains"},
		{boltdb.BucketPromptChainsByAgent, "prompt_chains_by_agent"},
		{boltdb.BucketPromptChainsByTask, "prompt_chains_by_task"},
		{boltdb.BucketCorpusDocuments, "corpus_documents"},
		{boltdb.BucketCorpusMetadata, "corpus_metadata"},
		{boltdb.BucketTasks, "tasks"},
		{boltdb.BucketTasksByStatus, "tasks_by_status"},
		{boltdb.BucketTasksByAgent, "tasks_by_agent"},
		{boltdb.BucketConfig, "config"},
	}

	for _, tc := range testCases {
		if tc.constant != tc.expected {
			t.Errorf("Expected bucket constant '%s', got '%s'", tc.expected, tc.constant)
		}
	}
}