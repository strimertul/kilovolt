package kv

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

	"go.uber.org/zap"
)

const test_namespace = "@test/"

func makeHubClient(t *testing.T, test func(hub *Hub, client *LocalClient)) {
	log, _ := zap.NewDevelopment()
	hub := createInMemoryHub(t, log)
	defer hub.Close()
	go hub.Run()

	client := NewLocalClient(ClientOptions{test_namespace}, log)
	defer client.Close()
	go client.Run()

	hub.AddClient(client)
	client.Wait()
	defer hub.RemoveClient(client)

	test(hub, client)
}

func prepareKey(t *testing.T, hub *Hub, key string, expected string) {
	err := hub.db.Set(test_namespace+key, expected)
	if err != nil {
		t.Fatal(err)
	}
}

func assertKey(t *testing.T, hub *Hub, key string, expected string) {
	val, err := hub.db.Get(test_namespace + key)
	if err != nil {
		if err == ErrorKeyNotFound {
			t.Errorf("Key '%s' not found", key)
		} else {
			t.Errorf("Error getting key '%s': %s", key, err)
		}
	}
	if val != expected {
		t.Errorf("Expected '%s', got '%s'", expected, val)
	}
}

func TestKeySet(t *testing.T) {
	makeHubClient(t, func(hub *Hub, client *LocalClient) {
		req, chn := client.MakeRequest(CmdWriteKey, map[string]interface{}{
			"key":  "test",
			"data": "test-value",
		})
		hub.incoming <- req
		mustSucceed(t, waitReply(t, chn))
		assertKey(t, hub, "test", "test-value")
	})
}

func TestKeyGet(t *testing.T) {
	makeHubClient(t, func(hub *Hub, client *LocalClient) {
		prepareKey(t, hub, "test", "test-value")
		req, chn := client.MakeRequest(CmdReadKey, map[string]interface{}{
			"key": "test",
		})
		hub.incoming <- req
		resp := mustSucceed(t, waitReply(t, chn))
		// Check that reply is correct
		if resp.Data.(string) != "test-value" {
			t.Fatalf("response value for kget expected to be \"testvalue\", got \"%v\"", resp.Data)
		}
	})
}

func TestKeySetBulk(t *testing.T) {
	makeHubClient(t, func(hub *Hub, client *LocalClient) {
		req, chn := client.MakeRequest(CmdWriteBulk, map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		})
		hub.incoming <- req
		mustSucceed(t, waitReply(t, chn))
		assertKey(t, hub, "key1", "value1")
		assertKey(t, hub, "key2", "value2")
	})
}

func TestKeyList(t *testing.T) {
	makeHubClient(t, func(hub *Hub, client *LocalClient) {
		prepareKey(t, hub, "key1", "test-value")
		prepareKey(t, hub, "key2", "test-value")
		req, chn := client.MakeRequest(CmdListKeys, map[string]interface{}{
			"prefix": "key",
		})
		hub.incoming <- req
		resp := mustSucceed(t, waitReply(t, chn))
		data := resp.Data.([]interface{})
		if len(data) != 2 {
			t.Fatalf("response value for klist expected to be a 2 item list, got \"%v\"", resp.Data)
		}
	})
}

func TestKeyGetBulk(t *testing.T) {
	makeHubClient(t, func(hub *Hub, client *LocalClient) {
		prepareKey(t, hub, "key1", "value1")
		prepareKey(t, hub, "key2", "value2")
		req, chn := client.MakeRequest(CmdReadBulk, map[string]interface{}{
			"keys": []string{"key1", "key2"},
		})
		hub.incoming <- req
		resp := mustSucceed(t, waitReply(t, chn))
		// Check that reply is correct
		values := resp.Data.(map[string]interface{})
		if values["key1"].(string) != "value1" || values["key2"].(string) != "value2" {
			t.Fatal("response values are different from what expected", values)
		}
	})
}

func TestKeyGetPrefix(t *testing.T) {
	makeHubClient(t, func(hub *Hub, client *LocalClient) {
		prepareKey(t, hub, "key1", "value1")
		prepareKey(t, hub, "key2", "value2")
		req, chn := client.MakeRequest(CmdReadPrefix, map[string]interface{}{
			"prefix": "key",
		})
		hub.incoming <- req
		resp := mustSucceed(t, waitReply(t, chn))
		// Check that reply is correct
		values := resp.Data.(map[string]interface{})
		if values["key1"].(string) != "value1" || values["key2"].(string) != "value2" {
			t.Fatal("response values are different from what expected", values)
		}
	})
}

