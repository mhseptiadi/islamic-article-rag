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

type googleProvider struct{}

func (googleProvider) GenerateStream(ctx context.Context, cfg Config, messages []map[string]interface{}, onChunk StreamChunkFn) (string, error) {
	if err := requireAPIKey(cfg, "google"); err != nil {
		return "", err
	}

	prompt := messagesToPrompt(messages)

	payload := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal google request: %w", err)
	}

	url := googleRequestURL(cfg)
	fmt.Println("url: ", url)
	fmt.Println("jsonData: ", string(jsonData))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("create google request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call google API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("google API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode google response: %w", err)
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("google API returned empty response")
	}

	answer := strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text)
	if answer == "" {
		return "", fmt.Errorf("google API returned empty response")
	}
	if onChunk != nil {
		if err := onChunk(answer); err != nil {
			return "", err
		}
	}

	return answer, nil
}

func googleRequestURL(cfg Config) string {
	if strings.Contains(cfg.APIURL, ":generateContent") {
		sep := "?"
		if strings.Contains(cfg.APIURL, "?") {
			sep = "&"
		}
		return cfg.APIURL + sep + "key=" + cfg.APIKey
	}

	return fmt.Sprintf(
		"%s/models/%s:generateContent?key=%s",
		strings.TrimSuffix(cfg.APIURL, "/"),
		cfg.Model,
		cfg.APIKey,
	)
}
