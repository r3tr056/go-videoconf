package interfaces

// BSON tags are load-bearing: queries and the unique index use these exact
// field names (the driver's tagless default lowercases everything, which
// silently broke joins).
type Socket struct {
	SessionID string `bson:"sessionId"`
	HashedURL string `bson:"hashedUrl"`
	SocketURL string `bson:"socketUrl"`
}

type Message struct {
	Type        string `json:"type"`
	UserID      string `json:"userID"`
	Description string `json:"description"`
	Candidate   string `json:"candidate"`
	To          string `json:"to"`
}
