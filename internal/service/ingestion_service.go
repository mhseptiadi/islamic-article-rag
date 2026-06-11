package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/mhseptiadi/islamic-article-rag/internal/model"
	"github.com/mhseptiadi/islamic-article-rag/internal/repository/qdrant"
	"github.com/mhseptiadi/islamic-article-rag/pkg/regexutil"
)

type IngestionService struct {
	embedder *EmbeddingClient
	vectors  *qdrant.VectorRepository
	articles *qdrant.ArticleRepository
}

func NewIngestionService(
	embedder *EmbeddingClient,
	vectors *qdrant.VectorRepository,
	articles *qdrant.ArticleRepository,
) *IngestionService {
	return &IngestionService{
		embedder: embedder,
		vectors:  vectors,
		articles: articles,
	}
}

func (s *IngestionService) IngestDirectory(ctx context.Context, rawDir string, windowSize, stepSize int) (int, error) {
	entries, err := os.ReadDir(rawDir)
	if err != nil {
		return 0, fmt.Errorf("read directory %s: %w", rawDir, err)
	}

	var allChunks []model.Chunk

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		fullPath := filepath.Join(rawDir, entry.Name())
		contentBytes, err := os.ReadFile(fullPath)
		if err != nil {
			return 0, fmt.Errorf("read file %s: %w", fullPath, err)
		}

		content := string(contentBytes)
		sourceURL := extractSourceURL(content)
		if sourceURL == "" {
			sourceURL = fmt.Sprintf("file://%s", entry.Name())
		}

		articleID := uuid.New().String()
		if err := s.articles.InsertArticle(ctx, model.Article{
			ID:   articleID,
			Text: content,
			URL:  sourceURL,
		}); err != nil {
			return 0, fmt.Errorf("insert article %s: %w", entry.Name(), err)
		}

		chunks, err := s.chunkFile(ctx, content, sourceURL, articleID, windowSize, stepSize)
		if err != nil {
			return 0, fmt.Errorf("process file %s: %w", entry.Name(), err)
		}

		allChunks = append(allChunks, chunks...)
		fmt.Printf("Processed file: %s\n", entry.Name())
	}

	if len(allChunks) == 0 {
		return 0, nil
	}

	if err := s.vectors.InsertChunks(ctx, allChunks); err != nil {
		return 0, err
	}

	return len(allChunks), nil
}

func (s *IngestionService) IngestArticle(ctx context.Context, articleID, title, body, sourceURL string) error {
	fullText := body
	if title != "" {
		fullText = title + "\n\n" + body
	}

	if err := s.articles.InsertArticle(ctx, model.Article{
		ID:   articleID,
		Text: fullText,
		URL:  sourceURL,
	}); err != nil {
		return err
	}

	paragraphs := splitParagraphs(body)
	chunks := make([]model.Chunk, 0, len(paragraphs))

	for i, paragraph := range paragraphs {
		refs := regexutil.ExtractQuranReferences(paragraph)
		refStrings := make([]string, len(refs))
		for j, ref := range refs {
			refStrings[j] = ref.Raw
		}

		paragraph = removeArabicText(paragraph)
		if !isEmbeddableChunk(paragraph) {
			continue
		}

		embeddings, err := s.embedder.Embed(ctx, []string{paragraph})
		if err != nil {
			return err
		}

		chunks = append(chunks, model.Chunk{
			ID:          articleID + "-" + strconv.Itoa(i),
			DenseVector: embeddings[0],
			Payload: model.Payload{
				Text: paragraph,
				Metadata: model.Metadata{
					ArticleID:    articleID,
					Title:        title,
					SourceURL:    sourceURL,
					QuranRefs:    refStrings,
					ParagraphIdx: i,
				},
			},
		})
	}

	return s.vectors.InsertChunks(ctx, chunks)
}

func (s *IngestionService) chunkFile(ctx context.Context, content, sourceURL, articleID string, windowSize, stepSize int) ([]model.Chunk, error) {
	paragraphs := strings.Split(content, "\n")
	var chunks []model.Chunk

	for i := 0; i < len(paragraphs); {
		end := i + windowSize
		if end > len(paragraphs) {
			end = len(paragraphs)
		}

		chunkText := strings.Join(paragraphs[i:end], "\n")
		chunkText = strings.TrimSpace(removeArabicText(chunkText))
		if !isEmbeddableChunk(chunkText) {
			if end == len(paragraphs) {
				break
			}
			i += stepSize
			continue
		}

		refs := regexutil.ExtractQuranReferences(chunkText)
		refStrings := make([]string, len(refs))
		for j, ref := range refs {
			refStrings[j] = ref.Raw
		}

		embeddings, err := s.embedder.Embed(ctx, []string{chunkText})
		if err != nil {
			return nil, err
		}

		chunks = append(chunks, model.Chunk{
			ID:          uuid.New().String(),
			DenseVector: embeddings[0],
			Payload: model.Payload{
				Text: chunkText,
				Metadata: model.Metadata{
					ArticleID: articleID,
					SourceURL: sourceURL,
					QuranRefs: refStrings,
				},
			},
		})

		if end == len(paragraphs) {
			break
		}
		i += stepSize
	}

	return chunks, nil
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

func extractSourceURL(text string) string {
	re := regexp.MustCompile(`https?://[^\s\)]+`)
	match := re.FindString(text)
	return strings.TrimRight(match, "\")]*")
}

func removeArabicText(text string) string {
	re := regexp.MustCompile(`[\x{0600}-\x{06FF}]+`)
	return re.ReplaceAllString(text, "")
}

func isEmbeddableChunk(text string) bool {
	for _, r := range text {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}
