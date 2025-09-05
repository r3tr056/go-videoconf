package signalling

import (
	"log"

	"github.com/r3tr056/go-videoconf/signalling-server/internal/monitoring"
	"github.com/sirupsen/logrus"
)

type MessageHandler struct {
	roomManager *RoomManager
	userManager *UserManager
	wsServer    *WebSocketServer
}

func NewMessageHandler(rm *RoomManager, um *UserManager, wsSrv *WebSocketServer) *MessageHandler {
	return &MessageHandler{
		roomManager: rm,
		userManager: um,
		wsServer:    wsSrv,
	}
}

// Routes incomming messages based on their type
func (h *MessageHandler) HandleMessage(client *Client, msg *Message) {
	metricsMsgType := msg.Type
	monitoring.IncMessagesReceived(metricsMsgType)

	switch msg.Type {
	case TypeJoinRoom:
		h.handleJoinRoom(client, msg)
	case TypeOffer:
		h.handleOffer(client, msg)
	case TypeAnswer:
		h.handleAnswer(client, msg)
	case TypeCandidate:
		h.handleCandidate(client, msg)
	case TypePing:
		h.handlePing(client)
	case TypeLeaveRoom:
		h.handleLeaveRoom(client)
	default:
		logrus.Warnf("Unknown message type: %s, Client: %s", msg.Type, client.userID)
		h.sendError(client, "Unknown message type")
	}
}

func (h *MessageHandler) unregisterClientFromRoom(c *Client) {
	if c.roomID != "" {
		room := h.roomManager.GetRoom(c.roomID)
		if room != nil {
			room.RemoveClient(c.userID)
			h.wsServer.BroadcastToRoom(room, Message{
				Type:   TypeUserLeft,
				RoomID: room.roomID,
				UserID: c.userID,
			}, "")
			logrus.Infof("Client %s left room %s", c.userID, c.roomID)
		}
	}
}

func (h *MessageHandler) sendError(c *Client, errorMessage string) {
	c.sendMessage(Message{
		Type:    TypeError,
		Payload: errorMessage,
	})
}

// Adds the client to the specified room
func (h *MessageHandler) joinRoom(c *Client, room *Room) {
	room.AddClient(c)
	c.roomID = room.roomID
	logrus.Infof("Client %s joined room %s", c.userID, c.roomID)
	c.send <- Message{
		Type:    TypeRoomCreated,
		RoomID:  room.roomID,
		UserID:  c.userID,
		Payload: room.roomID,
	}
}

func (h *MessageHandler) handlePing(c *Client) {
	c.send <- Message{Type: TypePong}
}

func (h *MessageHandler) handleLeaveRoom(c *Client) {
	h.unregisterClientFromRoom(c)
	c.send <- Message{Type: TypeUserLeft, RoomID: c.roomID, UserID: c.userID}
	c.roomID = ""
}

// handleJoinRoom processes join_room messages
func (h *MessageHandler) handleJoinRoom(c *Client, msg *Message) {
	roomID, ok := msg.Payload.(string)
	if !ok || roomID == "" {
		logrus.Warnf("Invalid room_id in join_room message, Client: %s", c.userID)
		h.sendError(c, "Invalid room_id for join_room")
		return
	}

	room := h.roomManager.GetRoom(roomID)
	if room == nil {
		room = h.roomManager.CreateRoom(roomID)
		h.wsServer.BroadcastRoomCreated(room)
	}

	h.joinRoom(c, room)

	h.wsServer.BroadcastToRoom(room, Message{
		Type:   TypeUserJoined,
		RoomID: room.roomID,
		UserID: c.userID,
	}, c.userID)
}

// handleOffer processes offer messages and forwards them to the target peer
func (h *MessageHandler) handleOffer(c *Client, msg *Message) {
	if msg.TargetID == "" {
		log.Printf("Invalid target_id in offer message, Client: %s", c.userID)
		h.sendError(c, "Invalid target_id for offer")
		return
	}
	h.forwardMessageToPeer(c, msg, msg.TargetID)
}

// handleAnswer processes answer messages and forwards them to the target peer
func (h *MessageHandler) handleAnswer(c *Client, msg *Message) {
	if msg.TargetID == "" {
		log.Printf("Invalid target_id in answer message, Client: %s", c.userID)
		h.sendError(c, "Invalid target_id for answer")
		return
	}
	h.forwardMessageToPeer(c, msg, msg.TargetID)
}

// handleCandidate processes candidate messages and forwards them to the target peer
func (h *MessageHandler) handleCandidate(c *Client, msg *Message) {
	if msg.TargetID == "" {
		logrus.Warnf("Invalid target_id in candidate message, Client: %s", c.userID)
		h.sendError(c, "Invalid target_id for candidate")
		return
	}
	h.forwardMessageToPeer(c, msg, msg.TargetID)
}

// forwardMessageToPeer finds the target client and sends the message
func (h *MessageHandler) forwardMessageToPeer(sender *Client, msg *Message, targetID string) {
	r := h.roomManager.GetRoom(sender.roomID)
	if r == nil {
		logrus.Warnf("Room not found while forwarding message, RoomID: %s, Client: %s", sender.roomID, sender.userID)
		h.sendError(sender, "Room not found")
		return
	}

	targetClient := r.GetClient(targetID)
	if targetClient == nil {
		logrus.Warnf("Target user not found in room, TargetID: %s, RoomID: %s, Client: %s", targetID, sender.roomID, sender.userID)
		h.sendError(sender, "Target user not found in room")
		return
	}

	msg.UserID = sender.userID
	targetClient.sendMessage(*msg)
	logrus.Debugf("Forwarded message type: %s from %s to %s in room %s", msg.Type, sender.userID, targetID, sender.roomID)
	monitoring.IncMessagesSent(msg.Type)
}
