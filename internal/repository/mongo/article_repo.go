package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/mhseptiadi/islamic-article-rag/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ArticleRepository struct {
	client     *mongo.Client
	collection *mongo.Collection
}

func NewArticleRepository(uri, database, collectionName string) (*ArticleRepository, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connect to mongodb: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("ping mongodb: %w", err)
	}

	collection := client.Database(database).Collection(collectionName)
	if _, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "url", Value: 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
	}); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("ensure url index: %w", err)
	}

	return &ArticleRepository{
		client:     client,
		collection: collection,
	}, nil
}

func (r *ArticleRepository) Close() error {
	if r.client == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return r.client.Disconnect(ctx)
}

func (r *ArticleRepository) InsertArticle(ctx context.Context, article model.Article) error {
	_, err := r.collection.ReplaceOne(
		ctx,
		bson.M{"_id": article.ID},
		article,
		options.Replace().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("upsert article: %w", err)
	}

	return nil
}

func (r *ArticleRepository) GetByURL(ctx context.Context, url string) (*model.Article, error) {
	if url == "" {
		return nil, nil
	}

	var article model.Article
	err := r.collection.FindOne(ctx, bson.M{"url": url}).Decode(&article)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get article by url: %w", err)
	}

	return &article, nil
}

func (r *ArticleRepository) GetByID(ctx context.Context, articleID string) (*model.Article, error) {
	var article model.Article
	err := r.collection.FindOne(ctx, bson.M{"_id": articleID}).Decode(&article)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get article: %w", err)
	}

	return &article, nil
}
