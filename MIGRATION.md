# Migration notes

## v3 to v4

- `cmd` has been removed from responses, you can still make requests without a `request_id` but you will get whatever value `cmd` would have had in the `request_id` field instead.
