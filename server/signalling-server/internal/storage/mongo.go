package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/r3tr056/go-videoconf/signalling-server/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	databaseName        = "signalling_db"
	roomsCollectionName = "rooms"
)

type MongoDBStorage struct {
	Client *mongo.Client
	db     *mongo.Database
}

func NewMongoDBStorage(cfg *config.Config) (*MongoDBStorage, error) {
	client, err := ConnectMongoDB(cfg.MongoDBURI)
	if err != nil {
		return nil, err
	}
	db := client.Database(databaseName)
	return &MongoDBStorage{Client: client, db: db}, nil
}

func ConnectMongoDB(mongoURI string) (*mongo.Client, error) {
	clientOptions := options.Client().ApplyURI(mongoURI)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	fmt.Println("Connected to MongoDB")
	return client, nil
}
