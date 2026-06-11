package model

type Article struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	URL  string `json:"url"`
}

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

type SparseVector struct {
	Indices []uint32  `json:"indices"`
	Values  []float32 `json:"values"`
}

func (v SparseVector) HasValues() bool {
	return len(v.Indices) > 0 && len(v.Values) > 0
}

type Chunk struct {
	ID           string        `json:"id"`
	DenseVector  []float32     `json:"dense_vector"`
	SparseVector *SparseVector `json:"sparse_vector,omitempty"`
	VectorType   string        `json:"vector_type"`
	Score        float64       `json:"score,omitempty"`
	Payload      Payload       `json:"payload"`
}
