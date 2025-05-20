package boltdb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/guild-ventures/guild-core/pkg/memory"
	bolt "go.etcd.io/bbolt"
)

// Store implements the memory.Store interface using BoltDB
type Store struct {
	db         *bolt.DB
	dbPath     string
	bucketList []string
}

// NewStore creates a new BoltDB store
func NewStore(dbPath string, options ...Option) (*Store, error) {
	// Create db directory if it doesn't exist
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	// Default options
	opts := &bolt.Options{
		Timeout: 5 * time.Second,
	}

	// Create or open the database
	db, err := bolt.Open(dbPath, 0600, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open bolt db: %w", err)
	}

	store := &Store{
		db:         db,
		dbPath:     dbPath,
		bucketList: AllBuckets(),
	}

	// Initialize buckets
	if err := store.initBuckets(); err != nil {
		db.Close()
		return nil, err
	}

	// Apply options
	for _, option := range options {
		option(store)
	}

	return store, nil
}

// Option is a function that configures the store
type Option func(*Store)

// WithCustomBuckets allows adding custom bucket names
func WithCustomBuckets(buckets ...string) Option {
	return func(s *Store) {
		s.bucketList = append(s.bucketList, buckets...)
	}
}

// initBuckets creates all required buckets if they don't exist
func (s *Store) initBuckets() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		for _, bucket := range s.bucketList {
			_, err := tx.CreateBucketIfNotExists([]byte(bucket))
			if err != nil {
				return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
			}
		}
		return nil
	})
}

// Put stores a value with the given key in the specified bucket
func (s *Store) Put(ctx context.Context, bucket, key string, value []byte) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		return b.Put([]byte(key), value)
	})
}

// Get retrieves a value by key from the specified bucket
func (s *Store) Get(ctx context.Context, bucket, key string) ([]byte, error) {
	var value []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		
		v := b.Get([]byte(key))
		if v == nil {
			return memory.ErrNotFound
		}
		
		// Copy the value as it might be invalidated after the transaction
		value = make([]byte, len(v))
		copy(value, v)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return value, nil
}

// Delete removes a key-value pair from the specified bucket
func (s *Store) Delete(ctx context.Context, bucket, key string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		return b.Delete([]byte(key))
	})
}

// List returns all keys in a bucket
func (s *Store) List(ctx context.Context, bucket string) ([]string, error) {
	var keys []string
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		
		return b.ForEach(func(k, v []byte) error {
			keys = append(keys, string(k))
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// ListKeys returns keys with the given prefix in a bucket
func (s *Store) ListKeys(ctx context.Context, bucket, prefix string) ([]string, error) {
	var keys []string
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		
		c := b.Cursor()
		prefixBytes := []byte(prefix)
		
		for k, _ := c.Seek(prefixBytes); k != nil && strings.HasPrefix(string(k), prefix); k, _ = c.Next() {
			keys = append(keys, string(k))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// Close closes the database
func (s *Store) Close() error {
	return s.db.Close()
}

// Transaction executes a function within a transaction
func (s *Store) Transaction(writable bool, fn func(*bolt.Tx) error) error {
	if writable {
		return s.db.Update(fn)
	}
	return s.db.View(fn)
}

// Path returns the database file path
func (s *Store) Path() string {
	return s.dbPath
}