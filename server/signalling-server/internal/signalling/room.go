package signalling

import (
	"context"
	"sync"
	"time"

	"github.com/r3tr056/go-videoconf/signalling-server/internal/monitoring"
	"github.com/r3tr056/go-videoconf/signalling-server/internal/storage"
	"github.com/sirupsen/logrus"
)

type Room struct {
	roomID  string
	clients map[string]*Client // UserID to Client
	mu      sync.Mutex         // Mutex to protect clients map
}

type RoomManager struct {
	rooms        map[string]*Room
	mu           sync.Mutex
	mongoStorage *storage.MongoDBStorage
}

func NewRoomManager(mongoStorage *storage.MongoDBStorage) *RoomManager {
	return &RoomManager{
		rooms:        make(map[string]*Room),
		mongoStorage: mongoStorage,
	}
}

func (rm *RoomManager) CreateRoom(roomID string) *Room {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if _, exists := rm.rooms[roomID]; exists {
		return rm.rooms[roomID]
	}

	r := &Room{
		roomID:  roomID,
		clients: make(map[string]*Client),
	}
	rm.rooms[roomID] = r

	ctx := context.Background()
	roomDoc := &storage.RoomDocument{
		RoomID:    roomID,
		CreatedAt: time.Now(),
	}
	if err := rm.mongoStorage.CreateRoomDocument(ctx, roomDoc); err != nil {
		logrus.Errorf("Failed to persist room to MongoDB: %v", err)
	} else {
		logrus.Debugf("Room persisted to MongoDB: %s", roomID)
	}

	logrus.Infof("Room created: %s", roomID)
	monitoring.SetActiveRooms(len(rm.rooms))
	return r
}

func (rm *RoomManager) GetRoom(roomID string) *Room {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if r, exists := rm.rooms[roomID]; exists {
		return r
	}

	ctx := context.Background()
	roomDoc, err := rm.mongoStorage.GetRoomDocument(ctx, roomID)
	if err != nil {
		logrus.Errorf("Error retreiving room from MongDB: %v", err)
		return nil
	}

	if roomDoc != nil {
		r := &Room{
			roomID:  roomDoc.RoomID,
			clients: make(map[string]*Client),
		}
		rm.rooms[roomID] = r
		logrus.Debugf("Room loaded from MongoDB to memory: %s", roomID)
		return r
	}

	return nil
}

func (rm *RoomManager) DeleteRoom(roomID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.rooms, roomID)

	ctx := context.Background()
	if err := rm.mongoStorage.DeleteRoomDocument(ctx, roomID); err != nil {
		logrus.Errorf("Failed to delete room from MongoDB: %v", err)
	} else {
		logrus.Debugf("Room deleted from MongoDB: %s", roomID)
	}

	logrus.Infof("Room deleted: %s", roomID)
	monitoring.SetActiveRooms(len(rm.rooms))
}

func (rm *RoomManager) GetRoomsCount() int {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	return len(rm.rooms)
}

func (r *Room) AddClient(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients[client.userID] = client
}

func (r *Room) RemoveClient(userID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.clients, userID)
}

func (r *Room) GetClient(userID string) *Client {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.clients[userID]
}

func (r *Room) GetClientCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.clients)
}
