package kv

import (
	"math/rand"
	"sync"
)

type clientList struct {
	data map[int64]Client
	mu   sync.Mutex
}

func newClientList() *clientList {
	return &clientList{
		data: make(map[int64]Client),
		mu:   sync.Mutex{},
	}
}

func (c *clientList) GetByID(id int64) (Client, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cl, ok := c.data[id]
	return cl, ok
}

func (c *clientList) AddClient(client Client) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	var uid int64
	for {
		// Generate Unique ID
		uid = rand.Int63()
		// Only exit if ID is not already assigned
		if _, ok := c.data[uid]; !ok {
			break
		}
	}

	client.SetUID(uid)
	c.data[uid] = client

	return uid
}

func (c *clientList) RemoveClient(client Client) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, client.UID())
}

func (c *clientList) Has(client Client) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.data[client.UID()]
	return ok
}

func (c *clientList) Clients() []Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	clients := []Client{}
	for _, cl := range c.data {
		clients = append(clients, cl)
	}
	return clients
}
