package service

import "context"

type EmbeddingClient struct {
	apiKey string
}

func NewEmbeddingClient(apiKey string) *EmbeddingClient {
	return &EmbeddingClient{apiKey: apiKey}
}

func (c *EmbeddingClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	_ = ctx
	_ = c.apiKey
	_ = texts
	return nil, nil
}
