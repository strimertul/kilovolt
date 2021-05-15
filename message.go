package kv

const ProtoVersion = "v4"

// Commands
const (
	CmdProtoVersion      = "version"
	CmdReadKey           = "kget"
	CmdReadBulk          = "kget-bulk"
	CmdReadPrefix        = "kget-all"
	CmdWriteKey          = "kset"
	CmdWriteBulk         = "kset-bulk"
	CmdSubscribeKey      = "ksub"
	CmdSubscribePrefix   = "ksub-prefix"
	CmdUnsubscribeKey    = "kunsub"
	CmdUnsubscribePrefix = "kunsub-prefix"
)

type ErrCode string

const (
	ErrServerError  = "server error"
	ErrInvalidFmt   = "invalid message format"
	ErrMissingParam = "required parameter missing"
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
	RequestID string  `json:"request_id,omitempty"`
}

type Response struct {
	CmdType   string      `json:"type"`
	Ok        bool        `json:"ok"`
	RequestID string      `json:"request_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

type Push struct {
	CmdType  string `json:"type"`
	Key      string `json:"key"`
	NewValue string `json:"new_value"`
}
