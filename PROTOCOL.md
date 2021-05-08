# Kilovolt Protocol

Kilovolt exposes a WebSocket server and speaks using text JSON messages.

**Note:** This documentation pertains to Kilovolt protocol version `v2`!

## Message format

### Client messages

Every client request comes in this format:

```json
{
	"command": "<cmd name>",
	"data": {
		<command args here>
	}
}
```

### Server messages

Messages coming from the server can be of three types:

#### Response

Responses are messages that are direct replies to client requests, they use this format:

```json
{
  "type": "response",
  "ok": true,
  "cmd": "<client request message as text>",
  "data": "<data, optional field>"
}
```

`cmd` is the text equivalent of the request the server is replying to, eg. `"{\"command\":\"kget\",\"data\":{\"key\":\"test\"}}"`. This is so you can have multiple requests running without having to block for each of them.

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
  "cmd": "<client request message as text>"
}
```

Check below for a list of all error codes.

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
  "cmd": "{\"command\":\"version\"}",
  "data": "v2"
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
  "cmd": "{\"command\":\"kget\",\"data\":{\"key\":\"my-key\"}}",
  "data": "key value"
}
```

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
  "ok": true,
  "cmd": "{\"command\":\"kset\",\"data\":{\"key\":\"my-key\",\"data\":\"key value\"}}"
}
```

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
  "ok": true,
  "cmd": "{\"command\":\"ksub\",\"data\":{\"key\":\"my-key\"}}"
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
  "ok": true,
  "cmd": "{\"command\":\"kunsub\",\"data\":{\"key\":\"my-key\"}}"
}
```

## Error codes

These are all the possible error codes that can be returned, make sure to check the `details` field for more information when possible!

| Error code                   | Description                                                                |
| ---------------------------- | -------------------------------------------------------------------------- |
| `invalid message format`     | Request received was not valid JSON                                        |
| `required parameter missing` | One or more required parameters were not supplied in the `data` dictionary |
| `server update failed`       | The underlying database returned error upon update                         |
| `unknown command`            | Command in request is not supported                                        |
