package logadapter

import (
	"fmt"

	"github.com/dgraph-io/badger/v4"
	"go.uber.org/zap"
)

type Zap2BadgerAdapter struct {
	logger *zap.Logger
}

var _ badger.Logger = (*Zap2BadgerAdapter)(nil)

func NewZap2BadgerAdapter(logger *zap.Logger) *Zap2BadgerAdapter {
	return &Zap2BadgerAdapter{
		logger: logger,
	}
}

func (a *Zap2BadgerAdapter) Debugf(format string, args ...interface{}) {
	a.logger.Debug(fmt.Sprintf(format, args...))
}

func (a *Zap2BadgerAdapter) Infof(format string, args ...interface{}) {
	a.logger.Info(fmt.Sprintf(format, args...))
}

func (a *Zap2BadgerAdapter) Warningf(format string, args ...interface{}) {
	a.logger.Warn(fmt.Sprintf(format, args...))
}

func (a *Zap2BadgerAdapter) Errorf(format string, args ...interface{}) {
	a.logger.Error(fmt.Sprintf(format, args...))
}
