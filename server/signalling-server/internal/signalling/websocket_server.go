package signalling

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/r3tr056/go-videoconf/signalling-server/config"
	"github.com/r3tr056/go-videoconf/signalling-server/internal/auth"
	"github.com/sirupsen/logrus"
)

type WebSocketServer struct {
	Config         *config.Config
	UserManager    *UserManager
	RoomManager    *RoomManager
	MessageHandler *MessageHandler
	authConfig     *auth.AuthConfig
}

func NewWebSocketServer(cfg *config.Config, um *UserManager, rm *RoomManager, msgHandler *MessageHandler, authCfg *auth.AuthConfig) *WebSocketServer {
	return &WebSocketServer{
		Config:         cfg,
		UserManager:    um,
		RoomManager:    rm,
		MessageHandler: msgHandler,
		authConfig:     authCfg,
	}
}

func (s *WebSocketServer) HandleWebSocketConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := s.UserManager.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.Errorf("WebSocket upgrade error: %v", err)
		return
	}

	client := s.UserManager.NewClient(conn)

	go s.WritePump(client)
	go s.ReadPump(client)
}

// Pumps messages from the WebSocket connection to the client's message handler.
// It also handles WebSocket control message like close, ping/pong, etc
func (s *WebSocketServer) ReadPump(client *Client) {
	defer func() {
		s.UserManager.unregisterClient(client)
		if client.roomID != "" {
			s.MessageHandler.unregisterClientFromRoom(client)
			s.BroadcastToRoom(s.RoomManager.GetRoom(client.roomID), Message{
				Type:   TypeUserLeft,
				RoomID: client.roomID,
				UserID: client.userID,
			}, "")
		}
		client.conn.Close()
		logrus.Debugf("Stopped readPump for client: %s", client.userID)
	}()

	client.conn.SetReadLimit(512)
	client.conn.SetReadDeadline(time.Now().Add(time.Minute * 5))
	client.conn.SetPongHandler(func(string) error {
		client.lastPong = time.Now()
		client.conn.SetReadDeadline(time.Now().Add(time.Minute * 5))
		return nil
	})

	for {
		_, messageBytes, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.Warnf("Unexpected close error: %v, Client: %s", err, client.userID)
			} else {
				logrus.Debugf("Read error: %v, Client: %s:", client.userID, err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			log.Printf("JSON unmarshal error: %v, Mesage: %s, Client: %s", err, string(messageBytes), client.userID)
			s.MessageHandler.sendError(client, "Invalid message format")
			continue
		}
	}
}

// Pumps messages from the send channel to the WebScoket connection.
// It also handles sending ping messages for connection health checks
func (s *WebSocketServer) WritePump(client *Client) {
	pingInterval := s.Config.GetPingInterval()
	ticket := time.NewTicker(pingInterval)
	defer func() {
		ticket.Stop()
		client.conn.Close()
		logrus.Debugf("Stopped writePump for client: %s", client.userID)
	}()

	for {
		select {
		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(time.Second * 10))
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				logrus.Errorf("Error getting writer: %v, Client: %s", err, client.userID)
				return
			}

			msgBytes, err := json.Marshal(message)
			if err != nil {
				logrus.Errorf("JSON marshal error: %v, Message: %+v, Client: %s", err, message, client.userID)
				continue
			}
			w.Write(msgBytes)

			if err := w.Close(); err != nil {
				logrus.Errorf("Error closing writer: %v, Client: %s", err, client.userID)
				return
			}

		case <-ticket.C:
			client.conn.SetWriteDeadline(time.Now().Add(time.Second * 10))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logrus.Debugf("Ping error: %v, Client: %s", err, client.userID)
				return
			}
		}
	}
}

// Sends a message to all clients in a room, optionally excluding a sender
func (s *WebSocketServer) BroadcastToRoom(room *Room, message Message, excludedUserID string) {
	room.mu.Lock()
	defer room.mu.Unlock()
	for _, client := range room.clients {
		if client.userID != excludedUserID {
			select {
			case client.send <- message:
			default:
				close(client.send)
				delete(room.clients, client.userID)
				log.Printf("Closed channel and removed disconnected client %s from room %s during broadcast.", client.userID, room.roomID)
			}
		}
	}
}

// Sends a `room_created` message to all clients currently conencted (outside of rooms)
// This could be used for a room listing feature.
func (s *WebSocketServer) BroadcastRoomCreated(room *Room) {
	clientsOutsideRooms := s.UserManager.getConnectedClientsOutsideRooms()
	for _, client := range clientsOutsideRooms {
		select {
		case client.send <- Message{
			Type:    TypeRoomCreated,
			RoomID:  room.roomID,
			Payload: room.roomID,
		}:

		default:
			close(client.send)
			log.Printf("Failed to broadcast room_created to client %s, channel closed.", client.userID)
		}
	}
}

func (s *WebSocketServer) GetRoomManager() *RoomManager {
	return s.RoomManager
}
