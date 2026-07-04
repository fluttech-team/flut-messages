package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Block struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	BlockerID string             `bson:"blocker_id"`
	BlockedID string             `bson:"blocked_id"`
	CreatedAt time.Time          `bson:"created_at"`
}
