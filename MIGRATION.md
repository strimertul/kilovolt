# Migration notes

## v4

- `cmd` has been removed from responses, please use `request_id` for correlating responses to requests
- If you are using the APIs, `RemapKeyFn` is now a string variable `namespace` as other methods of remapping keys are currently not supported

## v7

- API changed significantly, support for multiple DB backends.
- Logging library changed to logrus to zap.

## v10

- Calling `klogin` when authentication is not required will now return a `authentication not required` error