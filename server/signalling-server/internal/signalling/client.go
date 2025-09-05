package signalling

import (
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/r3tr056/go-videoconf/signalling-server/internal/monitoring"
	"github.com/sirupsen/logrus"
)

type Client struct {
	conn     *websocket.Conn
	userID   string
	roomID   string
	server   *UserManager
	send     chan Message // Channel to send messages to the client
	lastPong time.Time    // Track last pong message for health checks
}

type UserManager struct {
	clients    map[string]*Client
	clientsMu  sync.Mutex
	upgrader   websocket.Upgrader
	pingPeriod time.Duration
}

func NewUserManager(pingPeriod time.Duration) *UserManager {
	return &UserManager{
		clients: make(map[string]*Client),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		pingPeriod: pingPeriod,
	}
}

func (um *UserManager) NewClient(conn *websocket.Conn) *Client {
	client := &Client{
		conn:     conn,
		userID:   uuid.New().String(),
		server:   um,
		send:     make(chan Message, 256),
		lastPong: time.Now(),
	}
	um.registerClient(client)
	monitoring.IncConnectedClients()
	return client
}

func (um *UserManager) registerClient(client *Client) {
	um.clientsMu.Lock()
	defer um.clientsMu.Unlock()
	um.clients[client.userID] = client
	logrus.Infof("Client connected: %s, Total clients: %d", client.userID, len(um.clients))
}

func (um *UserManager) unregisterClient(client *Client) {
	um.clientsMu.Lock()
	defer um.clientsMu.Unlock()
	if _, ok := um.clients[client.userID]; ok {
		delete(um.clients, client.userID)
		monitoring.DecConnectedClients()
		logrus.Infof("Client disconnected: %s, Total clients: %d", client.userID, len(um.clients))
	}
}

func (um *UserManager) getClient(userID string) *Client {
	um.clientsMu.Lock()
	defer um.clientsMu.Unlock()
	return um.clients[userID]
}

func (um *UserManager) getConnectedClientsOutsideRooms() []*Client {
	um.clientsMu.Lock()
	defer um.clientsMu.Unlock()

	connectedClients := make([]*Client, 0)
	for _, client := range um.clients {
		if client.roomID == "" {
			connectedClients = append(connectedClients, client)
		}
	}
	return connectedClients
}

func (c *Client) sendMessage(message Message) {
	select {
	case c.send <- message:
	default:
		logrus.Warnf("Client send channel full, dropping message for user %s", c.userID)
		// Consider handling channel full scenarios more gracefully if needed
	}
}
