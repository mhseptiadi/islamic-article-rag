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

// StreamChunkFn is called for each LLM text chunk when streaming is available.
// Return a non-nil error to abort generation.
type StreamChunkFn func(chunk string) error

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
	return c.GenerateAnswerStream(ctx, question, contextBlocks, nil)
}

func (c *LLMClient) GenerateAnswerStream(ctx context.Context, question string, contextBlocks []string, onChunk StreamChunkFn) (string, error) {
	fmt.Println("c.provider: ", c.provider)
	switch c.provider {
	case "google":
		answer, err := c.generateGoogle(ctx, buildRAGPrompt(question, contextBlocks))
		if err != nil {
			return "", err
		}
		if onChunk != nil {
			if err := onChunk(answer); err != nil {
				return "", err
			}
		}
		return answer, nil
	case "groq":
		return c.generateGroqStream(ctx, question, contextBlocks, onChunk)
	default:
		answer, err := c.generateOllama(ctx, buildRAGPrompt(question, contextBlocks))
		if err != nil {
			return "", err
		}
		if onChunk != nil {
			if err := onChunk(answer); err != nil {
				return "", err
			}
		}
		return answer, nil
	}
}
