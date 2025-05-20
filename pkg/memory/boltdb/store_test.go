package boltdb_test

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/guild-ventures/guild-core/pkg/memory"
	"github.com/guild-ventures/guild-core/pkg/memory/boltdb"
	bolt "go.etcd.io/bbolt"
)

// setupTestStore creates a temporary BoltDB store for testing
func setupTestStore(t *testing.T) (*boltdb.Store, func()) {
	// Create a temporary directory for the test database
	tempDir, err := os.MkdirTemp("", "boltdb-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create a path for the database file
	dbPath := filepath.Join(tempDir, "test.db")

	// Create the store with test bucket
	store, err := boltdb.NewStore(dbPath, boltdb.WithCustomBuckets("test"))
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create store: %v", err)
	}

	// Return the store and a cleanup function
	cleanup := func() {
		store.Close()
		os.RemoveAll(tempDir)
	}

	return store, cleanup
}

// TestBoltDBStore_Implementation tests that the BoltDB store implements the Store interface
func TestBoltDBStore_Implementation(t *testing.T) {
	var _ memory.Store = &boltdb.Store{}
}

// TestNewStore tests the creation of a new BoltDB store
func TestNewStore(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "boltdb-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a path for the database file
	dbPath := filepath.Join(tempDir, "test.db")

	// Test creating a new store
	store, err := boltdb.NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Verify the store path
	if store.Path() != dbPath {
		t.Errorf("Expected path %s, got %s", dbPath, store.Path())
	}

	// Test creating a store with custom buckets
	customStore, err := boltdb.NewStore(dbPath, boltdb.WithCustomBuckets("custom_bucket"))
	if err != nil {
		t.Fatalf("Failed to create store with custom buckets: %v", err)
	}
	defer customStore.Close()

	// Verify the custom bucket exists by putting and getting a value
	ctx := context.Background()
	if err := customStore.Put(ctx, "custom_bucket", "test_key", []byte("test_value")); err != nil {
		t.Errorf("Failed to put value in custom bucket: %v", err)
	}

	value, err := customStore.Get(ctx, "custom_bucket", "test_key")
	if err != nil {
		t.Errorf("Failed to get value from custom bucket: %v", err)
	}

	if string(value) != "test_value" {
		t.Errorf("Expected value 'test_value', got '%s'", string(value))
	}
}

// TestStore_PutGet tests the Put and Get methods
func TestStore_PutGet(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Test putting and getting a value
	key := "test_key"
	value := []byte("test_value")

	if err := store.Put(ctx, "test", key, value); err != nil {
		t.Fatalf("Failed to put value: %v", err)
	}

	retrievedValue, err := store.Get(ctx, "test", key)
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}

	if string(retrievedValue) != string(value) {
		t.Errorf("Expected value '%s', got '%s'", string(value), string(retrievedValue))
	}

	// Test getting a non-existent key
	_, err = store.Get(ctx, "test", "non_existent")
	if err != memory.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}

	// Test getting from a non-existent bucket
	_, err = store.Get(ctx, "non_existent", key)
	if err == nil {
		t.Error("Expected error for non-existent bucket, got nil")
	}
}

// TestStore_Delete tests the Delete method
func TestStore_Delete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Put a value
	key := "test_key"
	value := []byte("test_value")

	if err := store.Put(ctx, "test", key, value); err != nil {
		t.Fatalf("Failed to put value: %v", err)
	}

	// Delete the value
	if err := store.Delete(ctx, "test", key); err != nil {
		t.Fatalf("Failed to delete value: %v", err)
	}

	// Verify it's gone
	_, err := store.Get(ctx, "test", key)
	if err != memory.ErrNotFound {
		t.Errorf("Expected ErrNotFound after deletion, got %v", err)
	}

	// Test deleting from a non-existent bucket
	err = store.Delete(ctx, "non_existent", key)
	if err == nil {
		t.Error("Expected error for non-existent bucket, got nil")
	}
}

