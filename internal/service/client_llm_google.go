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

func (c *LLMClient) generateGoogle(ctx context.Context, prompt string) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("google LLM requires LLM_API_KEY")
	}

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

	url := c.googleRequestURL()
	fmt.Println("url: ", url)
	fmt.Println("jsonData: ", string(jsonData))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("create google request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
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

	return answer, nil
}

func (c *LLMClient) googleRequestURL() string {
	if strings.Contains(c.apiURL, ":generateContent") {
		sep := "?"
		if strings.Contains(c.apiURL, "?") {
			sep = "&"
		}
		return c.apiURL + sep + "key=" + c.apiKey
	}

	return fmt.Sprintf(
		"%s/models/%s:generateContent?key=%s",
		strings.TrimSuffix(c.apiURL, "/"),
		c.model,
		c.apiKey,
	)
}
