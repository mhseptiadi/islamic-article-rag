package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (c *LLMClient) generateGroq(ctx context.Context, question string, contextBlocks []string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("groq LLM requires LLM_API_KEY")
	}

	payload := map[string]any{
		"messages":              buildRAGMessages(question, contextBlocks),
		"model":                 c.model,
		"temperature":           c.temperature,
		"max_completion_tokens": c.maxCompletionTokens,
		"top_p":                 c.topP,
		"stream":                c.stream,
		"reasoning_effort":      c.reasoningEffort,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal groq request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.groqRequestURL(), bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("create groq request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call groq API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("groq API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var answer strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return "", fmt.Errorf("decode groq stream chunk: %w", err)
		}
		if len(chunk.Choices) > 0 {
			answer.WriteString(chunk.Choices[0].Delta.Content)
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read groq stream: %w", err)
	}

	result := strings.TrimSpace(answer.String())
	if result == "" {
		return "", fmt.Errorf("groq API returned empty response")
	}

	return result, nil
}

func (c *LLMClient) groqRequestURL() string {
	if strings.Contains(c.apiURL, "groq.com") || strings.Contains(c.apiURL, "chat/completions") {
		return c.apiURL
	}
	return "https://api.groq.com/openai/v1/chat/completions"
}
