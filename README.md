# Kilovolt

Websocket-based Key-value store, can use many databases on the backend (currently BBolt/BadgerDB). Has a slim set of features (get/set/pub/sub), mostly what's needed for [strimertul](https://github.com/strimertul/strimertul) and [stulbe](https://github.com/strimertul/stulbe/)

## Drivers

To use kilovolt, you will need a database driver.

Official drivers exist for the current databases:

| Database | Driver module                                                       |
|----------|---------------------------------------------------------------------|
| BadgerDB | [strimertul/kv-badgerdb](https://github.com/strimertul/kv-badgerdb) |

If you have built a driver, feel free to submit a pull request to have it added to this README!

## Clients

We maintain a few libraries to interact with Kilovolt at [strimertul/kilovolt-clients](https://github.com/strimertul/kilovolt-clients).

If you don't find one that suits you, just write one yourself, I promise it's really simple! See [PROTOCOL.md](PROTOCOL.md) for all you'll need to implement to make it work.

## License

Most of the code here is based on [Gorilla Websocket's chat example](https://github.com/gorilla/websocket/tree/master/examples/chat), which is licensed under [BSD-2-Clause](LICENSE-gorilla) (see `LICENSE-gorilla`).

The entire project is licensed under [ISC](LICENSE) (see `LICENSE`).
