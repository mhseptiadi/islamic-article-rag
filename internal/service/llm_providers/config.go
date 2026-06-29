package llmproviders

import "net/http"

// StreamChunkFn is called for each LLM text chunk when streaming is available.
// Return a non-nil error to abort generation.
type StreamChunkFn func(chunk string) error

type Config struct {
	APIKey              string
	APIURL              string
	Model               string
	HTTPClient          *http.Client
	Temperature         float64
	MaxCompletionTokens int
	TopP                float64
	Stream              bool
	ReasoningEffort     string
}
