package service

import (
	"context"
	"fmt"

	"github.com/mhseptiadi/islamic-article-rag/internal/model"
	"github.com/mhseptiadi/islamic-article-rag/internal/repository/qdrant"
)

type QnAOrchestrator struct {
	embedder *EmbeddingClient
	llm      *LLMClient
	vectors  *qdrant.VectorRepository
}

func NewQnAOrchestrator(
	embedder *EmbeddingClient,
	llm *LLMClient,
	vectors *qdrant.VectorRepository,
) *QnAOrchestrator {
	return &QnAOrchestrator{
		embedder: embedder,
		llm:      llm,
		vectors:  vectors,
	}
}

type AskResult struct {
	Answer string        `json:"answer"`
	Chunks []model.Chunk `json:"chunks"`
}

func (o *QnAOrchestrator) Ask(ctx context.Context, question string) (*AskResult, error) {
	embeddings, err := o.embedder.Embed(ctx, []string{question})
	if err != nil {
		return nil, err
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings generated for question")
	}

	chunks, err := o.vectors.SearchSimilar(ctx, embeddings[0], 5)
	if err != nil {
		return nil, err
	}

	contextBlocks := make([]string, len(chunks))
	for i, chunk := range chunks {
		contextBlocks[i] = chunk.Payload.Text
	}

	answer, err := o.llm.GenerateAnswer(ctx, question, contextBlocks)
	if err != nil {
		return nil, err
	}

	return &AskResult{
		Answer: answer,
		Chunks: chunks,
	}, nil
}
