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

func GenerateOllama(ctx context.Context, cfg Config, prompt string, onChunk StreamChunkFn) (string, error) {
	payload := map[string]any{
		"model":  cfg.Model,
		"prompt": prompt,
		"stream": false,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal ollama request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.APIURL, bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("create ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call ollama API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("ollama API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode ollama response: %w", err)
	}
	answer := strings.TrimSpace(result.Response)
	if answer == "" {
		return "", fmt.Errorf("ollama API returned empty response")
	}
	if onChunk != nil {
		if err := onChunk(answer); err != nil {
			return "", err
		}
	}

	return answer, nil
}
