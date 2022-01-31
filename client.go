package kv

// Client is a middleman between the websocket connection and the hub.
type Client interface {
	Options() ClientOptions
	Close()

	SendMessage([]byte)
	SendJSON(interface{})

	SetUID(int64)
	UID() int64
}

// ClientOptions is a list of tweakable options for clients
type ClientOptions struct {
	// Adds a prefix to all key operations to restrict them to a namespace
	Namespace string
}
