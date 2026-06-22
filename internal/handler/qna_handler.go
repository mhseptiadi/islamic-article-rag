package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

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

	result, err := h.orchestrator.Ask(r.Context(), req.Question)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
