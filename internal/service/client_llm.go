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

type Message struct {
	Role    string
	Content string
}

type LLMClient struct {
	provider   string
	apiKey     string
	apiURL     string
	model      string
	httpClient *http.Client
}

func NewLLMClient(provider, apiKey, apiURL, model string) *LLMClient {
	return &LLMClient{
		provider:   provider,
		apiKey:     apiKey,
		apiURL:     apiURL,
		model:      model,
		httpClient: http.DefaultClient,
	}
}

func (c *LLMClient) GenerateAnswer(ctx context.Context, question string, contextBlocks []string) (string, error) {
	prompt := buildRAGPrompt(question, contextBlocks)
	fmt.Println("c.provider: ", c.provider)
	switch c.provider {
	case "google":
		return c.generateGoogle(ctx, prompt)
	default:
		return c.generateOllama(ctx, prompt)
	}
}

func (c *LLMClient) generateOllama(ctx context.Context, prompt string) (string, error) {
	payload := map[string]any{
		"model":  c.model,
		"prompt": prompt,
		"stream": false,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal ollama request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("create ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
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
	if strings.TrimSpace(result.Response) == "" {
		return "", fmt.Errorf("ollama API returned empty response")
	}

	return strings.TrimSpace(result.Response), nil
}

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

func buildRAGPrompt(question string, contextBlocks []string) string {
	var b strings.Builder

	// 	b.WriteString(`You are an intelligent and accurate Islamic AI assistant. You answer questions in both English and Indonesian.

	// Whenever you quote a verse from the Quran or a Hadith, you MUST wrap the exact quote inside specific XML tags with metadata attributes.

	// For the Quran:
	// Use the <quran> tag with chapter and verse attributes.
	// Example: Allah says, <quran chapter="2" verse="255">Allah! There is no deity except Him, the Ever-Living, the Sustainer of existence.</quran>

	// For Hadith:
	// Use the <hadith> tag with collection and number attributes. Valid collection keys are: bukhari, muslim, abudawud, tirmidhi, nasai, ibnmajah.
	// Example: The Prophet said: <hadith collection="bukhari" number="1">The reward of deeds depends upon the intentions.</hadith>

	// STRICT FORMATTING RULES:

	// No External Citations: NEVER write traditional citations outside the tags like "(QS. 2:255)" or "(HR. Bukhari no. 1)". The XML attributes are your only citation.

	// Pure Text Only: Only place the exact quote inside the tags. Do not put introductory words like "Narrated by..." inside the tags.

	// Universal Application: You must use this exact XML formatting regardless of whether you are replying in English or Indonesian.`)

	b.WriteString(`You are an Islamic AI assistant.
Rule 1: Answer in Indonesian language or English language.
Rule 2: Every time you quote the Quran, you must use this exact format: <quran chapter="number" verse="number">quote text</quran>.
Rule 3: Every time you quote a Hadith, you must use this exact format: <hadith collection="name" number="number">quote text</hadith>.
Rule 4: Do not write citations outside of these tags.`)

	messages := []Message{
		{Role: "user", Content: "What does the Quran say about fasting?"},
		{Role: "assistant", Content: "Fasting is prescribed for believers. Allah says: <quran chapter=\"2\" verse=\"183\">O you who have believed, decreed upon you is fasting as it was decreed upon those before you that you may become righteous.</quran>"},
		{Role: "user", Content: "Give me a hadith about intention."},
		{Role: "assistant", Content: "The Prophet emphasized intention deeply. He said: <hadith collection=\"bukhari\" number=\"1\">The reward of deeds depends upon the intentions.</hadith>"},
	}

	b.WriteString("\n\nExamples:\n\n")
	for _, msg := range messages {
		switch msg.Role {
		case "user":
			b.WriteString("Question: ")
		case "assistant":
			b.WriteString("Answer: ")
		}
		b.WriteString(msg.Content)
		b.WriteString("\n\n")
	}

	if len(contextBlocks) == 0 {
		b.WriteString("Articles:\n(no relevant articles found)\n\n")
	} else {
		b.WriteString("Articles:\n")
		for i, block := range contextBlocks {
			b.WriteString(fmt.Sprintf("[%d] %s\n\n", i+1, block))
		}
	}

	b.WriteString("Question: ")
	b.WriteString(question)
	b.WriteString("Answer:")

	return b.String()
}
