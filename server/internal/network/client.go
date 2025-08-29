package network

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn *websocket.Conn
	Id   string
	mu   sync.Mutex
}

func (c *Client) SendJSON(message any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.Conn.WriteJSON(message)
}
