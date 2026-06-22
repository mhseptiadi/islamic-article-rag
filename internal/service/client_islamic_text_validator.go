package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

type IslamicTextReplacement struct {
	Tag      string  `json:"tag"`
	Original string  `json:"original"`
	Matched  string  `json:"matched"`
	Score    float64 `json:"score"`
	Chapter  int     `json:"chapter"`
	Verse    int     `json:"verse"`
}

type IslamicTextValidationResponse struct {
	Text         string                   `json:"text"`
	ReplacedText string                   `json:"replaced_text"`
	Replacements []IslamicTextReplacement `json:"replacements"`
}

type IslamicTextValidatorClient struct {
	apiURL     string
	httpClient *http.Client
}

func NewIslamicTextValidatorClient(apiURL string) *IslamicTextValidatorClient {
	return &IslamicTextValidatorClient{
		apiURL:     apiURL,
		httpClient: http.DefaultClient,
	}
}

func (c *IslamicTextValidatorClient) Validate(ctx context.Context, text string) (*IslamicTextValidationResponse, error) {
	payload := map[string]string{"text": text}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal islamic text validator request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, normalizeHTTPURL(c.apiURL), bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create islamic text validator request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call islamic text validator API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("islamic text validator API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result IslamicTextValidationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode islamic text validator response: %w", err)
	}

	return &result, nil
}

var (
	quranTagPattern  = regexp.MustCompile(`(?is)<quran[^>]*>.*?</quran>`)
	hadithTagPattern = regexp.MustCompile(`(?is)<hadith[^>]*>.*?</hadith>`)
)

func ReplaceIslamicTagsOnValidationError(text string) string {
	text = quranTagPattern.ReplaceAllString(text, "{{quran validation error}}")
	text = hadithTagPattern.ReplaceAllString(text, "{{hadith validation error}}")
	return text
}

func normalizeHTTPURL(url string) string {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return "http://" + url
	}
	return url
}
