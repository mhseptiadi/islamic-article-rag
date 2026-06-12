package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mhseptiadi/islamic-article-rag/internal/config"
	"github.com/mhseptiadi/islamic-article-rag/internal/repository/qdrant"
	"github.com/mhseptiadi/islamic-article-rag/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	embedder := service.NewEmbeddingClient(cfg.EmbeddingAPIKey, cfg.OllamaEmbeddingURL, cfg.OllamaEmbeddingModel)

	vectors, err := qdrant.NewVectorRepository(cfg.QdrantHost, cfg.QdrantGRPCPort, cfg.QdrantCollection, cfg.MinSimilarityScore)
	if err != nil {
		log.Fatalf("connect to qdrant: %v", err)
	}
	defer vectors.Close()

	articles, err := qdrant.NewArticleRepository(cfg.QdrantHost, cfg.QdrantGRPCPort, cfg.QdrantArticleCollection)
	if err != nil {
		log.Fatalf("connect to qdrant articles: %v", err)
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
		fmt.Printf("\nSuccess: Ingested %d total chunks.\n", count)
	} else {
		fmt.Println("No chunks were generated. Make sure your data directory contains .md files.")
	}
}
