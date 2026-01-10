// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/lancekrogers/guild-core/pkg/memory/vector"
)

// MockVectorStore is a mock implementation of the VectorStore interface.
type MockVectorStore struct {
	mock.Mock
}

// SaveEmbedding mocks the SaveEmbedding method.
func (m *MockVectorStore) SaveEmbedding(ctx context.Context, embedding vector.Embedding) error {
	args := m.Called(ctx, embedding)
	return args.Error(0)
}

// QueryEmbeddings mocks the QueryEmbeddings method.
func (m *MockVectorStore) QueryEmbeddings(ctx context.Context, query string, limit int) ([]vector.EmbeddingMatch, error) {
	args := m.Called(ctx, query, limit)
	return args.Get(0).([]vector.EmbeddingMatch), args.Error(1)
}

// Close mocks the Close method.
func (m *MockVectorStore) Close() error {
	args := m.Called()
	return args.Error(0)
}
