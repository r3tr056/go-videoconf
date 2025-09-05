package storage

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type RoomDocument struct {
	RoomID          string    `bson:"room_id"`
	Name            string    `bson:"name"`
	Capacity        int32     `bson:"capacity"`
	Password        string    `bson:"password"`
	CreatedByUserId string    `bson:"created_by_user_id"`
	CreatedAt       time.Time `bson:"created_at"`
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

func (s *MongoDBStorage) UpdateRoomDocument(ctx context.Context, roomID string, updateDoc bson.M) (*RoomDocument, error) {
	collection := s.db.Collection(roomsCollectionName)
	filter := bson.M{"room_id": roomID}
	update := bson.M{"$set": updateDoc}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var updatedDoc RoomDocument
	err := collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updatedDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to update room document: %w", err)
	}
	return &updatedDoc, nil
}

func (s *MongoDBStorage) DeleteRoomDocument(ctx context.Context, roomID string) error {
	collection := s.db.Collection(roomsCollectionName)
	filter := bson.M{"room_id": roomID}
	_, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete room document: %w", err)
	}
	return nil
}
