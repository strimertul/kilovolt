package main

import (
	"flag"
	"net/http"

	kv "git.sr.ht/~ashkeel/kilovolt/v11"
	"go.uber.org/zap"
)

func main() {
	bind := flag.String("port", ":8080", "host:port to listen on")
	password := flag.String("password", "", "password to use (leave blank for no password)")
	flag.Parse()

	log, err := zap.NewDevelopment()
	checkErr(err)

	driver := kv.MakeBackend()
	hub, err := kv.NewHub(driver, kv.HubOptions{Password: *password}, log)
	checkErr(err)

	defer hub.Close()
	go hub.Run()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		hub.CreateWebsocketClient(w, r, kv.ClientOptions{})
	})
	checkErr(http.ListenAndServe(*bind, nil))
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
