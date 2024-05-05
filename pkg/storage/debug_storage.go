package storage

import (
	"errors"
	"strings"

	"github.com/dgate-io/dgate/pkg/util/tree/avl"
	"github.com/rs/zerolog"
)

type DebugStoreConfig struct {
	// Path to the directory where the files will be stored.
	// If the directory does not exist, it will be created.
	// If the directory exists, it will be used.
	Logger zerolog.Logger
}

type DebugStore struct {
	tree avl.Tree[string, []byte]
}

var _ Storage = &DebugStore{}

func NewDebugStore(cfg *DebugStoreConfig) *DebugStore {
	return &DebugStore{
		tree: avl.NewTree[string, []byte](),
	}
}

func (m *DebugStore) Connect() error {
	return nil
}

func (m *DebugStore) Get(key string) ([]byte, error) {
	if b, ok := m.tree.Find(key); ok {
		return b, nil
	}
	return nil, errors.New("key not found")
}

func (m *DebugStore) Set(key string, value []byte) error {
	m.tree.Insert(key, value)
	return nil
}

func (m *DebugStore) IterateValuesPrefix(prefix string, fn func(string, []byte) error) error {
	check := true
	m.tree.Each(func(k string, v []byte) bool {
		if strings.HasPrefix(k, prefix) {
			check = true
			if err := fn(k, v); err != nil {
				return false
			}
			return true
		}
		return check
	})
	return nil
}

func (m *DebugStore) IterateTxnPrefix(prefix string, fn func(StorageTxn, string) error) error {
	panic("implement me")
}

func (m *DebugStore) GetPrefix(prefix string, offset, limit int) ([]*KeyValue, error) {
	kvs := make([]*KeyValue, 0, limit)
	m.IterateValuesPrefix(prefix, func(key string, value []byte) error {
		if offset <= 0 {
			kvs = append(kvs, &KeyValue{
				Key:   key,
				Value: value,
			})
			if len(kvs) >= limit {
				return errors.New("limit reached")
			}
		} else {
			offset--
		}
		return nil
	})
	return kvs, nil
}

func (m *DebugStore) Delete(key string) error {
	if ok := m.tree.Delete(key); !ok {
		return errors.New("key not found")
	}
	return nil
}

func (m *DebugStore) Close() error {
	return nil
}
