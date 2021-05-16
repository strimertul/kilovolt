package kv

import (
	"math/rand"
	"strconv"
	"sync"
	_ "testing"

	jsoniter "github.com/json-iterator/go"
)

type mockClient struct {
	// Unique ID
	uid int64

	// Buffered channel of outbound messages.
	send chan []byte

	pending   map[string]chan interface{}
	pushes    chan Push
	responses chan Response

	options ClientOptions

	mu sync.Mutex
}

func newMockClient() *mockClient {
	return &mockClient{
		uid:       0,
		send:      make(chan []byte),
		pending:   make(map[string]chan interface{}),
		pushes:    make(chan Push, 100),
		responses: make(chan Response, 100),
		options: ClientOptions{
			Namespace: "@test/",
		},
		mu: sync.Mutex{},
	}
}

func (m *mockClient) Run() {
	for data := range m.send {
		var response Response
		jsoniter.ConfigFastest.Unmarshal(data, &response)
		// Check message
		if response.RequestID != "" {
			m.mu.Lock()
			// Get related channel
			chn, ok := m.pending[response.RequestID]
			if !ok {
				// Send to generic responses I guess??
				m.responses <- response
			} else {
				if response.Ok {
					chn <- response
				} else {
					// Must be an error, re-parse correctly
					var err Error
					jsoniter.ConfigFastest.Unmarshal(data, &err)
					chn <- err
				}
				delete(m.pending, response.RequestID)
			}
			m.mu.Unlock()
		} else {
			// Might be a push
			switch response.CmdType {
			case "push":
				var push Push
				jsoniter.ConfigFastest.Unmarshal(data, &push)
				m.pushes <- push
			}
		}

	}
}

func (c *mockClient) MakeRequest(cmd string, data map[string]interface{}) (rawMessage, <-chan interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var requestID string
	for {
		// Generate Unique ID
		requestID = strconv.FormatInt(rand.Int63(), 32)
		// Only exit if ID is not already assigned
		if _, ok := c.pending[requestID]; !ok {
			break
		}
	}
	chn := make(chan interface{}, 10)
	c.pending[requestID] = chn
	byt, _ := json.Marshal(Request{
		CmdName:   cmd,
		Data:      data,
		RequestID: requestID,
	})
	return rawMessage{c, byt}, chn
}

func (c *mockClient) SetUID(uid int64) {
	c.uid = uid
}

func (c *mockClient) UID() int64 {
	return c.uid
}

func (c *mockClient) SendJSON(data interface{}) {
	msg, _ := json.Marshal(data)
	c.send <- msg
}

func (c *mockClient) SendMessage(data []byte) {
	c.send <- data
}

func (c *mockClient) Options() ClientOptions {
	return c.options
}

func (c *mockClient) Close() {
	close(c.send)
}
