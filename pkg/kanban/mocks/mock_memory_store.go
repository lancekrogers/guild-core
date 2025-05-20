package mocks

import (
	"context"
	"strings"
	"sync"

	"github.com/guild-ventures/guild-core/pkg/memory"
)

// MockMemoryStore mocks the memory.Store interface for testing
type MockMemoryStore struct {
	data map[string]map[string][]byte
	mu   sync.RWMutex
}

// NewMockMemoryStore creates a new mock memory store
func NewMockMemoryStore() *MockMemoryStore {
	return &MockMemoryStore{
		data: make(map[string]map[string][]byte),
	}
}

// Put stores a value with the given key
func (m *MockMemoryStore) Put(ctx context.Context, bucket, key string, value []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.data[bucket]; !ok {
		m.data[bucket] = make(map[string][]byte)
	}

	// Make a copy of the value to prevent sharing memory
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	m.data[bucket][key] = valueCopy

	return nil
}

// Get retrieves a value by key
func (m *MockMemoryStore) Get(ctx context.Context, bucket, key string) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	bucketData, ok := m.data[bucket]
	if !ok {
		return nil, memory.StoreError{Message: "bucket not found"}
	}

	value, ok := bucketData[key]
	if !ok {
		return nil, memory.ErrNotFound
	}

	// Make a copy of the value to prevent sharing memory
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	return valueCopy, nil
}

// Delete removes a value by key
func (m *MockMemoryStore) Delete(ctx context.Context, bucket, key string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	bucketData, ok := m.data[bucket]
	if !ok {
		return memory.StoreError{Message: "bucket not found"}
	}

	delete(bucketData, key)

	return nil
}

// List returns all keys in a bucket
func (m *MockMemoryStore) List(ctx context.Context, bucket string) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	bucketData, ok := m.data[bucket]
	if !ok {
		return nil, memory.StoreError{Message: "bucket not found"}
	}

	keys := make([]string, 0, len(bucketData))
	for k := range bucketData {
		keys = append(keys, k)
	}

	return keys, nil
}

// ListKeys returns keys with the given prefix in a bucket
func (m *MockMemoryStore) ListKeys(ctx context.Context, bucket, prefix string) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	bucketData, ok := m.data[bucket]
	if !ok {
		return nil, memory.StoreError{Message: "bucket not found"}
	}

	keys := make([]string, 0)
	for k := range bucketData {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}

	return keys, nil
}

// Close closes the store
func (m *MockMemoryStore) Close() error {
	return nil
}

// Reset clears all data from the store
func (m *MockMemoryStore) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data = make(map[string]map[string][]byte)
}

// Dump returns a copy of all data in the store for debugging
func (m *MockMemoryStore) Dump() map[string]map[string][]byte {
	m.mu.RLock()
	defer m.mu.RUnlock()

	dump := make(map[string]map[string][]byte)
	for bucket, bucketData := range m.data {
		dump[bucket] = make(map[string][]byte)
		for k, v := range bucketData {
			valueCopy := make([]byte, len(v))
			copy(valueCopy, v)
			dump[bucket][k] = valueCopy
		}
	}

	return dump
}