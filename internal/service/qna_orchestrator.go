package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mhseptiadi/islamic-article-rag/internal/model"
	"github.com/mhseptiadi/islamic-article-rag/internal/repository/mongo"
	"github.com/mhseptiadi/islamic-article-rag/internal/repository/qdrant"
)

const (
	ContextSourceFullArticles = "full_articles"
	ContextSourceChunks       = "chunks"
)

type QnAOrchestrator struct {
	embedder       *EmbeddingClient
	llm            *LLMClient
	textValidator  *IslamicTextValidatorClient
	vectors        *qdrant.VectorRepository
	articles       *mongo.ArticleRepository
	qnaRecords     *mongo.QnARepository
	retrievalLimit int
	contextSource  string
	llmProvider    string
	llmModel       string
}

func NewQnAOrchestrator(
	embedder *EmbeddingClient,
	llm *LLMClient,
	textValidator *IslamicTextValidatorClient,
	vectors *qdrant.VectorRepository,
	articles *mongo.ArticleRepository,
	qnaRecords *mongo.QnARepository,
	retrievalLimit int,
	contextSource string,
	llmProvider string,
	llmModel string,
) *QnAOrchestrator {
	return &QnAOrchestrator{
		embedder:       embedder,
		llm:            llm,
		textValidator:  textValidator,
		vectors:        vectors,
		articles:       articles,
		qnaRecords:     qnaRecords,
		retrievalLimit: retrievalLimit,
		contextSource:  contextSource,
		llmProvider:    llmProvider,
		llmModel:       llmModel,
	}
}

type AskResult struct {
	RecordID                      string                         `json:"record_id"`
	Answer                        string                         `json:"answer"`
	IslamicTextValidationResponse *IslamicTextValidationResponse `json:"islamicTextValidationResponse"`
	FullArticles                  []model.Article                `json:"full_articles"`
	Chunks                        []model.Chunk                  `json:"chunks"`
}

var ErrInvalidFeedbackType = errors.New("invalid feedback_type")

func (o *QnAOrchestrator) Ask(ctx context.Context, question, clientIP string) (*AskResult, error) {
	embeddings, err := o.embedder.Embed(ctx, []string{question})
	if err != nil {
		return nil, err
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings generated for question")
	}

	chunks, err := o.vectors.HybridSearch(ctx, embeddings[0], question, o.retrievalLimit)
	if err != nil {
		return nil, err
	}

	var (
		fullArticles  []model.Article
		contextBlocks []string
	)

	switch o.contextSource {
	case ContextSourceChunks:
		contextBlocks = make([]string, len(chunks))
		for i := range chunks {
			contextBlocks[i] = formatChunkForContext(&chunks[i])
		}
	default:
		fullArticles, err = o.resolveFullArticles(ctx, chunks)
		if err != nil {
			return nil, err
		}
		contextBlocks = make([]string, len(fullArticles))
		for i := range fullArticles {
			contextBlocks[i] = formatArticleForContext(&fullArticles[i])
		}
	}

	answer, err := o.llm.GenerateAnswer(ctx, question, contextBlocks)
	if err != nil {
		return nil, err
	}

	validation, err := o.textValidator.Validate(ctx, answer)
	var validatedAnswer string
	if err != nil {
		validatedAnswer = ReplaceIslamicTagsOnValidationError(answer)
	} else {
		validatedAnswer = validation.ReplacedText
		if validatedAnswer == "" {
			validatedAnswer = answer
		}
	}

	recordID := uuid.New().String()
	if err := o.qnaRecords.Insert(ctx, model.QnARecord{
		ID:              recordID,
		Question:        question,
		Answer:          answer,
		ValidatedAnswer: validatedAnswer,
		LLMProvider:     o.llmProvider,
		LLMModel:        o.llmModel,
		ContextSource:   o.contextSource,
		ArticleIDs:      articleIDsFromArticles(fullArticles),
		Chunks:          chunksForQnARecord(chunks),
		CreatedAt:       time.Now().UTC(),
		IPAddress:       clientIP,
	}); err != nil {
		return nil, fmt.Errorf("record qna: %w", err)
	}

	return &AskResult{
		RecordID:                      recordID,
		Answer:                        validatedAnswer,
		IslamicTextValidationResponse: validation,
		FullArticles:                  fullArticles,
		Chunks:                        chunks,
	}, nil
}

func (o *QnAOrchestrator) SubmitFeedback(ctx context.Context, recordID string, feedbackType model.FeedbackType, comment string) error {
	if recordID == "" {
		return fmt.Errorf("record_id is required")
	}
	if !feedbackType.Valid() {
		return ErrInvalidFeedbackType
	}
	return o.qnaRecords.UpdateFeedback(ctx, recordID, feedbackType, comment)
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

func articleIDsFromArticles(articles []model.Article) []string {
	ids := make([]string, 0, len(articles))
	for _, article := range articles {
		if article.ID != "" {
			ids = append(ids, article.ID)
		}
	}
	return ids
}

func chunksForQnARecord(chunks []model.Chunk) []model.QnAChunk {
	refs := make([]model.QnAChunk, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk.ID == "" {
			continue
		}
		refs = append(refs, model.QnAChunk{
			ID:    chunk.ID,
			Score: chunk.Score,
		})
	}
	return refs
}

func formatChunkForContext(chunk *model.Chunk) string {
	var b strings.Builder
	meta := chunk.Payload.Metadata
	if meta.SourceURL != "" {
		b.WriteString("Source: ")
		b.WriteString(meta.SourceURL)
		b.WriteString("\n\n")
	}
	if meta.Title != "" {
		b.WriteString(meta.Title)
		b.WriteString("\n\n")
	}
	b.WriteString(chunk.Payload.Text)
	return b.String()
}
