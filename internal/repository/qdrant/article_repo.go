package qdrant

import (
	"context"
	"fmt"

	"github.com/mhseptiadi/islamic-article-rag/internal/model"
	"github.com/qdrant/go-client/qdrant"
)

const articleVectorSize = 1024

var articlePlaceholderVector = make([]float32, articleVectorSize)

type ArticleRepository struct {
	client         *qdrant.Client
	collectionName string
}

func NewArticleRepository(host string, grpcPort int, collectionName string) (*ArticleRepository, error) {
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: host,
		Port: grpcPort,
	})
	if err != nil {
		return nil, fmt.Errorf("connect to qdrant: %w", err)
	}

	return &ArticleRepository{
		client:         client,
		collectionName: collectionName,
	}, nil
}

func (r *ArticleRepository) Close() error {
	return r.client.Close()
}

func (r *ArticleRepository) InsertArticle(ctx context.Context, article model.Article) error {
	payload := qdrant.NewValueMap(map[string]any{
		"text": article.Text,
		"url":  article.URL,
	})

	point := &qdrant.PointStruct{
		Id:      pointIDFromString(article.ID),
		Vectors: qdrant.NewVectors(articlePlaceholderVector...),
		Payload: payload,
	}

	_, err := r.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: r.collectionName,
		Points:         []*qdrant.PointStruct{point},
	})
	if err != nil {
		return fmt.Errorf("upsert article: %w", err)
	}

	return nil
}

func (r *ArticleRepository) GetByID(ctx context.Context, articleID string) (*model.Article, error) {
	results, err := r.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: r.collectionName,
		Ids:            []*qdrant.PointId{pointIDFromString(articleID)},
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("get article: %w", err)
	}

	if len(results) == 0 {
		return nil, nil
	}

	payload := results[0].GetPayload()
	return &model.Article{
		ID:   articleID,
		Text: payloadString(payload, "text"),
		URL:  payloadString(payload, "url"),
	}, nil
}
