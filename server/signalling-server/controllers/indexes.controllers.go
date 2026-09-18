package controllers

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// EnsureIndexes creates the indexes joins depend on. Socket lookups run per
// join; hashedUrl is unique because the public URL must identify one session.
// It is idempotent and safe to call at every startup.
func EnsureIndexes(client *mongo.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := client.Database("vidchat").Collection("sockets").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "hashedUrl", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Fatal("Failed to ensure indexes:", err)
	}
}
