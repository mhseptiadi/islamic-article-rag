package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/mhseptiadi/islamic-article-rag/internal/config"
	"github.com/mhseptiadi/islamic-article-rag/internal/handler"
	"github.com/mhseptiadi/islamic-article-rag/internal/repository/mongo"
	"github.com/mhseptiadi/islamic-article-rag/internal/repository/qdrant"
	redisrepo "github.com/mhseptiadi/islamic-article-rag/internal/repository/redis"
	"github.com/mhseptiadi/islamic-article-rag/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	embedder := service.NewEmbeddingClient(cfg.EmbeddingAPIKey, cfg.EmbeddingURL, cfg.EmbeddingModel)
	llm := service.NewLLMClient(cfg.LLMProvider, cfg.LLMAPIKey, cfg.LLMApiURL, cfg.LLMModel)

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
		embedder, llm, vectors, articles, qnaRecords,
		cfg.QnARetrievalLimit, cfg.QnAContextSource,
		cfg.LLMProvider, cfg.LLMModel,
	)
	qnaHandler := handler.NewQnAHandler(orchestrator)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/ask", ipRateLimit(rateLimiter, qnaHandler.Ask))

	addr := fmt.Sprintf(":%s", cfg.HTTPPort)
	log.Printf("starting API server on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func ipRateLimit(limiter *redisrepo.RateLimitRepository, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed, err := limiter.AllowIP(r.Context(), clientIP(r))
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

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if ip := strings.TrimSpace(strings.Split(xff, ",")[0]); ip != "" {
			return ip
		}
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
