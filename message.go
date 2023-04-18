package kv

const ProtoVersion = "v9"

// Commands
const (
	CmdProtoVersion      = "version"
	CmdReadKey           = "kget"
	CmdReadBulk          = "kget-bulk"
	CmdReadPrefix        = "kget-all"
	CmdWriteKey          = "kset"
	CmdWriteBulk         = "kset-bulk"
	CmdRemoveKey         = "kdel"
	CmdSubscribeKey      = "ksub"
	CmdSubscribePrefix   = "ksub-prefix"
	CmdUnsubscribeKey    = "kunsub"
	CmdUnsubscribePrefix = "kunsub-prefix"
	CmdListKeys          = "klist"
	CmdAuthRequest       = "klogin"
	CmdAuthChallenge     = "kauth"
	CmdInternalClientID  = "_uid"
)

type ErrCode string

const (
	ErrServerError      ErrCode = "server error"
	ErrInvalidFmt       ErrCode = "invalid message format"
	ErrMissingParam     ErrCode = "required parameter missing"
	ErrUnknownCmd       ErrCode = "unknown command"
	ErrAuthNotInit      ErrCode = "authentication not initialized"
	ErrAuthFailed       ErrCode = "authentication failed"
	ErrAuthRequired     ErrCode = "authentication required"
	ErrAuthNotRequired  ErrCode = "authentication not required"
	ErrAuthNotSupported ErrCode = "authentication method not supported"
)

type AuthType string

const (
	AuthTypeChallenge   AuthType = "challenge"
	AuthTypeInteractive AuthType = "ask"
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

type Hello struct {
	CmdType string `json:"type"`
	Version string `json:"version"`
}
