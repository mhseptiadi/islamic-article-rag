package config

import "os"

type Config struct {
	HTTPPort        string
	QdrantURL       string
	EmbeddingAPIKey string
	LLMAPIKey       string
}

func Load() (*Config, error) {
	return &Config{
		HTTPPort:        getEnv("HTTP_PORT", "8080"),
		QdrantURL:       getEnv("QDRANT_URL", "http://localhost:6333"),
		EmbeddingAPIKey: os.Getenv("EMBEDDING_API_KEY"),
		LLMAPIKey:       os.Getenv("LLM_API_KEY"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
