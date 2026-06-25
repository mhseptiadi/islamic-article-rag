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

func (c *LLMClient) generateGroqStream(ctx context.Context, question string, contextBlocks []string, onChunk StreamChunkFn) (string, error) {
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

	if c.stream {
		return c.readGroqSSE(resp.Body, onChunk)
	}
	return c.readGroqJSON(resp.Body, onChunk)
}

func (c *LLMClient) readGroqSSE(body io.Reader, onChunk StreamChunkFn) (string, error) {
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

func (c *LLMClient) readGroqJSON(body io.Reader, onChunk StreamChunkFn) (string, error) {
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

func (c *LLMClient) groqRequestURL() string {
	if strings.Contains(c.apiURL, "groq.com") || strings.Contains(c.apiURL, "chat/completions") {
		return c.apiURL
	}
	return "https://api.groq.com/openai/v1/chat/completions"
}
