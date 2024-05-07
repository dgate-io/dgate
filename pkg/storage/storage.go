package storage

type Storage interface {
	StorageTxn
	Connect() error
	Close() error
}

type StorageTxn interface {
	Get(string) ([]byte, error)
	Set(string, []byte) error
	IterateValuesPrefix(prefix string, fn func(key string, val []byte) error) error
	IterateTxnPrefix(prefix string, fn func(txn StorageTxn, key string) error) error
	GetPrefix(prefix string, offset, limit int) ([]*KeyValue, error)
	Delete(string) error
}

type KeyValue struct {
	Key   string
	Value []byte
}
