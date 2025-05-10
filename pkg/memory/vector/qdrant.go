package vector

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/blockhead-consulting/guild/pkg/providers"
	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// DefaultQdrantCollection is the default collection name
	DefaultQdrantCollection = "guild_documents"
	
	// DefaultQdrantDimension is the default embedding dimension
	DefaultQdrantDimension = 1536 // OpenAI embedding dimension
)

// QdrantConfig contains configuration for the Qdrant vector store
type QdrantConfig struct {
	// Address is the address of the Qdrant server (e.g., "localhost:6334")
	Address string

	// Collection is the name of the collection to use
	Collection string

	// Dimension is the dimension of the vector embeddings
	Dimension uint64

	// EmbeddingProvider is the provider used to generate embeddings
	EmbeddingProvider EmbeddingProvider
}

// QdrantStore implements VectorStore using Qdrant
type QdrantStore struct {
	client            *qdrant.QdrantClient
	config            *QdrantConfig
	embeddingProvider EmbeddingProvider
	conn              *grpc.ClientConn
}

// NewQdrantStore creates a new Qdrant vector store
func NewQdrantStore(ctx context.Context, config *QdrantConfig) (*QdrantStore, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.Address == "" {
		return nil, fmt.Errorf("address cannot be empty")
	}

	if config.Collection == "" {
		config.Collection = DefaultQdrantCollection
	}

	if config.Dimension == 0 {
		config.Dimension = DefaultQdrantDimension
	}

	if config.EmbeddingProvider == nil {
		return nil, fmt.Errorf("embedding provider cannot be nil")
	}

	// Connect to Qdrant
	conn, err := grpc.Dial(config.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Qdrant: %w", err)
	}

	client := qdrant.NewQdrantClient(conn)

	// Create collection if it doesn't exist
	exists, err := collectionExists(ctx, client, config.Collection)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to check if collection exists: %w", err)
	}

	if !exists {
		if err := createCollection(ctx, client, config.Collection, config.Dimension); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to create collection: %w", err)
		}
	}

	return &QdrantStore{
		client:            client,
		config:            config,
		embeddingProvider: config.EmbeddingProvider,
		conn:              conn,
	}, nil
}

// Store stores a document with its embedding
func (s *QdrantStore) Store(ctx context.Context, doc *Document) error {
	var err error
	var embedding Vector

	// If the document doesn't have an embedding, generate one
	if doc.Embedding == nil {
		embedding, err = s.embeddingProvider.GetEmbedding(ctx, doc.Content)
		if err != nil {
			return fmt.Errorf("failed to generate embedding: %w", err)
		}
	} else {
		embedding = doc.Embedding
	}

	// Convert metadata to payload
	payload, err := mapToPayload(doc.Metadata)
	if err != nil {
		return fmt.Errorf("failed to convert metadata to payload: %w", err)
	}

	// Add content to payload
	contentPayload, err := mapToPayload(map[string]string{
		"content": doc.Content,
	})
	if err != nil {
		return fmt.Errorf("failed to convert content to payload: %w", err)
	}

	// Merge payloads
	for k, v := range contentPayload {
		payload[k] = v
	}

	// Convert embedding to float32 array
	vectors := make([]float32, len(embedding))
	for i, v := range embedding {
		vectors[i] = v
	}

	// Create point
	point := &qdrant.PointStruct{
		Id: &qdrant.PointId{
			PointIdOptions: &qdrant.PointId_Uuid{
				Uuid: doc.ID,
			},
		},
		Vectors: &qdrant.Vectors{
			VectorsOptions: &qdrant.Vectors_Vector{
				Vector: &qdrant.Vector{
					Data: vectors,
				},
			},
		},
		Payload: payload,
	}

	// Upsert point
	_, err = s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.config.Collection,
		Points:         []*qdrant.PointStruct{point},
	})

	if err != nil {
		return fmt.Errorf("failed to upsert point: %w", err)
	}

	return nil
}

