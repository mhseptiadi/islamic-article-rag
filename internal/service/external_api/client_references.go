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

type ReferencesResponse struct {
	Results []ReferenceResult `json:"results"`
}

type ReferenceResult struct {
	SourceType      string        `json:"source_type"`
	ReferenceName   string        `json:"reference_name"`
	ReferenceNumber string        `json:"reference_number"`
	Found           bool          `json:"found"`
	Error           string        `json:"error,omitempty"`
	Collection      string        `json:"collection,omitempty"`
	HadithNumber    int           `json:"hadith_number,omitempty"`
	Hadith          []HadithEntry `json:"hadith,omitempty"`
	Chapter         int           `json:"chapter,omitempty"`
	Verse           int           `json:"verse,omitempty"`
	Quran           []QuranEntry  `json:"quran,omitempty"`
}

type HadithEntry struct {
	ID           int                 `json:"id"`
	Collection   string              `json:"collection"`
	HadithNumber int                 `json:"hadith_number"`
	Reference    HadithBookReference `json:"reference"`
	Translations []HadithTranslation `json:"translations"`
}

type HadithBookReference struct {
	Book   int `json:"book"`
	Hadith int `json:"hadith"`
}

type HadithTranslation struct {
	Text     string `json:"text"`
	Language string `json:"language"`
	Grades   any    `json:"grades"`
}

type QuranEntry struct {
	ID       int    `json:"id"`
	Chapter  int    `json:"chapter"`
	Verse    int    `json:"verse"`
	Text     string `json:"text"`
	Language string `json:"language"`
	Source   string `json:"source"`
}

func (r ReferencesResponse) ToolContent() string {
	var contentBuilder strings.Builder
	contentBuilder.WriteString("RESULTS FOR YOUR REQUESTED BATCH:\n\n")

	for _, res := range r.Results {
		contentBuilder.WriteString(fmt.Sprintf("[MATCH FOR %s %s %s]:\n", res.SourceType, res.ReferenceName, res.ReferenceNumber))
		if !res.Found {
			if res.Error != "" {
				contentBuilder.WriteString(fmt.Sprintf("Error: %s\n", res.Error))
			}
			contentBuilder.WriteString("\n")
			continue
		}

		switch res.SourceType {
		case "hadith":
			for _, hadith := range res.Hadith {
				for _, translation := range hadith.Translations {
					contentBuilder.WriteString(fmt.Sprintf("[%s] %s\n", translation.Language, translation.Text))
				}
			}
		case "quran":
			for _, verse := range res.Quran {
				contentBuilder.WriteString(fmt.Sprintf("[%s/%s] chapter=%d verse=%d: %s\n", verse.Language, verse.Source, verse.Chapter, verse.Verse, verse.Text))
			}
		}
		contentBuilder.WriteString("\n")
	}
	return contentBuilder.String()
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

	fmt.Println("--------------------------------")
	fmt.Println("jsonData: ", string(jsonData))
	fmt.Println("c.apiURL: ", c.apiURL)

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

	fmt.Println("resp: ", resp)
	fmt.Println("-----------------")

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
