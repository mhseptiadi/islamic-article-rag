package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	// "strconv"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/mhseptiadi/islamic-article-rag/internal/model"
	"github.com/mhseptiadi/islamic-article-rag/internal/repository/mongo"
	"github.com/mhseptiadi/islamic-article-rag/internal/repository/qdrant"
	"github.com/mhseptiadi/islamic-article-rag/pkg/regexutil"
)

type IngestionService struct {
	embedder      *EmbeddingClient
	vectors       *qdrant.VectorRepository
	articles      *mongo.ArticleRepository
	maxChunkChars int
}

func NewIngestionService(
	embedder *EmbeddingClient,
	vectors *qdrant.VectorRepository,
	articles *mongo.ArticleRepository,
	maxChunkChars int,
) *IngestionService {
	return &IngestionService{
		embedder:      embedder,
		vectors:       vectors,
		articles:      articles,
		maxChunkChars: maxChunkChars,
	}
}

func (s *IngestionService) IngestDirectory(ctx context.Context, rawDir string, windowSize, stepSize int) (int, error) {
	entries, err := os.ReadDir(rawDir)
	if err != nil {
		return 0, fmt.Errorf("read directory %s: %w", rawDir, err)
	}

	errorsDir := filepath.Join(filepath.Dir(rawDir), "errors")
	doneDir := filepath.Join(filepath.Dir(rawDir), "done")
	if err := os.MkdirAll(errorsDir, 0o755); err != nil {
		return 0, fmt.Errorf("create errors directory %s: %w", errorsDir, err)
	}
	if err := os.MkdirAll(doneDir, 0o755); err != nil {
		return 0, fmt.Errorf("create done directory %s: %w", doneDir, err)
	}

	totalChunks := 0
	failedFiles := 0
	doneFiles := 0

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		fullPath := filepath.Join(rawDir, entry.Name())
		contentBytes, err := os.ReadFile(fullPath)
		if err != nil {
			failedFiles++
			fmt.Printf("Failed file %s: read: %v\n", entry.Name(), err)
			if moveErr := moveIngestionFile(fullPath, errorsDir); moveErr != nil {
				fmt.Printf("Failed to move %s to errors: %v\n", entry.Name(), moveErr)
			}
			continue
		}

		content := string(contentBytes)
		sourceURL := extractSourceURL(content)
		if sourceURL == "" {
			sourceURL = fmt.Sprintf("file://%s", entry.Name())
		}

		existing, err := s.articles.GetByURL(ctx, sourceURL)
		if err != nil {
			failedFiles++
			fmt.Printf("Failed file %s: check article: %v\n", entry.Name(), err)
			if moveErr := moveIngestionFile(fullPath, errorsDir); moveErr != nil {
				fmt.Printf("Failed to move %s to errors: %v\n", entry.Name(), moveErr)
			}
			continue
		}
		if existing != nil {
			fmt.Printf("Skipped file (duplicate URL): %s\n", entry.Name())
			doneFiles++
			if moveErr := moveIngestionFile(fullPath, doneDir); moveErr != nil {
				fmt.Printf("Failed to move %s to done: %v\n", entry.Name(), moveErr)
			}
			continue
		}

		articleID := uuid.New().String()

		chunks, err := s.chunkFile(ctx, content, sourceURL, articleID, windowSize, stepSize)
		if err != nil {
			failedFiles++
			fmt.Printf("Failed file %s: process file: %v\n", entry.Name(), err)
			if moveErr := moveIngestionFile(fullPath, errorsDir); moveErr != nil {
				fmt.Printf("Failed to move %s to errors: %v\n", entry.Name(), moveErr)
			}
			continue
		}

		if len(chunks) > 0 {
			if err := s.vectors.InsertChunks(ctx, chunks); err != nil {
				failedFiles++
				fmt.Printf("Failed file %s: insert chunks: %v\n", entry.Name(), err)
				if moveErr := moveIngestionFile(fullPath, errorsDir); moveErr != nil {
					fmt.Printf("Failed to move %s to errors: %v\n", entry.Name(), moveErr)
				}
				continue
			}
			totalChunks += len(chunks)
		}

		if err := s.articles.InsertArticle(ctx, model.Article{
			ID:   articleID,
			Text: content,
			URL:  sourceURL,
		}); err != nil {
			failedFiles++
			fmt.Printf("Failed file %s: insert article: %v\n", entry.Name(), err)
			if moveErr := moveIngestionFile(fullPath, errorsDir); moveErr != nil {
				fmt.Printf("Failed to move %s to errors: %v\n", entry.Name(), moveErr)
			}
			continue
		}

		fmt.Printf("Processed file: %s (%d chunks)\n", entry.Name(), len(chunks))
		doneFiles++
		if moveErr := moveIngestionFile(fullPath, doneDir); moveErr != nil {
			fmt.Printf("Failed to move %s to done: %v\n", entry.Name(), moveErr)
		}
	}

	if doneFiles > 0 {
		fmt.Printf("\nMoved %d completed file(s) to %s\n", doneFiles, doneDir)
	}
	if failedFiles > 0 {
		fmt.Printf("\nMoved %d failed file(s) to %s\n", failedFiles, errorsDir)
	}

	return totalChunks, nil
}

