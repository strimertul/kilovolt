package kv

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestWebsocketServer(t *testing.T) {
	log, _ := zap.NewDevelopment()
	hub := createInMemoryHub(t, log)
	defer hub.Close()
	go hub.Run()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		hub.CreateWebsocketClient(w, r, ClientOptions{})
	})
	httptest.NewServer(mux)
}
