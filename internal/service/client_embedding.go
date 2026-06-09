package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type EmbeddingClient struct {
	apiKey      string
	ollamaURL   string
	ollamaModel string
	httpClient  *http.Client
}

func NewEmbeddingClient(apiKey, ollamaURL, ollamaModel string) *EmbeddingClient {
	return &EmbeddingClient{
		apiKey:      apiKey,
		ollamaURL:   ollamaURL,
		ollamaModel: ollamaModel,
		httpClient:  http.DefaultClient,
	}
}

func (c *EmbeddingClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, 0, len(texts))
	for _, text := range texts {
		vector, err := c.embedOne(ctx, text)
		if err != nil {
			return nil, err
		}
		embeddings = append(embeddings, vector)
	}
	return embeddings, nil
}

func (c *EmbeddingClient) embedOne(ctx context.Context, text string) ([]float32, error) {
	payload := map[string]string{
		"model":  c.ollamaModel,
		"prompt": text,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.ollamaURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call embedding API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("embedding API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode embedding response: %w", err)
	}
	if len(result.Embedding) == 0 {
		return nil, fmt.Errorf("embedding API returned empty vector")
	}

	return result.Embedding, nil
}
