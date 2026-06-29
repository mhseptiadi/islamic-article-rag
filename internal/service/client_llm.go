package service

import (
	"context"
	"fmt"
	"net/http"

	llmproviders "github.com/mhseptiadi/islamic-article-rag/internal/service/llm_providers"
)

type Message struct {
	Role    string
	Content string
}

// StreamChunkFn is called for each LLM text chunk when streaming is available.
// Return a non-nil error to abort generation.
type StreamChunkFn = llmproviders.StreamChunkFn

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

func (c *LLMClient) providerConfig() llmproviders.Config {
	return llmproviders.Config{
		APIKey:              c.apiKey,
		APIURL:              c.apiURL,
		Model:               c.model,
		HTTPClient:          c.httpClient,
		Temperature:         c.temperature,
		MaxCompletionTokens: c.maxCompletionTokens,
		TopP:                c.topP,
		Stream:              c.stream,
		ReasoningEffort:     c.reasoningEffort,
	}
}

func (c *LLMClient) GenerateAnswerStream(ctx context.Context, question string, contextBlocks []string, onChunk StreamChunkFn) (string, error) {
	fmt.Println("c.provider: ", c.provider)
	cfg := c.providerConfig()
	switch c.provider {
	case "google":
		answer, err := llmproviders.GenerateGoogle(ctx, cfg, buildRAGPrompt(question, contextBlocks))
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
		return llmproviders.GenerateGroqStream(ctx, cfg, buildRAGMessages(question, contextBlocks), onChunk)
	default:
		answer, err := llmproviders.GenerateOllama(ctx, cfg, buildRAGPrompt(question, contextBlocks))
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
