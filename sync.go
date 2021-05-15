package kv

import (
	"math/rand"
	"sync"
)

type clientList struct {
	data map[int64]*Client
	mu   sync.Mutex
}

func newClientList() *clientList {
	return &clientList{
		data: make(map[int64]*Client),
		mu:   sync.Mutex{},
	}
}

func (c *clientList) GetByID(id int64) (*Client, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cl, ok := c.data[id]
	return cl, ok
}

func (c *clientList) AddClient(client *Client) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	for {
		// Generate Unique ID
		client.uid = rand.Int63()
		// Only exit if ID is not already assigned
		if _, ok := c.data[client.uid]; !ok {
			break
		}
	}

	c.data[client.uid] = client

	return client.uid
}

func (c *clientList) RemoveClient(client *Client) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, client.uid)
}

func (c *clientList) Has(client *Client) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.data[client.uid]
	return ok
}

func (c *clientList) Clients() []*Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	clients := []*Client{}
	for _, cl := range c.data {
		clients = append(clients, cl)
	}
	return clients
}
