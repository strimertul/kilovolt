package mapkv

import (
	"testing"
)

func TestNewBackend(t *testing.T) {
	MakeBackend()
}

func TestBackend_Get(t *testing.T) {
	db := MakeBackend()
	db.data["key"] = "value"

	val, err := db.Get("key")
	if err != nil {
		t.Fatal(err)
	}
	if val != "value" {
		t.Error("Expected value to be 'value'")
	}
}

func TestBackend_Set(t *testing.T) {
	db := MakeBackend()

	err := db.Set("key", "value")
	if err != nil {
		t.Fatal(err)
	}

	if val, ok := db.data["key"]; !ok || val != "value" {
		t.Fatal("key not found or value not set correctly")
	}
}

func TestBackend_GetBulk(t *testing.T) {
	db := MakeBackend()
	db.data["key1"] = "value1"
	db.data["key2"] = "value2"

	val, err := db.GetBulk([]string{"key1", "key2"})
	if err != nil {
		t.Fatal(err)
	}
	if val["key1"] != db.data["key1"] || val["key2"] != db.data["key2"] {
		t.Error("Expected values do not match")
	}
}

func TestBackend_SetBulk(t *testing.T) {
	db := MakeBackend()
	kv := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	err := db.SetBulk(kv)
	if err != nil {
		t.Fatal(err)
	}
	if val, ok := db.data["key1"]; !ok || val != "value1" {
		t.Fatal("key1 not found or value not set correctly")
	}
	if val, ok := db.data["key2"]; !ok || val != "value2" {
		t.Fatal("key2 not found or value not set correctly")
	}
}

func TestBackend_GetPrefix(t *testing.T) {
	db := MakeBackend()
	db.data["key1"] = "value1"
	db.data["key2"] = "value2"

	val, err := db.GetPrefix("key")
	if err != nil {
		t.Fatal(err)
	}
	if len(val) != 2 || val["key1"] != db.data["key1"] || val["key2"] != db.data["key2"] {
		t.Error("Expected values do not match")
	}
}

func TestBackend_Delete(t *testing.T) {
	db := MakeBackend()
	db.data["key"] = "value"

	err := db.Delete("key")
	if err != nil {
		t.Fatal(err)
	}

	_, ok := db.data["key"]
	if ok {
		t.Fatal("key not deleted")
	}
}

func TestBackend_List(t *testing.T) {
	db := MakeBackend()
	db.data["key1"] = "value1"
	db.data["key2"] = "value2"

	val, err := db.List("key")
	if err != nil {
		t.Fatal(err)
	}
	if len(val) != 2 {
		t.Error("Expected 2 keys")
	}
}
