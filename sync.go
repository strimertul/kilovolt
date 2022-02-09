package kv

import (
	"errors"
	"math/rand"
	"sync"
)

var (
	ErrClientNotFound = errors.New("client not found")
)

type authChallenge struct {
	Challenge []byte
	Salt      []byte
}

type clientData struct {
	client        Client
	challenge     authChallenge
	authenticated bool
}

type clientList struct {
	data map[int64]clientData
	mu   sync.RWMutex
}

func newClientList() *clientList {
	return &clientList{
		data: make(map[int64]clientData),
		mu:   sync.RWMutex{},
	}
}

func (c *clientList) GetByID(id int64) (Client, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cl, ok := c.data[id]
	return cl.client, ok
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
	c.data[uid] = clientData{
		client:        client,
		authenticated: false,
	}

	return uid
}

func (c *clientList) RemoveClient(client Client) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, client.UID())
}

func (c *clientList) Has(client Client) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.data[client.UID()]
	return ok
}

func (c *clientList) Clients() []Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	clients := []Client{}
	for _, cl := range c.data {
		clients = append(clients, cl.client)
	}
	return clients
}

func (c *clientList) SetChallenge(id int64, challenge authChallenge) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, ok := c.data[id]
	if !ok {
		return ErrClientNotFound
	}

	data.challenge = challenge
	c.data[id] = data

	return nil
}

func (c *clientList) Challenge(id int64) (authChallenge, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	data, ok := c.data[id]
	if !ok {
		return authChallenge{}, ErrClientNotFound
	}

	return data.challenge, nil
}

func (c *clientList) SetAuthenticated(id int64, authenticated bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, ok := c.data[id]
	if !ok {
		return ErrClientNotFound
	}

	data.authenticated = authenticated
	c.data[id] = data

	return nil
}

func (c *clientList) Authenticated(id int64) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	data, ok := c.data[id]
	if !ok {
		return false
	}

	return data.authenticated
}
