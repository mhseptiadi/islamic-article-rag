package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/mhseptiadi/islamic-article-rag/internal/model"
	"github.com/mhseptiadi/islamic-article-rag/internal/repository/qdrant"
	"github.com/mhseptiadi/islamic-article-rag/pkg/regexutil"
)

type IngestionService struct {
	embedder *EmbeddingClient
	vectors  *qdrant.VectorRepository
}

func NewIngestionService(embedder *EmbeddingClient, vectors *qdrant.VectorRepository) *IngestionService {
	return &IngestionService{
		embedder: embedder,
		vectors:  vectors,
	}
}

func (s *IngestionService) IngestArticle(ctx context.Context, articleID, title, body string) error {
	paragraphs := splitParagraphs(body)
	chunks := make([]model.Chunk, 0, len(paragraphs))

	for i, paragraph := range paragraphs {
		refs := regexutil.ExtractQuranReferences(paragraph)
		refStrings := make([]string, len(refs))
		for j, ref := range refs {
			refStrings[j] = ref.Raw
		}

		embeddings, err := s.embedder.Embed(ctx, []string{paragraph})
		if err != nil {
			return err
		}

		chunks = append(chunks, model.Chunk{
			ID:     articleID + "-" + strconv.Itoa(i),
			Vector: embeddings[0],
			Payload: model.Payload{
				Text: paragraph,
				Metadata: model.Metadata{
					ArticleID:    articleID,
					Title:        title,
					QuranRefs:    refStrings,
					ParagraphIdx: i,
				},
			},
		})
	}

	return s.vectors.InsertChunks(ctx, chunks)
}

func splitParagraphs(body string) []string {
	raw := strings.Split(body, "\n\n")
	parts := make([]string, 0, len(raw))
	for _, p := range raw {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}
