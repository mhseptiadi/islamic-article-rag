package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mhseptiadi/islamic-article-rag/internal/config"
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

	ingestion := service.NewIngestionService(embedder, vectors, articles, cfg.MaxChunkChars)

	count, err := ingestion.IngestDirectory(
		context.Background(),
		cfg.RawArticlesDir,
		cfg.ChunkWindowSize,
		cfg.ChunkStepSize,
	)
	if err != nil {
		log.Fatalf("ingest articles: %v", err)
	}

	if count > 0 {
		fmt.Printf("\nDone: Ingested %d total chunks.\n", count)
	} else {
		fmt.Println("No chunks were generated. Check data/raw_articles, data/done, or data/errors for file status.")
	}
}
