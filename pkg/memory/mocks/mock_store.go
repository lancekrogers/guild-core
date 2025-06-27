// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package mocks

import (
	"context"
	"strings"
	"sync"

	"github.com/lancekrogers/guild/pkg/memory"
)

// MockStore implements memory.Store interface for testing
type MockStore struct {
	mu      sync.RWMutex
	buckets map[string]map[string][]byte
	closed  bool

	// For tracking calls in tests
	PutCalls    int
	GetCalls    int
	DeleteCalls int
	ListCalls   int

	// Optional errors to return
	PutError    error
	GetError    error
	DeleteError error
	ListError   error
}

// NewMockStore creates a new mock store
func NewMockStore() *MockStore {
	return &MockStore{
		buckets: make(map[string]map[string][]byte),
	}
}

// Put stores a value with the given key
func (m *MockStore) Put(ctx context.Context, bucket, key string, value []byte) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.PutCalls++

	if m.PutError != nil {
		return m.PutError
	}

	if m.closed {
		return memory.StoreError{Message: "store is closed"}
	}

	// Initialize the bucket if it doesn't exist
	if _, ok := m.buckets[bucket]; !ok {
		m.buckets[bucket] = make(map[string][]byte)
	}

	// Copy the value
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	m.buckets[bucket][key] = valueCopy
	return nil
}

// Get retrieves a value by key
func (m *MockStore) Get(ctx context.Context, bucket, key string) ([]byte, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	m.GetCalls++

	if m.GetError != nil {
		return nil, m.GetError
	}

	if m.closed {
		return nil, memory.StoreError{Message: "store is closed"}
	}

	bucketMap, ok := m.buckets[bucket]
	if !ok {
		return nil, memory.StoreError{Message: "bucket not found"}
	}

	value, ok := bucketMap[key]
	if !ok {
		return nil, memory.ErrNotFound
	}

	// Return a copy of the value
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	return valueCopy, nil
}

// Delete removes a value by key
func (m *MockStore) Delete(ctx context.Context, bucket, key string) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.DeleteCalls++

	if m.DeleteError != nil {
		return m.DeleteError
	}

	if m.closed {
		return memory.StoreError{Message: "store is closed"}
	}

	bucketMap, ok := m.buckets[bucket]
	if !ok {
		return memory.StoreError{Message: "bucket not found"}
	}

	delete(bucketMap, key)
	return nil
}

// List returns all keys in a bucket
func (m *MockStore) List(ctx context.Context, bucket string) ([]string, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	m.ListCalls++

	if m.ListError != nil {
		return nil, m.ListError
	}

	if m.closed {
		return nil, memory.StoreError{Message: "store is closed"}
	}

	bucketMap, ok := m.buckets[bucket]
	if !ok {
		return nil, memory.StoreError{Message: "bucket not found"}
	}

	keys := make([]string, 0, len(bucketMap))
	for k := range bucketMap {
		keys = append(keys, k)
	}

	return keys, nil
}

// ListKeys returns keys with the given prefix in a bucket
func (m *MockStore) ListKeys(ctx context.Context, bucket, prefix string) ([]string, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	m.ListCalls++

	if m.ListError != nil {
		return nil, m.ListError
	}

	if m.closed {
		return nil, memory.StoreError{Message: "store is closed"}
	}

	bucketMap, ok := m.buckets[bucket]
	if !ok {
		return nil, memory.StoreError{Message: "bucket not found"}
	}

	keys := make([]string, 0)
	for k := range bucketMap {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}

	return keys, nil
}

// Close closes the store
func (m *MockStore) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true
	return nil
}

// Reset resets the mock store to a clean state
func (m *MockStore) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.buckets = make(map[string]map[string][]byte)
	m.PutCalls = 0
	m.GetCalls = 0
	m.DeleteCalls = 0
	m.ListCalls = 0
	m.closed = false
	m.PutError = nil
	m.GetError = nil
	m.DeleteError = nil
	m.ListError = nil
}

// SetError sets an error to be returned by the specified method
func (m *MockStore) SetError(method string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch method {
	case "Put":
		m.PutError = err
	case "Get":
		m.GetError = err
	case "Delete":
		m.DeleteError = err
	case "List", "ListKeys":
		m.ListError = err
	}
}