func moveIngestionFile(srcPath, destDir string) error {
	destPath := filepath.Join(destDir, filepath.Base(srcPath))
	if _, err := os.Stat(destPath); err == nil {
		ext := filepath.Ext(destPath)
		base := strings.TrimSuffix(filepath.Base(destPath), ext)
		destPath = filepath.Join(destDir, fmt.Sprintf("%s-%s%s", base, uuid.New().String()[:8], ext))
	}
	return os.Rename(srcPath, destPath)
}

// func (s *IngestionService) IngestArticle(ctx context.Context, articleID, title, body, sourceURL string) error {
// 	existing, err := s.articles.GetByURL(ctx, sourceURL)
// 	if err != nil {
// 		return err
// 	}
// 	if existing != nil {
// 		return nil
// 	}

// 	fullText := body
// 	if title != "" {
// 		fullText = title + "\n\n" + body
// 	}

// 	if err := s.articles.InsertArticle(ctx, model.Article{
// 		ID:   articleID,
// 		Text: fullText,
// 		URL:  sourceURL,
// 	}); err != nil {
// 		return err
// 	}

// 	paragraphs := splitParagraphs(body)
// 	chunks := make([]model.Chunk, 0, len(paragraphs))

// 	for i, paragraph := range paragraphs {
// 		paragraph = removeArabicText(paragraph)
// 		if !isEmbeddableChunk(paragraph) {
// 			continue
// 		}

// 		refs := regexutil.ExtractQuranReferences(paragraph)
// 		refStrings := make([]string, len(refs))
// 		for j, ref := range refs {
// 			refStrings[j] = ref.Raw
// 		}

// 		subChunks := splitByMaxChars(paragraph, s.maxChunkChars)
// 		for j, subText := range subChunks {
// 			embeddings, err := s.embedder.Embed(ctx, []string{subText})
// 			if err != nil {
// 				return err
// 			}

// 			chunkID := articleID + "-" + strconv.Itoa(i)
// 			if len(subChunks) > 1 {
// 				chunkID += "-" + strconv.Itoa(j)
// 			}

// 			chunks = append(chunks, model.Chunk{
// 				ID:          chunkID,
// 				DenseVector: embeddings[0],
// 				Payload: model.Payload{
// 					Text: subText,
// 					Metadata: model.Metadata{
// 						ArticleID:    articleID,
// 						Title:        title,
// 						SourceURL:    sourceURL,
// 						QuranRefs:    refStrings,
// 						ParagraphIdx: i,
// 					},
// 				},
// 			})
// 		}
// 	}

// 	return s.vectors.InsertChunks(ctx, chunks)
// }

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

		for _, subText := range splitByMaxChars(chunkText, s.maxChunkChars) {
			embeddings, err := s.embedder.Embed(ctx, []string{subText})
			if err != nil {
				return nil, err
			}

			chunks = append(chunks, model.Chunk{
				ID:          uuid.New().String(),
				DenseVector: embeddings[0],
				Payload: model.Payload{
					Text: subText,
					Metadata: model.Metadata{
						ArticleID: articleID,
						SourceURL: sourceURL,
						QuranRefs: refStrings,
					},
				},
			})
		}

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

func splitByMaxChars(text string, maxChars int) []string {
	text = strings.TrimSpace(text)
	if maxChars <= 0 || len(text) <= maxChars {
		return []string{text}
	}

	var parts []string
	for len(text) > maxChars {
		cut := maxChars
		if idx := strings.LastIndex(text[:cut], "\n"); idx > maxChars/2 {
			cut = idx
		} else if idx := strings.LastIndex(text[:cut], " "); idx > maxChars/2 {
			cut = idx
		}
		parts = append(parts, strings.TrimSpace(text[:cut]))
		text = strings.TrimSpace(text[cut:])
	}
	if text != "" {
		parts = append(parts, text)
	}
	return parts
}