// StoreMany stores multiple documents with their embeddings
func (s *QdrantStore) StoreMany(ctx context.Context, docs []*Document) error {
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

	// Create points
	points := make([]*qdrant.PointStruct, len(docs))
	for i, doc := range docs {
		// Convert metadata to payload
		payload, err := mapToPayload(doc.Metadata)
		if err != nil {
			return fmt.Errorf("failed to convert metadata to payload: %w", err)
		}

		// Add content to payload
		contentPayload, err := mapToPayload(map[string]string{
			"content": doc.Content,
		})
		if err != nil {
			return fmt.Errorf("failed to convert content to payload: %w", err)
		}

		// Merge payloads
		for k, v := range contentPayload {
			payload[k] = v
		}

		// Convert embedding to float32 array
		vectors := make([]float32, len(doc.Embedding))
		for j, v := range doc.Embedding {
			vectors[j] = v
		}

		// Create point
		points[i] = &qdrant.PointStruct{
			Id: &qdrant.PointId{
				PointIdOptions: &qdrant.PointId_Uuid{
					Uuid: doc.ID,
				},
			},
			Vectors: &qdrant.Vectors{
				VectorsOptions: &qdrant.Vectors_Vector{
					Vector: &qdrant.Vector{
						Data: vectors,
					},
				},
			},
			Payload: payload,
		}
	}

	// Upsert points
	_, err := s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.config.Collection,
		Points:         points,
	})

	if err != nil {
		return fmt.Errorf("failed to upsert points: %w", err)
	}

	return nil
}

// Retrieve retrieves a document by ID
func (s *QdrantStore) Retrieve(ctx context.Context, id string) (*Document, error) {
	// Create point ID
	pointID := &qdrant.PointId{
		PointIdOptions: &qdrant.PointId_Uuid{
			Uuid: id,
		},
	}

	// Retrieve point
	resp, err := s.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: s.config.Collection,
		Ids:            []*qdrant.PointId{pointID},
		WithVectors:    true,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve point: %w", err)
	}

	if len(resp.Result) == 0 {
		return nil, fmt.Errorf("document not found")
	}

	// Convert point to document
	point := resp.Result[0]
	doc, err := pointToDocument(point)
	if err != nil {
		return nil, fmt.Errorf("failed to convert point to document: %w", err)
	}

	return doc, nil
}

