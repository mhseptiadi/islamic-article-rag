package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
)

const (
	windowSize = 3 // Number of paragraphs per chunk
	stepSize   = 2 // Slide forward by 2 paragraphs (1 paragraph overlap)
	collection = "indonesian_articles"
	dataDir    = "data/raw_articles"
)

func main() {
	// 1. Connect to Qdrant (Local or Cloud)
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: "localhost",
		Port: 6334, // gRPC port
	})
	if err != nil {
		log.Fatalf("Failed to connect to Qdrant: %v", err)
	}
	defer client.Close()

	// 2. Read all files in the directory
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		log.Fatalf("Failed to read directory %s: %v", dataDir, err)
	}

	var allPoints []*qdrant.PointStruct

	for _, entry := range entries {
		// Skip directories and non-markdown files
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		fullPath := filepath.Join(dataDir, entry.Name())
		contentBytes, err := os.ReadFile(fullPath)
		if err != nil {
			log.Printf("Failed to read file %s: %v", fullPath, err)
			continue
		}

		content := string(contentBytes)

		// Dynamically extract the URL from the file content
		sourceURL := extractSourceURL(content)
		if sourceURL == "" {
			// Fallback if no URL is found in the text
			sourceURL = fmt.Sprintf("file://%s", entry.Name())
		}

		// 3. Execute Sliding Window Chunking for this file
		paragraphs := strings.Split(content, "\n")

		for i := 0; i < len(paragraphs); {
			end := i + windowSize
			if end > len(paragraphs) {
				end = len(paragraphs)
			}

			fmt.Println("i: ", i)

			// Combine paragraphs for this chunk
			chunkText := strings.Join(paragraphs[i:end], "\n")
			// fmt.Println("chunkText: ", chunkText)

			// Clean up the chunk
			chunkText = removeArabicText(chunkText)
			// fmt.Println("chunkText no arabic: ", chunkText)

			// Extract metadata
			// koranRefs := extractQuranRefs(chunkText)
			// fmt.Println("koranRefs: ", koranRefs)

			// Call Embedding API (Cohere/OpenAI)
			vector := generateEmbedding(chunkText)
			// fmt.Println("vector: ", vector)

			// Build the Qdrant Payload (Metadata)
			payload := qdrant.NewValueMap(map[string]any{
				"chunk_text": chunkText,
				"source_url": sourceURL,
				// "koran_refs": koranRefs,
			})

			fmt.Println("payload: ", payload)

			fmt.Println("-------------------")

			// Create the Qdrant Point and add to batch
			allPoints = append(allPoints, &qdrant.PointStruct{
				Id:      qdrant.NewIDUUID(generateDeterministicUUID(chunkText)),
				Vectors: qdrant.NewVectors(vector...),
				Payload: payload,
			})

			// Break if we've reached the end of the paragraphs array
			if end == len(paragraphs) {
				break
			}
			i += stepSize
		}

		fmt.Printf("Processed file: %s\n", entry.Name())
	}

	// 4. Batch Upsert all chunks into Vector Database
	if len(allPoints) > 0 {
		_, err = client.Upsert(context.Background(), &qdrant.UpsertPoints{
			CollectionName: collection,
			Points:         allPoints,
		})
		if err != nil {
			log.Fatalf("Failed to upsert points to Qdrant: %v", err)
		}
		fmt.Printf("\nSuccess: Ingested %d total chunks across %d files.\n", len(allPoints), len(entries))
	} else {
		fmt.Println("No chunks were generated. Make sure your data directory contains .md files.")
	}
}

// ----------------------------------------------------------------------------
// Helper Functions
// ----------------------------------------------------------------------------

// extractSourceURL finds the first http/https link in the text.
func extractSourceURL(text string) string {
	re := regexp.MustCompile(`https?://[^\s\)]+`)
	match := re.FindString(text)
	match = strings.TrimRight(match, "\")]*")
	return match
}

// extractQuranRefs scans the text for patterns like "(QS. Al-Anbiya': 107)"
func extractQuranRefs(text string) []string {
	var refs []string
	re := regexp.MustCompile(`(?i)\(QS\.?\s+([^:]+):\s*(\d+)\)`)
	matches := re.FindAllStringSubmatch(text, -1)

	for _, match := range matches {
		if len(match) == 3 {
			surahName := strings.TrimSpace(match[1])
			ayahNumber := strings.TrimSpace(match[2])

			normalizedID := fmt.Sprintf("surah_%s_ayah_%s",
				strings.ToLower(strings.ReplaceAll(surahName, " ", "_")),
				ayahNumber,
			)
			refs = append(refs, normalizedID)
		}
	}
	return refs
}

// removeArabicText strips Unicode Arabic blocks to prevent LLM hallucination
func removeArabicText(text string) string {
	re := regexp.MustCompile(`[\x{0600}-\x{06FF}]+`)
	return re.ReplaceAllString(text, "")
}

// generateEmbedding simulates calling an Embedding API
func generateEmbedding(text string) []float32 {
	// TODO: Replace with actual HTTP client call to Cohere/OpenAI
	// return []float32{0.12, 0.45, -0.32, 0.88}
	// Ollama runs locally on port 11434

	url := "http://localhost:11434/api/embeddings"

	// Create the JSON payload
	payload := map[string]string{
		"model":  "bge-m3",
		"prompt": text,
	}
	jsonData, _ := json.Marshal(payload)

	// Make the HTTP request
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Failed to call Ollama: %v", err)
	}
	defer resp.Body.Close()

	// Parse the response
	var result struct {
		Embedding []float32 `json:"embedding"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	fmt.Println("result: ", result)
	fmt.Println("result.Embedding: ", result.Embedding)
	fmt.Println("result.Embedding length: ", len(result.Embedding))

	return result.Embedding
}

// generateDeterministicUUID generates a stable ID so re-running the script updates chunks instead of duplicating
func generateDeterministicUUID(text string) string {
	// TODO: Use google/uuid package with NewHash() in production
	// return "550e8400-e29b-41d4-a716-446655440000"
	// use uuid v7
	uuid := uuid.New()
	return uuid.String()
}
