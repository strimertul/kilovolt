package kv

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"

	"go.uber.org/zap"
)

type commandHandlerFn func(*Hub, Client, Request)

var handlers = map[string]commandHandlerFn{
	CmdReadKey:           cmdReadKey,
	CmdReadBulk:          cmdReadBulk,
	CmdReadPrefix:        cmdReadPrefix,
	CmdWriteKey:          cmdWriteKey,
	CmdWriteBulk:         cmdWriteBulk,
	CmdRemoveKey:         cmdRemoveKey,
	CmdSubscribeKey:      cmdSubscribeKey,
	CmdUnsubscribeKey:    cmdUnsubscribeKey,
	CmdSubscribePrefix:   cmdSubscribePrefix,
	CmdUnsubscribePrefix: cmdUnsubscribePrefix,
	CmdProtoVersion:      cmdProtoVersion,
	CmdListKeys:          cmdListKeys,
	CmdAuthRequest:       cmdAuthRequest,
	CmdAuthChallenge:     cmdAuthChallenge,
	CmdInternalClientID:  cmdInternalClientID,
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

	data, err := h.db.Get(realKey)
	if err != nil {
		if err == ErrorKeyNotFound {
			client.SendJSON(Response{"response", true, msg.RequestID, ""})
			h.logger.Debug("get for non-existent key", zap.Int64("client", client.UID()), zap.String("key", realKey))
			return
		} else {
			sendErr(client, ErrServerError, err.Error(), msg.RequestID)
			return
		}
	}
	client.SendJSON(Response{"response", true, msg.RequestID, data})
	h.logger.Debug("get key", zap.Int64("client", client.UID()), zap.String("key", realKey))
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

	results, err := h.db.GetBulk(realKeys)
	if err != nil {
		sendErr(client, ErrServerError, "server error: "+err.Error(), msg.RequestID)
		return
	}

	// Remap keys if necessary
	out := make(map[string]string)
	for key, value := range results {
		out[key[len(options.Namespace):]] = value
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

	results, err := h.db.GetPrefix(realPrefix)
	if err != nil {
		sendErr(client, ErrServerError, err.Error(), msg.RequestID)
		return
	}

	// Remap keys if necessary
	out := make(map[string]string)
	for key, value := range results {
		out[key[len(options.Namespace):]] = value
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

	err := h.db.Set(realKey, data)
	if err != nil {
		sendErr(client, ErrServerError, err.Error(), msg.RequestID)
		return
	}
	// Send OK response
	client.SendJSON(Response{"response", true, msg.RequestID, nil})

	h.subscriptions.KeyChanged(realKey, data)
	h.logger.Debug("modified key", zap.Int64("client", client.UID()), zap.String("key", realKey))
}

func cmdRemoveKey(h *Hub, client Client, msg Request) {
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

	err := h.db.Delete(realKey)
	if err != nil {
		sendErr(client, ErrServerError, err.Error(), msg.RequestID)
		return
	}
	// Send OK response
	client.SendJSON(Response{"response", true, msg.RequestID, nil})

	h.subscriptions.KeyChanged(realKey, "")
	h.logger.Debug("removed key", zap.Int64("client", client.UID()), zap.String("key", realKey))
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

	err := h.db.SetBulk(kvs)
	if err != nil {
		sendErr(client, ErrServerError, err.Error(), msg.RequestID)
		return
	}
	// Send OK response
	client.SendJSON(Response{"response", true, msg.RequestID, nil})

	for k, v := range kvs {
		h.subscriptions.KeyChanged(k, v)
	}
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

	h.subscriptions.SubscribeKey(client.UID(), realKey)
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

	h.subscriptions.SubscribePrefix(client.UID(), realPrefix)
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

	h.subscriptions.UnsubscribeKey(client.UID(), realKey)
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
	h.subscriptions.UnsubscribePrefix(client.UID(), realPrefix)
	h.logger.Debug("unsubscribed from prefix", zap.Int64("client", client.UID()), zap.String("prefix", realPrefix))

	// Send OK response
	client.SendJSON(Response{"response", true, msg.RequestID, nil})
}

func cmdProtoVersion(_ *Hub, client Client, msg Request) {
	client.SendJSON(Response{"response", true, msg.RequestID, ProtoVersion})
}

func cmdInternalClientID(_ *Hub, client Client, msg Request) {
	client.SendJSON(Response{"response", true, msg.RequestID, strconv.FormatInt(client.UID(), 10)})
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

	out, err := h.db.List(realPrefix)
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
