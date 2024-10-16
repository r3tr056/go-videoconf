package interfaces

import (
	"sync"
	"github.com/gorilla/websocket"
)

type Connection struct {
	Socket *websocket.Conn
	mu sync.Mutex
}

func (c *Connection) Send(message Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Socket.WriteJSON(message)
}