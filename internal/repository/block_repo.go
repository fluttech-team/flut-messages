package repository

import (
	"context"
	"time"

	"github.com/flutapp/chat-service/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type BlockRepository interface {
	Create(ctx context.Context, blockerID string, blockedID string) (*domain.Block, error)
	Delete(ctx context.Context, blockerID string, blockedID string) error
	IsBlocked(ctx context.Context, blockerID string, blockedID string) (bool, error)
	GetBlockedList(ctx context.Context, userID string) ([]string, error)
}

type blockRepo struct {
	collection *mongo.Collection
}

func NewBlockRepository(db *mongo.Database) (BlockRepository, error) {
	collection := db.Collection("blocks")

	// Create indexes
	indexModel := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "blocker_id", Value: 1},
				{Key: "blocked_id", Value: 1},
			},
		},
	}

	_, err := collection.Indexes().CreateMany(context.Background(), indexModel)
	if err != nil {
		return nil, err
	}

	return &blockRepo{
		collection: collection,
	}, nil
}

func (r *blockRepo) Create(ctx context.Context, blockerID string, blockedID string) (*domain.Block, error) {
	block := &domain.Block{
		ID:        primitive.NewObjectID(),
		BlockerID: blockerID,
		BlockedID: blockedID,
		CreatedAt: time.Now(),
	}

	result, err := r.collection.InsertOne(ctx, block)
	if err != nil {
		return nil, err
	}

	block.ID = result.InsertedID.(primitive.ObjectID)
	return block, nil
}

func (r *blockRepo) Delete(ctx context.Context, blockerID string, blockedID string) error {
	filter := bson.M{
		"blocker_id": blockerID,
		"blocked_id": blockedID,
	}

	_, err := r.collection.DeleteOne(ctx, filter)
	return err
}

func (r *blockRepo) IsBlocked(ctx context.Context, blockerID string, blockedID string) (bool, error) {
	filter := bson.M{
		"blocker_id": blockerID,
		"blocked_id": blockedID,
	}

	var block domain.Block
	err := r.collection.FindOne(ctx, filter).Decode(&block)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (r *blockRepo) GetBlockedList(ctx context.Context, userID string) ([]string, error) {
	filter := bson.M{
		"blocker_id": userID,
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var blocks []*domain.Block
	err = cursor.All(ctx, &blocks)
	if err != nil {
		return nil, err
	}

	blockedList := make([]string, 0, len(blocks))
	for _, block := range blocks {
		blockedList = append(blockedList, block.BlockedID)
	}

	return blockedList, nil
}
