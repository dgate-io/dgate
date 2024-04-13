package logger

import (
	"io"
	"log"

	"github.com/hashicorp/go-hclog"
	"github.com/rs/zerolog"
)

type ZeroHCLogger struct {
	zerolog.Logger
}

var _ hclog.Logger = (*ZeroHCLogger)(nil)

func NewZeroHCLogger(logger zerolog.Logger) *ZeroHCLogger {
	return &ZeroHCLogger{logger}
}

func NewNopHCLogger() *ZeroHCLogger {
	return &ZeroHCLogger{zerolog.Nop()}
}

func (l *ZeroHCLogger) IsTrace() bool {
	return l.Logger.GetLevel() == zerolog.TraceLevel
}

func (l *ZeroHCLogger) IsDebug() bool {
	return l.Logger.GetLevel() == zerolog.DebugLevel
}

func (l *ZeroHCLogger) IsInfo() bool {
	return l.Logger.GetLevel() == zerolog.InfoLevel
}

func (l *ZeroHCLogger) IsWarn() bool {
	return l.Logger.GetLevel() == zerolog.WarnLevel
}

func (l *ZeroHCLogger) IsError() bool {
	return l.Logger.GetLevel() == zerolog.ErrorLevel
}

func (l *ZeroHCLogger) Trace(format string, args ...interface{}) {
	l.Logger.Trace().Fields(args).Msg(format)
}

func (l *ZeroHCLogger) Debug(format string, args ...interface{}) {
	l.Logger.Debug().Fields(args).Msg(format)
}

func (l *ZeroHCLogger) Info(format string, args ...interface{}) {
	l.Logger.Info().Fields(args).Msg(format)
}

func (l *ZeroHCLogger) Warn(format string, args ...interface{}) {
	l.Logger.Warn().Fields(args).Msg(format)
}

func (l *ZeroHCLogger) Error(format string, args ...interface{}) {
	l.Logger.Error().Fields(args).Msg(format)
}

func (l *ZeroHCLogger) Log(level hclog.Level, format string, args ...interface{}) {
	switch level {
	case hclog.Trace:
		l.Logger.Trace().Fields(args).Msg(format)
	case hclog.Debug:
		l.Logger.Debug().Fields(args).Msg(format)
	case hclog.Info:
		l.Logger.Info().Fields(args).Msg(format)
	case hclog.Warn:
		l.Logger.Warn().Fields(args).Msg(format)
	case hclog.Error:
		l.Logger.Error().Fields(args).Msg(format)
	default:
		log.Fatalf("unknown level %d", level)
	}
}

func (l *ZeroHCLogger) GetLevel() hclog.Level {
	switch l.Logger.GetLevel() {
	case zerolog.TraceLevel:
		return hclog.Trace
	case zerolog.DebugLevel:
		return hclog.Debug
	case zerolog.InfoLevel:
		return hclog.Info
	case zerolog.WarnLevel:
		return hclog.Warn
	case zerolog.ErrorLevel:
		return hclog.Error
	default:
		log.Printf("unknown level %d", l.Logger.GetLevel())
		return hclog.NoLevel
	}
}

func (l *ZeroHCLogger) SetLevel(level hclog.Level) {
	switch level {
	case hclog.Trace:
		l.Logger = l.Logger.Level(zerolog.TraceLevel)
	case hclog.Debug:
		l.Logger = l.Logger.Level(zerolog.DebugLevel)
	case hclog.Info:
		l.Logger = l.Logger.Level(zerolog.InfoLevel)
	case hclog.Warn:
		l.Logger = l.Logger.Level(zerolog.WarnLevel)
	case hclog.Error:
		l.Logger = l.Logger.Level(zerolog.ErrorLevel)
	default:
		log.Fatalf("unknown level %d", level)
	}
}

func (l *ZeroHCLogger) Name() string {
	return ""
}

func (l *ZeroHCLogger) Named(name string) hclog.Logger {
	return &ZeroHCLogger{l.Logger.With().Str("name", name).Logger()}
}

func (l *ZeroHCLogger) ResetNamed(name string) hclog.Logger {
	return &ZeroHCLogger{l.Logger.With().Str("name", name).Logger()}
}

func (l *ZeroHCLogger) With(args ...interface{}) hclog.Logger {
	return &ZeroHCLogger{l.Logger.With().Fields(args).Logger()}
}

func (l *ZeroHCLogger) StandardLogger(opts *hclog.StandardLoggerOptions) *log.Logger {
	return log.New(l.Logger, "", 0)
}

func (l *ZeroHCLogger) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer {
	return l.Logger
}

func (l *ZeroHCLogger) ImpliedArgs() []interface{} {
	return nil
}
