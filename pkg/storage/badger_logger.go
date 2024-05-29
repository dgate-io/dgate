package storage

import (
	"fmt"

	"github.com/dgraph-io/badger/v4"
	"go.uber.org/zap"
)

type badgerLoggerAdapter struct {
	logger *zap.Logger
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

func newBadgerLoggerAdapter(component string, logger *zap.Logger) badger.Logger {
	return &badgerLoggerAdapter{
		logger: logger.Named(component),
	}
}
