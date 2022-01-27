package main

import (
	"flag"
	"net/http"

	"github.com/strimertul/kilovolt/v7/drivers/badgerdb"

	"go.uber.org/zap"

	"github.com/dgraph-io/badger/v3"
	kv "github.com/strimertul/kilovolt/v7"
)

func main() {
	// Get cmd line parameters
	bind := flag.String("bind", "localhost:4338", "HTTP server bind in format addr:port")
	dbfile := flag.String("dbdir", "data", "Path to strimertul database dir")
	password := flag.String("password", "", "Optional password for authentication")
	flag.Parse()

	// Loading routine
	options := badger.DefaultOptions(*dbfile)
	db, err := badger.Open(options)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	logger, _ := zap.NewDevelopment()

	// Initialize KV (required)
	hub, err := kv.NewHub(badgerdb.NewBadgerBackend(db), kv.HubOptions{Password: *password}, logger)
	if err != nil {
		panic(err)
	}
	go hub.Run()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		kv.ServeWs(hub, w, r)
	})

	// Start HTTP server
	http.ListenAndServe(*bind, nil)
}
