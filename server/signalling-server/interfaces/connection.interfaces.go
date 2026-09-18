package interfaces

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Connection struct {
	Socket *websocket.Conn
	mu     sync.Mutex
}

// Send serializes all data writes for this connection. A stuck peer can only
// stall its own sender for the write deadline, never forever.
func (c *Connection) Send(message Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Socket.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return c.Socket.WriteJSON(message)
}

// Ping sends a heartbeat under the same mutex as Send: gorilla forbids
// concurrent writes, so the ping ticker must share this lock.
func (c *Connection) Ping() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Socket.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return c.Socket.WriteMessage(websocket.PingMessage, nil)
}
