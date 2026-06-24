package service

import (
	"context"
	"fmt"
	"net/http"
)

type Message struct {
	Role    string
	Content string
}

type LLMClient struct {
	provider   string
	apiKey     string
	apiURL     string
	model      string
	httpClient *http.Client

	temperature         float64
	maxCompletionTokens int
	topP                float64
	stream              bool
	reasoningEffort     string
}

func NewLLMClient(provider, apiKey, apiURL, model string, temperature float64, maxCompletionTokens int, topP float64, stream bool, reasoningEffort string) *LLMClient {
	return &LLMClient{
		provider:            provider,
		apiKey:              apiKey,
		apiURL:              apiURL,
		model:               model,
		httpClient:          http.DefaultClient,
		temperature:         temperature,
		maxCompletionTokens: maxCompletionTokens,
		topP:                topP,
		stream:              stream,
		reasoningEffort:     reasoningEffort,
	}
}

func (c *LLMClient) GenerateAnswer(ctx context.Context, question string, contextBlocks []string) (string, error) {
	fmt.Println("c.provider: ", c.provider)
	switch c.provider {
	case "google":
		return c.generateGoogle(ctx, buildRAGPrompt(question, contextBlocks))
	case "groq":
		return c.generateGroq(ctx, question, contextBlocks)
	default:
		return c.generateOllama(ctx, buildRAGPrompt(question, contextBlocks))
	}
}
