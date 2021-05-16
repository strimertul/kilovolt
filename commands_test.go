package kv

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestCommands(t *testing.T) {
	log := logrus.New()
	log.Level = logrus.TraceLevel

	hub := createInMemoryHub(t, log)
	defer hub.Close()
	go hub.Run()

	client1 := newMockClient()
	defer client1.Close()
	go client1.Run()

	t.Run("kset", func(t *testing.T) {
		req, chn := client1.MakeRequest(CmdWriteKey, map[string]interface{}{
			"key":  "test",
			"data": "testvalue",
		})
		hub.incoming <- req
		mustSucceed(t, waitReply(t, chn))
	})

	t.Run("kget", func(t *testing.T) {
		req, chn := client1.MakeRequest(CmdReadKey, map[string]interface{}{
			"key": "test",
		})
		hub.incoming <- req
		resp := mustSucceed(t, waitReply(t, chn))
		// Check that reply is correct
		if resp.Data.(string) != "testvalue" {
			t.Fatalf("response value for kget expected to be \"testvalue\", got \"%v\"", resp.Data)
		}
	})

	t.Run("kset-bulk", func(t *testing.T) {
		req, chn := client1.MakeRequest(CmdWriteBulk, map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		})
		hub.incoming <- req
		mustSucceed(t, waitReply(t, chn))
	})

	t.Run("kget-bulk", func(t *testing.T) {
		req, chn := client1.MakeRequest(CmdReadBulk, map[string]interface{}{
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

	t.Run("kget-all", func(t *testing.T) {
		req, chn := client1.MakeRequest(CmdReadPrefix, map[string]interface{}{
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

	t.Run("kget inexistant", func(t *testing.T) {
		req, chn := client1.MakeRequest(CmdReadKey, map[string]interface{}{
			"key": "__ this key doesn't exist I swear __",
		})
		hub.incoming <- req
		resp := mustSucceed(t, waitReply(t, chn))
		// Check that reply is correct (empty)
		if resp.Data.(string) != "" {
			t.Fatalf("response value for kget expected to be empty, got \"%v\"", resp.Data)
		}
	})

	//
	// Error conditions
	//

	// Missing parameters
	t.Run("errors", func(t *testing.T) {
		noParams := []string{
			CmdReadKey, CmdReadBulk, CmdReadPrefix, CmdWriteKey, CmdWriteBulk,
			CmdSubscribeKey, CmdSubscribePrefix, CmdUnsubscribeKey, CmdUnsubscribePrefix,
		}
		for _, cmd := range noParams {
			t.Run(cmd+" with wrong key", func(t *testing.T) {
				t.Parallel()
				req, chn := client1.MakeRequest(cmd, map[string]interface{}{
					"@dingus": "bogus",
				})
				hub.incoming <- req
				resp := mustFail(t, waitReply(t, chn))
				// Check that reply is correct (empty)
				if resp.Error != ErrMissingParam {
					t.Fatalf("error value for kget expected to be \"%s\", got \"%s\"", ErrMissingParam, resp.Error)
				}
			})
		}

		wrongType := map[string]map[string]interface{}{
			CmdReadKey:           {"key": 1234},
			CmdReadBulk:          {"keys": 1234},
			CmdReadPrefix:        {"prefix": 1234},
			CmdWriteKey:          {"key": 1234, "data": 1234},
			CmdWriteBulk:         {"test": 1234},
			CmdSubscribeKey:      {"key": 1234},
			CmdSubscribePrefix:   {"prefix": 1234},
			CmdUnsubscribeKey:    {"key": 1234},
			CmdUnsubscribePrefix: {"prefix": 1234},
		}
		for cmd, data := range wrongType {
			t.Run(cmd+" with invalid key type", func(t *testing.T) {
				t.Parallel()
				req, chn := client1.MakeRequest(cmd, data)
				hub.incoming <- req
				resp := mustFail(t, waitReply(t, chn))
				// Check that reply is correct (empty)
				if resp.Error != ErrMissingParam {
					t.Fatalf("error value for kget expected to be \"%s\", got \"%s\"", ErrMissingParam, resp.Error)
				}
			})
		}
	})
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
