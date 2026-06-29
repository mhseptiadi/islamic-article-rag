package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"

	"github.com/mhseptiadi/islamic-article-rag/internal/config"
	"github.com/mhseptiadi/islamic-article-rag/internal/service/external_api"
	"github.com/mhseptiadi/islamic-article-rag/internal/handler"
	"github.com/mhseptiadi/islamic-article-rag/internal/repository/mongo"
	"github.com/mhseptiadi/islamic-article-rag/internal/repository/qdrant"
	redisrepo "github.com/mhseptiadi/islamic-article-rag/internal/repository/redis"
	"github.com/mhseptiadi/islamic-article-rag/internal/service"
)

//go:embed web/*
var webFS embed.FS

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	embedder := service.NewEmbeddingClient(cfg.EmbeddingProvider, cfg.EmbeddingAPIKey, cfg.EmbeddingURL, cfg.EmbeddingModel)
	referencesClient := external_api.NewReferencesClient(cfg.ReferencesAPIURL)
	llm := service.NewLLMClient(cfg.LLMProvider, cfg.LLMAPIKey, cfg.LLMApiURL, cfg.LLMModel, cfg.LLMTemperature, cfg.LLMMaxCompletionTokens, cfg.LLMTopP, cfg.LLMStream, cfg.LLMReasoningEffort, referencesClient)
	topicDetector := service.NewTopicDetectorClient(cfg.LLMTopicDetectorAPIKey, cfg.LLMTopicDetectorAPIURL, cfg.LLMTopicDetectorModel)
	textValidator := external_api.NewIslamicTextValidatorClient(cfg.IslamicTextValidatorURL)

	vectors, err := qdrant.NewVectorRepository(cfg.QdrantHost, cfg.QdrantAPIKey, cfg.QdrantGRPCPort, cfg.QdrantCollection, cfg.MinSimilarityScore)
	if err != nil {
		log.Fatalf("connect to qdrant: %v", err)
	}
	defer vectors.Close()

	articles, err := mongo.NewArticleRepository(cfg.MongoURI, cfg.MongoDatabase, cfg.MongoArticlesCollection)
	if err != nil {
		log.Fatalf("connect to mongodb articles: %v", err)
	}
	defer articles.Close()

	qnaRecords, err := mongo.NewQnARepository(cfg.MongoURI, cfg.MongoDatabase, cfg.MongoQnACollection)
	if err != nil {
		log.Fatalf("connect to mongodb qna records: %v", err)
	}
	defer qnaRecords.Close()

	rateLimiter, err := redisrepo.NewRateLimitRepository(cfg.RedisURL, cfg.MaxIPRequestPerMinute)
	if err != nil {
		log.Fatalf("connect to redis: %v", err)
	}
	defer rateLimiter.Close()

	orchestrator := service.NewQnAOrchestrator(
		embedder, llm, topicDetector, textValidator, vectors, articles, qnaRecords,
		cfg.QnARetrievalLimit, cfg.QnAContextSource,
		cfg.LLMProvider, cfg.LLMModel,
	)
	qnaHandler := handler.NewQnAHandler(orchestrator, cfg.MaxQuestionChars)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/ask", ipRateLimit(rateLimiter, qnaHandler.Ask))
	mux.HandleFunc("POST /api/v1/feedback", qnaHandler.Feedback)
	registerWebRoutes(mux)

	addr := fmt.Sprintf(":%s", cfg.HTTPPort)
	log.Printf("starting API server on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func ipRateLimit(limiter *redisrepo.RateLimitRepository, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed, err := limiter.AllowIP(r.Context(), handler.ClientIP(r))
		if err != nil {
			http.Error(w, "rate limit check failed", http.StatusInternalServerError)
			return
		}
		if !allowed {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}

func registerWebRoutes(mux *http.ServeMux) {
	webRoot, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("load web assets: %v", err)
	}

	fileServer := http.FileServer(http.FS(webRoot))
	mux.Handle("GET /{$}", fileServer)
	mux.Handle("GET /css/", fileServer)
	mux.Handle("GET /js/", fileServer)
}