// Query performs a similarity search on the vector store
func (s *QdrantStore) Query(ctx context.Context, embedding Vector, limit int) ([]*QueryResult, error) {
	if limit <= 0 {
		limit = 10 // Default limit
	}

	// Convert embedding to float32 array
	vector := make([]float32, len(embedding))
	for i, v := range embedding {
		vector[i] = v
	}

	// Create search request
	resp, err := s.client.Search(ctx, &qdrant.SearchPoints{
		CollectionName: s.config.Collection,
		Vector:         vector,
		Limit:          uint64(limit),
		WithVectors:    true,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to search points: %w", err)
	}

	// Convert search results to query results
	results := make([]*QueryResult, len(resp.Result))
	for i, point := range resp.Result {
		doc, err := pointToDocument(point.Point)
		if err != nil {
			return nil, fmt.Errorf("failed to convert point to document: %w", err)
		}

		results[i] = &QueryResult{
			Document: doc,
			Score:    point.Score,
		}
	}

	return results, nil
}

// QueryByText performs a similarity search using text
func (s *QdrantStore) QueryByText(ctx context.Context, text string, limit int) ([]*QueryResult, error) {
	// Generate embedding for the query text
	embedding, err := s.embeddingProvider.GetEmbedding(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Query using the embedding
	return s.Query(ctx, embedding, limit)
}

// Delete removes a document from the vector store
func (s *QdrantStore) Delete(ctx context.Context, id string) error {
	// Create point ID
	pointID := &qdrant.PointId{
		PointIdOptions: &qdrant.PointId_Uuid{
			Uuid: id,
		},
	}

	// Delete point
	_, err := s.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: s.config.Collection,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Ids{
				Ids: &qdrant.PointsIdsList{
					Ids: []*qdrant.PointId{pointID},
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to delete point: %w", err)
	}

	return nil
}

// DeleteMany removes multiple documents from the vector store
func (s *QdrantStore) DeleteMany(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	// Create point IDs
	pointIDs := make([]*qdrant.PointId, len(ids))
	for i, id := range ids {
		pointIDs[i] = &qdrant.PointId{
			PointIdOptions: &qdrant.PointId_Uuid{
				Uuid: id,
			},
		}
	}

	// Delete points
	_, err := s.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: s.config.Collection,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Ids{
				Ids: &qdrant.PointsIdsList{
					Ids: pointIDs,
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to delete points: %w", err)
	}

	return nil
}

// Close closes the vector store
func (s *QdrantStore) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

// Helper functions

// collectionExists checks if a collection exists
func collectionExists(ctx context.Context, client *qdrant.QdrantClient, name string) (bool, error) {
	resp, err := client.ListCollections(ctx, &qdrant.ListCollectionsRequest{})
	if err != nil {
		return false, err
	}

	for _, collection := range resp.Collections {
		if collection.Name == name {
			return true, nil
		}
	}

	return false, nil
}

// createCollection creates a new collection
func createCollection(ctx context.Context, client *qdrant.QdrantClient, name string, dimension uint64) error {
	_, err := client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: name,
		VectorsConfig: map[string]*qdrant.VectorParams{
			"": {
				Size:     dimension,
				Distance: qdrant.Distance_Cosine,
			},
		},
	})

	if err != nil {
		return err
	}

	// Wait for the collection to be created
	for i := 0; i < 10; i++ {
		resp, err := client.CollectionInfo(ctx, &qdrant.CollectionInfoRequest{
			CollectionName: name,
		})
		if err == nil && resp.Result.Status == qdrant.CollectionStatus_Green {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timed out waiting for collection to be created")
}

// mapToPayload converts a map to a Qdrant payload
func mapToPayload(m map[string]string) (map[string]*qdrant.Value, error) {
	payload := make(map[string]*qdrant.Value)
	for k, v := range m {
		payload[k] = &qdrant.Value{
			Kind: &qdrant.Value_StringValue{
				StringValue: v,
			},
		}
	}
	return payload, nil
}

// pointToDocument converts a Qdrant point to a document
func pointToDocument(point *qdrant.PointStruct) (*Document, error) {
	// Extract ID
	var id string
	switch pID := point.Id.PointIdOptions.(type) {
	case *qdrant.PointId_Uuid:
		id = pID.Uuid
	case *qdrant.PointId_Num:
		id = fmt.Sprintf("%d", pID.Num)
	default:
		return nil, fmt.Errorf("unsupported point ID type")
	}

	// Extract embedding
	var embedding Vector
	switch vec := point.Vectors.VectorsOptions.(type) {
	case *qdrant.Vectors_Vector:
		embedding = make(Vector, len(vec.Vector.Data))
		for i, v := range vec.Vector.Data {
			embedding[i] = v
		}
	default:
		return nil, fmt.Errorf("unsupported vector type")
	}

	// Extract content and metadata
	content := ""
	metadata := make(map[string]string)
	for k, v := range point.Payload {
		if k == "content" {
			switch val := v.Kind.(type) {
			case *qdrant.Value_StringValue:
				content = val.StringValue
			}
		} else {
			switch val := v.Kind.(type) {
			case *qdrant.Value_StringValue:
				metadata[k] = val.StringValue
			case *qdrant.Value_IntValue:
				metadata[k] = fmt.Sprintf("%d", val.IntValue)
			case *qdrant.Value_BoolValue:
				metadata[k] = fmt.Sprintf("%t", val.BoolValue)
			}
		}
	}

	return &Document{
		ID:        id,
		Content:   content,
		Metadata:  metadata,
		Embedding: embedding,
	}, nil
}

// OpenAIEmbeddingProvider implements EmbeddingProvider using OpenAI embeddings
type OpenAIEmbeddingProvider struct {
	client providers.LLMClient
	model  string
}

// NewOpenAIEmbeddingProvider creates a new OpenAI embedding provider
func NewOpenAIEmbeddingProvider(client providers.LLMClient, model string) *OpenAIEmbeddingProvider {
	if model == "" {
		model = "text-embedding-3-small" // Default model
	}
	
	return &OpenAIEmbeddingProvider{
		client: client,
		model:  model,
	}
}

// GetEmbedding generates an embedding for the given text
func (p *OpenAIEmbeddingProvider) GetEmbedding(ctx context.Context, text string) (Vector, error) {
	req := &providers.EmbeddingRequest{
		Text:  text,
		Model: p.model,
	}
	
	resp, err := p.client.CreateEmbedding(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}
	
	// Convert to Vector
	embedding := make(Vector, len(resp.Embedding))
	for i, v := range resp.Embedding {
		embedding[i] = float32(v)
	}
	
	return embedding, nil
}

// GetEmbeddings generates embeddings for multiple texts
func (p *OpenAIEmbeddingProvider) GetEmbeddings(ctx context.Context, texts []string) ([]Vector, error) {
	if len(texts) == 0 {
		return []Vector{}, nil
	}
	
	req := &providers.EmbeddingRequest{
		Texts: texts,
		Model: p.model,
	}
	
	resp, err := p.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}
	
	// Convert to Vectors
	embeddings := make([]Vector, len(resp.Embeddings))
	for i, e := range resp.Embeddings {
		embedding := make(Vector, len(e))
		for j, v := range e {
			embedding[j] = float32(v)
		}
		embeddings[i] = embedding
	}
	
	return embeddings, nil
}