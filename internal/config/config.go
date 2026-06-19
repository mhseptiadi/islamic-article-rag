package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort string
	// QdrantURL               string
	QdrantHost              string
	QdrantAPIKey            string
	QdrantGRPCPort          int
	QdrantCollection        string
	QdrantArticleCollection string
	MongoURI                string
	MongoDatabase           string
	MongoArticlesCollection string
	MongoQnACollection      string
	EmbeddingAPIKey         string
	LLMProvider             string
	LLMAPIKey               string
	LLMApiURL               string
	LLMModel                string

	RawArticlesDir       string
	ChunkWindowSize      int
	ChunkStepSize        int
	MaxChunkChars        int
	OllamaEmbeddingURL   string
	OllamaEmbeddingModel string
	MinSimilarityScore   float64
	QnARetrievalLimit    int
	QnAContextSource     string

	RedisURL              string
	MaxIPRequestPerMinute int
	MaxRequestPerMinute   int
	MaxRequestPerDay      int
	MaxQuestionChars      int
}

func Load() (*Config, error) {
	fileEnv, err := loadFileEnv()
	if err != nil {
		return nil, fmt.Errorf("load .env: %w", err)
	}

	provider := getEnv("LLM_PROVIDER", "ollama", fileEnv)

	return &Config{
		HTTPPort: getEnv("HTTP_PORT", "8080", fileEnv),
		// QdrantURL:               getEnv("QDRANT_URL", "http://localhost:6333", fileEnv),
		QdrantHost:              getEnv("QDRANT_HOST", "localhost", fileEnv),
		QdrantAPIKey:            getEnv("QDRANT_API_KEY", "", fileEnv),
		QdrantGRPCPort:          getEnvInt("QDRANT_GRPC_PORT", 6334, fileEnv),
		QdrantCollection:        getEnv("QDRANT_COLLECTION", "indonesian_articles", fileEnv),
		QdrantArticleCollection: getEnv("QDRANT_ARTICLE_COLLECTION", "indonesian_articles_full", fileEnv),
		MongoURI:                getEnv("MONGO_URI", "mongodb://localhost:27017", fileEnv),
		MongoDatabase:           getEnv("MONGO_DATABASE", "islamic_article_rag", fileEnv),
		MongoArticlesCollection: getEnv("MONGO_ARTICLES_COLLECTION", "articles", fileEnv),
		MongoQnACollection:      getEnv("MONGO_QNA_COLLECTION", "qna_records", fileEnv),
		EmbeddingAPIKey:         getEnv("EMBEDDING_API_KEY", "", fileEnv),
		LLMProvider:             provider,
		LLMAPIKey:               getEnv("LLM_API_KEY", "", fileEnv),
		LLMApiURL:               getEnv("LLM_API_URL", "http://localhost:11434/api/generate", fileEnv),
		LLMModel:                getEnv("LLM_MODEL", "qwen2.5:7b", fileEnv),
		RawArticlesDir:          getEnv("RAW_ARTICLES_DIR", "data/raw_articles", fileEnv),
		ChunkWindowSize:         getEnvInt("CHUNK_WINDOW_SIZE", 3, fileEnv),
		ChunkStepSize:           getEnvInt("CHUNK_STEP_SIZE", 2, fileEnv),
		MaxChunkChars:           getEnvInt("MAX_CHUNK_CHARS", 6000, fileEnv),
		OllamaEmbeddingURL:      getEnv("OLLAMA_EMBEDDING_URL", "http://localhost:11434/api/embeddings", fileEnv),
		OllamaEmbeddingModel:    getEnv("OLLAMA_EMBEDDING_MODEL", "bge-m3", fileEnv),
		MinSimilarityScore:      getEnvFloat("MIN_SIMILARITY_SCORE", 0.40, fileEnv),
		QnARetrievalLimit:       getEnvInt("QNA_RETRIEVAL_LIMIT", 5, fileEnv),
		QnAContextSource:        getEnv("QNA_CONTEXT_SOURCE", "chunks", fileEnv),
		RedisURL:                getEnv("REDIS_URL", "redis://localhost:6379", fileEnv),
		MaxIPRequestPerMinute:   getEnvInt("MAX_IP_REQUESTS_PER_MINUTE", 5, fileEnv),
		MaxRequestPerMinute:     getEnvInt("MAX_REQUESTS_PER_MINUTE", 30, fileEnv),
		MaxRequestPerDay:        getEnvInt("MAX_REQUESTS_PER_DAY", 1000, fileEnv),
		MaxQuestionChars:        getEnvInt("MAX_QUESTION_CHARS", 200, fileEnv),
	}, nil
}

func getEnv(key, fallback string, fileEnv map[string]string) string {
	if v, ok := fileEnv[key]; ok {
		return v
	}
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int, fileEnv map[string]string) int {
	v := getEnv(key, "", fileEnv)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvFloat(key string, fallback float64, fileEnv map[string]string) float64 {
	v := getEnv(key, "", fileEnv)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return n
}

func loadFileEnv() (map[string]string, error) {
	path, err := findDotEnv()
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return godotenv.Unmarshal(string(stripSemicolonComments(data)))
}

func findDotEnv() (string, error) {
	if path := os.Getenv("ENV_FILE"); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		path := filepath.Join(dir, ".env")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", os.ErrNotExist
}

func stripSemicolonComments(data []byte) []byte {
	var cleaned strings.Builder
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), ";") {
			continue
		}
		cleaned.WriteString(line)
		cleaned.WriteByte('\n')
	}
	return []byte(cleaned.String())
}
