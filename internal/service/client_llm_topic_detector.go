package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

const (
	topicDetectorSystemPrompt = `you are a topic detector. detect if the user question is related to islamic topic or not. answer with "1" if yes and "0" if not`
	OffTopicAnswer            = "I'm sorry, I can only answer questions about Islamic topics."
)

type TopicDetectorClient struct {
	apiKey     string
	apiURL     string
	model      string
	httpClient *http.Client
}

func NewTopicDetectorClient(apiKey, apiURL, model string) *TopicDetectorClient {
	return &TopicDetectorClient{
		apiKey:     apiKey,
		apiURL:     apiURL,
		model:      model,
		httpClient: http.DefaultClient,
	}
}

func (c *TopicDetectorClient) Enabled() bool {
	return c.apiKey != "" && c.apiURL != "" && c.model != ""
}

func (c *TopicDetectorClient) IsIslamicTopic(ctx context.Context, question string) (bool, error) {
	if !c.Enabled() {
		return true, nil
	}

	answer, err := c.detectGroq(ctx, question)
	if err != nil {
		return false, err
	}

	return parseTopicDetectorResponse(answer)
}

func parseTopicDetectorResponse(raw string) (bool, error) {
	switch strings.TrimSpace(raw) {
	case "1":
		return true, nil
	case "0":
		return false, nil
	default:
		return false, fmt.Errorf("unexpected topic detector response: %q", raw)
	}
}
