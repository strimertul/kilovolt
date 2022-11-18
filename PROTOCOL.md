# Kilovolt Protocol

Kilovolt exposes a WebSocket server and speaks using text JSON messages.

**Note:** This documentation pertains to Kilovolt protocol version `v9`! If you are coming from previous versions, check the [migration notes](MIGRATION.md).

## Message format

### Client messages

Every client request comes in this format:

```json
{
	"command": "<cmd name>",
	"request_id": "<any string here>",
	"data": {
		<command args here>
	}
}
```

`request_id` is a client-generated string that will be added to the response. This is so you can have multiple requests running without having to block for each of them. If you don't provide a request_id, the server will make one for you depending on the request.

### Server messages

Messages coming from the server can be of three types:

#### Hello

The Hello message is delivered as soon as a connection is established and contains the version of the protocol used by the server.

```json
{
  "type": "hello",
  "version": "v7"
}
```

#### Response

Responses are messages that are direct replies to client requests, they use this format:

```json
{
  "type": "response",
  "ok": true,
  "request_id": "<request_id from request>",
  "data": "<data, optional field>"
}
```

If a `request_id` field is not provided, the response `request_id` field will be added to the response with the text equivalent of the request the server is replying to, eg. `"{\"command\":\"kget\",\"data\":{\"key\":\"test\"}}"`. This is an older version of `request_id` that did not require client-generated IDs.

`data` is optional and may not be provided if the request does not expect it, eg. a "set key" request.

#### Push

A push is a server message that's triggered when someone else writes to a key you are subscribed to.

They follow this format:

```json
{
  "type": "push",
  "key": "<affected key>",
  "new_value": "<new value>"
}
```

#### Errors

If your request supplied invalid parameters or a server error was encountered, the server will return an error reponse instead of a normal response.

They follow this format:

```json
{
  "ok": false,
  "error": "<error code>",
  "details": "<text details>",
  "request_id": "<request_id from request>"
}
```

Check below for a list of all error codes.

## Authentication

As a websocket server, Kilovolt servers are accessible from any webpage you might visit and any process open in your computer. To protect from unauthorized access, Kilovolt supports setting an optional password and making client go through an authentication phase before any command can be called (except for informative ones like `version`).

Authentication is performed in two steps:

- The client sends a `klogin`, asking for authentication. The server replies with a challenge and a salt.
- The client encodes the HMAC-SHA256 of the challenge using the password and the salt, and sends it back using the `kauth` command.

Both challenge, salt and resulting hash are encoded using base64.

This is what the flow should look like:

```
Client: klogin {}
Server: response { challenge: "MC45NDU0NTU2MDk3ODI2OTU1", salt: "MTIyLjI5MzkzMzQ0MjczMDA3" }
Client: kauth { hash: base64_encode(HMAC-SHA256(base64_decode(challenge), (password + base64_decode(salt)))) }
Server: response { ok: true }
```

## Supported commands

### `version` - Get Kilovolt protocol version

Get a string representation of the Kilovolt server's supported protocol version

#### Example

Request

```json
{ "command": "version" }
```

Response

```json
{
  "type": "response",
  "ok": true,
  "data": "v4"
}
```

### `klogin` - Generate authentication challenge

Generates a challenge to authenticate clients.

Request

```json
{ "command": "klogin" }
```

Response

```json
{
  "type": "response",
  "ok": true,
  "data": {
    "challenge": "z9hUVNfu1rQJw1VWGYUjrkj2KCla2pI5YKVMqqQPZ1A=",
    "salt": "CFs4DalF5p4L0cxbhK3eQm8mFUmsqWJtY/paN/Df2ZU="
  }
}
```

### `kauth` - Submit authentication challenge

