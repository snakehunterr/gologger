package gologger

import (
	"errors"
	"fmt"
	"io"
	"runtime"

	"github.com/snakehunterr/gologger/loggers"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap/zapcore"

	"github.com/rs/zerolog"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
)

type LoggerService struct {
	ServiceName string

	ModuleName string

	consoleLogger     *loggers.ConsoleLogger
	fileLogger        *loggers.FileLogger
	sentryLogger      *loggers.SentryLogger
	openObserveLogger *loggers.OpenObserveLogger

	loggers []Logger
	childs  []*LoggerService
}

func NewLoggerService(name string) *LoggerService {
	ls := &LoggerService{
		ServiceName: name,
	}

	ls.loggers = []Logger{
		ls.consoleLogger,
		ls.fileLogger,
		ls.sentryLogger,
		ls.openObserveLogger,
	}

	return ls
}

func (ls *LoggerService) NewChild(name string) (*LoggerService, error) {
	if ls == nil {
		return nil, errors.New("*LoggerService is nil")
	}

	newls := &LoggerService{
		ServiceName:   name,
		consoleLogger: ls.consoleLogger,
		fileLogger:    ls.fileLogger,
		sentryLogger:  ls.sentryLogger,
	}

	if ls.openObserveLogger != nil {
		if err := newls.WithOpenObserveLogger(ls.openObserveLogger.GetConfig()); err != nil {
			return nil, fmt.Errorf("newls.WithOpenObserveLogger: %w", err)
		}
	}

	newls.loggers = []Logger{
		newls.consoleLogger,
		newls.fileLogger,
		newls.sentryLogger,
		newls.openObserveLogger,
	}

	ls.childs = append(ls.childs, newls)

	return newls, nil
}

func (ls *LoggerService) WithModuleName(name string) *LoggerService {
	if ls == nil {
		return ls
	}

	ls.ModuleName = name
	return ls
}

// Tracer returns the OTel tracer backing the OpenObserve logger, for
// starting spans:
//
//	ctx, span := logger.Tracer().Start(ctx, "operation-name")
//	defer span.End()
//
// If no OpenObserve logger has been configured via WithOpenObserveLogger,
// a no-op tracer is returned so calling code never needs a nil check.
func (ls *LoggerService) Tracer() trace.Tracer {
	if ls == nil || ls.openObserveLogger == nil {
		return nooptrace.NewTracerProvider().Tracer("noop")
	}
	return ls.openObserveLogger.Tracer()
}

// Meter returns the OTel meter backing the OpenObserve logger, for creating
// instruments:
//
//	counter, err := logger.Meter().Int64Counter("requests_total")
//
// If no OpenObserve logger has been configured via WithOpenObserveLogger,
// a no-op meter is returned so calling code never needs a nil check.
func (ls *LoggerService) Meter() metric.Meter {
	if ls == nil || ls.openObserveLogger == nil {
		return noopmetric.NewMeterProvider().Meter("noop")
	}
	return ls.openObserveLogger.Meter()
}

func (ls *LoggerService) WithConsoleLogger(config *ConsoleLoggerConfig) error {
	if ls == nil {
		return loggers.LoggerError("*LoggerService is nil")
	}

	if ls.consoleLogger != nil {
		if err := ls.consoleLogger.Close(); err != nil {
			return fmt.Errorf("ls.consoleLogger.Close: %w", err)
		}
	}

	var err error
	if ls.consoleLogger, err = loggers.NewConsoleLogger(config); err != nil {
		return fmt.Errorf("loggers.NewConsoleLogger: %w", err)
	}

	return nil
}

func (ls *LoggerService) WithFileLogger(config *FileLoggerConfig) error {
	if ls == nil {
		return loggers.LoggerError("*LoggerService is nil")
	}

	if ls.fileLogger != nil {
		if err := ls.fileLogger.Close(); err != nil {
			return fmt.Errorf("ls.fileLogger.Close: %w", err)
		}
	}

	var err error
	if ls.fileLogger, err = loggers.NewFileLogger(config); err != nil {
		return fmt.Errorf("loggers.NewFileLogger: %w", err)
	}

	return nil
}