func TestKeyGetEmpty(t *testing.T) {
	makeHubClient(t, func(hub *Hub, client *LocalClient) {
		req, chn := client.MakeRequest(CmdReadKey, map[string]interface{}{
			"key": "test",
		})
		hub.incoming <- req
		resp := mustSucceed(t, waitReply(t, chn))
		// Check that reply is correct (empty)
		if resp.Data.(string) != "" {
			t.Fatalf("response value for kget expected to be empty, got \"%v\"", resp.Data)
		}
	})
}

func TestErrorMissingParam(t *testing.T) {
	noParams := []string{
		CmdReadKey, CmdReadBulk, CmdReadPrefix, CmdWriteKey,
		CmdSubscribeKey, CmdSubscribePrefix, CmdUnsubscribeKey, CmdUnsubscribePrefix,
	}
	for _, cmd := range noParams {
		t.Run(cmd+" with wrong key", func(t *testing.T) {
			makeHubClient(t, func(hub *Hub, client *LocalClient) {
				req, chn := client.MakeRequest(cmd, map[string]interface{}{
					"@dingus": "bogus",
				})
				hub.incoming <- req
				resp := mustFail(t, waitReply(t, chn))
				// Check that reply is correct (empty)
				if resp.Error != ErrMissingParam {
					t.Fatalf("error value for %s expected to be \"%s\", got \"%s\"", cmd, ErrMissingParam, resp.Error)
				}
			})
		})
	}
}

func TestErrorWrongType(t *testing.T) {
	wrongType := map[string]map[string]interface{}{
		CmdReadKey:           {"key": 1234},
		CmdReadBulk:          {"keys": 1234},
		CmdReadPrefix:        {"prefix": 1234},
		CmdWriteKey:          {"key": 1234, "data": 1234},
		CmdSubscribeKey:      {"key": 1234},
		CmdSubscribePrefix:   {"prefix": 1234},
		CmdUnsubscribeKey:    {"key": 1234},
		CmdUnsubscribePrefix: {"prefix": 1234},
	}
	for cmd, data := range wrongType {
		t.Run(cmd+" with invalid key type", func(t *testing.T) {
			makeHubClient(t, func(hub *Hub, client *LocalClient) {
				req, chn := client.MakeRequest(cmd, data)
				hub.incoming <- req
				resp := mustFail(t, waitReply(t, chn))
				// Check that reply is correct (empty)
				if resp.Error != ErrMissingParam {
					t.Fatalf("error value for kget expected to be \"%s\", got \"%s\"", ErrMissingParam, resp.Error)
				}
			})
		})
	}

	// kset-bulk is special, returns InvalidFmt on wrong format
	t.Run(CmdWriteBulk+" with invalid key type", func(t *testing.T) {
		makeHubClient(t, func(hub *Hub, client *LocalClient) {
			req, chn := client.MakeRequest(CmdWriteBulk, map[string]interface{}{"test": 1234})
			hub.incoming <- req
			resp := mustFail(t, waitReply(t, chn))
			// Check that reply is correct (empty)
			if resp.Error != ErrInvalidFmt {
				t.Fatalf("error value for kget expected to be \"%s\", got \"%s\"", ErrInvalidFmt, resp.Error)
			}
		})
	})
}

func TestKeySubscription(t *testing.T) {
	makeHubClient(t, func(hub *Hub, client *LocalClient) {
		// Subscribe to key
		req, chn := client.MakeRequest(CmdSubscribeKey, map[string]interface{}{
			"key": "sub-test",
		})
		hub.incoming <- req
		mustSucceed(t, waitReply(t, chn))

		// Check that subscription is in database
		prefixedKey := client.options.Namespace + "sub-test"
		lst := hub.subscriptions.GetSubscribers(prefixedKey)
		if len(lst) < 1 {
			t.Fatal("subscribe failed, subscription not present")
		}

		// Modify key
		req, chn = client.MakeRequest(CmdWriteKey, map[string]interface{}{
			"key":  "sub-test",
			"data": "yo this is a new value!",
		})
		hub.incoming <- req
		mustSucceed(t, waitReply(t, chn))

		// Check for pushes
		select {
		case <-time.After(10 * time.Second):
			t.Fatal("push took too long to arrive")
		case push := <-client.pushes:
			if push.Key != "sub-test" || push.NewValue != "yo this is a new value!" {
				t.Fatal("wrong push received", push)
			}
		}

		// Unsubscribe to key
		req, chn = client.MakeRequest(CmdUnsubscribeKey, map[string]interface{}{
			"key": "sub-test",
		})
		hub.incoming <- req
		mustSucceed(t, waitReply(t, chn))

		// Check that subscription is not in database anymore
		lst = hub.subscriptions.GetSubscribers(prefixedKey)
		if len(lst) > 0 {
			t.Fatal("unsubscribe failed, subscription still present")
		}
	})
}

