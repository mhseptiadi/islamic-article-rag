package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/mhseptiadi/islamic-article-rag/internal/model"
	"github.com/mhseptiadi/islamic-article-rag/internal/repository/qdrant"
)

type QnAOrchestrator struct {
	embedder *EmbeddingClient
	llm      *LLMClient
	vectors  *qdrant.VectorRepository
	articles *qdrant.ArticleRepository
}

func NewQnAOrchestrator(
	embedder *EmbeddingClient,
	llm *LLMClient,
	vectors *qdrant.VectorRepository,
	articles *qdrant.ArticleRepository,
) *QnAOrchestrator {
	return &QnAOrchestrator{
		embedder: embedder,
		llm:      llm,
		vectors:  vectors,
		articles: articles,
	}
}

type AskResult struct {
	Answer       string          `json:"answer"`
	FullArticles []model.Article `json:"full_articles"`
	Chunks       []model.Chunk   `json:"chunks"`
}

func (o *QnAOrchestrator) Ask(ctx context.Context, question string) (*AskResult, error) {
	embeddings, err := o.embedder.Embed(ctx, []string{question})
	if err != nil {
		return nil, err
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings generated for question")
	}

	chunks, err := o.vectors.HybridSearch(ctx, embeddings[0], question, 5)
	if err != nil {
		return nil, err
	}

	fullArticles, err := o.resolveFullArticles(ctx, chunks)
	if err != nil {
		return nil, err
	}

	contextBlocks := make([]string, len(fullArticles))
	for i := range fullArticles {
		contextBlocks[i] = formatArticleForContext(&fullArticles[i])
	}

	answer, err := o.llm.GenerateAnswer(ctx, question, contextBlocks)
	if err != nil {
		return nil, err
	}

	return &AskResult{
		Answer:       answer,
		FullArticles: fullArticles,
		Chunks:       chunks,
	}, nil
}

func (o *QnAOrchestrator) resolveFullArticles(ctx context.Context, chunks []model.Chunk) ([]model.Article, error) {
	refs := uniqueArticleRefs(chunks)
	fullArticles := make([]model.Article, 0, len(refs))

	for _, ref := range refs {
		var (
			article *model.Article
			err     error
		)
		switch {
		case ref.articleID != "":
			article, err = o.articles.GetByID(ctx, ref.articleID)
		case ref.sourceURL != "":
			article, err = o.articles.GetByURL(ctx, ref.sourceURL)
		default:
			continue
		}
		if err != nil {
			return nil, err
		}
		if article == nil {
			continue
		}

		fullArticles = append(fullArticles, *article)
	}

	return fullArticles, nil
}

type articleRef struct {
	articleID string
	sourceURL string
}

func uniqueArticleRefs(chunks []model.Chunk) []articleRef {
	seenIDs := make(map[string]bool)
	seenURLs := make(map[string]bool)
	refs := make([]articleRef, 0, len(chunks))

	for _, chunk := range chunks {
		articleID := chunk.Payload.Metadata.ArticleID
		sourceURL := chunk.Payload.Metadata.SourceURL

		if articleID != "" {
			if seenIDs[articleID] {
				continue
			}
			seenIDs[articleID] = true
			if sourceURL != "" {
				seenURLs[sourceURL] = true
			}
			refs = append(refs, articleRef{articleID: articleID, sourceURL: sourceURL})
			continue
		}

		if sourceURL != "" && !seenURLs[sourceURL] {
			seenURLs[sourceURL] = true
			refs = append(refs, articleRef{sourceURL: sourceURL})
		}
	}

	return refs
}

func formatArticleForContext(article *model.Article) string {
	var b strings.Builder
	if article.URL != "" {
		b.WriteString("Source: ")
		b.WriteString(article.URL)
		b.WriteString("\n\n")
	}
	b.WriteString(article.Text)
	return b.String()
}
