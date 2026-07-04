package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func NewMongoClient(uri string) (*mongo.Client, error) {
	return mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
}

func GetDatabase(client *mongo.Client, dbName string) *mongo.Database {
	return client.Database(dbName)
}
