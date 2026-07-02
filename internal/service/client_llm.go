package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mhseptiadi/islamic-article-rag/internal/model"
	"github.com/mhseptiadi/islamic-article-rag/internal/service/external_api"
	llm_providers "github.com/mhseptiadi/islamic-article-rag/internal/service/llm_providers"
)

type Message struct {
	Role    string
	Content string
}

// StreamChunkFn is called for each LLM text chunk when streaming is available.
// Return a non-nil error to abort generation.
type StreamChunkFn = llm_providers.StreamChunkFn

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

	referencesClient *external_api.ReferencesClient
}

func NewLLMClient(provider, apiKey, apiURL, model string, temperature float64, maxCompletionTokens int, topP float64, stream bool, reasoningEffort string, referencesClient *external_api.ReferencesClient) *LLMClient {
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
		referencesClient:    referencesClient,
	}
}

func (c *LLMClient) GenerateAnswer(ctx context.Context, question string, contextBlocks []string) (string, error) {
	return c.GenerateAnswerStream(ctx, question, contextBlocks, nil)
}

func (c *LLMClient) providerConfig() llm_providers.Config {
	return llm_providers.Config{
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
		return llm_providers.GenerateGoogle(ctx, cfg, buildRAGPrompt(question, contextBlocks), onChunk)
	case "groq":
		return llm_providers.GenerateGroqStream(ctx, cfg, buildRAGMessages(question, contextBlocks, false), onChunk)
	default:
		return llm_providers.GenerateOllama(ctx, cfg, buildRAGPrompt(question, contextBlocks), onChunk)
	}
}

// GenerateAgenticStream is your new entry point from the HTTP Handler
func (c *LLMClient) GenerateAgenticStream(ctx context.Context, question string, contextBlocks []string, onChunk StreamChunkFn) (string, error) {
	messages := buildRAGMessages(question, contextBlocks, true)
	cfg := c.providerConfig()

	// PHASE 1: The Hidden Agent Check (Disable streaming for clean JSON parsing)
	cfg.Stream = false

	// Execute the hidden Groq call (You will need to update readGroqJSON to return the full GroqResponse struct)
	rawResp, err := llm_providers.ExecuteGroqRequest(ctx, cfg, messages)
	if err != nil {
		return "", err
	}

	fmt.Println("rawResp: ", rawResp)

	// build new messages after tools
	messages = buildRAGMessages(question, contextBlocks, false)

	// Check if the LLM decided to call your batch validator
	if len(rawResp.Choices[0].Message.ToolCalls) > 0 {
		toolCall := rawResp.Choices[0].Message.ToolCalls[0]

		if toolCall.Function.Name == "validate_islamic_text" {
			fmt.Println("🤖 Agent requested scripture validation...")

			// 1. Parse the batch array requested by the LLM
			var args model.ToolArguments
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				return "", fmt.Errorf("parse validate_islamic_text arguments: %w", err)
			}

			// 2. HTTP POST to the references microservice
			if c.referencesClient == nil {
				return "", fmt.Errorf("references client is not configured")
			}
			refsResp, err := c.referencesClient.Lookup(ctx, args.References)
			if err != nil {
				return "", err
			}
			verifiedText := refsResp.ToolContent()

			fmt.Println("verifiedText: ", verifiedText)

			// 3. Append the LLM's tool call request to history
			messages = append(messages, map[string]interface{}{
				"role":       "assistant",
				"content":    "",
				"tool_calls": []model.ToolCall{toolCall},
			})

			// 4. Append the Validator's response to history
			messages = append(messages, map[string]interface{}{
				"role":         "tool",
				"tool_call_id": toolCall.ID,
				"name":         toolCall.Function.Name,
				"content":      verifiedText,
				// "content": refsResp,
			})
		}
	} else {
		// If no tools were called, append the standard text response
		messages = append(messages, map[string]interface{}{
			"role":    "assistant",
			"content": rawResp.Choices[0].Message.Content,
		})
	}

	fmt.Println("--------------------------------")
	// to json string
	jsonMessages, err := json.Marshal(messages)
	if err != nil {
		return "", fmt.Errorf("marshal messages: %w", err)
	}
	fmt.Println("messages: ", string(jsonMessages))
	fmt.Println("--------------------------------")

	// PHASE 2: The Visible Stream
	// Turn streaming back on and call Groq a second time to stream the final verified answer to the SPA
	cfg.Stream = true
	return llm_providers.GenerateGroqStream(ctx, cfg, messages, onChunk)
}
