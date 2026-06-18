package model

import "time"

type QnARecord struct {
	ID            string    `json:"id" bson:"_id"`
	Question      string    `json:"question" bson:"question"`
	Answer        string    `json:"answer" bson:"answer"`
	LLMProvider   string    `json:"llm_provider,omitempty" bson:"llm_provider,omitempty"`
	LLMModel      string    `json:"llm_model,omitempty" bson:"llm_model,omitempty"`
	ContextSource string    `json:"context_source,omitempty" bson:"context_source,omitempty"`
	ArticleIDs    []string  `json:"article_ids,omitempty" bson:"article_ids,omitempty"`
	Chunks        []QnAChunk `json:"chunks,omitempty" bson:"chunks,omitempty"`
	CreatedAt     time.Time `json:"created_at" bson:"created_at"`
}

type QnAChunk struct {
	ID    string  `json:"id" bson:"id"`
	Score float64 `json:"score" bson:"score"`
}
