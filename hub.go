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
	Client Client
	Data   []byte
}

type Hub struct {
	clients    *clientList
	incoming   chan rawMessage
	register   chan Client
	unregister chan Client

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
		register:   make(chan Client, 10),
		unregister: make(chan Client, 10),
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
		for _, clientid := range subscribers {
			client, ok := h.clients.GetByID(clientid)
			if ok {
				options := client.Options()
				submsg, _ := json.Marshal(Push{"push", string(kv.Key[len(options.Namespace):]), string(kv.Value)})
				client.SendMessage(submsg)
			}
		}
	}
	return nil
}

func (h *Hub) handleCmd(client Client, message rawMessage) {
	// Decode request
	var msg Request
	err := json.Unmarshal(message.Data, &msg)
	if err != nil {
		sendErr(message.Client, ErrInvalidFmt, err.Error(), msg.RequestID)
		return
	}

	// Get handler for command
	handler, ok := handlers[msg.CmdName]
	if !ok {
		// No handler found, send invalid command
		sendErr(client, ErrUnknownCmd, fmt.Sprintf("command \"%s\" is mistyped or not supported", msg.CmdName), msg.RequestID)
		return
	}

	// Run handler
	handler(h, client, msg)
}

func sendErr(client Client, err ErrCode, details string, requestID string) {
	client.SendJSON(Error{false, err, details, requestID})
}

func (h *Hub) Run() {
	h.logger.Info("running")
	for {
		select {
		case client := <-h.register:
			// Generate ID
			h.clients.AddClient(client)

			// Send welcome message
			client.SendJSON(Hello{CmdType: "hello", Version: ProtoVersion})

		case client := <-h.unregister:
			// Unsubscribe from all keys
			if err := dbUnsubscribeFromAll(h.memdb, client); err != nil {
				h.logger.WithError(err).WithField("clientid", client.UID()).Error("error removing subscriptions for client")
			}

			// Delete entry and close channel
			h.clients.RemoveClient(client)
			client.Close()

		case message := <-h.incoming:
			h.handleCmd(message.Client, message)
		}
	}
}
