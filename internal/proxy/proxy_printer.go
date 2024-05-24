package proxy

import "log/slog"

type (
	ProxyPrinter struct {
		logger *slog.Logger
		// logs   []*printerLog
	}
	// printerLog struct {
	// 	time  time.Time
	// 	level string
	// 	msg   string
	// }
)

func NewProxyPrinter(logger *slog.Logger) *ProxyPrinter {
	return &ProxyPrinter{
		logger: logger,
		// logs:   make([]*printerLog, 0),
	}
}

func (pp *ProxyPrinter) Error(s string) {
	// pp.logs = append(pp.logs, &printerLog{
	// 	time.Now(), "error", s})
	pp.logger.Error(s)
}

func (pp *ProxyPrinter) Warn(s string) {
	// pp.logs = append(pp.logs, &printerLog{
	// 	time.Now(), "warn", s})
	pp.logger.Warn(s)
}

func (pp *ProxyPrinter) Log(s string) {
	// pp.logs = append(pp.logs, &printerLog{
	// 	time.Now(), "info", s})
	pp.logger.Debug(s)
}
