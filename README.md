# Kilovolt

Websocket-based Key-value store, can use many databases on the backend. Has a slim set of features (get/set/pub/sub), mostly what's needed for [strimertul](https://git.sr.ht/~ashkeel/strimertul).

## Drivers

To use kilovolt, you will need a database driver.

Official drivers exist for the current databases:

| Database   | Driver module                         |
|------------|---------------------------------------|
| [Pebble]   | [strimertul/kilovolt-driver-pebble]   |
| [BadgerDB] | [strimertul/kv-badgerdb] (deprecated) |

[BadgerDB]: https://github.com/dgraph-io/badger
[strimertul/kv-badgerdb]: https://github.com/strimertul/kv-badgerdb
[Pebble]: https://github.com/cockroachdb/pebble
[strimertul/kilovolt-driver-pebble]: https://git.sr.ht/~ashkeel/kilovolt-driver-pebble 

If you have built a driver, feel free to submit a just send a patch request to [strimertul-devel](https://lists.sr.ht/~ashkeel/strimertul-devel) or [email me](mailto:ash@nebula.cafe) to have it added to this README!

### Go mod and git.sr.ht

Due to Google's [aggressive behavior and refusal to conform to standard internet "don't be a dick" code](https://drewdevault.com/2022/05/25/Google-has-been-DDoSing-sourcehut.html) (aka DDoSsing), you will need to bypass GOPROXY to be able to clone this repository, like this:

```sh
export GOPRIVATE=git.sr.ht
```

## Clients

We maintain a few libraries to interact with Kilovolt, you can find a list in the [wiki](https://man.sr.ht/~ashkeel/kilovolt/clients.md).

If you don't find one that suits you, just write one yourself, I promise it's really simple! See [PROTOCOL.md](PROTOCOL.md) for all you'll need to implement to make it work.

## License

Most of the code here is based on [Gorilla Websocket's chat example](https://github.com/gorilla/websocket/tree/master/examples/chat), which is licensed under [BSD-2-Clause](LICENSE-gorilla) (see `LICENSE-gorilla`).

The entire project is licensed under [ISC](LICENSE) (see `LICENSE`).
