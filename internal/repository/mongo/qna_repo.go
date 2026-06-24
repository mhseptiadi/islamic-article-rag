package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mhseptiadi/islamic-article-rag/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrQnARecordNotFound = errors.New("qna record not found")

type QnARepository struct {
	client     *mongo.Client
	collection *mongo.Collection
}

func NewQnARepository(uri, database, collectionName string) (*QnARepository, error) {
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
		Keys: bson.D{{Key: "created_at", Value: -1}},
	}); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("ensure created_at index: %w", err)
	}

	return &QnARepository{
		client:     client,
		collection: collection,
	}, nil
}

func (r *QnARepository) Close() error {
	if r.client == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return r.client.Disconnect(ctx)
}

func (r *QnARepository) Insert(ctx context.Context, record model.QnARecord) error {
	_, err := r.collection.InsertOne(ctx, record)
	if err != nil {
		return fmt.Errorf("insert qna record: %w", err)
	}
	return nil
}

func (r *QnARepository) UpdateFeedback(ctx context.Context, id string, feedbackType model.FeedbackType, comment string) error {
	now := time.Now().UTC()
	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"feedback_type": string(feedbackType),
			"comment":       comment,
			"feedback_at":   now,
		},
	})
	if err != nil {
		return fmt.Errorf("update qna feedback: %w", err)
	}
	if result.MatchedCount == 0 {
		return ErrQnARecordNotFound
	}
	return nil
}
