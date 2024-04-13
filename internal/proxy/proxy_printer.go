package proxy

import (
	"github.com/rs/zerolog"
)

type (
	ProxyPrinter struct {
		logger zerolog.Logger
		// logs   []*printerLog
	}
	// printerLog struct {
	// 	time  time.Time
	// 	level string
	// 	msg   string
	// }
)

func NewProxyPrinter(logger zerolog.Logger) *ProxyPrinter {
	return &ProxyPrinter{
		logger: logger,
		// logs:   make([]*printerLog, 0),
	}
}

func (pp *ProxyPrinter) Error(s string) {
	// pp.logs = append(pp.logs, &printerLog{
	// 	time.Now(), "error", s})
	pp.logger.Error().Msg(s)
}

func (pp *ProxyPrinter) Warn(s string) {
	// pp.logs = append(pp.logs, &printerLog{
	// 	time.Now(), "warn", s})
	pp.logger.Warn().Msg(s)
}

func (pp *ProxyPrinter) Log(s string) {
	// pp.logs = append(pp.logs, &printerLog{
	// 	time.Now(), "info", s})
	pp.logger.Debug().Msg(s)
}
