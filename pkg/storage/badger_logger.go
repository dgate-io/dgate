package storage

import (
	"fmt"
	"log/slog"

	"github.com/dgraph-io/badger/v4"
)

type badgerLoggerAdapter struct {
	logger *slog.Logger
}

func (b *badgerLoggerAdapter) Errorf(format string, args ...any) {
	b.logger.Error(fmt.Sprintf(format, args...))
}

func (b *badgerLoggerAdapter) Warningf(format string, args ...any) {
	b.logger.Warn(fmt.Sprintf(format, args...))
}

func (b *badgerLoggerAdapter) Infof(format string, args ...any) {
	b.logger.Info(fmt.Sprintf(format, args...))
}

func (b *badgerLoggerAdapter) Debugf(format string, args ...any) {
	b.logger.Debug(fmt.Sprintf(format, args...))
}

func newBadgerLoggerAdapter(component string, logger *slog.Logger) badger.Logger {
	logger = logger.WithGroup(component)
	return &badgerLoggerAdapter{
		logger: logger,
	}
}
