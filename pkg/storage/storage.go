package storage

type Storage interface {
	StorageTxn
	Txn(write bool, fn func(txn StorageTxn) error) error
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
