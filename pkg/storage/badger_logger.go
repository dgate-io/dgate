package storage

import (
	"github.com/dgraph-io/badger/v4"
	"github.com/rs/zerolog"
)

type badgerLoggerAdapter struct {
	logger zerolog.Logger
}

func (b *badgerLoggerAdapter) Errorf(format string, args ...any) {
	b.logger.Error().Msgf(format, args...)
}

func (b *badgerLoggerAdapter) Warningf(format string, args ...any) {
	b.logger.Warn().Msgf(format, args...)
}

func (b *badgerLoggerAdapter) Infof(format string, args ...any) {
	b.logger.Info().Msgf(format, args...)
}

func (b *badgerLoggerAdapter) Debugf(format string, args ...any) {
	b.logger.Debug().Msgf(format, args...)
}

func newBadgerLoggerAdapter(component string, logger zerolog.Logger) badger.Logger {
	// logger := fsConfig.Logger.Hook(zerolog.HookFunc(func(e *zerolog.Event, level zerolog.Level, msg string) {
	// 	e.Str("storage", "filestore::badger")
	// }))
	logger = logger.With().Str("component", component).Logger()
	return &badgerLoggerAdapter{
		logger: logger,
	}
}
