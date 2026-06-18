package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/mhseptiadi/islamic-article-rag/internal/config"
	"github.com/mhseptiadi/islamic-article-rag/internal/handler"
	"github.com/mhseptiadi/islamic-article-rag/internal/repository/mongo"
	"github.com/mhseptiadi/islamic-article-rag/internal/repository/qdrant"
	"github.com/mhseptiadi/islamic-article-rag/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	embedder := service.NewEmbeddingClient(cfg.EmbeddingAPIKey, cfg.OllamaEmbeddingURL, cfg.OllamaEmbeddingModel)
	llm := service.NewLLMClient(cfg.LLMProvider, cfg.LLMAPIKey, cfg.LLMApiURL, cfg.LLMModel)

	vectors, err := qdrant.NewVectorRepository(cfg.QdrantHost, cfg.QdrantGRPCPort, cfg.QdrantCollection, cfg.MinSimilarityScore)
	if err != nil {
		log.Fatalf("connect to qdrant: %v", err)
	}
	defer vectors.Close()

	articles, err := mongo.NewArticleRepository(cfg.MongoURI, cfg.MongoDatabase, cfg.MongoArticlesCollection)
	if err != nil {
		log.Fatalf("connect to mongodb articles: %v", err)
	}
	defer articles.Close()

	orchestrator := service.NewQnAOrchestrator(embedder, llm, vectors, articles, cfg.QnARetrievalLimit, cfg.QnAContextSource)
	qnaHandler := handler.NewQnAHandler(orchestrator)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/ask", qnaHandler.Ask)

	addr := fmt.Sprintf(":%s", cfg.HTTPPort)
	log.Printf("starting API server on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
