package mapkv

import (
	"strings"

	"github.com/strimertul/kilovolt/v7/drivers"
)

// Backend is an in-memory map[string]string driver. Should not be used!
type Backend struct {
	data map[string]string
}

func MakeBackend() *Backend {
	return &Backend{
		data: make(map[string]string),
	}
}

func (b *Backend) Get(key string) (string, error) {
	val, ok := b.data[key]
	if !ok {
		return "", drivers.ErrorKeyNotFound
	}
	return val, nil
}

func (b *Backend) Set(key string, value string) error {
	b.data[key] = value
	return nil
}

func (b *Backend) Delete(key string) error {
	delete(b.data, key)
	return nil
}

func (b *Backend) SetBulk(data map[string]string) error {
	for k, v := range data {
		b.data[k] = v
	}
	return nil
}

func (b *Backend) GetBulk(keys []string) (map[string]string, error) {
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

func (b *Backend) List(prefix string) ([]string, error) {
	keys := make([]string, 0, len(b.data))
	for k := range b.data {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

func (b *Backend) GetPrefix(prefix string) (map[string]string, error) {
	result := make(map[string]string)
	for k, v := range b.data {
		if strings.HasPrefix(k, prefix) {
			result[k] = v
		}
	}
	return result, nil
}
