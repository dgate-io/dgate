package storage

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"
)

type FileStoreConfig struct {
	Directory string `koanf:"dir"`
	Logger    *zap.Logger
}

type FileStore struct {
	logger     *zap.Logger
	bucketName []byte
	directory  string
	db         *bolt.DB
}

type FileStoreTxn struct {
	txn    *bolt.Tx
	ro     bool
	bucket *bolt.Bucket
}

var _ Storage = (*FileStore)(nil)
var _ StorageTxn = (*FileStoreTxn)(nil)

var (
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

	if fsConfig.Logger == nil {
		fsConfig.Logger = zap.NewNop()
	}

	return &FileStore{
		directory:  fsConfig.Directory,
		logger:     fsConfig.Logger.Named("boltstore::bolt"),
		bucketName: []byte("dgate"),
	}
}

func (s *FileStore) Connect() (err error) {
	if err = os.MkdirAll(s.directory, 0755); err != nil {
		return err
	}
	filePath := path.Join(s.directory, "dgate.db")
	if s.db, err = bolt.Open(filePath, 0755, nil); err != nil {
		return err
	} else if tx, err := s.db.Begin(true); err != nil {
		return err
	} else {
		_, err = tx.CreateBucketIfNotExists(s.bucketName)
		if err != nil {
			return err
		}
		return tx.Commit()
	}
}

func (s *FileStore) Txn(write bool, fn func(StorageTxn) error) error {
	txFn := func(txn *bolt.Tx) (err error) {
		return fn(s.newTxn(txn))
	}
	if write {
		return s.db.Update(txFn)
	}
	return s.db.View(txFn)
}

func (s *FileStore) newTxn(txn *bolt.Tx) *FileStoreTxn {
	if bucket := txn.Bucket(s.bucketName); bucket != nil {
		return &FileStoreTxn{
			txn:    txn,
			bucket: bucket,
		}
	}
	panic("bucket not found")
}

func (s *FileStore) Get(key string) ([]byte, error) {
	var value []byte
	return value, s.db.View(func(txn *bolt.Tx) (err error) {
		value, err = s.newTxn(txn).Get(key)
		return err
	})
}

func (s *FileStore) Set(key string, value []byte) error {
	return s.db.Update(func(txn *bolt.Tx) error {
		return s.newTxn(txn).Set(key, value)
	})
}

func (s *FileStore) Delete(key string) error {
	return s.db.Update(func(txn *bolt.Tx) error {
		return s.newTxn(txn).Delete(key)
	})
}

func (s *FileStore) IterateValuesPrefix(prefix string, fn func(string, []byte) error) error {
	return s.db.View(func(txn *bolt.Tx) error {
		return s.newTxn(txn).IterateValuesPrefix(prefix, fn)
	})
}

func (s *FileStore) IterateTxnPrefix(prefix string, fn func(StorageTxn, string) error) error {
	return s.db.Update(func(txn *bolt.Tx) error {
		return s.newTxn(txn).IterateTxnPrefix(prefix, fn)
	})
}

func (s *FileStore) GetPrefix(prefix string, offset, limit int) ([]*KeyValue, error) {
	var list []*KeyValue
	err := s.db.View(func(txn *bolt.Tx) error {
		val, err := s.newTxn(txn).GetPrefix(prefix, offset, limit)
		if err != nil {
			return fmt.Errorf("failed to get prefix: %w", err)
		}
		list = val
		return nil
	})
	return list, err
}

func (s *FileStore) Close() error {
	return s.db.Close()
}

func (tx *FileStoreTxn) Get(key string) ([]byte, error) {
	return tx.bucket.Get([]byte(key)), nil
}

func (tx *FileStoreTxn) Set(key string, value []byte) error {
	if tx.ro {
		return ErrTxnReadOnly
	}
	return tx.bucket.Put([]byte(key), value)
}

func (tx *FileStoreTxn) Delete(key string) error {
	if tx.ro {
		return ErrTxnReadOnly
	}
	return tx.bucket.Delete([]byte(key))
}

func (tx *FileStoreTxn) IterateValuesPrefix(prefix string, fn func(string, []byte) error) error {
	c := tx.bucket.Cursor()
	pre := []byte(prefix)
	for k, v := c.Seek(pre); bytes.HasPrefix(k, pre); k, v = c.Next() {
		if err := fn(string(k), v); err != nil {
			return err
		}
	}
	return nil
}

func (tx *FileStoreTxn) IterateTxnPrefix(prefix string, fn func(StorageTxn, string) error) error {
	c := tx.bucket.Cursor()
	pre := []byte(prefix)
	for k, _ := c.Seek(pre); bytes.HasPrefix(k, pre); k, _ = c.Next() {
		if err := fn(tx, string(k)); err != nil {
			return err
		}
	}
	return nil
}

func (s *FileStoreTxn) GetPrefix(prefix string, offset, limit int) ([]*KeyValue, error) {
	list := make([]*KeyValue, 0)
	c := s.bucket.Cursor()
	pre := []byte(prefix)
	for k, v := c.Seek(pre); bytes.HasPrefix(k, pre); k, v = c.Next() {
		if offset > 0 {
			offset--
			continue
		}
		if limit == 0 {
			break
		}
		list = append(list, &KeyValue{Key: string(k), Value: v})
		limit--
	}
	return list, nil
}
