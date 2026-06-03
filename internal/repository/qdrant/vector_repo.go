package qdrant

import (
	"context"

	"github.com/mhseptiadi/islamic-article-rag/internal/model"
)

type VectorRepository struct {
	baseURL        string
	collectionName string
}

func NewVectorRepository(baseURL, collectionName string) *VectorRepository {
	return &VectorRepository{
		baseURL:        baseURL,
		collectionName: collectionName,
	}
}

func (r *VectorRepository) InsertChunks(ctx context.Context, chunks []model.Chunk) error {
	_ = ctx
	_ = chunks
	return nil
}

func (r *VectorRepository) SearchSimilar(ctx context.Context, vector []float32, limit int) ([]model.Chunk, error) {
	_ = ctx
	_ = vector
	_ = limit
	return nil, nil
}
