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
	provider       string
	apiKey         string
	embeddingURL   string
	embeddingModel string
	httpClient     *http.Client
}

func NewEmbeddingClient(provider, apiKey, embeddingURL, embeddingModel string) *EmbeddingClient {
	return &EmbeddingClient{
		provider:       provider,
		apiKey:         apiKey,
		embeddingURL:   embeddingURL,
		embeddingModel: embeddingModel,
		httpClient:     http.DefaultClient,
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
	switch c.provider {
	case "deepinfra":
		return c.embedDeepInfra(ctx, text)
	default:
		return c.embedOllama(ctx, text)
	}
}

func (c *EmbeddingClient) embedOllama(ctx context.Context, text string) ([]float32, error) {
	payload := map[string]string{
		"model":  c.embeddingModel,
		"prompt": text,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal ollama embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.embeddingURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create ollama embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call ollama embedding API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("ollama embedding API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode ollama embedding response: %w", err)
	}
	if len(result.Embedding) == 0 {
		return nil, fmt.Errorf("ollama embedding API returned empty vector")
	}

	return result.Embedding, nil
}

func (c *EmbeddingClient) embedDeepInfra(ctx context.Context, text string) ([]float32, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("deepinfra embedding requires EMBEDDING_API_KEY")
	}

	payload := map[string]string{
		"input":            text,
		"model":            c.embeddingModel,
		"encoding_format":  "float",
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal deepinfra embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.embeddingURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create deepinfra embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call deepinfra embedding API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("deepinfra embedding API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode deepinfra embedding response: %w", err)
	}
	if len(result.Data) == 0 || len(result.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("deepinfra embedding API returned empty vector")
	}

	return result.Data[0].Embedding, nil
}
