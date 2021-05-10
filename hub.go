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

type clientList map[*Client]bool

type Hub struct {
	clients    clientList
	incoming   chan rawMessage
	register   chan *Client
	unregister chan *Client

	subscribers map[string]clientList

	db *badger.DB

	logger logrus.FieldLogger
}

var json = jsoniter.ConfigDefault

func NewHub(db *badger.DB, logger logrus.FieldLogger) *Hub {
	if logger == nil {
		logger = logrus.New()
	}

	hub := &Hub{
		incoming:    make(chan rawMessage, 10),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		clients:     make(clientList),
		subscribers: make(map[string]clientList),
		db:          db,
		logger:      logger,
	}

	go func() {
		db.Subscribe(context.Background(), hub.update, []byte{})
	}()

	return hub
}

func (h *Hub) update(kvs *pb.KVList) error {
	for _, kv := range kvs.Kv {
		key := string(kv.Key)

		// Check for subscribers
		if subscribers, ok := h.subscribers[key]; ok {
			// Notify subscribers
			submsg, _ := json.Marshal(wsPush{"push", key, string(kv.Value)})
			for client := range subscribers {
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
	var msg wsRequest
	messageID := string(message.Data)
	err := json.Unmarshal(message.Data, &msg)
	if err != nil {
		message.Client.sendErr(ErrInvalidFmt, err.Error(), messageID)
		return
	}

	switch msg.CmdName {
	case CmdReadKey:
		// Check params
		key, ok := msg.Data["key"].(string)
		if !ok {
			client.sendErr(ErrMissingParam, "invalid or missing 'key' parameter", messageID)
			return
		}

		// Remap key if necessary
		realKey := key
		if client.options.RemapKeyFn != nil {
			realKey = client.options.RemapKeyFn(realKey)
		}

		h.db.View(func(tx *badger.Txn) error {
			val, err := tx.Get([]byte(realKey))
			if err != nil {
				if err == badger.ErrKeyNotFound {
					client.sendJSON(wsGenericResponse{"response", true, messageID, string("")})
					h.logger.WithFields(logrus.Fields{
						"client": client.conn.RemoteAddr(),
						"key":    string(realKey),
					}).Debug("get for inexistant key")
					return nil
				}
				return err
			}
			byt, err := val.ValueCopy(nil)
			if err != nil {
				return err
			}
			client.sendJSON(wsGenericResponse{"response", true, messageID, string(byt)})
			h.logger.WithFields(logrus.Fields{
				"client": client.conn.RemoteAddr(),
				"key":    string(realKey),
			}).Debug("get key")
			return nil
		})
	case CmdWriteKey:
		// Check params
		key, ok := msg.Data["key"].(string)
		if !ok {
			client.sendErr(ErrMissingParam, "invalid or missing 'key' parameter", messageID)
			return
		}
		data, ok := msg.Data["data"].(string)
		if !ok {
			client.sendErr(ErrMissingParam, "invalid or missing 'data' parameter", messageID)
			return
		}

		// Remap key if necessary
		realKey := key
		if client.options.RemapKeyFn != nil {
			realKey = client.options.RemapKeyFn(realKey)
		}

		err := h.db.Update(func(tx *badger.Txn) error {
			return tx.Set([]byte(realKey), []byte(data))
		})
		if err != nil {
			client.sendErr(ErrUpdateFailed, err.Error(), messageID)
		}
		// Send OK response
		client.sendJSON(wsEmptyResponse{"response", true, messageID})

		h.logger.WithFields(logrus.Fields{
			"client": client.conn.RemoteAddr(),
			"key":    string(realKey),
		}).Debug("modified key")
	case CmdSubscribeKey:
		// Check params
		key, ok := msg.Data["key"].(string)
		if !ok {
			client.sendErr(ErrMissingParam, "invalid or missing 'key' parameter", messageID)
			return
		}

		// Remap key if necessary
		realKey := key
		if client.options.RemapKeyFn != nil {
			realKey = client.options.RemapKeyFn(realKey)
		}

		_, ok = h.subscribers[realKey]
		if !ok {
			h.subscribers[realKey] = make(clientList)
		}
		h.subscribers[realKey][client] = true
		h.logger.WithFields(logrus.Fields{
			"client": client.conn.RemoteAddr(),
			"key":    string(realKey),
		}).Debug("subscribed to key")
		// Send OK response
		client.sendJSON(wsEmptyResponse{"response", true, messageID})
	case CmdUnsubscribeKey:
		// Check params
		key, ok := msg.Data["key"].(string)
		if !ok {
			client.sendErr(ErrMissingParam, "invalid or missing 'key' parameter", messageID)
			return
		}

		// Remap key if necessary
		realKey := key
		if client.options.RemapKeyFn != nil {
			realKey = client.options.RemapKeyFn(realKey)
		}

		_, ok = h.subscribers[realKey]
		if !ok {
			// No subscription, just say we're done
			client.sendJSON(wsEmptyResponse{"response", true, messageID})
			return
		}
		if _, ok := h.subscribers[realKey][client]; !ok {
			// No subscription from specific client, just say we're done
			client.sendJSON(wsEmptyResponse{"response", true, messageID})
			return
		}
		delete(h.subscribers[realKey], client)
		h.logger.WithFields(logrus.Fields{
			"client": client.conn.RemoteAddr(),
			"key":    string(realKey),
		}).Debug("unsubscribed to key")
		// Send OK response
		client.sendJSON(wsEmptyResponse{"response", true, messageID})
	case CmdProtoVersion:
		client.sendJSON(wsGenericResponse{"response", true, messageID, ProtoVersion})
	default:
		client.sendErr(ErrUnknownCmd, fmt.Sprintf("command \"%s\" is mistyped or not supported", msg.CmdName), messageID)
	}
}

func (h *Hub) Run() {
	h.logger.Info("running")
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			// Make sure client is considered active first
			if _, ok := h.clients[client]; !ok {
				continue
			}
			// Unsubscribe from all keys
			for key := range h.subscribers {
				delete(h.subscribers[key], client)
			}
			// Delete entry and close channel
			delete(h.clients, client)
			close(client.send)
		case message := <-h.incoming:
			h.handleCmd(message.Client, message)
		}
	}
}
