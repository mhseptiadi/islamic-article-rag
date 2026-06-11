package qdrant

import (
	"github.com/mhseptiadi/islamic-article-rag/internal/model"
	"github.com/qdrant/go-client/qdrant"
)

const (
	DenseVectorName  = "dense"
	SparseVectorName = "sparse"
)

func newDenseVectors(vector []float32) *qdrant.Vectors {
	return newNamedVectors(vector, nil)
}

func newSparseVectors(sparse model.SparseVector) *qdrant.Vectors {
	return qdrant.NewVectorsMap(map[string]*qdrant.Vector{
		SparseVectorName: qdrant.NewVectorSparse(sparse.Indices, sparse.Values),
	})
}

// newNamedVectors builds a point with dense and optional sparse vectors.
// Sparse values should be term frequencies only; Qdrant applies collection-level
// IDF (BM25) when the sparse vector is configured with modifier "idf".
func newNamedVectors(dense []float32, sparse *model.SparseVector) *qdrant.Vectors {
	vectors := map[string]*qdrant.Vector{
		DenseVectorName: qdrant.NewVectorDense(dense),
	}
	if sparse != nil && sparse.HasValues() {
		vectors[SparseVectorName] = qdrant.NewVectorSparse(sparse.Indices, sparse.Values)
	}
	return qdrant.NewVectorsMap(vectors)
}
