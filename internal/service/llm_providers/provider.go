package llm_providers

import (
	"context"
	"fmt"
	"strings"

	"github.com/mhseptiadi/islamic-article-rag/internal/model"
)

// Provider is the common interface for RAG answer generation providers.
type Provider interface {
	GenerateStream(ctx context.Context, cfg Config, messages []map[string]interface{}, onChunk StreamChunkFn) (string, error)
}

// AgentResponse holds a non-streaming agent decision from providers that support tool calls.
type AgentResponse struct {
	Content   string
	ToolCalls []model.ToolCall
}

// AgentCapable is implemented by providers that support tool-calling agent workflows.
type AgentCapable interface {
	Provider
	ExecuteAgentRequest(ctx context.Context, cfg Config, messages []map[string]interface{}) (*AgentResponse, error)
}

var providers = map[string]Provider{
	"google": googleProvider{},
	"groq":   groqProvider{},
	"ollama": ollamaProvider{},
}

// GetProvider returns the provider for the given name. Unknown names fall back to ollama.
func GetProvider(name string) Provider {
	if p, ok := providers[name]; ok {
		return p
	}
	return ollamaProvider{}
}

func messagesToPrompt(messages []map[string]interface{}) string {
	var b strings.Builder
	for i, msg := range messages {
		role, _ := msg["role"].(string)
		content, _ := msg["content"].(string)
		switch role {
		case "system":
			b.WriteString(content)
			b.WriteString("\n\nExamples:\n\n")
		case "user":
			if i != len(messages)-1 {
				b.WriteString("Question: ")
			}
			b.WriteString(content)
			if i != len(messages)-1 {
				b.WriteString("\n\n")
			}
		case "assistant":
			b.WriteString("Answer: ")
			b.WriteString(content)
			b.WriteString("\n\n")
		}
	}
	b.WriteString("Answer:")
	return b.String()
}

func requireAPIKey(cfg Config, providerName string) error {
	if cfg.APIKey == "" {
		return fmt.Errorf("%s LLM requires LLM_API_KEY", providerName)
	}
	return nil
}
