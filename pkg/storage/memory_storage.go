package storage

import "go.uber.org/zap"

type MemoryStoreConfig struct {
	// Path to the directory where the files will be stored.
	// If the directory does not exist, it will be created.
	// If the directory exists, it will be used.
	Logger *zap.Logger
}

type MemoryStore struct {
	*FileStore
}

var _ Storage = &MemoryStore{}

func NewMemoryStore(cfg *MemoryStoreConfig) *MemoryStore {
	return &MemoryStore{
		FileStore: &FileStore{
			inMemory: true,
			logger: newBadgerLoggerAdapter(
				"memstore::badger", cfg.Logger,
			),
		},
	}
}
