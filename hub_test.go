package kv

import (
	"testing"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/sirupsen/logrus"
)

func TestHub(t *testing.T) {
	log := logrus.New()
	log.Level = logrus.TraceLevel

	var hub *Hub
	t.Run("create", func(t *testing.T) {
		hub = createInMemoryHub(t, log)
	})

	// Run hub routines on a separate goroutine
	go hub.Run()

	client := newMockClient(log)
	t.Run("register client", func(t *testing.T) {
		hub.register <- client
		// Wait for hello or timeout
		select {
		case <-time.After(10 * time.Second):
			t.Fatal("server took too long to take action")
		case <-client.send:
		}
		if len(hub.clients.Clients()) < 1 {
			t.Fatal("client not registered")
		}
	})

	t.Run("unregister client", func(t *testing.T) {
		hub.unregister <- client
		// Wait for close or timeout
		select {
		case <-time.After(10 * time.Second):
			t.Fatal("server took too long to take action")
		case <-client.send:
		}
		if len(hub.clients.Clients()) > 0 {
			t.Fatal("client not removed")
		}
	})
}

func createInMemoryHub(t *testing.T, log logrus.FieldLogger) *Hub {
	// Open in-memory DB
	options := badger.DefaultOptions("").WithInMemory(true).WithLogger(log)
	db, err := badger.Open(options)
	if err != nil {
		t.Fatal("db initialization failed", err.Error())
	}

	// Create hub with in-mem DB
	hub, err := NewHub(db, HubOptions{}, log)
	if err != nil {
		t.Fatal("hub initialization failed", err.Error())
	}

	return hub
}
