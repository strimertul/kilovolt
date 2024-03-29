# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## 11.0.1 - 2023-11-03

### Fixed

- Ping deadlines are not coded right and dropped all connections after a minute, this has been temporarily turned off

## 11.0.0 - 2023-11-03

### Changed

- Changed websocket library to nhooyr.io/websocket, should be faster and better maintained
- Closing a hub will now stop any goroutine managing connections

## 10.0.0 - 2023-04-18

### Added

- New authentication method: interactive/"ask", check PROTOCOL.md for an overview.

### Changed

- Calling `klogin` when authentication is not required will now return a `authentication not required` error

## 9.1.0 - 2023-02-13

### Fixed

- Removed two buffered channels from local client that would fill up and block the read loop

## 9.0.1 - 2022-11-18

### Changed

- "_uid" now returns the ID as a string to prevent rounding errors in JSON parsers

## 9.0.0 - 2022-11-18

New Protocol version (`v9`)

### Added

- Added utility internal function `_uid`

## 8.0.5 - 2022-03-25

### Changed

- `LocalClient` subscriptions callbacks are now called in a separate goroutine to avoid deadlocks when using the client inside the callback.

## 8.0.3 - 2022-02-09

### Changed

- Expose `Hub.CreateWebsocketClient`
- Client list now uses a RWMutex for concurrent read accesses

## 8.0.2 - 2022-02-01

### Changed

- Moved "received" logging from notice to debug level

## 8.0.1 - 2022-02-01

### Changed

- Added `SetAuthenticated` method for skipping authentication

## 8.0.0 - 2022-01-31

New protocol version (`v8`)

### Added

- Added `kdel` to delete keys
