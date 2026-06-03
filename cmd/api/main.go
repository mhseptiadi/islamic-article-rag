package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/mhseptiadi/islamic-article-rag/internal/config"
	"github.com/mhseptiadi/islamic-article-rag/internal/handler"
	"github.com/mhseptiadi/islamic-article-rag/internal/repository/qdrant"
	"github.com/mhseptiadi/islamic-article-rag/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	embedder := service.NewEmbeddingClient(cfg.EmbeddingAPIKey)
	llm := service.NewLLMClient(cfg.LLMAPIKey)
	vectors := qdrant.NewVectorRepository(cfg.QdrantURL, "article_chunks")

	orchestrator := service.NewQnAOrchestrator(embedder, llm, vectors)
	qnaHandler := handler.NewQnAHandler(orchestrator)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/ask", qnaHandler.Ask)

	addr := fmt.Sprintf(":%s", cfg.HTTPPort)
	log.Printf("starting API server on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
