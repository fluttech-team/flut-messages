package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Conversation struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	ParticipantIDs []string           `bson:"participant_ids"`
	CreatedAt      time.Time          `bson:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at"`

	LastMessage *MessagePreview `bson:"last_message"`

	UnreadCount map[string]int `bson:"unread_count"` // {userA: 2, userB: 0}
}

type MessagePreview struct {
	ID        primitive.ObjectID `bson:"id"`
	Text      string             `bson:"text"`
	SenderID  string             `bson:"sender_id"`
	CreatedAt time.Time          `bson:"created_at"`
}
