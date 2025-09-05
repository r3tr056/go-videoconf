package signalling

import "encoding/json"

const (
	TypeJoinRoom    = "join_room"
	TypeRoomCreated = "room_created"
	TypeUserJoined  = "user_joined"
	TypeUserLeft    = "user_left"
	TypeOffer       = "offer"
	TypeAnswer      = "answer"
	TypeCandidate   = "candidate"
	TypeError       = "error"
	TypePing        = "ping"
	TypePong        = "pong"
	TypeLeaveRoom   = "leave_room"
)

// Message struct to encapsulate signalling messages
type Message struct {
	Type     string      `json:"type"`
	RoomID   string      `json:"room_id,omitempty"`
	UserID   string      `json:"user_id,omitempty"`
	TargetID string      `json:"target_id,omitempty"` // For directed messages
	Payload  interface{} `json:"payload,omitempty"`
}

func (m *Message) ToJSONBytes() ([]byte, error) {
	return json.Marshal(m)
}

func MessageFromJSONBytes(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
