package room

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/r3tr056/go-videoconf/room-service/internal/monitoring"
	"github.com/r3tr056/go-videoconf/room-service/internal/storage"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

type RoomManager struct {
	mongoStorage  *storage.MongoDBStorage
	activeRooms   map[string]*Room
	activeRoomsMu sync.RWMutex
}

func NewRoomManager(mongoStorage *storage.MongoDBStorage) *RoomManager {
	rm := &RoomManager{
		mongoStorage: mongoStorage,
		activeRooms:  make(map[string]*Room),
	}
	monitoring.SetGetActiveRoomCountFn(rm.GetActiveRoomCount)
	return rm
}

func (rm *RoomManager) GetActiveRoomCount() int {
	rm.activeRoomsMu.RLock()
	defer rm.activeRoomsMu.RUnlock()
	return len(rm.activeRooms)
}

func (rm *RoomManager) CreateRoom(ctx context.Context, name string, capacity int32, password string, createdByUserId string) (*Room, error) {
	roomID := uuid.New().String()
	now := time.Now()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logrus.Errorf("Failed to hash password: %v", err)
		return nil, fmt.Errorf("failed to create room: %w", err)
	}

	roomDoc := &storage.RoomDocument{
		RoomID:          roomID,
		Name:            name,
		Capacity:        capacity,
		Password:        string(hashedPassword),
		CreatedByUserId: createdByUserId,
		CreatedAt:       now,
	}

	if err := rm.mongoStorage.CreateRoomDocument(ctx, roomDoc); err != nil {
		logrus.Errorf("Failed to create room document in MongoDB: %v", err)
		return nil, fmt.Errorf("failed to create room: %w", err)
	}

	room := &Room{
		RoomID:          roomID,
		Name:            name,
		Capacity:        capacity,
		Password:        password,
		CreatedByUserId: createdByUserId,
		CreatedAt:       now,
	}
	rm.activeRooms[roomID] = room
	monitoring.IncRoomsCreated()
	monitoring.SetGetActiveRoomCountFn(rm.GetActiveRoomCount)
	return room, nil
}

func (rm *RoomManager) AuthenticateRoomPassword(ctx context.Context, roomID string, password string) (bool, error) {
	room, err := rm.GetRoom(ctx, roomID)
	if err != nil {
		return false, err
	}
	if room == nil {
		return false, nil
	}

	err = bcrypt.CompareHashAndPassword([]byte(room.Password), []byte(password))
	return err == nil, nil
}

func (rm *RoomManager) GetRoom(ctx context.Context, roomID string) (*Room, error) {
	if r, ok := rm.activeRooms[roomID]; ok {
		return r, nil
	}

	roomDoc, err := rm.mongoStorage.GetRoomDocument(ctx, roomID)
	if err != nil {
		logrus.Errorf("Failed to get room document from MongoDB: %v", err)
		return nil, fmt.Errorf("failed to get room: %w", err)
	}
	if roomDoc == nil {
		return nil, nil
	}

	room := &Room{
		RoomID:          roomDoc.RoomID,
		Name:            roomDoc.Name,
		Capacity:        roomDoc.Capacity,
		Password:        roomDoc.Password,
		CreatedByUserId: roomDoc.CreatedByUserId,
		CreatedAt:       roomDoc.CreatedAt,
	}
	rm.activeRooms[roomID] = room
	return room, nil
}

func (rm *RoomManager) UpdateRoom(ctx context.Context, roomID string, name *string, capacity *int32, password *string) (*Room, error) {
	updateDoc := bson.M{}
	if name != nil {
		updateDoc["name"] = *name
	}
	if capacity != nil {
		updateDoc["capacity"] = *capacity
	}
	if password != nil {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
		if err != nil {
			logrus.Errorf("Failed to hash password: %v", err)
			return nil, fmt.Errorf("failed to update room: %w", err)
		}
		updateDoc["password"] = string(hashedPassword)
	}

	updatedRoomDoc, err := rm.mongoStorage.UpdateRoomDocument(ctx, roomID, updateDoc)
	if err != nil {
		logrus.Errorf("Failed to update room document in MongoDB: %v", err)
		return nil, fmt.Errorf("failed to update room: %w", err)
	}

	updatedRoom := &Room{
		RoomID:          updatedRoomDoc.RoomID,
		Name:            updatedRoomDoc.Name,
		Capacity:        updatedRoomDoc.Capacity,
		Password:        updatedRoomDoc.Password,
		CreatedByUserId: updatedRoomDoc.CreatedByUserId,
		CreatedAt:       updatedRoomDoc.CreatedAt,
	}
	rm.activeRooms[roomID] = updatedRoom
	return updatedRoom, nil
}

func (rm *RoomManager) DeleteRoom(ctx context.Context, roomID string) error {
	if err := rm.mongoStorage.DeleteRoomDocument(ctx, roomID); err != nil {
		logrus.Errorf("Failed to delete room document from MongoDB: %v", err)
		return fmt.Errorf("failed to delete room: %w", err)
	}
	delete(rm.activeRooms, roomID)
	monitoring.IncRoomsDeleted()
	monitoring.SetGetActiveRoomCountFn(rm.GetActiveRoomCount)
	return nil
}

func (rm *RoomManager) AddParticipant(ctx context.Context, roomID string, userID string, role string) (*Participant, error) {
	participantID := uuid.New().String()
	now := time.Now()

	participantDoc := &storage.ParticipantDocument{
		ParticipantID: participantID,
		UserID:        userID,
		RoomID:        roomID,
		Role:          role,
		JoinedAt:      now,
	}

	if err := rm.mongoStorage.AddParticipantDocument(ctx, participantDoc); err != nil {
		logrus.Errorf("Failed to add participant document to MongoDB: %v", err)
		return nil, fmt.Errorf("failed to add participant: %w", err)
	}

	participant := &Participant{
		ParticipantID: participantID,
		UserID:        userID,
		RoomID:        roomID,
		Role:          role,
		JoinedAt:      now,
	}
	monitoring.IncParticipantsAdded()
	return participant, nil
}

func (rm *RoomManager) RemoveParticipant(ctx context.Context, participantID string) error {
	if err := rm.mongoStorage.RemoveParticipantDocument(ctx, participantID); err != nil {
		logrus.Errorf("Failed to remove participant document from MongoDB: %v", err)
		return fmt.Errorf("failed to remove participant: %w", err)
	}
	monitoring.IncParticipantsRemoved()
	return nil
}

func (rm *RoomManager) ListParticipants(ctx context.Context, roomID string) ([]*Participant, error) {
	participantDocs, err := rm.mongoStorage.ListParticipantDocumentsByRoom(ctx, roomID)
	if err != nil {
		logrus.Errorf("Failed to list participant documents from MongoDB: %v", err)
		return nil, fmt.Errorf("failed to list participants: %w", err)
	}

	participants := make([]*Participant, len(participantDocs))
	for i, doc := range participantDocs {
		participants[i] = &Participant{
			ParticipantID: doc.ParticipantID,
			UserID:        doc.UserID,
			RoomID:        doc.RoomID,
			Role:          doc.Role,
			JoinedAt:      doc.JoinedAt,
		}
	}
	return participants, nil
}
