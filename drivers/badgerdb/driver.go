package badgerdb

import (
	"github.com/strimertul/kilovolt/v7/drivers"

	"github.com/dgraph-io/badger/v3"
)

type Backend struct {
	db *badger.DB
}

func NewBadgerBackend(db *badger.DB) Backend {
	return Backend{db}
}

func (b Backend) Get(key string) (string, error) {
	var out string
	err := b.db.View(func(tx *badger.Txn) error {
		val, err := tx.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return drivers.ErrorKeyNotFound
			}
			return err
		}
		byt, err := val.ValueCopy(nil)
		if err != nil {
			return err
		}
		out = string(byt)
		return nil
	})
	return out, err
}

func (b Backend) GetBulk(keys []string) (map[string]string, error) {
	out := make(map[string]string)
	err := b.db.View(func(tx *badger.Txn) error {
		for _, key := range keys {
			val, err := tx.Get([]byte(key))
			if err != nil {
				if err == badger.ErrKeyNotFound {
					out[key] = ""
					continue
				}
				return err
			}
			byt, err := val.ValueCopy(nil)
			if err != nil {
				return err
			}
			out[key] = string(byt)
		}
		return nil
	})
	return out, err
}

func (b Backend) GetPrefix(prefix string) (map[string]string, error) {
	out := make(map[string]string)
	err := b.db.View(func(tx *badger.Txn) error {
		opt := badger.DefaultIteratorOptions
		opt.Prefix = []byte(prefix)
		it := tx.NewIterator(opt)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			byt, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			out[string(item.Key())] = string(byt)
		}
		return nil
	})
	return out, err
}

func (b Backend) Set(key, value string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return tx.Set([]byte(key), []byte(value))
	})
}

func (b Backend) SetBulk(kv map[string]string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		for k, v := range kv {
			err := tx.Set([]byte(k), []byte(v))
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (b Backend) Delete(key string) error {
	return b.db.Update(func(tx *badger.Txn) error {
		return tx.Delete([]byte(key))
	})
}

func (b Backend) List(prefix string) ([]string, error) {
	var out []string
	err := b.db.View(func(tx *badger.Txn) error {
		opt := badger.DefaultIteratorOptions
		opt.Prefix = []byte(prefix)
		it := tx.NewIterator(opt)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			out = append(out, string(item.Key()))
		}
		return nil
	})
	return out, err
}
