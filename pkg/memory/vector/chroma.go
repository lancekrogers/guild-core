package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// ChromaConfig contains configuration for the Chroma vector store
type ChromaConfig struct {
	// URL is the address of the Chroma server (e.g., "http://localhost:8000")
	URL string

	// CollectionName is the name of the collection to use
	CollectionName string

	// EmbeddingProvider is the provider used to generate embeddings
	EmbeddingProvider EmbeddingProvider
}

// ChromaStore implements VectorStore using Chroma
type ChromaStore struct {
	config            *ChromaConfig
	embeddingProvider EmbeddingProvider
	collectionID      string
	httpClient        *http.Client
}

// NewChromaStore creates a new Chroma vector store
func NewChromaStore(ctx context.Context, config *ChromaConfig) (*ChromaStore, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.URL == "" {
		return nil, fmt.Errorf("URL cannot be empty")
	}

	if config.CollectionName == "" {
		return nil, fmt.Errorf("collection name cannot be empty")
	}

	if config.EmbeddingProvider == nil {
		return nil, fmt.Errorf("embedding provider cannot be nil")
	}

	// Clean URL
	config.URL = strings.TrimSuffix(config.URL, "/")

	// Create HTTP client
	httpClient := &http.Client{}

	// Create store
	store := &ChromaStore{
		config:            config,
		embeddingProvider: config.EmbeddingProvider,
		httpClient:        httpClient,
	}

	// Initialize collection
	if err := store.initCollection(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize collection: %w", err)
	}

	return store, nil
}

// initCollection initializes the collection
func (s *ChromaStore) initCollection(ctx context.Context) error {
	// Check if collection exists
	collections, err := s.listCollections(ctx)
	if err != nil {
		return fmt.Errorf("failed to list collections: %w", err)
	}

	for _, collection := range collections {
		if collection.Name == s.config.CollectionName {
			s.collectionID = collection.ID
			return nil
		}
	}

	// Collection doesn't exist, create it
	collection, err := s.createCollection(ctx, s.config.CollectionName)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	s.collectionID = collection.ID
	return nil
}

// Collection represents a Chroma collection
type Collection struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// listCollections lists all collections
func (s *ChromaStore) listCollections(ctx context.Context) ([]Collection, error) {
	url := fmt.Sprintf("%s/api/v1/collections", s.config.URL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list collections: %s", string(bodyBytes))
	}

	var result struct {
		Collections []Collection `json:"collections"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Collections, nil
}

// createCollection creates a new collection
func (s *ChromaStore) createCollection(ctx context.Context, name string) (*Collection, error) {
	url := fmt.Sprintf("%s/api/v1/collections", s.config.URL)

	type createCollectionRequest struct {
		Name     string `json:"name"`
		Metadata struct{} `json:"metadata"`
	}

	request := createCollectionRequest{
		Name:     name,
		Metadata: struct{}{},
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create collection: %s", string(bodyBytes))
	}

	var collection Collection
	if err := json.NewDecoder(resp.Body).Decode(&collection); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &collection, nil
}

// Store stores a document with its embedding
func (s *ChromaStore) Store(ctx context.Context, doc *Document) error {
	docs := []*Document{doc}
	return s.StoreMany(ctx, docs)
}

// StoreMany stores multiple documents with their embeddings
func (s *ChromaStore) StoreMany(ctx context.Context, docs []*Document) error {
	if len(docs) == 0 {
		return nil
	}

	// Generate embeddings for documents that don't have them
	docsNeedingEmbeddings := []int{}
	textsToEmbed := []string{}

	for i, doc := range docs {
		if doc.Embedding == nil {
			docsNeedingEmbeddings = append(docsNeedingEmbeddings, i)
			textsToEmbed = append(textsToEmbed, doc.Content)
		}
	}

	if len(textsToEmbed) > 0 {
		embeddings, err := s.embeddingProvider.GetEmbeddings(ctx, textsToEmbed)
		if err != nil {
			return fmt.Errorf("failed to generate embeddings: %w", err)
		}

		for i, idx := range docsNeedingEmbeddings {
			docs[idx].Embedding = embeddings[i]
		}
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/upsert", s.config.URL, s.collectionID)

	// Prepare request
	type upsertRequest struct {
		IDs       []string          `json:"ids"`
		Embeddings [][]float32     `json:"embeddings"`
		Metadatas []map[string]interface{} `json:"metadatas"`
		Documents []string          `json:"documents"`
	}

	request := upsertRequest{
		IDs:       make([]string, len(docs)),
		Embeddings: make([][]float32, len(docs)),
		Metadatas: make([]map[string]interface{}, len(docs)),
		Documents: make([]string, len(docs)),
	}

	for i, doc := range docs {
		// Use the provided ID or generate a new one
		id := doc.ID
		if id == "" {
			id = uuid.New().String()
			docs[i].ID = id
		}
		request.IDs[i] = id

		// Add embedding
		request.Embeddings[i] = doc.Embedding

		// Add content
		request.Documents[i] = doc.Content

		// Add metadata
		metadata := make(map[string]interface{})
		for k, v := range doc.Metadata {
			metadata[k] = v
		}
		request.Metadatas[i] = metadata
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upsert documents: %s", string(bodyBytes))
	}

	return nil
}

// Retrieve retrieves a document by ID
func (s *ChromaStore) Retrieve(ctx context.Context, id string) (*Document, error) {
	url := fmt.Sprintf("%s/api/v1/collections/%s/get", s.config.URL, s.collectionID)

	type getRequest struct {
		IDs              []string `json:"ids"`
		IncludeEmbeddings bool    `json:"include_embeddings"`
	}

	request := getRequest{
		IDs:              []string{id},
		IncludeEmbeddings: true,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to retrieve document: %s", string(bodyBytes))
	}

	var result struct {
		IDs       []string          `json:"ids"`
		Embeddings [][]float32     `json:"embeddings"`
		Metadatas []map[string]interface{} `json:"metadatas"`
		Documents []string          `json:"documents"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.IDs) == 0 {
		return nil, fmt.Errorf("document not found")
	}

	// Build document from response
	doc := &Document{
		ID:      result.IDs[0],
		Content: result.Documents[0],
	}

	// Extract embedding
	if len(result.Embeddings) > 0 {
		embedding := make(Vector, len(result.Embeddings[0]))
		for i, v := range result.Embeddings[0] {
			embedding[i] = v
		}
		doc.Embedding = embedding
	}

	// Extract metadata
	if len(result.Metadatas) > 0 {
		metadata := make(map[string]string)
		for k, v := range result.Metadatas[0] {
			if str, ok := v.(string); ok {
				metadata[k] = str
			} else {
				metadata[k] = fmt.Sprintf("%v", v)
			}
		}
		doc.Metadata = metadata
	}

	return doc, nil
}

