package room

import "time"

type Room struct {
	RoomID          string    `json:"roomId"`
	Name            string    `json:"name"`
	Capacity        int32     `json:"capacity"`
	Password        string    `json:"password"`
	CreatedByUserId string    `json:"createdByUserId"`
	CreatedAt       time.Time `json:"createdAt"`
}

type Participant struct {
	ParticipantID string    `json:"participantId"`
	UserID        string    `json:"userId"`
	RoomID        string    `json:"roomId"`
	Role          string    `json:"role"`
	JoinedAt      time.Time `json:"joinedAt"`
}
