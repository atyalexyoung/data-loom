package network

import "sync"

type ClientHub struct {
	mu      sync.RWMutex
	clients map[string]*Client
}

func NewClientHub() *ClientHub {
	return &ClientHub{
		clients: make(map[string]*Client),
	}
}

// AddClient adds a client to the list of clients for a given key
func (c *ClientHub) AddClient(client *Client) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.clients[client.Id] = client
}

// RemoveClient removes a client from the list for a given key.
func (c *ClientHub) RemoveClient(client *Client) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.clients, client.Id)
}

func (c *ClientHub) GetClient(id string) *Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.clients[id]
}
