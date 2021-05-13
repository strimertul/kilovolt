package kv

import "sync"

type clientList struct {
	data map[*Client]bool
	mu   sync.Mutex
}

func newClientList() *clientList {
	return &clientList{
		data: make(map[*Client]bool),
		mu:   sync.Mutex{},
	}
}

func (c *clientList) AddClient(client *Client) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[client] = true
}

func (c *clientList) RemoveClient(client *Client) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, client)
}

func (c *clientList) Has(client *Client) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.data[client]
	return ok && c.data[client]
}
