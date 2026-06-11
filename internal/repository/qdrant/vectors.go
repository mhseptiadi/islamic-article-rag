package qdrant

import (
	"github.com/mhseptiadi/islamic-article-rag/internal/model"
	"github.com/qdrant/go-client/qdrant"
)

const (
	DenseVectorName  = "dense"
	SparseVectorName = "sparse"
	BM25Model        = "qdrant/bm25"
)

func newDenseVectors(vector []float32) *qdrant.Vectors {
	return newNamedVectors(vector, "")
}

func newSparseVectors(sparse model.SparseVector) *qdrant.Vectors {
	return qdrant.NewVectorsMap(map[string]*qdrant.Vector{
		SparseVectorName: qdrant.NewVectorSparse(sparse.Indices, sparse.Values),
	})
}

// newNamedVectors builds a point with dense and BM25 sparse vectors.
// Sparse is generated server-side from sparseText; Qdrant applies collection-level
// IDF when the sparse vector is configured with modifier "idf".
func newNamedVectors(dense []float32, sparseText string) *qdrant.Vectors {
	vectors := map[string]*qdrant.Vector{
		DenseVectorName: qdrant.NewVectorDense(dense),
	}
	if sparseText != "" {
		vectors[SparseVectorName] = qdrant.NewVectorDocument(&qdrant.Document{
			Model: BM25Model,
			Text:  sparseText,
		})
	}
	return qdrant.NewVectorsMap(vectors)
}
