package logadapter

import (
	"context"
	"io"
	"log"

	"github.com/hashicorp/go-hclog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Zap2HCLogAdapter struct {
	ctx    context.Context
	logger *zap.Logger
}

func NewZap2HCLogAdapter(logger *zap.Logger) *Zap2HCLogAdapter {
	return &Zap2HCLogAdapter{context.Background(), logger}
}

func (l *Zap2HCLogAdapter) IsTrace() bool {
	return l.logger.Core().Enabled(hc2zapLevel(hclog.Trace))
}

func (l *Zap2HCLogAdapter) IsDebug() bool {
	return l.logger.Core().Enabled(hc2zapLevel(hclog.Debug))
}

func (l *Zap2HCLogAdapter) IsInfo() bool {
	return l.logger.Core().Enabled(hc2zapLevel(hclog.Info))
}

func (l *Zap2HCLogAdapter) IsWarn() bool {
	return l.logger.Core().Enabled(hc2zapLevel(hclog.Warn))
}

func (l *Zap2HCLogAdapter) IsError() bool {
	return l.logger.Core().Enabled(hc2zapLevel(hclog.Error))
}

func (l *Zap2HCLogAdapter) Trace(format string, args ...interface{}) {}

func (l *Zap2HCLogAdapter) Debug(format string, args ...interface{}) {
	l.logger.Debug(format)
}

func (l *Zap2HCLogAdapter) Info(format string, args ...interface{}) {
	l.logger.Info(format)
}

func (l *Zap2HCLogAdapter) Warn(format string, args ...interface{}) {
	l.logger.Warn(format)
}

func (l *Zap2HCLogAdapter) Error(format string, args ...interface{}) {
	l.logger.Error(format)
}

func (l *Zap2HCLogAdapter) Log(level hclog.Level, format string, args ...interface{}) {
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

func (l *Zap2HCLogAdapter) GetLevel() hclog.Level {
	return zap2hcLogLevel(l.logger.Level())
}

func (l *Zap2HCLogAdapter) SetLevel(level hclog.Level) {}

func (l *Zap2HCLogAdapter) Name() string {
	return l.logger.Name()
}

func (l *Zap2HCLogAdapter) Named(name string) hclog.Logger {
	return &Zap2HCLogAdapter{l.ctx, l.logger.Named(name)}
}

func (l *Zap2HCLogAdapter) ResetNamed(name string) hclog.Logger {
	return &Zap2HCLogAdapter{l.ctx, l.logger.Named(name)}
}

func (l *Zap2HCLogAdapter) With(args ...any) hclog.Logger {
	return &Zap2HCLogAdapter{l.ctx, l.logger.Sugar().With(args...).Desugar()}
}

func (l *Zap2HCLogAdapter) StandardLogger(opts *hclog.StandardLoggerOptions) *log.Logger {
	return zap.NewStdLog(l.logger)
}

func (l *Zap2HCLogAdapter) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer {
	return l.StandardLogger(opts).Writer()
}

func (l *Zap2HCLogAdapter) ImpliedArgs() []interface{} {
	return nil
}

func hc2zapLevel(lvl hclog.Level) zapcore.Level {
	switch lvl {
	case hclog.Debug, hclog.Trace, hclog.NoLevel:
		return zap.DebugLevel
	case hclog.Info, hclog.DefaultLevel:
		return zap.InfoLevel
	case hclog.Warn:
		return zap.WarnLevel
	case hclog.Error, hclog.Off:
		return zap.ErrorLevel
	default:
		return zap.InfoLevel
	}
}

func zap2hcLogLevel(lvl zapcore.Level) hclog.Level {
	switch lvl {
	case zap.DebugLevel:
		return hclog.Debug
	case zap.InfoLevel:
		return hclog.Info
	case zap.WarnLevel:
		return hclog.Warn
	case zap.ErrorLevel, zap.PanicLevel, zap.DPanicLevel, zap.FatalLevel:
		return hclog.Error
	default:
		return hclog.NoLevel
	}
}