func TestPrefixSubscription(t *testing.T) {
	makeHubClient(t, func(hub *Hub, client *LocalClient) {

		// Subscribe to key
		req, chn := client.MakeRequest(CmdSubscribePrefix, map[string]interface{}{
			"prefix": "sub-",
		})
		hub.incoming <- req
		mustSucceed(t, waitReply(t, chn))

		// Check that subscription is in database
		prefixedKey := client.options.Namespace + "sub-test-1234"
		lst := hub.subscriptions.GetSubscribers(prefixedKey)
		if len(lst) < 1 {
			t.Fatal("subscribe failed, subscription not present")
		}

		// Modify key
		req, chn = client.MakeRequest(CmdWriteKey, map[string]interface{}{
			"key":  "sub-test-1234",
			"data": "yo this is a new value!",
		})
		hub.incoming <- req
		mustSucceed(t, waitReply(t, chn))

		// Check for pushes
		select {
		case <-time.After(10 * time.Second):
			t.Fatal("push took too long to arrive")
		case push := <-client.pushes:
			if push.Key != "sub-test-1234" || push.NewValue != "yo this is a new value!" {
				t.Fatal("wrong push received", push)
			}
		}

		// Unsubscribe to key
		req, chn = client.MakeRequest(CmdUnsubscribePrefix, map[string]interface{}{
			"prefix": "sub-",
		})
		hub.incoming <- req
		mustSucceed(t, waitReply(t, chn))

		// Check that subscription is not in database anymore
		lst = hub.subscriptions.GetSubscribers(prefixedKey)
		if len(lst) > 0 {
			t.Fatal("unsubscribe failed, subscription still present")
		}
	})

}

func TestAuthentication(t *testing.T) {
	const password = "test"

	log, _ := zap.NewDevelopment()

	hub := createInMemoryHub(t, log)
	hub.SetOptions(HubOptions{Password: password})
	defer hub.Close()
	go hub.Run()

	client := NewLocalClient(ClientOptions{test_namespace}, log)
	defer client.Close()
	go client.Run()

	hub.AddClient(client)
	client.Wait()
	defer hub.RemoveClient(client)

	// Make sure client is not authenticated
	if hub.clients.Authenticated(client.UID()) {
		t.Fatal("client just connected and is already considered authenticated")
	}

	// Make authentication request
	req, chn := client.MakeRequest(CmdAuthRequest, map[string]interface{}{})
	hub.incoming <- req
	challenge := mustSucceed(t, waitReply(t, chn))
	data := challenge.Data.(map[string]interface{})

	// Decode challenge
	challengeBytes, err := base64.StdEncoding.DecodeString(data["challenge"].(string))
	if err != nil {
		t.Fatal("failed to decode challenge", err.Error())
	}
	saltBytes, err := base64.StdEncoding.DecodeString(data["salt"].(string))
	if err != nil {
		t.Fatal("failed to decode salt", err.Error())
	}

	// Create hash from password and challenge
	hash := hmac.New(sha256.New, append([]byte(password), saltBytes...))
	hash.Write(challengeBytes)
	hashBytes := hash.Sum(nil)

	// Send auth challenge
	req, chn = client.MakeRequest(CmdAuthChallenge, map[string]interface{}{
		"hash": base64.StdEncoding.EncodeToString(hashBytes),
	})
	hub.incoming <- req

	mustSucceed(t, waitReply(t, chn))

	// Make sure client is authenticated now
	if !hub.clients.Authenticated(client.UID()) {
		t.Fatal("client just authenticated but considered not authenticated")
	}
}

func waitReply(t *testing.T, chn <-chan interface{}) interface{} {
	// Wait for response or timeout
	select {
	case <-time.After(10 * time.Second):
		t.Fatal("server took too long to respond")
	case response := <-chn:
		return response
	}
	panic("unreacheable")
}

func mustSucceed(t *testing.T, resp interface{}) Response {
	switch v := resp.(type) {
	case Response:
		return v
	case Error:
		t.Fatalf("received server error: [%s] %s", v.Error, v.Details)
	default:
		t.Fatalf("received unexpected type: %T", v)
	}
	panic("unreacheable")
}

func mustFail(t *testing.T, resp interface{}) Error {
	switch v := resp.(type) {
	case Response:
		t.Fatalf("received response with data: %v", v.Data)
	case Error:
		return v
	default:
		t.Fatalf("received unexpected type: %T", v)
	}
	panic("unreacheable")
}