// Query performs a similarity search on the vector store
func (s *ChromaStore) Query(ctx context.Context, embedding Vector, limit int) ([]*QueryResult, error) {
	if limit <= 0 {
		limit = 10 // Default limit
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/query", s.config.URL, s.collectionID)

	type queryRequest struct {
		QueryEmbeddings [][]float32 `json:"query_embeddings"`
		NResults        int         `json:"n_results"`
		IncludeEmbeddings bool      `json:"include_embeddings"`
	}

	request := queryRequest{
		QueryEmbeddings: [][]float32{embedding},
		NResults:        limit,
		IncludeEmbeddings: true,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to query documents: %s", string(bodyBytes))
	}

	var result struct {
		IDs       [][]string          `json:"ids"`
		Embeddings [][]float32       `json:"embeddings"`
		Metadatas [][]map[string]interface{} `json:"metadatas"`
		Documents [][]string          `json:"documents"`
		Distances [][]float32       `json:"distances"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.IDs) == 0 || len(result.IDs[0]) == 0 {
		return []*QueryResult{}, nil
	}

	// Build query results from response
	queryResults := make([]*QueryResult, len(result.IDs[0]))
	for i := range result.IDs[0] {
		doc := &Document{
			ID:      result.IDs[0][i],
			Content: result.Documents[0][i],
		}

		// Extract embedding
		if len(result.Embeddings) > 0 && i < len(result.Embeddings[0]) {
			embedding := make(Vector, len(result.Embeddings[0][i]))
			for j, v := range result.Embeddings[0][i] {
				embedding[j] = v
			}
			doc.Embedding = embedding
		}

		// Extract metadata
		if len(result.Metadatas) > 0 && i < len(result.Metadatas[0]) {
			metadata := make(map[string]string)
			for k, v := range result.Metadatas[0][i] {
				if str, ok := v.(string); ok {
					metadata[k] = str
				} else {
					metadata[k] = fmt.Sprintf("%v", v)
				}
			}
			doc.Metadata = metadata
		}

		// Calculate score (1 - distance) since Chroma returns distances
		var score float32 = 1.0
		if len(result.Distances) > 0 && i < len(result.Distances[0]) {
			score = 1.0 - result.Distances[0][i]
		}

		queryResults[i] = &QueryResult{
			Document: doc,
			Score:    score,
		}
	}

	return queryResults, nil
}

// QueryByText performs a similarity search using text
func (s *ChromaStore) QueryByText(ctx context.Context, text string, limit int) ([]*QueryResult, error) {
	// Generate embedding for the query text
	embedding, err := s.embeddingProvider.GetEmbedding(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Query using the embedding
	return s.Query(ctx, embedding, limit)
}

// Delete removes a document from the vector store
func (s *ChromaStore) Delete(ctx context.Context, id string) error {
	return s.DeleteMany(ctx, []string{id})
}

// DeleteMany removes multiple documents from the vector store
func (s *ChromaStore) DeleteMany(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	url := fmt.Sprintf("%s/api/v1/collections/%s/delete", s.config.URL, s.collectionID)

	type deleteRequest struct {
		IDs []string `json:"ids"`
	}

	request := deleteRequest{
		IDs: ids,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete documents: %s", string(bodyBytes))
	}

	return nil
}

// Close closes the vector store
func (s *ChromaStore) Close() error {
	// No resources to close for HTTP client
	return nil
}