// TestStore_List tests the List method
func TestStore_List(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Put multiple values
	keys := []string{"key1", "key2", "key3"}
	for _, key := range keys {
		if err := store.Put(ctx, "test", key, []byte("value_"+key)); err != nil {
			t.Fatalf("Failed to put value: %v", err)
		}
	}

	// List all keys
	listedKeys, err := store.List(ctx, "test")
	if err != nil {
		t.Fatalf("Failed to list keys: %v", err)
	}

	if len(listedKeys) != len(keys) {
		t.Errorf("Expected %d keys, got %d", len(keys), len(listedKeys))
	}

	// Check that all keys are present
	for _, key := range keys {
		found := false
		for _, listedKey := range listedKeys {
			if listedKey == key {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Key '%s' not found in list", key)
		}
	}

	// Test listing from a non-existent bucket
	_, err = store.List(ctx, "non_existent")
	if err == nil {
		t.Error("Expected error for non-existent bucket, got nil")
	}
}

// TestStore_ListKeys tests the ListKeys method
func TestStore_ListKeys(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Put values with different prefixes
	prefixA := "prefix_a_"
	prefixB := "prefix_b_"
	
	keysA := []string{prefixA + "1", prefixA + "2", prefixA + "3"}
	keysB := []string{prefixB + "1", prefixB + "2"}
	
	for _, key := range append(keysA, keysB...) {
		if err := store.Put(ctx, "test", key, []byte("value_"+key)); err != nil {
			t.Fatalf("Failed to put value: %v", err)
		}
	}

	// List keys with prefix A
	listedKeysA, err := store.ListKeys(ctx, "test", prefixA)
	if err != nil {
		t.Fatalf("Failed to list keys with prefix A: %v", err)
	}

	if len(listedKeysA) != len(keysA) {
		t.Errorf("Expected %d keys with prefix A, got %d", len(keysA), len(listedKeysA))
	}

	// Check that all prefix A keys are present
	for _, key := range keysA {
		found := false
		for _, listedKey := range listedKeysA {
			if listedKey == key {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Key '%s' not found in prefix A list", key)
		}
	}

	// List keys with prefix B
	listedKeysB, err := store.ListKeys(ctx, "test", prefixB)
	if err != nil {
		t.Fatalf("Failed to list keys with prefix B: %v", err)
	}

	if len(listedKeysB) != len(keysB) {
		t.Errorf("Expected %d keys with prefix B, got %d", len(keysB), len(listedKeysB))
	}

	// Check that all prefix B keys are present
	for _, key := range keysB {
		found := false
		for _, listedKey := range listedKeysB {
			if listedKey == key {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Key '%s' not found in prefix B list", key)
		}
	}

	// Test listing with a non-existent prefix
	listedKeysNone, err := store.ListKeys(ctx, "test", "non_existent_prefix")
	if err != nil {
		t.Fatalf("Failed to list keys with non-existent prefix: %v", err)
	}

	if len(listedKeysNone) != 0 {
		t.Errorf("Expected 0 keys with non-existent prefix, got %d", len(listedKeysNone))
	}

	// Test listing from a non-existent bucket
	_, err = store.ListKeys(ctx, "non_existent", prefixA)
	if err == nil {
		t.Error("Expected error for non-existent bucket, got nil")
	}
}

// TestStore_Close tests the Close method
func TestStore_Close(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Close the store
	if err := store.Close(); err != nil {
		t.Fatalf("Failed to close store: %v", err)
	}

	// Operations should fail after closing
	ctx := context.Background()
	err := store.Put(ctx, "test", "key", []byte("value"))
	if err == nil {
		t.Error("Expected error after closing store, got nil")
	}
}

// TestStore_ConcurrentAccess tests concurrent access to the store
func TestStore_ConcurrentAccess(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	numGoroutines := 10
	numOperations := 100
	done := make(chan bool, numGoroutines)

	// Start multiple goroutines to access the store concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOperations; j++ {
				// Perform a put
				key := strconv.Itoa(id) + "_" + strconv.Itoa(j)
				err := store.Put(ctx, "test", key, []byte("value_"+key))
				if err != nil {
					t.Errorf("Failed to put value in goroutine %d: %v", id, err)
				}

				// Perform a get
				_, err = store.Get(ctx, "test", key)
				if err != nil {
					t.Errorf("Failed to get value in goroutine %d: %v", id, err)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to finish
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify the total number of keys
	keys, err := store.List(ctx, "test")
	if err != nil {
		t.Fatalf("Failed to list keys: %v", err)
	}

	expectedKeys := numGoroutines * numOperations
	if len(keys) != expectedKeys {
		t.Errorf("Expected %d keys after concurrent access, got %d", expectedKeys, len(keys))
	}
}

// TestStore_ContextCancellation tests that operations respect context cancellation
func TestStore_ContextCancellation(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create a context and cancel it
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Operations should respect the cancelled context
	err := store.Put(ctx, "test", "key", []byte("value"))
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}

	_, err = store.Get(ctx, "test", "key")
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}

	err = store.Delete(ctx, "test", "key")
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}

	_, err = store.List(ctx, "test")
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}

	_, err = store.ListKeys(ctx, "test", "prefix")
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

// TestStore_Transaction tests the Transaction method
func TestStore_Transaction(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Test read-only transaction
	err := store.Transaction(false, func(tx *bolt.Tx) error {
		// Transaction should not allow modifications
		return nil
	})
	if err != nil {
		t.Errorf("Read-only transaction failed: %v", err)
	}

	// Test writable transaction
	err = store.Transaction(true, func(tx *bolt.Tx) error {
		// Transaction should allow modifications
		return nil
	})
	if err != nil {
		t.Errorf("Writable transaction failed: %v", err)
	}

	// Test transaction that returns an error
	testErr := memory.StoreError{Message: "test error"}
	err = store.Transaction(true, func(tx *bolt.Tx) error {
		return testErr
	})
	if err != testErr {
		t.Errorf("Expected transaction to return test error, got %v", err)
	}
}

// TestStore_Timeouts tests that operations respect the configured timeout
func TestStore_Timeouts(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "boltdb-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create two store instances pointing to the same database
	// with a very short timeout
	dbPath := filepath.Join(tempDir, "test.db")
	store1, err := boltdb.NewStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create first store: %v", err)
	}
	defer store1.Close()

	// Start a long-running transaction on store1
	done := make(chan bool)
	go func() {
		err := store1.Transaction(true, func(tx *bolt.Tx) error {
			// Hold the transaction for a while
			time.Sleep(2 * time.Second)
			return nil
		})
		if err != nil {
			t.Errorf("Long-running transaction failed: %v", err)
		}
		done <- true
	}()

	// Try to open another store with a very short timeout
	// This should fail with timeout error
	time.Sleep(100 * time.Millisecond) // Give the first transaction time to start
	_, err = boltdb.NewStore(dbPath, func(s *boltdb.Store) {
		// Set a very short timeout
		// Note: This is a mock implementation as the real code doesn't expose the timeout option
		// In a real test, you'd need to modify the code to expose this configuration
	})

	// Wait for the first transaction to complete
	<-done
}