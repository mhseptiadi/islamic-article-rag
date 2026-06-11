package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type LLMClient struct {
	apiKey      string
	ollamaURL   string
	ollamaModel string
	httpClient  *http.Client
}

func NewLLMClient(apiKey, ollamaURL, ollamaModel string) *LLMClient {
	return &LLMClient{
		apiKey:      apiKey,
		ollamaURL:   ollamaURL,
		ollamaModel: ollamaModel,
		httpClient:  http.DefaultClient,
	}
}

func (c *LLMClient) GenerateAnswer(ctx context.Context, question string, contextBlocks []string) (string, error) {
	payload := map[string]any{
		"model":  c.ollamaModel,
		"prompt": buildRAGPrompt(question, contextBlocks),
		"stream": false,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal LLM request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.ollamaURL, bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("create LLM request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call LLM API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM API returned status %d", resp.StatusCode)
	}

	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode LLM response: %w", err)
	}
	if strings.TrimSpace(result.Response) == "" {
		return "", fmt.Errorf("LLM API returned empty response")
	}

	return strings.TrimSpace(result.Response), nil
}

func buildRAGPrompt(question string, contextBlocks []string) string {
	var b strings.Builder

	b.WriteString("You are a helpful assistant answering questions about Islamic articles.\n")
	b.WriteString("Use only the full articles below. If the articles are insufficient, say you cannot answer from the available sources.\n\n")
	b.WriteString("Answer in Indonesian language or English language.\n\n")

	if len(contextBlocks) == 0 {
		b.WriteString("Articles:\n(no relevant articles found)\n\n")
	} else {
		b.WriteString("Articles:\n")
		for i, block := range contextBlocks {
			b.WriteString(fmt.Sprintf("[%d] %s\n\n", i+1, block))
		}
	}

	b.WriteString("Question: ")
	b.WriteString(question)
	b.WriteString("\n\nAnswer:")

	return b.String()
}
