package kv

import (
	"context"
	"fmt"

	"github.com/dgraph-io/badger/v3"
	"github.com/dgraph-io/badger/v3/pb"
	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
)

type rawMessage struct {
	Client *Client
	Data   []byte
}

type Hub struct {
	clients    *clientList
	incoming   chan rawMessage
	register   chan *Client
	unregister chan *Client

	db    *badger.DB
	memdb *badger.DB

	logger logrus.FieldLogger
}

var json = jsoniter.ConfigDefault

func NewHub(db *badger.DB, logger logrus.FieldLogger) (*Hub, error) {
	if logger == nil {
		logger = logrus.New()
	}

	// Create temporary DB for subscriptions
	inmemdb, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		return nil, err
	}

	hub := &Hub{
		incoming:   make(chan rawMessage, 10),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    newClientList(),
		db:         db,
		memdb:      inmemdb,
		logger:     logger,
	}

	go func() {
		err := db.Subscribe(context.Background(), hub.update, []byte{})
		if err != nil {
			logger.WithError(err).Error("db subscription halted because of error")
		}
	}()

	return hub, nil
}

func (h *Hub) Close() {
	h.memdb.Close()
}

func (h *Hub) update(kvs *pb.KVList) error {
	for _, kv := range kvs.Kv {
		// Get subscribers
		subscribers, err := dbGetSubscribersForKey(h.memdb, kv.Key)
		if err != nil {
			return err
		}

		// Notify subscribers
		submsg, _ := json.Marshal(Push{"push", string(kv.Key), string(kv.Value)})
		for _, clientid := range subscribers {
			client, ok := h.clients.GetByID(clientid)
			if ok {
				client.send <- submsg
			}
		}
	}
	return nil
}

func (h *Hub) ReadKey(key string) (string, error) {
	tx := h.db.NewTransaction(false)
	defer tx.Discard()

	val, err := tx.Get([]byte(key))
	if err != nil {
		return "", err
	}
	byt, err := val.ValueCopy(nil)
	return string(byt), err
}

func (h *Hub) WriteKey(key string, data string) error {
	tx := h.db.NewTransaction(true)
	defer tx.Discard()

	err := tx.Set([]byte(key), []byte(data))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (h *Hub) handleCmd(client *Client, message rawMessage) {
	// Decode request
	var msg Request
	err := json.Unmarshal(message.Data, &msg)
	if err != nil {
		message.Client.sendErr(ErrInvalidFmt, err.Error(), msg.RequestID)
		return
	}
	if msg.RequestID == "" {
		msg.RequestID = string(message.Data)
	}

	// Get handler for command
	handler, ok := handlers[msg.CmdName]
	if !ok {
		// No handler found, send invalid command
		client.sendErr(ErrUnknownCmd, fmt.Sprintf("command \"%s\" is mistyped or not supported", msg.CmdName), msg.RequestID)
		return
	}

	// Run handler
	handler(h, client, msg)
}

func (h *Hub) Run() {
	h.logger.Info("running")
	for {
		select {
		case client := <-h.register:
			// Generate ID
			h.clients.AddClient(client)
		case client := <-h.unregister:
			// Unsubscribe from all keys
			if err := dbUnsubscribeFromAll(h.memdb, client); err != nil {
				h.logger.WithError(err).WithField("clientid", client.uid).Error("error removing subscriptions for client")
			}
			// Delete entry and close channel
			h.clients.RemoveClient(client)
			close(client.send)

		case message := <-h.incoming:
			h.handleCmd(message.Client, message)
		}
	}
}
