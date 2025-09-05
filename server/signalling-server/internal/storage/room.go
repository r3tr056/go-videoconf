package storage

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type RoomDocument struct {
	RoomID    string    `bson:"room_id"`
	CreatedAt time.Time `bson:"created_at"`
}

func (s *MongoDBStorage) CreateRoomDocument(ctx context.Context, doc *RoomDocument) error {
	collection := s.db.Collection(roomsCollectionName)
	_, err := collection.InsertOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("failed to insert room document: %w", err)
	}
	return nil
}

func (s *MongoDBStorage) GetRoomDocument(ctx context.Context, roomID string) (*RoomDocument, error) {
	collection := s.db.Collection(roomsCollectionName)
	filter := bson.M{"room_id": roomID}
	var doc RoomDocument
	err := collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get room document: %w", err)
	}
	return &doc, nil
}

func (s *MongoDBStorage) DeleteRoomDocument(ctx context.Context, roomID string) error {
	collection := s.db.Collection(roomsCollectionName)
	filter := bson.M{"room_id": roomID}
	_, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delte room document: %w", err)
	}
	return nil
}

func (s *MongoDBStorage) ListRoomDocuments(ctx context.Context) ([]*RoomDocument, error) {
	collection := s.db.Collection(roomsCollectionName)
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to list room documents: %w", err)
	}
	defer cursor.Close(ctx)

	var rooms []*RoomDocument
	for cursor.Next(ctx) {
		var doc RoomDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("failed to decode room document: %w", err)
		}
		rooms = append(rooms, &doc)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error while listing room documents: %w", err)
	}

	return rooms, nil
}
