package main

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/r3tr056/go-videoconf/signalling-server/interfaces"
)

// MessageRouter is the SFU seam (P2.4): the WebSocket handler routes every
// signalling message through this interface. v1 fans out with an in-memory
// mesh relay; a v2 SFU can implement the same interface to take over media
// switching without changing wshandler.
type MessageRouter interface {
	// Add registers (or re-registers) a member and returns its handle.
	Add(socket, userID string, conn *websocket.Conn) *interfaces.Connection
	// Remove drops a member, closing its socket.
	Remove(socket, userID string)
	// Route delivers msg from sender to the other members of socket.
	Route(socket, sender string, msg interfaces.Message)
	// Members returns the current member ids of socket.
	Members(socket string) []string
}

// MeshRelay is the v1 MessageRouter: in-memory broadcast fan-out. All methods
// are safe for concurrent use.
type MeshRelay struct {
	mu    sync.RWMutex
	rooms map[string]map[string]*interfaces.Connection
}

func NewMeshRelay() *MeshRelay {
	return &MeshRelay{rooms: make(map[string]map[string]*interfaces.Connection)}
}

func (m *MeshRelay) Add(socket, userID string, conn *websocket.Conn) *interfaces.Connection {
	m.mu.Lock()
	defer m.mu.Unlock()
	room := m.rooms[socket]
	if room == nil {
		room = make(map[string]*interfaces.Connection)
		m.rooms[socket] = room
	}
	if room[userID] == nil {
		room[userID] = &interfaces.Connection{Socket: conn}
	} else {
		// Rejoin replaces the stale socket; the old read loop exits on
		// its own once the client is gone.
		room[userID].Socket = conn
	}
	return room[userID]
}

func (m *MeshRelay) Remove(socket, userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	room := m.rooms[socket]
	if room == nil {
		return
	}
	if c := room[userID]; c != nil {
		c.Socket.Close()
		delete(room, userID)
	}
	if len(room) == 0 {
		delete(m.rooms, socket)
	}
}

func (m *MeshRelay) Route(socket, sender string, msg interfaces.Message) {
	// Targeted delivery: offers, answers, and ICE candidates carry a `to`
	// member. Without this, every offer would fork answers from the whole
	// room and the mesh would never converge past two members.
	if msg.To != "" {
		m.mu.RLock()
		target := m.rooms[socket][msg.To]
		m.mu.RUnlock()
		if target != nil {
			if err := target.Send(msg); err != nil {
				m.prune(socket, msg.To, target)
			}
		}
		return
	}

	m.mu.RLock()
	room := m.rooms[socket]
	peers := make(map[string]*interfaces.Connection, len(room))
	for user, c := range room {
		if user != sender {
			peers[user] = c
		}
	}
	m.mu.RUnlock()

	for user, c := range peers {
		if err := c.Send(msg); err != nil {
			m.prune(socket, user, c)
		}
	}
}

// prune drops a dead connection, but only if the room entry is still the
// same connection (a rejoin may have replaced it).
func (m *MeshRelay) prune(socket, user string, c *interfaces.Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cur := m.rooms[socket][user]; cur == c {
		cur.Socket.Close()
		delete(m.rooms[socket], user)
	}
}

func (m *MeshRelay) Members(socket string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	room := m.rooms[socket]
	out := make([]string, 0, len(room))
	for user := range room {
		out = append(out, user)
	}
	return out
}

// rateLimiter is a small rolling-window limiter: at most maxHits messages per
// window. Signalling is low-rate; anything beyond this is abuse or a bug.
type rateLimiter struct {
	mu      sync.Mutex
	hits    []time.Time
	maxHits int
	window  time.Duration
}

func newRateLimiter(maxHits int, window time.Duration) *rateLimiter {
	return &rateLimiter{maxHits: maxHits, window: window}
}

func (l *rateLimiter) allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-l.window)
	kept := l.hits[:0]
	for _, t := range l.hits {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	l.hits = kept
	if len(l.hits) >= l.maxHits {
		return false
	}
	l.hits = append(l.hits, now)
	return true
}
