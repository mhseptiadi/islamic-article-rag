package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (c *TopicDetectorClient) detectGroq(ctx context.Context, question string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("groq topic detector requires LLM_TOPIC_DETECTOR_API_KEY")
	}

	payload := map[string]any{
		"messages": []map[string]string{
			{"role": "system", "content": topicDetectorSystemPrompt},
			{"role": "user", "content": question},
		},
		"model":                 c.model,
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.groqRequestURL(), bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("create groq topic detector request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
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

func (c *TopicDetectorClient) groqRequestURL() string {
	if strings.Contains(c.apiURL, "groq.com") || strings.Contains(c.apiURL, "chat/completions") {
		return c.apiURL
	}
	return "https://api.groq.com/openai/v1/chat/completions"
}
