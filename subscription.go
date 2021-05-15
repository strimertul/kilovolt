package kv

import (
	"bytes"
	"encoding/gob"

	"github.com/dgraph-io/badger/v3"
)

const subscriptionDBPrefix = "subscribers/"
const subscriptionKeyDBPrefix = subscriptionDBPrefix + "key/"
const subscriptionPrefixDBPrefix = subscriptionDBPrefix + "prefix/"

func dbSubscribeToKey(db *badger.DB, client *Client, key string) error {
	return db.Update(func(tx *badger.Txn) error {
		// Get subscribers for key
		var keylist []int64
		key := []byte(subscriptionKeyDBPrefix + key)
		list, err := tx.Get(key)
		if err != nil {
			if err != badger.ErrKeyNotFound {
				return err
			}
		} else {
			byt, err := list.ValueCopy(nil)
			if err != nil {
				return err
			}
			err = gob.NewDecoder(bytes.NewBuffer(byt)).Decode(&keylist)
			if err != nil {
				return err
			}
		}
		keylist = append(keylist, client.uid)
		b := new(bytes.Buffer)
		err = gob.NewEncoder(b).Encode(keylist)
		if err != nil {
			return err
		}
		return tx.Set(key, b.Bytes())
	})
}

func dbUnsubscribeFromKey(db *badger.DB, client *Client, key string) error {
	return db.Update(func(tx *badger.Txn) error {
		// Get subscribers for key
		var keylist []int64
		fullkey := []byte(subscriptionKeyDBPrefix + key)
		list, err := tx.Get(fullkey)
		if err != nil {
			if err != badger.ErrKeyNotFound {
				return err
			}
		} else {
			byt, err := list.ValueCopy(nil)
			if err != nil {
				return err
			}
			err = gob.NewDecoder(bytes.NewBuffer(byt)).Decode(&keylist)
			if err != nil {
				return err
			}
		}
		// Search and remove client ID from list
		newList := keylist
		for i, cid := range keylist {
			if cid == client.uid {
				newList = append(keylist[:i], keylist[i+1:]...)
			}
		}
		b := new(bytes.Buffer)
		err = gob.NewEncoder(b).Encode(newList)
		if err != nil {
			return err
		}
		return tx.Set(fullkey, b.Bytes())
	})
}

func dbSubscribeToPrefix(db *badger.DB, client *Client, prefix string) error {
	return db.Update(func(tx *badger.Txn) error {
		// Get subscribers for key
		var keylist []int64
		fullkey := []byte(subscriptionKeyDBPrefix + prefix)
		list, err := tx.Get(fullkey)
		if err != nil {
			if err != badger.ErrKeyNotFound {
				return err
			}
		} else {
			byt, err := list.ValueCopy(nil)
			if err != nil {
				return err
			}
			err = gob.NewDecoder(bytes.NewBuffer(byt)).Decode(&keylist)
			if err != nil {
				return err
			}
		}
		keylist = append(keylist, client.uid)
		b := new(bytes.Buffer)
		err = gob.NewEncoder(b).Encode(keylist)
		if err != nil {
			return err
		}
		return tx.Set(fullkey, b.Bytes())
	})
}

func dbUnsubscribeFromPrefix(db *badger.DB, client *Client, prefix string) error {
	return db.Update(func(tx *badger.Txn) error {
		// Get subscribers for key
		var keylist []int64
		fullkey := []byte(subscriptionKeyDBPrefix + prefix)
		list, err := tx.Get(fullkey)
		if err != nil {
			if err != badger.ErrKeyNotFound {
				return err
			}
		} else {
			byt, err := list.ValueCopy(nil)
			if err != nil {
				return err
			}
			err = gob.NewDecoder(bytes.NewBuffer(byt)).Decode(&keylist)
			if err != nil {
				return err
			}
		}
		// Search and remove client ID from list
		newList := keylist
		for i, cid := range keylist {
			if cid == client.uid {
				newList = append(keylist[:i], keylist[i+1:]...)
			}
		}
		b := new(bytes.Buffer)
		err = gob.NewEncoder(b).Encode(newList)
		if err != nil {
			return err
		}
		return tx.Set(fullkey, b.Bytes())
	})
}

func dbUnsubscribeFromAll(db *badger.DB, client *Client) error {
	return db.Update(func(tx *badger.Txn) error {
		// Remove from remaining subscriptions
		for key := range client.subscriptions {
			var keylist []int64
			list, err := tx.Get([]byte(key))
			if err != nil {
				if err != badger.ErrKeyNotFound {
					return err
				}
			} else {
				byt, err := list.ValueCopy(nil)
				if err != nil {
					return err
				}
				err = gob.NewDecoder(bytes.NewBuffer(byt)).Decode(&keylist)
				if err != nil {
					return err
				}
			}
			// Search and remove client ID from list
			newList := keylist
			for i, cid := range keylist {
				if cid == client.uid {
					newList = append(keylist[:i], keylist[i+1:]...)
				}
			}
			b := new(bytes.Buffer)
			err = gob.NewEncoder(b).Encode(newList)
			if err != nil {
				return err
			}
			err = tx.Set([]byte(key), b.Bytes())
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func dbGetSubscribersForKey(db *badger.DB, key []byte) ([]int64, error) {
	subscribers := []int64{}
	err := db.View(func(tx *badger.Txn) error {
		// Get subscribers for key
		list, err := tx.Get([]byte(subscriptionKeyDBPrefix + string(key)))
		if err != nil {
			if err != badger.ErrKeyNotFound {
				return err
			}
		} else {
			byt, err := list.ValueCopy(nil)
			if err != nil {
				return err
			}
			keylist := []int64{}
			err = gob.NewDecoder(bytes.NewBuffer(byt)).Decode(&keylist)
			if err != nil {
				return err
			}
			subscribers = append(subscribers, keylist...)
		}

		// Get subscribers for prefix
		opt := badger.DefaultIteratorOptions
		opt.Prefix = []byte(subscriptionPrefixDBPrefix)
		it := tx.NewIterator(opt)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			// Check if key is valid prefix
			if bytes.HasPrefix(key, it.Item().Key()[len(opt.Prefix):]) {
				byt, err := item.ValueCopy(nil)
				if err != nil {
					return err
				}
				keylist := []int64{}
				err = gob.NewDecoder(bytes.NewBuffer(byt)).Decode(&keylist)
				if err != nil {
					return err
				}
				subscribers = append(subscribers, keylist...)
			}
		}
		return nil
	})

	return subscribers, err
}
