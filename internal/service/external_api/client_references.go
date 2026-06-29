package external_api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mhseptiadi/islamic-article-rag/internal/model"
)

type VerifiedReference struct {
	SourceType      string `json:"source_type"`
	ReferenceName   string `json:"reference_name"`
	ReferenceNumber string `json:"reference_number"`
	Arabic          string `json:"arabic,omitempty"`
	Translation     string `json:"translation,omitempty"`
}

type ReferencesResponse struct {
	Text       string              `json:"text,omitempty"`
	References []VerifiedReference `json:"references"`
}

func (r ReferencesResponse) ToolContent() string {
	if strings.TrimSpace(r.Text) != "" {
		return r.Text
	}
	data, err := json.Marshal(r)
	if err != nil {
		return fmt.Sprintf(`{"error":"marshal references response: %s"}`, err)
	}
	return string(data)
}

type ReferencesClient struct {
	apiURL     string
	httpClient *http.Client
}

func NewReferencesClient(apiURL string) *ReferencesClient {
	return &ReferencesClient{
		apiURL:     apiURL,
		httpClient: http.DefaultClient,
	}
}

func (c *ReferencesClient) Lookup(ctx context.Context, references []model.ScriptureReference) (*ReferencesResponse, error) {
	payload := model.ToolArguments{References: references}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal references request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, normalizeHTTPURL(c.apiURL), bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create references request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call references API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("references API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result ReferencesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode references response: %w", err)
	}

	return &result, nil
}
