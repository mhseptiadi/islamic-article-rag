package qdrant

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/mhseptiadi/islamic-article-rag/internal/model"
	"github.com/qdrant/go-client/qdrant"
)

type VectorRepository struct {
	client             *qdrant.Client
	collectionName     string
	minSimilarityScore float32
}

func NewVectorRepository(host string, apiKey string, grpcPort int, collectionName string, minSimilarityScore float64) (*VectorRepository, error) {
	host = normalizeQdrantHost(host)
	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   host,
		APIKey: apiKey,
		Port:   grpcPort,
		UseTLS: apiKey != "",
	})
	if err != nil {
		return nil, fmt.Errorf("connect to qdrant: %w", err)
	}

	return &VectorRepository{
		client:             client,
		collectionName:     collectionName,
		minSimilarityScore: float32(minSimilarityScore),
	}, nil
}

func (r *VectorRepository) Close() error {
	return r.client.Close()
}

func (r *VectorRepository) InsertChunks(ctx context.Context, chunks []model.Chunk) error {
	if len(chunks) == 0 {
		return nil
	}

	points := make([]*qdrant.PointStruct, len(chunks))
	for i, chunk := range chunks {
		payload := qdrant.NewValueMap(map[string]any{
			"chunk_text":    chunk.Payload.Text,
			"article_id":    chunk.Payload.Metadata.ArticleID,
			"source_url":    chunk.Payload.Metadata.SourceURL,
			"title":         chunk.Payload.Metadata.Title,
			"paragraph_idx": chunk.Payload.Metadata.ParagraphIdx,
		})

		points[i] = &qdrant.PointStruct{
			Id:      pointIDFromString(chunk.ID),
			Vectors: newNamedVectors(chunk.DenseVector, chunk.Payload.Text),
			Payload: payload,
		}
	}

	_, err := r.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: r.collectionName,
		Points:         points,
	})
	if err != nil {
		return fmt.Errorf("upsert points: %w", err)
	}

	return nil
}

// func (r *VectorRepository) SearchSimilar(ctx context.Context, vector []float32, limit int) ([]model.Chunk, error) {
// 	if len(vector) == 0 {
// 		return nil, fmt.Errorf("search vector is empty")
// 	}

// 	if limit <= 0 {
// 		limit = 5
// 	}
// 	queryLimit := uint64(limit)
// 	scoreThreshold := r.minSimilarityScore

// 	results, err := r.client.Query(ctx, &qdrant.QueryPoints{
// 		CollectionName: r.collectionName,
// 		Query:          qdrant.NewQueryDense(vector),
// 		Using:          qdrant.PtrOf(DenseVectorName),
// 		Limit:          &queryLimit,
// 		ScoreThreshold: &scoreThreshold,
// 		WithPayload:    qdrant.NewWithPayload(true),
// 	})
// 	if err != nil {
// 		return nil, fmt.Errorf("query similar points: %w", err)
// 	}

// 	chunks := make([]model.Chunk, 0, len(results))
// 	for _, point := range results {
// 		if point.GetScore() < r.minSimilarityScore {
// 			continue
// 		}
// 		chunks = append(chunks, scoredPointToChunk(point))
// 	}

// 	return chunks, nil
// }

// HybridSearch combines dense semantic search with sparse BM25 keyword search
// using Reciprocal Rank Fusion (RRF).
func (r *VectorRepository) HybridSearch(ctx context.Context, denseVector []float32, queryText string, limit int) ([]model.Chunk, error) {
	if len(denseVector) == 0 {
		return nil, fmt.Errorf("dense query vector is empty")
	}
	if strings.TrimSpace(queryText) == "" {
		return nil, fmt.Errorf("query text is empty")
	}

	if limit <= 0 {
		limit = 5
	}

	queryLimit := uint64(limit)
	prefetchLimit := queryLimit * 2
	if prefetchLimit < 10 {
		prefetchLimit = 10
	}

	results, err := r.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: r.collectionName,
		Prefetch: []*qdrant.PrefetchQuery{
			{
				Query: qdrant.NewQueryDense(denseVector),
				Using: qdrant.PtrOf(DenseVectorName),
				Limit: qdrant.PtrOf(prefetchLimit),
			},
			{
				Query: qdrant.NewQueryDocument(&qdrant.Document{
					Model: BM25Model,
					Text:  queryText,
				}),
				Using: qdrant.PtrOf(SparseVectorName),
				Limit: qdrant.PtrOf(prefetchLimit),
			},
		},
		Query:       qdrant.NewQueryFusion(qdrant.Fusion_RRF),
		Limit:       qdrant.PtrOf(queryLimit),
		WithPayload: qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("hybrid search: %w", err)
	}

	chunks := make([]model.Chunk, 0, len(results))
	for _, point := range results {
		// fmt.Println("point", point.GetScore())
		if point.GetScore() < r.minSimilarityScore {
			continue
		}
		chunks = append(chunks, scoredPointToChunk(point))
	}

	return chunks, nil
}

func scoredPointToChunk(point *qdrant.ScoredPoint) model.Chunk {
	payload := point.GetPayload()

	return model.Chunk{
		ID:    pointIDString(point.GetId()),
		Score: float64(point.GetScore()),
		Payload: model.Payload{
			Text: payloadString(payload, "chunk_text"),
			Metadata: model.Metadata{
				ArticleID:    payloadString(payload, "article_id"),
				Title:        payloadString(payload, "title"),
				SourceURL:    payloadString(payload, "source_url"),
				ParagraphIdx: payloadInt(payload, "paragraph_idx"),
				QuranRefs:    payloadStringList(payload, "koran_refs"),
			},
		},
	}
}

func pointIDString(id *qdrant.PointId) string {
	if id == nil {
		return ""
	}
	if uuid := id.GetUuid(); uuid != "" {
		return uuid
	}
	return strconv.FormatUint(id.GetNum(), 10)
}

func payloadString(payload map[string]*qdrant.Value, key string) string {
	val, ok := payload[key]
	if !ok || val == nil {
		return ""
	}
	return val.GetStringValue()
}

func payloadInt(payload map[string]*qdrant.Value, key string) int {
	val, ok := payload[key]
	if !ok || val == nil {
		return 0
	}
	if i := val.GetIntegerValue(); i != 0 {
		return int(i)
	}
	if d := val.GetDoubleValue(); d != 0 {
		return int(d)
	}
	return 0
}

func payloadStringList(payload map[string]*qdrant.Value, key string) []string {
	val, ok := payload[key]
	if !ok || val == nil {
		return nil
	}

	list := val.GetListValue()
	if list == nil {
		return nil
	}

	out := make([]string, 0, len(list.GetValues()))
	for _, item := range list.GetValues() {
		if s := item.GetStringValue(); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func normalizeQdrantHost(host string) string {
	host = strings.TrimSpace(host)
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	if i := strings.Index(host, "/"); i >= 0 {
		host = host[:i]
	}
	return host
}
