package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Message struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	ConversationID  primitive.ObjectID `bson:"conversation_id"`
	SenderID        string             `bson:"sender_id"`
	ReceiverID      string             `bson:"receiver_id"`
	Text            string             `bson:"text"`
	Status          string             `bson:"status"` // "pending", "delivered", "read"
	CreatedAt       time.Time          `bson:"created_at"`
	DeliveredAt     *time.Time         `bson:"delivered_at"`
	ReadAt          *time.Time         `bson:"read_at"`
	EditedAt        *time.Time         `bson:"edited_at"`
	IsDeleted       bool               `bson:"is_deleted"`
	Attachments     []Attachment       `bson:"attachments"`
	TemplateType    *string            `bson:"template_type,omitempty"` // system/template messages migrated from backend-flut
}

type Attachment struct {
	Type string `bson:"type"` // "image", "file"
	URL  string `bson:"url"`
	Name string `bson:"name"`
}
