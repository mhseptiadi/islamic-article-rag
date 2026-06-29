package llmproviders

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

func GenerateGroqStream(ctx context.Context, cfg Config, messages []map[string]string, onChunk StreamChunkFn) (string, error) {
	if cfg.APIKey == "" {
		return "", fmt.Errorf("groq LLM requires LLM_API_KEY")
	}

	payload := map[string]any{
		"messages":              messages,
		"model":                 cfg.Model,
		"temperature":           cfg.Temperature,
		"max_completion_tokens": cfg.MaxCompletionTokens,
		"top_p":                 cfg.TopP,
		"stream":                cfg.Stream,
		"reasoning_effort":      cfg.ReasoningEffort,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal groq request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, groqRequestURL(cfg), bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("create groq request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call groq API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("groq API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if cfg.Stream {
		return readGroqSSE(resp.Body, onChunk)
	}
	return readGroqJSON(resp.Body, onChunk)
}

func readGroqSSE(body io.Reader, onChunk StreamChunkFn) (string, error) {
	var answer strings.Builder
	scanner := bufio.NewScanner(body)
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
		if len(chunk.Choices) == 0 {
			continue
		}
		text := chunk.Choices[0].Delta.Content
		if text == "" {
			continue
		}
		answer.WriteString(text)
		if onChunk != nil {
			if err := onChunk(text); err != nil {
				return "", err
			}
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

func readGroqJSON(body io.Reader, onChunk StreamChunkFn) (string, error) {
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode groq response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("groq API returned empty response")
	}

	answer := strings.TrimSpace(result.Choices[0].Message.Content)
	if answer == "" {
		return "", fmt.Errorf("groq API returned empty response")
	}
	if onChunk != nil {
		if err := onChunk(answer); err != nil {
			return "", err
		}
	}

	return answer, nil
}

func groqRequestURL(cfg Config) string {
	if strings.Contains(cfg.APIURL, "groq.com") || strings.Contains(cfg.APIURL, "chat/completions") {
		return cfg.APIURL
	}
	return "https://api.groq.com/openai/v1/chat/completions"
}
