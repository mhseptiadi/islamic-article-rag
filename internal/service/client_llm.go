package service

import "context"

type LLMClient struct {
	apiKey string
}

func NewLLMClient(apiKey string) *LLMClient {
	return &LLMClient{apiKey: apiKey}
}

func (c *LLMClient) GenerateAnswer(ctx context.Context, question string, contextBlocks []string) (string, error) {
	_ = ctx
	_ = c.apiKey
	_ = question
	_ = contextBlocks
	return "", nil
}
