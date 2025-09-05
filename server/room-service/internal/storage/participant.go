package storage

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ParticipantDocument struct {
	ParticipantID string    `bson:"participant_id"`
	UserID        string    `bson:"user_id"`
	RoomID        string    `bson:"room_id"`
	Role          string    `bson:"role"`
	JoinedAt      time.Time `bson:"joined_at"`
}

func (s *MongoDBStorage) AddParticipantDocument(ctx context.Context, doc *ParticipantDocument) error {
	collection := s.db.Collection(participantsCollectionName)
	_, err := collection.InsertOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("failed to insert participant document: %w", err)
	}
	return nil
}

func (s *MongoDBStorage) GetParticipantDocument(ctx context.Context, participantID string) (*ParticipantDocument, error) {
	collection := s.db.Collection(participantsCollectionName)
	filter := bson.M{"participant_id": participantID}
	var doc ParticipantDocument
	err := collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get participant document: %w", err)
	}
	return &doc, nil
}

func (s *MongoDBStorage) RemoveParticipantDocument(ctx context.Context, participantID string) error {
	collection := s.db.Collection(participantsCollectionName)
	filter := bson.M{"participant_id": participantID}
	_, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to remove participant document: %w", err)
	}
	return nil
}

func (s *MongoDBStorage) ListParticipantDocumentsByRoom(ctx context.Context, roomID string) ([]*ParticipantDocument, error) {
	collection := s.db.Collection(participantsCollectionName)
	filter := bson.M{"room_id": roomID}
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find participant documents: %w", err)
	}
	defer cursor.Close(ctx)

	var participants []*ParticipantDocument
	for cursor.Next(ctx) {
		var doc ParticipantDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("failed to decode participant document: %w", err)
		}
		participants = append(participants, &doc)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error while listing participant documents: %w", err)
	}

	return participants, nil
}
