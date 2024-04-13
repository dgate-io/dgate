package storage

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
	"github.com/rs/zerolog"
)

type FileStoreConfig struct {
	Directory string `koanf:"dir"`
	Logger    zerolog.Logger

	inMemory bool
}

type FileStore struct {
	directory string
	inMemory  bool
	logger    badger.Logger
	db        *badger.DB
}

type FileStoreTxn struct {
	txn *badger.Txn
	ro  bool
}

var _ Storage = &FileStore{}
var _ StorageTxn = &FileStoreTxn{}

var (
	// ErrStoreLocked is returned when the storage is locked.
	ErrStoreLocked error = errors.New("storage is locked")
	// ErrKeyNotFound is returned when the key is not found.
	ErrKeyNotFound error = errors.New("key not found")
	// ErrTxnReadOnly is returned when the transaction is read only.
	ErrTxnReadOnly error = errors.New("transaction is read only")
)

func NewFileStore(fsConfig *FileStoreConfig) *FileStore {
	if fsConfig == nil {
		fsConfig = &FileStoreConfig{}
	}
	if fsConfig.Directory == "" {
		panic("directory is required")
	} else {
		// Remove trailing slash if it exists.
		fsConfig.Directory = strings.TrimSuffix(fsConfig.Directory, "/")
	}

	logger := fsConfig.Logger.Hook(zerolog.HookFunc(func(e *zerolog.Event, level zerolog.Level, msg string) {
		e.Str("storage", "filestore::badger")
	}))

	return &FileStore{
		directory: fsConfig.Directory,
		logger: newBadgerLoggerAdapter(
			"filestore::badger",
			logger.Level(zerolog.InfoLevel),
		),
		inMemory: fsConfig.inMemory,
	}
}

func newFileStoreTxn(txn *badger.Txn) *FileStoreTxn {
	return &FileStoreTxn{
		txn: txn,
	}
}

func (s *FileStore) Connect() error {
	var opts badger.Options
	var err error
	if s.inMemory {
		opts = badger.DefaultOptions("").
			WithCompression(options.Snappy).
			WithInMemory(true).
			WithLogger(s.logger)
	} else {
		// Create the directory if it does not exist.
		if _, err := os.Stat(s.directory); os.IsNotExist(err) {
			err := os.MkdirAll(s.directory, 0755)
			if err != nil {
				return errors.New("failed to create directory - " + s.directory + ":  " + err.Error())
			}
		}

		opts = badger.DefaultOptions(s.directory).
			WithReadOnly(false).
			WithInMemory(s.inMemory).
			WithCompression(options.Snappy).
			WithLogger(s.logger)
	}
	s.db, err = badger.Open(opts)
	if err != nil {
		return err
	}
	return nil
}

func (s *FileStore) Get(key string) ([]byte, error) {
	var value []byte
	err := s.db.View(func(txn *badger.Txn) error {
		val, err := newFileStoreTxn(txn).Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrKeyNotFound
			}
			return err
		}
		value = val
		return nil
	})
	return value, err
}

func (s *FileStore) Set(key string, value []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return newFileStoreTxn(txn).Set(key, value)
	})
}

func (s *FileStore) IterateValuesPrefix(prefix string, fn func(string, []byte) error) error {
	return s.db.View(func(txn *badger.Txn) error {
		return newFileStoreTxn(txn).IterateValuesPrefix(prefix, fn)
	})
}

func (s *FileStore) IterateTxnPrefix(prefix string, fn func(StorageTxn, string) error) error {
	return s.db.View(func(txn *badger.Txn) error {
		return newFileStoreTxn(txn).IterateTxnPrefix(prefix, fn)
	})
}

func (s *FileStore) GetPrefix(prefix string, offset, limit int) ([]*KeyValue, error) {
	var return_list []*KeyValue
	err := s.db.View(func(txn *badger.Txn) error {
		val, err := newFileStoreTxn(txn).GetPrefix(prefix, offset, limit)
		if err != nil {
			return fmt.Errorf("failed to get prefix: %w", err)
		}
		return_list = val
		return nil
	})
	return return_list, err
}

func (s *FileStore) Delete(key string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

func (s *FileStore) Txn(readOnly bool, fn func(StorageTxn) error) error {
	txFunc := s.db.View
	if !readOnly {
		txFunc = s.db.Update
	}
	return txFunc(func(txn *badger.Txn) error {
		return fn(&FileStoreTxn{
			txn: txn,
			ro:  readOnly,
		})
	})
}

func (s *FileStore) Close() error {
	return s.db.Close()
}

func (tx *FileStoreTxn) Get(key string) ([]byte, error) {
	item, err := tx.txn.Get([]byte(key))
	if err != nil {
		return nil, err
	}
	val, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err
	}
	return val, nil
}

func (tx *FileStoreTxn) Set(key string, value []byte) error {
	if tx.ro {
		return ErrTxnReadOnly
	}
	return tx.txn.Set([]byte(key), value)
}

func (tx *FileStoreTxn) Delete(key string) error {
	if tx.ro {
		return ErrTxnReadOnly
	}
	return tx.txn.Delete([]byte(key))
}

func (tx *FileStoreTxn) IterateValuesPrefix(prefix string, fn func(string, []byte) error) error {
	iter := tx.txn.NewIterator(badger.IteratorOptions{
		Prefix: []byte(prefix),
	})
	defer iter.Close()
	for iter.Rewind(); iter.Valid(); iter.Next() {
		item := iter.Item()
		val, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		if err := fn(string(item.Key()), val); err != nil {
			return err
		}
	}
	return nil
}

func (tx *FileStoreTxn) IterateTxnPrefix(prefix string, fn func(StorageTxn, string) error) error {
	iter := tx.txn.NewIterator(badger.IteratorOptions{
		Prefix: []byte(prefix),
	})
	defer iter.Close()
	for iter.Rewind(); iter.Valid(); iter.Next() {
		item := iter.Item()
		if err := fn(tx, string(item.Key())); err != nil {
			return err
		}
	}
	return nil
}

func (s *FileStoreTxn) GetPrefix(prefix string, offset, limit int) ([]*KeyValue, error) {
	return_list := make([]*KeyValue, 0)
	iter := s.txn.NewIterator(badger.IteratorOptions{
		Prefix: []byte(prefix),
	})
	defer iter.Close()
	for iter.Rewind(); iter.Valid(); iter.Next() {
		if offset > 0 {
			offset -= 1
			continue
		}
		item := iter.Item()
		val, err := item.ValueCopy(nil)
		if err != nil {
			return nil, fmt.Errorf("error copying value: %v", err)
		}
		return_list = append(return_list, &KeyValue{
			Key:   string(item.Key()),
			Value: val,
		})
		if limit -= 1; limit == 0 {
			break
		}
	}

	return return_list, nil
}
