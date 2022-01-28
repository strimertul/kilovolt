package kv

import (
	"sort"
	"strings"
)

// mapkv is an in-memory map[string]string driver. Should not be used!
type mapkv struct {
	data map[string]string
}

func MakeBackend() *mapkv {
	return &mapkv{
		data: make(map[string]string),
	}
}

func (b *mapkv) Get(key string) (string, error) {
	val, ok := b.data[key]
	if !ok {
		return "", ErrorKeyNotFound
	}
	return val, nil
}

func (b *mapkv) Set(key string, value string) error {
	b.data[key] = value
	return nil
}

func (b *mapkv) Delete(key string) error {
	delete(b.data, key)
	return nil
}

func (b *mapkv) SetBulk(data map[string]string) error {
	for k, v := range data {
		b.data[k] = v
	}
	return nil
}

func (b *mapkv) GetBulk(keys []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, k := range keys {
		v, err := b.Get(k)
		if err != nil {
			result[k] = ""
		} else {
			result[k] = v
		}
	}
	return result, nil
}

func (b *mapkv) List(prefix string) ([]string, error) {
	keys := make([]string, 0, len(b.data))
	for k := range b.data {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	sort.Sort(sort.StringSlice(keys))
	return keys, nil
}

func (b *mapkv) GetPrefix(prefix string) (map[string]string, error) {
	result := make(map[string]string)
	for k, v := range b.data {
		if strings.HasPrefix(k, prefix) {
			result[k] = v
		}
	}
	return result, nil
}
