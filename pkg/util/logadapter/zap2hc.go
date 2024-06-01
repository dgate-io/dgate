package logadapter

import (
	"context"
	"fmt"
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

func (l *Zap2HCLogAdapter) Trace(format string, args ...any) {}

func prepArgs(args ...any) []zap.Field {
	if len(args) == 0 {
		return []zap.Field{}
	} else if len(args)%2 != 0 {
		args = append(args, "MISSING")
	}
	fields := make([]zap.Field, 0, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		key, val := args[i].(string), args[i+1]
		switch t := val.(type) {
		case hclog.Format:
			val = fmt.Sprintf(t[0].(string), t[1:]...)
		}
		fields = append(fields, zap.Any(key, val))
	}
	return fields
}

func (l *Zap2HCLogAdapter) Debug(format string, args ...any) {
	l.Log(hclog.Debug, format, args...)
}

func (l *Zap2HCLogAdapter) Info(format string, args ...any) {
	l.Log(hclog.Info, format, args...)
}

func (l *Zap2HCLogAdapter) Warn(format string, args ...any) {
	l.Log(hclog.Warn, format, args...)
}

func (l *Zap2HCLogAdapter) Error(format string, args ...any) {
	l.Log(hclog.Error, format, args...)
}

func (l *Zap2HCLogAdapter) Log(level hclog.Level, format string, args ...any) {
	switch level {
	case hclog.Debug:
		l.logger.Debug(format, prepArgs(args...)...)
	case hclog.Info:
		l.logger.Info(format, prepArgs(args...)...)
	case hclog.Warn:
		l.logger.Warn(format, prepArgs(args...)...)
	case hclog.Error:
		l.logger.Error(format, prepArgs(args...)...)
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

func (l *Zap2HCLogAdapter) ImpliedArgs() []any {
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
