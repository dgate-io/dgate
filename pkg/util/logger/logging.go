package logger

import (
	"context"
	"io"
	"log"
	"log/slog"

	"github.com/hashicorp/go-hclog"
)

type SLogHCAdapter struct {
	ctx    context.Context
	logger *slog.Logger
	level  *slog.LevelVar
}

func NewSLogHCAdapter(logger *slog.Logger) *SLogHCAdapter {
	return &SLogHCAdapter{context.Background(), logger, new(slog.LevelVar)}
}

func (l *SLogHCAdapter) IsTrace() bool { return false }

func (l *SLogHCAdapter) IsDebug() bool {
	return l.logger.Handler().Enabled(l.ctx, slog.LevelDebug)
}

func (l *SLogHCAdapter) IsInfo() bool {
	return l.logger.Handler().Enabled(l.ctx, slog.LevelInfo)
}

func (l *SLogHCAdapter) IsWarn() bool {
	return l.logger.Handler().Enabled(l.ctx, slog.LevelWarn)
}

func (l *SLogHCAdapter) IsError() bool {
	return l.logger.Handler().Enabled(l.ctx, slog.LevelError)
}

func (l *SLogHCAdapter) Trace(format string, args ...interface{}) {}

func (l *SLogHCAdapter) Debug(format string, args ...interface{}) {
	l.logger.Debug(format)
}

func (l *SLogHCAdapter) Info(format string, args ...interface{}) {
	l.logger.Info(format)
}

func (l *SLogHCAdapter) Warn(format string, args ...interface{}) {
	l.logger.Warn(format)
}

func (l *SLogHCAdapter) Error(format string, args ...interface{}) {
	l.logger.Error(format)
}

func (l *SLogHCAdapter) Log(level hclog.Level, format string, args ...interface{}) {
	switch level {
	case hclog.Debug:
		l.Debug(format, args...)
	case hclog.Info:
		l.Info(format, args...)
	case hclog.Warn:
		l.Warn(format, args...)
	case hclog.Error:
		l.Error(format, args...)
	}
}

func (l *SLogHCAdapter) GetLevel() hclog.Level {
	return sl2hcLogLevel(l.level.Level())
}

func (l *SLogHCAdapter) SetLevel(level hclog.Level) {
	l.level.Set(hc2slLogLevel(level))
}

func (l *SLogHCAdapter) Name() string {
	return ""
}

func (l *SLogHCAdapter) Named(name string) hclog.Logger {
	return &SLogHCAdapter{l.ctx, l.logger.With("name", name), l.level}
}

func (l *SLogHCAdapter) ResetNamed(name string) hclog.Logger {
	return &SLogHCAdapter{l.ctx, l.logger.With("name", nil), l.level}
}

func (l *SLogHCAdapter) With(args ...any) hclog.Logger {
	return &SLogHCAdapter{l.ctx, l.logger.With(args), l.level}
}

func (l *SLogHCAdapter) StandardLogger(opts *hclog.StandardLoggerOptions) *log.Logger {
	return slog.NewLogLogger(l.logger.Handler(), slog.LevelInfo)
}

func (l *SLogHCAdapter) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer {
	return slog.NewLogLogger(l.logger.Handler(), slog.LevelInfo).Writer()
}

func (l *SLogHCAdapter) ImpliedArgs() []interface{} {
	return nil
}

func hc2slLogLevel(lvl hclog.Level) slog.Level {
	switch lvl {
	case hclog.Debug, hclog.Trace, hclog.NoLevel:
		return slog.LevelDebug
	case hclog.Info, hclog.DefaultLevel:
		return slog.LevelInfo
	case hclog.Warn:
		return slog.LevelWarn
	case hclog.Error, hclog.Off:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func sl2hcLogLevel(lvl slog.Level) hclog.Level {
	switch lvl {
	case slog.LevelDebug:
		return hclog.Debug
	case slog.LevelInfo:
		return hclog.Info
	case slog.LevelWarn:
		return hclog.Warn
	case slog.LevelError:
		return hclog.Error
	default:
		return hclog.NoLevel
	}
}
