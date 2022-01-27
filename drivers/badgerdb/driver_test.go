package badgerdb

import (
	"testing"

	"github.com/dgraph-io/badger/v3"
)

func TestNewBadgerBackend(t *testing.T) {
	withDB(t, func(db *badger.DB) {
		_ = NewBadgerBackend(db)
	})
}

func TestBackend_Get(t *testing.T) {
	withDB(t, func(db *badger.DB) {
		err := db.Update(func(txn *badger.Txn) error {
			return txn.Set([]byte("key"), []byte("value"))
		})
		if err != nil {
			t.Fatal(err)
		}
		backend := NewBadgerBackend(db)
		val, err := backend.Get("key")
		if err != nil {
			t.Fatal(err)
		}
		if val != "value" {
			t.Error("Expected value to be 'value'")
		}
	})
}

func TestBackend_Set(t *testing.T) {
	withDB(t, func(db *badger.DB) {
		backend := NewBadgerBackend(db)
		err := backend.Set("key", "value")
		if err != nil {
			t.Fatal(err)
		}
		err = db.View(func(txn *badger.Txn) error {
			val, err := txn.Get([]byte("key"))
			if err != nil {
				return err
			}
			byt, err := val.ValueCopy(nil)
			if err != nil {
				return err
			}
			if string(byt) != "value" {
				t.Error("Expected value to be 'value'")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestBackend_GetBulk(t *testing.T) {
	kv := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	withDB(t, func(db *badger.DB) {
		err := db.Update(func(txn *badger.Txn) error {
			for k, v := range kv {
				err := txn.Set([]byte(k), []byte(v))
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		backend := NewBadgerBackend(db)
		val, err := backend.GetBulk([]string{"key1", "key2"})
		if err != nil {
			t.Fatal(err)
		}
		if val["key1"] != kv["key1"] || val["key2"] != kv["key2"] {
			t.Error("Expected values do not match")
		}
	})
}

func TestBackend_SetBulk(t *testing.T) {
	kv := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	withDB(t, func(db *badger.DB) {
		backend := NewBadgerBackend(db)
		err := backend.SetBulk(kv)
		if err != nil {
			t.Fatal(err)
		}
		err = db.View(func(txn *badger.Txn) error {
			for k, v := range kv {
				val, err := txn.Get([]byte(k))
				if err != nil {
					return err
				}
				byt, err := val.ValueCopy(nil)
				if err != nil {
					return err
				}
				if string(byt) != v {
					t.Errorf("Expected value for '%s' to be '%s'", k, v)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestBackend_GetPrefix(t *testing.T) {
	kv := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	withDB(t, func(db *badger.DB) {
		err := db.Update(func(txn *badger.Txn) error {
			for k, v := range kv {
				err := txn.Set([]byte(k), []byte(v))
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		backend := NewBadgerBackend(db)
		val, err := backend.GetPrefix("key")
		if err != nil {
			t.Fatal(err)
		}
		if len(val) != 2 || val["key1"] != kv["key1"] || val["key2"] != kv["key2"] {
			t.Error("Expected values do not match")
		}
	})
}

func TestBackend_Delete(t *testing.T) {
	withDB(t, func(db *badger.DB) {
		err := db.Update(func(txn *badger.Txn) error {
			return txn.Set([]byte("key"), []byte("value"))
		})
		if err != nil {
			t.Fatal(err)
		}
		backend := NewBadgerBackend(db)
		err = backend.Delete("key")
		if err != nil {
			t.Fatal(err)
		}
		err = db.View(func(txn *badger.Txn) error {
			_, err := txn.Get([]byte("key"))
			if err != nil {
				if err == badger.ErrKeyNotFound {
					return nil
				}
				return err
			}
			t.Error("Key not deleted")
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestBackend_List(t *testing.T) {
	kv := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	withDB(t, func(db *badger.DB) {
		err := db.Update(func(txn *badger.Txn) error {
			for k, v := range kv {
				err := txn.Set([]byte(k), []byte(v))
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		backend := NewBadgerBackend(db)
		val, err := backend.List("key")
		if err != nil {
			t.Fatal(err)
		}
		if len(val) != 2 {
			t.Error("Expected 2 keys")
		}
	})
}

func withDB(t *testing.T, test func(db *badger.DB)) {
	// Open in-memory DB
	options := badger.DefaultOptions("").WithInMemory(true)
	db, err := badger.Open(options)
	if err != nil {
		t.Fatal("db initialization failed", err.Error())
	}
	defer db.Close()

	test(db)
}
