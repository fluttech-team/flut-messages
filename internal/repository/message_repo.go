package repository

import (
	"context"
	"time"

	"github.com/flutapp/chat-service/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MessageRepository interface {
	Insert(ctx context.Context, msg *domain.Message) (*domain.Message, error)
	FindByConversationID(ctx context.Context, convID primitive.ObjectID, limit int64, offset int64) ([]*domain.Message, error)
	FindByID(ctx context.Context, id primitive.ObjectID) (*domain.Message, error)
	UpdateStatus(ctx context.Context, id primitive.ObjectID, status string) error
	MarkDeleted(ctx context.Context, id primitive.ObjectID) error
	Update(ctx context.Context, id primitive.ObjectID, text string) error
	SearchByText(ctx context.Context, convID primitive.ObjectID, query string) ([]*domain.Message, error)
	FindUnreadByUserID(ctx context.Context, userID string) ([]*domain.Message, error)
}

type messageRepo struct {
	collection *mongo.Collection
}

func NewMessageRepository(db *mongo.Database) (MessageRepository, error) {
	collection := db.Collection("messages")

	// Create indexes
	indexModels := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "conversation_id", Value: 1},
				{Key: "created_at", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "receiver_id", Value: 1},
				{Key: "status", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "text", Value: "text"},
			},
		},
	}

	_, err := collection.Indexes().CreateMany(context.Background(), indexModels)
	if err != nil {
		return nil, err
	}

	return &messageRepo{
		collection: collection,
	}, nil
}

func (r *messageRepo) Insert(ctx context.Context, msg *domain.Message) (*domain.Message, error) {
	// Generate new ObjectID
	msg.ID = primitive.NewObjectID()

	// Set created_at
	msg.CreatedAt = time.Now()

	// Set status to "pending"
	msg.Status = "pending"

	// Set is_deleted to false
	msg.IsDeleted = false

	// Insert document
	result, err := r.collection.InsertOne(ctx, msg)
	if err != nil {
		return nil, err
	}

	msg.ID = result.InsertedID.(primitive.ObjectID)
	return msg, nil
}

func (r *messageRepo) FindByConversationID(ctx context.Context, convID primitive.ObjectID, limit int64, offset int64) ([]*domain.Message, error) {
	filter := bson.M{
		"conversation_id": convID,
		"is_deleted":      false,
	}

	opts := options.Find().
		SetSort(bson.M{"created_at": -1}).
		SetLimit(limit).
		SetSkip(offset)

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []*domain.Message
	err = cursor.All(ctx, &messages)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

func (r *messageRepo) FindByID(ctx context.Context, id primitive.ObjectID) (*domain.Message, error) {
	filter := bson.M{
		"_id": id,
	}

	var message domain.Message
	err := r.collection.FindOne(ctx, filter).Decode(&message)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &message, nil
}

func (r *messageRepo) UpdateStatus(ctx context.Context, id primitive.ObjectID, status string) error {
	filter := bson.M{
		"_id": id,
	}

	update := bson.M{
		"$set": bson.M{
			"status": status,
		},
	}

	// Add timestamp fields based on status
	if status == "delivered" {
		update["$set"].(bson.M)["delivered_at"] = time.Now()
	} else if status == "read" {
		update["$set"].(bson.M)["read_at"] = time.Now()
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *messageRepo) MarkDeleted(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{
		"_id": id,
	}

	update := bson.M{
		"$set": bson.M{
			"is_deleted": true,
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *messageRepo) Update(ctx context.Context, id primitive.ObjectID, text string) error {
	filter := bson.M{
		"_id": id,
	}

	update := bson.M{
		"$set": bson.M{
			"text":      text,
			"edited_at": time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *messageRepo) SearchByText(ctx context.Context, convID primitive.ObjectID, query string) ([]*domain.Message, error) {
	filter := bson.M{
		"$text": bson.M{
			"$search": query,
		},
		"conversation_id": convID,
		"is_deleted":      false,
	}

	opts := options.Find().
		SetSort(bson.M{"created_at": -1})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []*domain.Message
	err = cursor.All(ctx, &messages)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

func (r *messageRepo) FindUnreadByUserID(ctx context.Context, userID string) ([]*domain.Message, error) {
	filter := bson.M{
		"receiver_id": userID,
		"status": bson.M{
			"$in": []string{"pending", "delivered"},
		},
		"is_deleted": false,
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []*domain.Message
	err = cursor.All(ctx, &messages)
	if err != nil {
		return nil, err
	}

	return messages, nil
}
