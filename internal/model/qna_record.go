package model

import "time"

// enum feedback type
// good: Answer is good.
// hallucinated: The answer is hallucinated (fake info not in the text).
// wrong_context: The answer is true, but it doesn't match the user's actual question.
// wrong_citation: Wrong Quran or Hadith recitation/translation.
// incomplete: The answer is incomplete.
// false_refusal: The LLM refused to answer the question..
// other: Something else.

type FeedbackType string

const (
	FeedbackTypeGood          FeedbackType = "good"
	FeedbackTypeHallucinated  FeedbackType = "hallucinated"
	FeedbackTypeWrongContext  FeedbackType = "wrong_context"
	FeedbackTypeWrongCitation FeedbackType = "wrong_citation"
	FeedbackTypeIncomplete    FeedbackType = "incomplete"
	FeedbackTypeFalseRefusal  FeedbackType = "false_refusal"
	FeedbackTypeOther         FeedbackType = "other"
)

func (t FeedbackType) Valid() bool {
	switch t {
	case FeedbackTypeGood, FeedbackTypeHallucinated, FeedbackTypeWrongContext,
		FeedbackTypeWrongCitation, FeedbackTypeIncomplete, FeedbackTypeFalseRefusal, FeedbackTypeOther:
		return true
	default:
		return false
	}
}

type QnARecord struct {
	ID              string     `json:"id" bson:"_id"`
	Question        string     `json:"question" bson:"question"`
	Answer          string     `json:"answer" bson:"answer"`
	ValidatedAnswer string     `json:"validated_answer" bson:"validated_answer"`
	LLMProvider     string     `json:"llm_provider,omitempty" bson:"llm_provider,omitempty"`
	LLMModel        string     `json:"llm_model,omitempty" bson:"llm_model,omitempty"`
	ContextSource   string     `json:"context_source,omitempty" bson:"context_source,omitempty"`
	ArticleIDs      []string   `json:"article_ids,omitempty" bson:"article_ids,omitempty"`
	Chunks          []QnAChunk `json:"chunks,omitempty" bson:"chunks,omitempty"`
	CreatedAt       time.Time  `json:"created_at" bson:"created_at"`

	IPAddress string `json:"ip_address,omitempty" bson:"ip_address,omitempty"`

	// feedback
	FeedbackType string    `json:"feedback_type,omitempty" bson:"feedback_type,omitempty"`
	Comment      string    `json:"comment,omitempty" bson:"comment,omitempty"`
	FeedbackAt   time.Time `json:"feedback_at,omitempty" bson:"feedback_at,omitempty"`
}

type QnAChunk struct {
	ID    string  `json:"id" bson:"id"`
	Score float64 `json:"score" bson:"score"`
}
