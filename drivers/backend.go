package drivers

import "errors"

var (
	ErrorKeyNotFound = errors.New("key not found")
)

type Backend interface {
	Get(key string) (string, error)
	GetBulk(keys []string) (map[string]string, error)
	GetPrefix(prefix string) (map[string]string, error)
	Set(key string, value string) error
	SetBulk(kv map[string]string) error
	Delete(key string) error
	List(prefix string) ([]string, error)
}
