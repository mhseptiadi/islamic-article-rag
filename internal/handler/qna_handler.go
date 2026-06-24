package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/mhseptiadi/islamic-article-rag/internal/model"
	mongorepo "github.com/mhseptiadi/islamic-article-rag/internal/repository/mongo"
	"github.com/mhseptiadi/islamic-article-rag/internal/service"
)

type QnAHandler struct {
	orchestrator      *service.QnAOrchestrator
	maxQuestionChars int
}

func NewQnAHandler(orchestrator *service.QnAOrchestrator, maxQuestionChars int) *QnAHandler {
	return &QnAHandler{
		orchestrator:      orchestrator,
		maxQuestionChars: maxQuestionChars,
	}
}

type askRequest struct {
	Question string `json:"question"`
}

func (h *QnAHandler) Ask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req askRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Question == "" {
		http.Error(w, "question is required", http.StatusBadRequest)
		return
	}

	if h.maxQuestionChars > 0 && len(req.Question) > h.maxQuestionChars {
		http.Error(w, fmt.Sprintf("question exceeds maximum length of %d characters", h.maxQuestionChars), http.StatusBadRequest)
		return
	}

	result, err := h.orchestrator.Ask(r.Context(), req.Question, ClientIP(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

const maxFeedbackCommentChars = 1000

type feedbackRequest struct {
	RecordID     string `json:"record_id"`
	FeedbackType string `json:"feedback_type"`
	Comment      string `json:"comment"`
}

func (h *QnAHandler) Feedback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req feedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.RecordID == "" {
		http.Error(w, "record_id is required", http.StatusBadRequest)
		return
	}
	if req.FeedbackType == "" {
		http.Error(w, "feedback_type is required", http.StatusBadRequest)
		return
	}
	if len(req.Comment) > maxFeedbackCommentChars {
		http.Error(w, fmt.Sprintf("comment exceeds maximum length of %d characters", maxFeedbackCommentChars), http.StatusBadRequest)
		return
	}

	err := h.orchestrator.SubmitFeedback(r.Context(), req.RecordID, model.FeedbackType(req.FeedbackType), req.Comment)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidFeedbackType):
			http.Error(w, "invalid feedback_type", http.StatusBadRequest)
		case errors.Is(err, mongorepo.ErrQnARecordNotFound):
			http.Error(w, "qna record not found", http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