Submits the authentication challenge to the server and authenticates the client if correct. Refer to the [Authentication section](#authentication) for more information on how to calculate the authentication challenge.

Request

```json
{
  "command": "kauth",
  "data": { "hash": "NG3GPDGkR791t6SnPl0RV2Wj9msbkie3h7VmlKHY6mo=" }
}
```

Response

```json
{
  "type": "response",
  "ok": true
}
```

### `kget` - Get Key

Read key from database, this will return an empty string if the key is not in the database.

Required data:

| Parameter | Description |
| --------- | ----------- |
| key       | Key to read |

All values are string.

#### Example

Request

```json
{ "command": "kget", "data": { "key": "my-key" } }
```

Response

```json
{
  "type": "response",
  "ok": true,
  "data": "key value"
}
```

### `kget-bulk` - Get multiple keys

TODO

### `kget-all` - Get all keys with given prefix

TODO

### `kset` - Set key

#### Example

Request

```json
{ "command": "kset", "data": { "key": "my-key", "data": "key value" } }
```

Response

```json
{
  "type": "response",
  "ok": true
}
```

### `kset-bulk` - Set multiple keys

TODO

### `kdel` - Remove key

TODO

### `ksub` - Subscribe to key

Subscribe to key changes and receive pushes every time someone writes to it.

Required data:

| Parameter | Description         |
| --------- | ------------------- |
| key       | Key to subscribe to |

#### Example

Request

```json
{ "command": "ksub", "data": { "key": "my-key" } }
```

Response

```json
{
  "type": "response",
  "ok": true
}
```

Push (later on)

```json
{ "type": "push", "key": "my-key", "new_value": "changed value" }
```

### `kunsub` - Unsubscribe to key

Remove subscription to key changes.

Required data:

| Parameter | Description             |
| --------- | ----------------------- |
| key       | Key to unsubscribe from |

#### Example

Request

```json
{ "command": "kunsub", "data": { "key": "my-key" } }
```

Response

```json
{
  "type": "response",
  "ok": true
}
```

### `ksub-prefix` - Subscribe to prefix

Subscribe to changes of any key with a given prefix and receive pushes every time someone writes to them.

Required data:

| Parameter | Description            |
| --------- | ---------------------- |
| prefix    | Prefix to subscribe to |

#### Example

Request

```json
{ "command": "ksub-prefix", "data": { "prefix": "key" } }
```

Response

```json
{
  "type": "response",
  "ok": true
}
```

Push (later on)

```json
{ "type": "push", "key": "key-name", "new_value": "changed value" }
```

### `kunsub-prefix` - Unsubscribe from prefix

Remove subscription to prefix changes.

Required data:

| Parameter | Description                |
| --------- | -------------------------- |
| prefix    | Prefix to unsubscribe from |

#### Example

Request

```json
{ "command": "kunsub-prefix", "data": { "prefix": "key" } }
```

Response

```json
{
  "type": "response",
  "ok": true
}
```

### `klist` - Get list of keys (with optional prefix)

List keys with an optional given prefix.

Required data:

| Parameter | Description |
| --------- | ----------- |
| prefix    | Key prefix  |

#### Example

Request

```json
{ "command": "klist", "data": { "prefix": "key" } }
```

Response

```json
{
  "type": "response",
  "ok": true,
  "data": ["key1", "key2"]
}
```

## Internal commands

These commands are used in special occasions (like custom authentication systems). The schema for these commands can be quite unstable!


### `_uid` - Get internal client ID

Gets the internal client ID (e.g. for setting authentication status externally)

#### Example

Request

```json
{ "command": "_uid" }
```

Response

```json
{
  "type": "response",
  "ok": true,
  "data": "42"
}
```

## Error codes

These are all the possible error codes that can be returned, make sure to check the `details` field for more information when possible!

| Error code                       | Description                                                                |
| -------------------------------- | -------------------------------------------------------------------------- |
| `invalid message format`         | Request received was not valid JSON                                        |
| `required parameter missing`     | One or more required parameters were not supplied in the `data` dictionary |
| `server error`                   | The underlying database returned error                                     |
| `unknown command`                | Command in request is not supported                                        |
| "authentication not initialized" | Trying to solve a challenge that wasn't initiated                          |
| "authentication failed"          | Challenge is invalid                                                       |
| "authentication required"        | Trying to use a command without having authenticated first                 |
