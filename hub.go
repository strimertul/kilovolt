package kv

import (
	"context"
	"fmt"

	"github.com/dgraph-io/badger/v3"
	"github.com/dgraph-io/badger/v3/pb"
	jsoniter "github.com/json-iterator/go"
	cmap "github.com/orcaman/concurrent-map"
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

	subscribers cmap.ConcurrentMap // map[string]clientList

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
		clients:     newClientList(),
		subscribers: cmap.New(), // make(map[string]clientList),
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
		if subscribers, ok := h.subscribers.Get(key); ok {
			// Notify subscribers
			submsg, _ := json.Marshal(Push{"push", key, string(kv.Value)})
			for _, client := range subscribers.(*clientList).Clients() {
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
	var msg Request
	messageID := string(message.Data)
	err := json.Unmarshal(message.Data, &msg)
	if err != nil {
		message.Client.sendErr(ErrInvalidFmt, err.Error(), messageID, msg.RequestID)
		return
	}
	if msg.RequestID != "" {
		messageID = ""
	}

	switch msg.CmdName {
	case CmdReadKey:
		// Check params
		key, ok := msg.Data["key"].(string)
		if !ok {
			client.sendErr(ErrMissingParam, "invalid or missing 'key' parameter", messageID, msg.RequestID)
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
					client.sendJSON(Response{"response", true, messageID, msg.RequestID, string("")})
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
			client.sendJSON(Response{"response", true, messageID, msg.RequestID, string(byt)})
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
			client.sendErr(ErrMissingParam, "invalid or missing 'key' parameter", messageID, msg.RequestID)
			return
		}
		data, ok := msg.Data["data"].(string)
		if !ok {
			client.sendErr(ErrMissingParam, "invalid or missing 'data' parameter", messageID, msg.RequestID)
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
			client.sendErr(ErrUpdateFailed, err.Error(), messageID, msg.RequestID)
			return
		}
		// Send OK response
		client.sendJSON(Response{"response", true, messageID, msg.RequestID, nil})

		h.logger.WithFields(logrus.Fields{
			"client": client.conn.RemoteAddr(),
			"key":    string(realKey),
		}).Debug("modified key")
	case CmdSubscribeKey:
		// Check params
		key, ok := msg.Data["key"].(string)
		if !ok {
			client.sendErr(ErrMissingParam, "invalid or missing 'key' parameter", messageID, msg.RequestID)
			return
		}

		// Remap key if necessary
		realKey := key
		if client.options.RemapKeyFn != nil {
			realKey = client.options.RemapKeyFn(realKey)
		}

		var subs *clientList
		if h.subscribers.Has(realKey) {
			data, _ := h.subscribers.Get(realKey)
			subs = data.(*clientList)
		} else {
			subs = newClientList()
		}
		subs.AddClient(client)
		h.subscribers.Set(realKey, subs)
		h.logger.WithFields(logrus.Fields{
			"client": client.conn.RemoteAddr(),
			"key":    string(realKey),
		}).Debug("subscribed to key")
		// Send OK response
		client.sendJSON(Response{"response", true, messageID, msg.RequestID, nil})
	case CmdUnsubscribeKey:
		// Check params
		key, ok := msg.Data["key"].(string)
		if !ok {
			client.sendErr(ErrMissingParam, "invalid or missing 'key' parameter", messageID, msg.RequestID)
			return
		}

		// Remap key if necessary
		realKey := key
		if client.options.RemapKeyFn != nil {
			realKey = client.options.RemapKeyFn(realKey)
		}

		data, ok := h.subscribers.Get(realKey)
		if !ok {
			// No subscription, just say we're done
			client.sendJSON(Response{"response", true, messageID, msg.RequestID, nil})
			return
		}
		subs := data.(*clientList)
		if subs.Has(client); !ok {
			// No subscription from specific client, just say we're done
			client.sendJSON(Response{"response", true, messageID, msg.RequestID, nil})
			return
		}
		subs.RemoveClient(client)
		h.subscribers.Set(realKey, subs)

		h.logger.WithFields(logrus.Fields{
			"client": client.conn.RemoteAddr(),
			"key":    string(realKey),
		}).Debug("unsubscribed to key")
		// Send OK response
		client.sendJSON(Response{"response", true, messageID, msg.RequestID, nil})
	case CmdProtoVersion:
		client.sendJSON(Response{"response", true, messageID, msg.RequestID, ProtoVersion})
	default:
		client.sendErr(ErrUnknownCmd, fmt.Sprintf("command \"%s\" is mistyped or not supported", msg.CmdName), messageID, msg.RequestID)
	}
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
			for k := range h.subscribers.IterBuffered() {
				k.Val.(*clientList).RemoveClient(client)
			}
			// Delete entry and close channel
			h.clients.RemoveClient(client)
			close(client.send)

		case message := <-h.incoming:
			h.handleCmd(message.Client, message)
		}
	}
}