func (ls *LoggerService) WithSentryLogger(config *SentryLoggerConfig) error {
	if ls == nil {
		return loggers.LoggerError("*LoggerService is nil")
	}

	if ls.sentryLogger != nil {
		if err := ls.sentryLogger.Close(); err != nil {
			return fmt.Errorf("ls.sentryLogger.Close: %w", err)
		}
	}

	var err error
	if ls.sentryLogger, err = loggers.NewSentryLogger(config); err != nil {
		return fmt.Errorf("loggers.NewSentryLogger: %w", err)
	}

	return nil
}

func (ls *LoggerService) WithOpenObserveLogger(config *OpenObserveLoggerConfig) error {
	if ls == nil {
		return loggers.LoggerError("*LoggerService is nil")
	}

	if ls.openObserveLogger != nil {
		if err := ls.openObserveLogger.Close(); err != nil {
			return fmt.Errorf("ls.openObserveLogger.Close: %w", err)
		}
	}

	var err error
	if ls.openObserveLogger, err = loggers.NewOpenObserveLogger(ls.ServiceName, config); err != nil {
		return fmt.Errorf("loggers.NewOpenObserveLogger: %w", err)
	}

	return nil
}

func callerFuncName() string {
	pc, _, _, ok := runtime.Caller(3)

	if !ok {
		return "<unknown>"
	}

	return runtime.FuncForPC(pc).Name()
}

func (ls *LoggerService) callToLoggers(fn func(logger Logger) *zerolog.Event) LoggerEvents {
	if ls == nil {
		return nil
	}

	events := make([]*zerolog.Event, len(ls.loggers))

	fnName := callerFuncName()

	for i, logger := range ls.loggers {
		if logger == nil {
			continue
		}

		events[i] = fn(logger).
			Str("service", ls.ServiceName).
			Str("module", ls.ModuleName).
			Str("caller_name", fnName)
	}

	return events
}

func (ls *LoggerService) Trace() LoggerEvents {
	return ls.callToLoggers(func(logger Logger) *zerolog.Event { return logger.Trace() })
}

func (ls *LoggerService) Debug() LoggerEvents {
	return ls.callToLoggers(func(logger Logger) *zerolog.Event { return logger.Debug() })
}

func (ls *LoggerService) Info() LoggerEvents {
	return ls.callToLoggers(func(logger Logger) *zerolog.Event { return logger.Info() })
}

func (ls *LoggerService) Warn() LoggerEvents {
	return ls.callToLoggers(func(logger Logger) *zerolog.Event { return logger.Warn() })
}

func (ls *LoggerService) Error() LoggerEvents {
	return ls.callToLoggers(func(logger Logger) *zerolog.Event { return logger.Error() })
}

func (ls *LoggerService) GetMinZerologLevel() zerolog.Level {
	level := zerolog.Disabled

	for _, logger := range ls.loggers {
		if logger == nil {
			continue
		}

		lvl := logger.GetLevel()
		if level > lvl {
			level = lvl
		}
	}

	return level
}

func (ls *LoggerService) GetMinZapLevel() zapcore.Level {
	switch ls.GetMinZerologLevel() {
	case zerolog.TraceLevel, zerolog.DebugLevel:
		return zapcore.DebugLevel
	case zerolog.InfoLevel:
		return zapcore.InfoLevel
	case zerolog.WarnLevel:
		return zapcore.WarnLevel
	case zerolog.ErrorLevel:
		return zapcore.ErrorLevel
	default:
		return zapcore.ErrorLevel
	}
}

func (ls *LoggerService) Close() error {
	if ls == nil {
		return nil
	}

	for _, logger := range ls.loggers {
		if logger == nil {
			continue
		}

		if closer, ok := logger.(io.Closer); ok {
			if err := closer.Close(); err != nil {
				return fmt.Errorf("%T.Close: %w", logger, err)
			}
		}
	}

	for _, logger := range ls.childs {
		if err := logger.Close(); err != nil {
			return fmt.Errorf("logger.Close: %w", err)
		}
	}

	return nil
}
