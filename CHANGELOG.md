# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [9.1.0] - 2023-02-13

### Fixed

- Removed two buffered channels from local client that would fill up and block the read loop

## [9.0.1] - 2022-11-18

### Changed

- "_uid" now returns the ID as a string to prevent rounding errors in JSON parsers

## [9.0.0] - 2022-11-18

New Protocol version (`v9`)

### Added

- Added utility internal function `_uid`

## [8.0.5] - 2022-03-25

### Changed

- `LocalClient` subscriptions callbacks are now called in a separate goroutine to avoid deadlocks when using the client inside the callback.

## [8.0.3] - 2022-02-09

### Changed

- Expose `Hub.CreateWebsocketClient`
- Client list now uses a RWMutex for concurrent read accesses

## [8.0.2] - 2022-02-01

### Changed

- Moved "received" logging from notice to debug level

## [8.0.1] - 2022-02-01

### Changed

- Added `SetAuthenticated` method for skipping authentication

## [8.0.0] - 2022-01-31

New protocol version (`v8`)

### Added

- Added `kdel` to delete keys

[current]: https://github.com/strimertul/kilovolt/compare/v9.0.1...HEAD
[9.0.1]: https://github.com/strimertul/kilovolt/compare/v9.0.0...v9.0.1
[9.0.0]: https://github.com/strimertul/kilovolt/compare/v8.0.5...v9.0.0
[8.0.5]: https://github.com/strimertul/kilovolt/compare/v8.0.4...v8.0.5
[8.0.4]: https://github.com/strimertul/kilovolt/compare/v8.0.3...v8.0.4
[8.0.3]: https://github.com/strimertul/kilovolt/compare/v8.0.2...v8.0.3
[8.0.2]: https://github.com/strimertul/kilovolt/compare/v8.0.1...v8.0.2
[8.0.1]: https://github.com/strimertul/kilovolt/compare/v8.0.0...v8.0.1
[8.0.0]: https://github.com/strimertul/kilovolt/compare/v7.2.4...v8.0.0
