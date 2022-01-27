package kv

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"

	"go.uber.org/zap"

	"github.com/dgraph-io/badger/v3"
)

type commandHandlerFn func(*Hub, Client, Request)

var handlers = map[string]commandHandlerFn{
	CmdReadKey:           cmdReadKey,
	CmdReadBulk:          cmdReadBulk,
	CmdReadPrefix:        cmdReadPrefix,
	CmdWriteKey:          cmdWriteKey,
	CmdWriteBulk:         cmdWriteBulk,
	CmdSubscribeKey:      cmdSubscribeKey,
	CmdUnsubscribeKey:    cmdUnsubscribeKey,
	CmdSubscribePrefix:   cmdSubscribePrefix,
	CmdUnsubscribePrefix: cmdUnsubscribePrefix,
	CmdProtoVersion:      cmdProtoVersion,
	CmdListKeys:          cmdListKeys,
	CmdAuthRequest:       cmdAuthRequest,
	CmdAuthChallenge:     cmdAuthChallenge,
}

func cmdReadKey(h *Hub, client Client, msg Request) {
	if !requireAuth(h, client, msg) {
		return
	}

	// Check params
	key, ok := msg.Data["key"].(string)
	if !ok {
		sendErr(client, ErrMissingParam, "invalid or missing 'key' parameter", msg.RequestID)
		return
	}

	// Remap key if necessary
	options := client.Options()
	realKey := options.Namespace + key

	err := h.db.View(func(tx *badger.Txn) error {
		val, err := tx.Get([]byte(realKey))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				client.SendJSON(Response{"response", true, msg.RequestID, ""})
				h.logger.Debug("get for inexistant key", zap.Int64("client", client.UID()), zap.String("key", realKey))
				return nil
			}
			return err
		}
		byt, err := val.ValueCopy(nil)
		if err != nil {
			return err
		}
		client.SendJSON(Response{"response", true, msg.RequestID, string(byt)})
		h.logger.Debug("get key", zap.Int64("client", client.UID()), zap.String("key", realKey))
		return nil
	})

	if err != nil {
		sendErr(client, ErrServerError, err.Error(), msg.RequestID)
		return
	}
}

func cmdReadBulk(h *Hub, client Client, msg Request) {
	if !requireAuth(h, client, msg) {
		return
	}

	// Check params
	keys, ok := msg.Data["keys"].([]interface{})
	if !ok {
		sendErr(client, ErrMissingParam, "invalid or missing 'keys' parameter", msg.RequestID)
		return
	}

	options := client.Options()
	realKeys := make([]string, len(keys))
	for index, key := range keys {
		// Remap key if necessary
		realKeys[index], ok = key.(string)
		if !ok {
			sendErr(client, ErrMissingParam, "invalid entry in 'keys' parameter", msg.RequestID)
			return
		}
		realKeys[index] = options.Namespace + realKeys[index]
	}

	out := make(map[string]string)
	err := h.db.View(func(tx *badger.Txn) error {
		for index, key := range realKeys {
			val, err := tx.Get([]byte(key))
			if err != nil {
				if err == badger.ErrKeyNotFound {
					out[key] = ""
					continue
				}
				return err
			}
			byt, err := val.ValueCopy(nil)
			if err != nil {
				return err
			}
			out[keys[index].(string)] = string(byt)
		}
		return nil
	})
	if err != nil {
		sendErr(client, ErrServerError, "server error: "+err.Error(), msg.RequestID)
		return
	}
	h.logger.Debug("get multi key", zap.Int64("client", client.UID()), zap.Strings("keys", realKeys))
	client.SendJSON(Response{"response", true, msg.RequestID, out})
}

func cmdReadPrefix(h *Hub, client Client, msg Request) {
	if !requireAuth(h, client, msg) {
		return
	}

	// Check params
	prefix, ok := msg.Data["prefix"].(string)
	if !ok {
		sendErr(client, ErrMissingParam, "invalid or missing 'prefix' parameter", msg.RequestID)
		return
	}

	// Remap key if necessary
	options := client.Options()
	realPrefix := options.Namespace + prefix

	out := make(map[string]string)
	err := h.db.View(func(tx *badger.Txn) error {
		opt := badger.DefaultIteratorOptions
		opt.Prefix = []byte(realPrefix)
		it := tx.NewIterator(opt)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			byt, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			key := string(item.Key())
			out[prefix+key[len(realPrefix):]] = string(byt)
		}
		return nil
	})
	if err != nil {
		sendErr(client, ErrServerError, err.Error(), msg.RequestID)
		return
	}
	h.logger.Debug("get all (prefix)", zap.Int64("client", client.UID()), zap.String("prefix", prefix))
	client.SendJSON(Response{"response", true, msg.RequestID, out})
}

