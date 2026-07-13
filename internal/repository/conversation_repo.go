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

type ConversationRepository interface {
	Create(ctx context.Context, conversation *domain.Conversation) (*domain.Conversation, error)
	FindByApplicationID(ctx context.Context, applicationID string) (*domain.Conversation, error)
	FindByID(ctx context.Context, id primitive.ObjectID) (*domain.Conversation, error)
	FindByUserID(ctx context.Context, userID string, limit int64, offset int64) ([]*domain.Conversation, error)
	UpdateLastMessage(ctx context.Context, conversationID primitive.ObjectID, message *domain.MessagePreview) error
	UpdateUnreadCount(ctx context.Context, conversationID primitive.ObjectID, userID string, increment int) error
}

type conversationRepo struct {
	collection *mongo.Collection
}

func NewConversationRepository(db *mongo.Database) (ConversationRepository, error) {
	collection := db.Collection("conversations")

	// Create indexes
	indexModel := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "participant_ids", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "participant_ids", Value: 1},
				{Key: "updated_at", Value: -1},
			},
		},
		{
			Keys:    bson.D{{Key: "application_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}

	_, err := collection.Indexes().CreateMany(context.Background(), indexModel)
	if err != nil {
		return nil, err
	}

	return &conversationRepo{
		collection: collection,
	}, nil
}

func (r *conversationRepo) Create(ctx context.Context, conversation *domain.Conversation) (*domain.Conversation, error) {
	// Generate new ObjectID
	conversation.ID = primitive.NewObjectID()

	// Set timestamps
	now := time.Now()
	conversation.CreatedAt = now
	conversation.UpdatedAt = now

	// Initialize UnreadCount map if nil
	if conversation.UnreadCount == nil {
		conversation.UnreadCount = make(map[string]int)
	}

	// Insert document
	result, err := r.collection.InsertOne(ctx, conversation)
	if err != nil {
		return nil, err
	}

	conversation.ID = result.InsertedID.(primitive.ObjectID)
	return conversation, nil
}

func (r *conversationRepo) FindByParticipants(ctx context.Context, userIDs []string) (*domain.Conversation, error) {
	filter := bson.M{
		"participant_ids": bson.M{
			"$all": userIDs,
		},
	}

	var conversation domain.Conversation
	err := r.collection.FindOne(ctx, filter).Decode(&conversation)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &conversation, nil
}

func (r *conversationRepo) FindByApplicationID(ctx context.Context, applicationID string) (*domain.Conversation, error) {
	var conversation domain.Conversation
	err := r.collection.FindOne(ctx, bson.M{"application_id": applicationID}).Decode(&conversation)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &conversation, nil
}

func (r *conversationRepo) FindByID(ctx context.Context, id primitive.ObjectID) (*domain.Conversation, error) {
	filter := bson.M{
		"_id": id,
	}

	var conversation domain.Conversation
	err := r.collection.FindOne(ctx, filter).Decode(&conversation)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &conversation, nil
}

func (r *conversationRepo) FindByUserID(ctx context.Context, userID string, limit int64, offset int64) ([]*domain.Conversation, error) {
	filter := bson.M{
		"participant_ids": userID,
	}

	opts := options.Find().
		SetSort(bson.M{"updated_at": -1}).
		SetLimit(limit).
		SetSkip(offset)

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var conversations []*domain.Conversation
	err = cursor.All(ctx, &conversations)
	if err != nil {
		return nil, err
	}

	return conversations, nil
}

func (r *conversationRepo) UpdateLastMessage(ctx context.Context, conversationID primitive.ObjectID, message *domain.MessagePreview) error {
	filter := bson.M{
		"_id": conversationID,
	}

	update := bson.M{
		"$set": bson.M{
			"last_message": message,
			"updated_at":   time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *conversationRepo) UpdateUnreadCount(ctx context.Context, conversationID primitive.ObjectID, userID string, increment int) error {
	filter := bson.M{
		"_id": conversationID,
	}

	update := bson.M{
		"$inc": bson.M{
			"unread_count." + userID: increment,
		},
	}

	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}
