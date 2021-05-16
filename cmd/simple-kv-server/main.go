package main

import (
	"flag"
	"net/http"

	"github.com/dgraph-io/badger/v3"
	"github.com/sirupsen/logrus"

	kv "github.com/strimertul/kilovolt/v4"
)

var log = logrus.New()

func wrapLogger(module string) logrus.FieldLogger {
	return log.WithField("module", module)
}

func parseLogLevel(level string) logrus.Level {
	switch level {
	case "error":
		return logrus.ErrorLevel
	case "warn", "warning":
		return logrus.WarnLevel
	case "info", "notice":
		return logrus.InfoLevel
	case "debug":
		return logrus.DebugLevel
	case "trace":
		return logrus.TraceLevel
	default:
		return logrus.InfoLevel
	}
}

func main() {
	// Get cmd line parameters
	bind := flag.String("bind", "localhost:4338", "HTTP server bind in format addr:port")
	dbfile := flag.String("dbdir", "data", "Path to strimertul database dir")
	loglevel := flag.String("loglevel", "info", "Logging level (debug, info, warn, error)")
	flag.Parse()

	log.SetLevel(parseLogLevel(*loglevel))

	// Loading routine
	options := badger.DefaultOptions(*dbfile)
	options.Logger = wrapLogger("db")
	db, err := badger.Open(options)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Initialize KV (required)
	hub, err := kv.NewHub(db, wrapLogger("kv"))
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
