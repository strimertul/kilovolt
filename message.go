package kv

// Commands
const (
	CmdProtoVersion   = "version"
	CmdReadKey        = "kget"
	CmdWriteKey       = "kset"
	CmdSubscribeKey   = "ksub"
	CmdUnsubscribeKey = "kunsub"
)

const ProtoVersion = "v3"

type ErrCode string

const (
	ErrInvalidFmt   = "invalid message format"
	ErrMissingParam = "required parameter missing"
	ErrUpdateFailed = "server update failed"
	ErrUnknownCmd   = "unknown command"
)

type Request struct {
	CmdName   string                 `json:"command"`
	RequestID string                 `json:"request_id,omitempty"`
	Data      map[string]interface{} `json:"data"`
}

type Error struct {
	Ok        bool    `json:"ok"`
	Error     ErrCode `json:"error"`
	Details   string  `json:"details"`
	Cmd       string  `json:"cmd,omitempty"`
	RequestID string  `json:"request_id,omitempty"`
}

type Response struct {
	CmdType   string      `json:"type"`
	Ok        bool        `json:"ok"`
	Cmd       string      `json:"cmd"`
	RequestID string      `json:"request_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

type Push struct {
	CmdType  string `json:"type"`
	Key      string `json:"key"`
	NewValue string `json:"new_value"`
}
