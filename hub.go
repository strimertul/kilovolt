package kv

import (
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	mrand "math/rand"

	"go.uber.org/zap"

	"github.com/dgraph-io/badger/v3/pb"
	jsoniter "github.com/json-iterator/go"
)

type rawMessage struct {
	Client Client
	Data   []byte
}

type HubOptions struct {
	Password string
}

type Hub struct {
	options       HubOptions
	clients       *clientList
	incoming      chan rawMessage
	register      chan Client
	unregister    chan Client
	subscriptions *SubscriptionManager

	db Driver

	logger *zap.Logger
}

var json = jsoniter.ConfigDefault

func NewHub(db Driver, options HubOptions, logger *zap.Logger) (*Hub, error) {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	clients := newClientList()
	subscriptions := makeSubscriptionManager()

	hub := &Hub{
		incoming:      make(chan rawMessage, 10),
		register:      make(chan Client, 10),
		unregister:    make(chan Client, 10),
		clients:       clients,
		db:            db,
		logger:        logger,
		options:       options,
		subscriptions: subscriptions,
	}

	subscriptions.hub = hub

	return hub, nil
}

func (hub *Hub) Close() {
}

func (hub *Hub) SetOptions(options HubOptions) {
	hub.options = options
}

func (hub *Hub) update(kvs *pb.KVList) error {
	for _, kv := range kvs.Kv {
		// Get subscribers
		subscribers := hub.subscriptions.GetSubscribers(string(kv.Key))

		// Notify subscribers
		for _, clientid := range subscribers {
			client, ok := hub.clients.GetByID(clientid)
			if ok {
				options := client.Options()
				submsg, _ := json.Marshal(Push{"push", string(kv.Key[len(options.Namespace):]), string(kv.Value)})
				client.SendMessage(submsg)
			}
		}
	}
	return nil
}

func (hub *Hub) handleCmd(client Client, message rawMessage) {
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
	handler(hub, client, msg)
}

func (hub *Hub) randomBytes() []byte {
	saltBytes := make([]byte, 32)
	_, err := crand.Read(saltBytes)
	if err != nil {
		hub.logger.Warn("failed to generate secure bytes, falling back to insecure PRNG", zap.Error(err))
		insecureBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(insecureBytes, mrand.Uint64())
		return insecureBytes
	}
	return saltBytes
}

func sendErr(client Client, err ErrCode, details string, requestID string) {
	client.SendJSON(Error{false, err, details, requestID})
}

func (hub *Hub) Run() {
	hub.logger.Info("running")
	for {
		select {
		case client := <-hub.register:
			// Generate ID
			hub.clients.AddClient(client)

			// Send welcome message
			client.SendJSON(Hello{CmdType: "hello", Version: ProtoVersion})

		case client := <-hub.unregister:
			// Unsubscribe from all keys
			hub.subscriptions.UnsubscribeAll(client.UID())

			// Delete entry and close channel
			hub.clients.RemoveClient(client)
			client.Close()

		case message := <-hub.incoming:
			hub.handleCmd(message.Client, message)
		}
	}
}
