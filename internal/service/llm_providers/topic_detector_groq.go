package llm_providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func DetectTopicGroq(ctx context.Context, cfg Config, systemPrompt, question string) (string, error) {
	if cfg.APIKey == "" {
		return "", fmt.Errorf("groq topic detector requires LLM_TOPIC_DETECTOR_API_KEY")
	}

	payload := map[string]any{
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": question},
		},
		"model":                 cfg.Model,
		"temperature":           0,
		"max_completion_tokens": 128,
		"top_p":                 1,
		"stream":                false,
		"reasoning_effort":      "low",
		"stop":                  nil,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal groq topic detector request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, groqRequestURL(cfg), bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("create groq topic detector request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call groq topic detector API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", fmt.Errorf("read groq topic detector response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("groq topic detector API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("decode groq topic detector response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("groq topic detector API returned empty response: %s", strings.TrimSpace(string(body)))
	}

	answer := strings.TrimSpace(result.Choices[0].Message.Content)
	if answer == "" {
		return "", fmt.Errorf("groq topic detector API returned empty content: %s", strings.TrimSpace(string(body)))
	}

	return answer, nil
}
