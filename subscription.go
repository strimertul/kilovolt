package kv

import (
	"bytes"
	"encoding/gob"
	"strconv"

	"github.com/dgraph-io/badger/v3"
)

const subscriptionDBPrefix = "subscribers/"
const subscriptionKeyDBPrefix = subscriptionDBPrefix + "key/"
const subscriptionPrefixDBPrefix = subscriptionDBPrefix + "prefix/"
const clientSubscriptionsPrefix = "client-subscriptions/"

func dbSubscribeToKey(db *badger.DB, client Client, key string) error {
	return db.Update(func(tx *badger.Txn) error {
		// Get subscribers for key
		fullkey := []byte(subscriptionKeyDBPrefix + key)
		keylist, err := txGetSubscriberList(tx, fullkey)
		if err != nil {
			return err
		}

		// Add UID to list and save it to DB
		keylist = append(keylist, client.UID())
		return txSetGob(tx, fullkey, keylist)
	})
}

func dbUnsubscribeFromKey(db *badger.DB, client Client, key string) error {
	return db.Update(func(tx *badger.Txn) error {
		// Get subscribers for key
		fullkey := []byte(subscriptionKeyDBPrefix + key)
		keylist, err := txGetSubscriberList(tx, fullkey)
		if err != nil {
			return err
		}

		// Search and remove client ID from list
		keylist = removeFromList(keylist, client.UID())
		return txSetGob(tx, fullkey, keylist)
	})
}

func dbSubscribeToPrefix(db *badger.DB, client Client, prefix string) error {
	return db.Update(func(tx *badger.Txn) error {
		// Get subscribers for key
		var keylist []int64
		fullkey := []byte(subscriptionKeyDBPrefix + prefix)
		keylist, err := txGetSubscriberList(tx, fullkey)
		if err != nil {
			return err
		}

		// Add UID to list and save it to DB
		keylist = append(keylist, client.UID())
		return txSetGob(tx, fullkey, keylist)
	})
}

func dbUnsubscribeFromPrefix(db *badger.DB, client Client, prefix string) error {
	// Prepare key and UID
	uid := client.UID()
	fullkey := []byte(subscriptionKeyDBPrefix + prefix)

	return db.Update(func(tx *badger.Txn) error {
		// Remove subscription from client's own list
		subscriptions, err := txGetUserSubscription(tx, uid)
		if err != nil {
			return err
		}
		delete(subscriptions, string(fullkey))
		err = txSetUserSubscription(tx, uid, subscriptions)
		if err != nil {
			return err
		}

		// Get subscribers for key
		keylist, err := txGetSubscriberList(tx, fullkey)
		if err != nil {
			return err
		}

		// Search and remove client ID from list
		keylist = removeFromList(keylist, uid)
		return txSetGob(tx, fullkey, keylist)
	})
}

func dbUnsubscribeFromAll(db *badger.DB, client Client) error {
	// Get UID beforehand
	uid := client.UID()

	return db.Update(func(tx *badger.Txn) error {
		// Get subscriptions for client
		subscriptions, err := txGetUserSubscription(tx, uid)
		if err != nil {
			return err
		}
		if subscriptions == nil {
			return nil
		}

		// Remove from remaining subscriptions
		for key := range subscriptions {
			bkey := []byte(key)
			keylist, err := txGetSubscriberList(tx, bkey)
			if err != nil {
				return err
			}

			// Search and remove client ID from list
			newList := removeFromList(keylist, client.UID())
			err = txSetGob(tx, bkey, newList)
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
		keylist, err := txGetSubscriberList(tx, []byte(subscriptionKeyDBPrefix+string(key)))
		if err != nil {
			return err
		}
		subscribers = append(subscribers, keylist...)

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

func removeFromList(lst []int64, item int64) []int64 {
	for i, cid := range lst {
		if cid == item {
			return append(lst[:i], lst[i+1:]...)
		}
	}
	return lst
}

func txGetSubscriberList(tx *badger.Txn, key []byte) ([]int64, error) {
	var keylist []int64
	err := txGetGob(tx, key, &keylist)
	if err == badger.ErrKeyNotFound {
		return []int64{}, nil
	}
	return keylist, err
}

func txGetUserSubscription(tx *badger.Txn, uid int64) (map[string]bool, error) {
	var subscriptions map[string]bool

	key := []byte(clientSubscriptionsPrefix + strconv.FormatInt(uid, 10))
	err := txGetGob(tx, key, subscriptions)
	if err == badger.ErrKeyNotFound {
		return nil, nil
	}

	return subscriptions, err
}

func txSetUserSubscription(tx *badger.Txn, uid int64, subscriptions map[string]bool) error {
	key := []byte(clientSubscriptionsPrefix + strconv.FormatInt(uid, 10))
	return txSetGob(tx, key, subscriptions)
}

func txGetGob(tx *badger.Txn, key []byte, dst interface{}) error {
	sub, err := tx.Get(key)
	if err != nil {
		return err
	}
	byt, err := sub.ValueCopy(nil)
	if err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewBuffer(byt)).Decode(dst)
}

func txSetGob(tx *badger.Txn, key []byte, data interface{}) error {
	b := new(bytes.Buffer)
	err := gob.NewEncoder(b).Encode(data)
	if err != nil {
		return err
	}
	return tx.Set([]byte(key), b.Bytes())
}