func cmdWriteKey(h *Hub, client Client, msg Request) {
	if !requireAuth(h, client, msg) {
		return
	}

	// Check params
	key, ok := msg.Data["key"].(string)
	if !ok {
		sendErr(client, ErrMissingParam, "invalid or missing 'key' parameter", msg.RequestID)
		return
	}
	data, ok := msg.Data["data"].(string)
	if !ok {
		sendErr(client, ErrMissingParam, "invalid or missing 'data' parameter", msg.RequestID)
		return
	}

	// Remap key if necessary
	options := client.Options()
	realKey := options.Namespace + key

	err := h.db.Update(func(tx *badger.Txn) error {
		return tx.Set([]byte(realKey), []byte(data))
	})
	if err != nil {
		sendErr(client, ErrServerError, err.Error(), msg.RequestID)
		return
	}
	// Send OK response
	client.SendJSON(Response{"response", true, msg.RequestID, nil})

	h.logger.Debug("modified key", zap.Int64("client", client.UID()), zap.String("key", realKey))
}

func cmdWriteBulk(h *Hub, client Client, msg Request) {
	if !requireAuth(h, client, msg) {
		return
	}

	options := client.Options()
	// Copy data over
	kvs := make(map[string]string)
	for k, v := range msg.Data {
		strval, ok := v.(string)
		if !ok {
			sendErr(client, ErrInvalidFmt, fmt.Sprintf("invalid value for key \"%s\"", k), msg.RequestID)
			return
		}
		kvs[options.Namespace+k] = strval
	}

	err := h.db.Update(func(tx *badger.Txn) error {
		for k, v := range kvs {
			err := tx.Set([]byte(k), []byte(v))
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		sendErr(client, ErrServerError, err.Error(), msg.RequestID)
		return
	}
	// Send OK response
	client.SendJSON(Response{"response", true, msg.RequestID, nil})

	h.logger.Debug("bulk modify keys", zap.Int64("client", client.UID()))
}

func cmdSubscribeKey(h *Hub, client Client, msg Request) {
	if !requireAuth(h, client, msg) {
		return
	}

	// Check params
	key, ok := msg.Data["key"].(string)
	if !ok {
		sendErr(client, ErrMissingParam, "invalid or missing 'key' parameter", msg.RequestID)
		return
	}

	// Remap key if necessary
	options := client.Options()
	realKey := options.Namespace + key

	if err := dbSubscribeToKey(h.memdb, client, realKey); err != nil {
		sendErr(client, ErrServerError, err.Error(), msg.RequestID)
		return
	}
	h.logger.Debug("subscribed to key", zap.Int64("client", client.UID()), zap.String("key", realKey))
	// Send OK response
	client.SendJSON(Response{"response", true, msg.RequestID, nil})
}

func cmdSubscribePrefix(h *Hub, client Client, msg Request) {
	if !requireAuth(h, client, msg) {
		return
	}

	// Check params
	prefix, ok := msg.Data["prefix"].(string)
	if !ok {
		sendErr(client, ErrMissingParam, "invalid or missing 'prefix' parameter", msg.RequestID)
		return
	}

	// Remap key if necessary
	options := client.Options()
	realPrefix := options.Namespace + prefix

	if err := dbSubscribeToPrefix(h.memdb, client, realPrefix); err != nil {
		sendErr(client, ErrServerError, err.Error(), msg.RequestID)
		return
	}
	h.logger.Debug("subscribed to prefix", zap.Int64("client", client.UID()), zap.String("prefix", realPrefix))
	// Send OK response
	client.SendJSON(Response{"response", true, msg.RequestID, nil})
}

func cmdUnsubscribeKey(h *Hub, client Client, msg Request) {
	if !requireAuth(h, client, msg) {
		return
	}

	// Check params
	key, ok := msg.Data["key"].(string)
	if !ok {
		sendErr(client, ErrMissingParam, "invalid or missing 'key' parameter", msg.RequestID)
		return
	}

	// Remap key if necessary
	options := client.Options()
	realKey := options.Namespace + key

	if err := dbUnsubscribeFromKey(h.memdb, client, realKey); err != nil {
		sendErr(client, ErrServerError, err.Error(), msg.RequestID)
		return
	}
	h.logger.Debug("unsubscribed to key", zap.Int64("client", client.UID()), zap.String("key", realKey))
	// Send OK response
	client.SendJSON(Response{"response", true, msg.RequestID, nil})

}

func cmdUnsubscribePrefix(h *Hub, client Client, msg Request) {
	if !requireAuth(h, client, msg) {
		return
	}

	// Check params
	prefix, ok := msg.Data["prefix"].(string)
	if !ok {
		sendErr(client, ErrMissingParam, "invalid or missing 'prefix' parameter", msg.RequestID)
		return
	}

	// Remap key if necessary
	options := client.Options()
	realPrefix := options.Namespace + prefix

	// Add to prefix subscriber map
	if err := dbUnsubscribeFromPrefix(h.memdb, client, realPrefix); err != nil {
		sendErr(client, ErrServerError, err.Error(), msg.RequestID)
		return
	}
	h.logger.Debug("unsubscribed from prefix", zap.Int64("client", client.UID()), zap.String("prefix", realPrefix))

	// Send OK response
	client.SendJSON(Response{"response", true, msg.RequestID, nil})
}

func cmdProtoVersion(_ *Hub, client Client, msg Request) {
	client.SendJSON(Response{"response", true, msg.RequestID, ProtoVersion})
}

func cmdListKeys(h *Hub, client Client, msg Request) {
	if !requireAuth(h, client, msg) {
		return
	}

	var prefix string

	// Check params
	if prefixRaw, ok := msg.Data["prefix"]; ok {
		prefix, _ = prefixRaw.(string)
	}

	// Remap key if necessary
	options := client.Options()
	realPrefix := options.Namespace + prefix

	var out []string
	err := h.db.View(func(tx *badger.Txn) error {
		opt := badger.DefaultIteratorOptions
		opt.Prefix = []byte(realPrefix)
		it := tx.NewIterator(opt)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			out = append(out, string(item.Key()))
		}
		return nil
	})
	if err != nil {
		sendErr(client, ErrServerError, err.Error(), msg.RequestID)
		return
	}
	// If no keys are found, return empty array instead of null
	if len(out) < 1 {
		out = []string{}
	}
	h.logger.Debug("list keys", zap.Int64("client", client.UID()), zap.String("prefix", prefix))
	client.SendJSON(Response{"response", true, msg.RequestID, out})
}

func cmdAuthRequest(h *Hub, client Client, msg Request) {
	// Create challenge
	challenge := authChallenge{
		Challenge: h.randomBytes(),
		Salt:      h.randomBytes(),
	}
	_ = h.clients.SetChallenge(client.UID(), challenge)

	// Send challenge
	client.SendJSON(Response{"response", true, msg.RequestID, struct {
		Challenge string `json:"challenge"`
		Salt      string `json:"salt"`
	}{
		base64.StdEncoding.EncodeToString(challenge.Challenge[:]),
		base64.StdEncoding.EncodeToString(challenge.Salt[:]),
	}})
}

func cmdAuthChallenge(h *Hub, client Client, msg Request) {
	// Check params
	challenge, ok := msg.Data["hash"].(string)
	if !ok {
		sendErr(client, ErrMissingParam, "invalid or missing 'challenge' parameter", msg.RequestID)
		return
	}

	// Decode challenge
	challengeBytes, err := base64.StdEncoding.DecodeString(challenge)
	if err != nil {
		sendErr(client, ErrInvalidFmt, "invalid 'challenge' parameter", msg.RequestID)
		return
	}

	// Get UID
	uid := client.UID()

	// Generate hash
	challengeData, err := h.clients.Challenge(uid)
	if err != nil {
		sendErr(client, ErrAuthNotInit, "you must start an authentication challenge first", msg.RequestID)
		return
	}
	hash := hmac.New(sha256.New, append([]byte(h.options.Password), challengeData.Salt...))
	hash.Write(challengeData.Challenge)
	hashBytes := hash.Sum(nil)

	// Check if hash matches
	if subtle.ConstantTimeCompare(hashBytes, challengeBytes) != 1 {
		sendErr(client, ErrAuthFailed, "authentication failed", msg.RequestID)
		return
	}

	_ = h.clients.SetAuthenticated(uid, true)

	// Send OK response
	client.SendJSON(Response{"response", true, msg.RequestID, nil})
}

func requireAuth(h *Hub, client Client, msg Request) bool {
	// Exit early if we don't have a password (no auth required)
	if h.options.Password == "" {
		return true
	}

	if !h.clients.Authenticated(client.UID()) {
		sendErr(client, ErrAuthRequired, "authentication required", msg.RequestID)
		return false
	}

	return true
}
