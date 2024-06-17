package proxy

import (
	"go.uber.org/zap"
)

type (
	ProxyPrinter struct {
		logger *zap.Logger
	}
)

// NewProxyPrinter creates a new ProxyPrinter.
func NewProxyPrinter(logger *zap.Logger, lvl zap.AtomicLevel) *ProxyPrinter {
	newLogger := logger.WithOptions(zap.IncreaseLevel(lvl))
	if !logger.Core().Enabled(lvl.Level()) {
		logger.Warn("the desired log level is lower than the global log level")
	}
	return &ProxyPrinter{newLogger}
}

// Error logs a message at error level.
func (pp *ProxyPrinter) Error(s string) {
	pp.logger.Error(s)
}

// Warn logs a message at warn level.
func (pp *ProxyPrinter) Warn(s string) {
	pp.logger.Warn(s)
}

// Log logs a message at debug level.
func (pp *ProxyPrinter) Log(s string) {
	pp.logger.Debug(s)
}

/*
	Note: The following methods are not used but are included for completeness.
*/

// Info logs a message at info level.
func (pp *ProxyPrinter) Info(s string) {
	pp.logger.Info(s)
}

// Debug logs a message at debug level.
func (pp *ProxyPrinter) Debug(s string) {
	pp.logger.Debug(s)
}
