package model

type Metadata struct {
	ArticleID    string   `json:"article_id"`
	Title        string   `json:"title"`
	SourceURL    string   `json:"source_url,omitempty"`
	QuranRefs    []string `json:"quran_refs,omitempty"`
	ParagraphIdx int      `json:"paragraph_idx"`
}

type Payload struct {
	Text     string   `json:"text"`
	Metadata Metadata `json:"metadata"`
}

type Chunk struct {
	ID      string    `json:"id"`
	Vector  []float32 `json:"vector"`
	Score   float64   `json:"score,omitempty"`
	Payload Payload   `json:"payload"`
}
