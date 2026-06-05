package config

import (
	"os"
	"strconv"
)

type Config struct {
	HTTPPort        string
	QdrantURL       string
	QdrantHost      string
	QdrantGRPCPort  int
	QdrantCollection string
	EmbeddingAPIKey string
	LLMAPIKey       string

	RawArticlesDir       string
	ChunkWindowSize      int
	ChunkStepSize        int
	OllamaEmbeddingURL   string
	OllamaEmbeddingModel string
}

func Load() (*Config, error) {
	return &Config{
		HTTPPort:             getEnv("HTTP_PORT", "8080"),
		QdrantURL:            getEnv("QDRANT_URL", "http://localhost:6333"),
		QdrantHost:           getEnv("QDRANT_HOST", "localhost"),
		QdrantGRPCPort:       getEnvInt("QDRANT_GRPC_PORT", 6334),
		QdrantCollection:     getEnv("QDRANT_COLLECTION", "indonesian_articles"),
		EmbeddingAPIKey:      os.Getenv("EMBEDDING_API_KEY"),
		LLMAPIKey:            os.Getenv("LLM_API_KEY"),
		RawArticlesDir:       getEnv("RAW_ARTICLES_DIR", "data/raw_articles"),
		ChunkWindowSize:      getEnvInt("CHUNK_WINDOW_SIZE", 3),
		ChunkStepSize:        getEnvInt("CHUNK_STEP_SIZE", 2),
		OllamaEmbeddingURL:   getEnv("OLLAMA_EMBEDDING_URL", "http://localhost:11434/api/embeddings"),
		OllamaEmbeddingModel: getEnv("OLLAMA_EMBEDDING_MODEL", "bge-m3"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
