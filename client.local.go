package kv

import (
	"math/rand"
	"strconv"
	"sync"

	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
)

type SubscriptionCallback func(key string, value string)

type LocalClient struct {
	// Unique ID
	uid int64

	// Buffered channel of outbound messages.
	send chan []byte

	subscriptions *subscriptionManager
	callbacks     map[int64]SubscriptionCallback
	pending       map[string]chan interface{}
	responses     chan Response

	logger  *zap.Logger
	options ClientOptions

	mu    sync.Mutex
	ready chan bool
}

func NewLocalClient(options ClientOptions, log *zap.Logger) *LocalClient {
	if log == nil {
		log, _ = zap.NewProduction()
	}

	return &LocalClient{
		uid:           0,
		send:          make(chan []byte),
		subscriptions: makeSubscriptionManager(),
		callbacks:     make(map[int64]SubscriptionCallback),
		pending:       make(map[string]chan interface{}),
		responses:     make(chan Response, 100),
		logger:        log,
		options:       options,
		mu:            sync.Mutex{},
		ready:         make(chan bool),
	}
}

func (m *LocalClient) Run() {
	for data := range m.send {
		m.logger.Debug("received from server", zap.String("data", string(data)))
		var response Response
		err := jsoniter.ConfigFastest.Unmarshal(data, &response)
		if err != nil {
			m.logger.Error("failed to unmarshal response", zap.Error(err))
			continue
		}

		// Check message
		if response.RequestID != "" {
			func() {
				m.mu.Lock()
				defer m.mu.Unlock()

				// Get related channel
				chn, ok := m.pending[response.RequestID]
				if !ok {
					m.logger.Warn("received response for an unmatched request", zap.String("request-id", response.RequestID))
					return
				}
				defer delete(m.pending, response.RequestID)

				if response.Ok {
					chn <- response
					return
				}

				// Must be an error, reparse correctly
				var err Error
				parseErr := jsoniter.ConfigFastest.Unmarshal(data, &err)
				if parseErr != nil {
					m.logger.Error("failed to unmarshal data", zap.Error(parseErr))
					return
				}
				chn <- err
			}()
			continue
		}

		// Might be a push
		switch response.CmdType {
		case "push":
			var push Push
			err = jsoniter.ConfigFastest.Unmarshal(data, &push)
			if err != nil {
				m.logger.Error("failed to unmarshal push", zap.Error(err))
				continue
			}
			subscriberIds := m.subscriptions.GetSubscribers(push.Key)
			for _, subscriberId := range subscriberIds {
				callback, ok := m.callbacks[subscriberId]
				if ok {
					go callback(push.Key, push.NewValue)
				}
			}
		case "hello":
			m.ready <- true
		}
	}
}

func (c *LocalClient) Wait() {
	<-c.ready
}

func (c *LocalClient) MakeRequest(cmd string, data map[string]interface{}) (Message, <-chan interface{}) {
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
	return Message{c, byt}, chn
}

func (c *LocalClient) createCallback(callback SubscriptionCallback) (id int64) {
	for {
		id = rand.Int63()
		if _, ok := c.callbacks[id]; !ok {
			c.callbacks[id] = callback
			return
		}
	}
}

func (c *LocalClient) SetKeySubCallback(key string, callback SubscriptionCallback) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Generate random, unused ID
	id := c.createCallback(callback)
	c.subscriptions.SubscribeKey(id, key)
	return id
}

func (c *LocalClient) SetPrefixSubCallback(key string, callback SubscriptionCallback) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Generate random, unused ID
	id := c.createCallback(callback)
	c.subscriptions.SubscribePrefix(id, key)
	return id
}

func (c *LocalClient) UnsetCallback(id int64) {
	_, ok := c.callbacks[id]
	if !ok {
		return
	}
	c.mu.Lock()
	c.subscriptions.UnsubscribeAll(id)
	delete(c.callbacks, id)
	c.mu.Unlock()
}

func (c *LocalClient) SetUID(uid int64) {
	c.uid = uid
}

func (c *LocalClient) UID() int64 {
	return c.uid
}

func (c *LocalClient) SendJSON(data interface{}) {
	msg, _ := json.Marshal(data)
	c.send <- msg
}

func (c *LocalClient) SendMessage(data []byte) {
	c.send <- data
}

func (c *LocalClient) Options() ClientOptions {
	return c.options
}

func (c *LocalClient) Close() {
	if c.send != nil {
		close(c.send)
		c.send = nil
	}
}
