# Migration notes

## v3 to v4

- `cmd` has been removed from responses, please use `request_id` for correlating responses to requests
- If you are using the APIs, `RemapKeyFn` is now a string variable `namespace` as other methods of remapping keys are currently not supported

## v4 to v6

*No breaking changes*

## v6 to v7

- API changed significantly, support for multiple DB backends.
- Logging library changed to logrus to zap.

## v7 to v8

*No breaking changes*
