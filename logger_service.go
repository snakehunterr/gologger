package gologger

import (
	"fmt"
	"io"

	"github.com/snakehunterr/gologger/loggers"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/rs/zerolog"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
)

type LoggerService struct {
	serviceName       string
	moduleName        string
	consoleLogger     *loggers.ConsoleLogger
	fileLogger        *loggers.FileLogger
	sentryLogger      *loggers.SentryLogger
	openObserveLogger *loggers.OpenObserveLogger
	loggers           []Logger
	childs            []*LoggerService
}

func NewLoggerService(serviceName, moduleName string) *LoggerService {
	return &LoggerService{
		serviceName: serviceName,
		moduleName:  moduleName,
		loggers:     make([]Logger, 0, 4),
	}
}

func (ls *LoggerService) WithModuleName(name string) *LoggerService {
	if ls == nil {
		return ls
	}

	newls := &LoggerService{
		serviceName: ls.serviceName,
		moduleName:  name,
		loggers:     make([]Logger, 0, 4),
	}

	ls.childs = append(ls.childs, newls)

	if ls.consoleLogger != nil {
		newls.consoleLogger = ls.consoleLogger
		newls.loggers = append(newls.loggers, ls.consoleLogger)
	}

	if ls.fileLogger != nil {
		newls.fileLogger = ls.fileLogger
		newls.loggers = append(newls.loggers, ls.fileLogger)
	}

	if ls.sentryLogger != nil {
		newls.sentryLogger = ls.sentryLogger
		newls.loggers = append(newls.loggers, ls.sentryLogger)
	}

	if ls.openObserveLogger != nil {
		newls.openObserveLogger = ls.openObserveLogger
		newls.loggers = append(newls.loggers, ls.openObserveLogger)
	}

	return newls
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
		return nil
	}

	var err error
	if ls.consoleLogger, err = loggers.NewConsoleLogger(config); err != nil {
		return fmt.Errorf("loggers.NewConsoleLogger: %w", err)
	}

	ls.loggers = append(ls.loggers, ls.consoleLogger)

	return nil
}

func (ls *LoggerService) WithFileLogger(config *FileLoggerConfig) error {
	if ls == nil {
		return loggers.LoggerError("*LoggerService is nil")
	}

	if ls.fileLogger != nil {
		return nil
	}

	var err error
	if ls.fileLogger, err = loggers.NewFileLogger(config); err != nil {
		return fmt.Errorf("loggers.NewFileLogger: %w", err)
	}

	ls.loggers = append(ls.loggers, ls.fileLogger)

	return nil
}

func (ls *LoggerService) WithSentryLogger(config *SentryLoggerConfig) error {
	if ls == nil {
		return loggers.LoggerError("*LoggerService is nil")
	}

	if ls.sentryLogger != nil {
		return nil
	}

	var err error
	if ls.sentryLogger, err = loggers.NewSentryLogger(config); err != nil {
		return fmt.Errorf("loggers.NewSentryLogger: %w", err)
	}

	ls.loggers = append(ls.loggers, ls.sentryLogger)

	return nil
}

func (ls *LoggerService) WithOpenObserveLogger(config *OpenObserveLoggerConfig) error {
	if ls == nil {
		return loggers.LoggerError("*LoggerService is nil")
	}

	if ls.openObserveLogger != nil {
		return nil
	}

	var err error
	if ls.openObserveLogger, err = loggers.NewOpenObserveLogger(config); err != nil {
		return fmt.Errorf("loggers.NewOpenObserveLogger: %w", err)
	}

	ls.loggers = append(ls.loggers, ls.openObserveLogger)

	return nil
}

func (ls *LoggerService) callToLoggers(fn func(logger Logger) *zerolog.Event) LoggerEvents {
	if ls == nil {
		return nil
	}

	events := make([]*zerolog.Event, len(ls.loggers))

	for i, logger := range ls.loggers {
		events[i] = fn(logger).
			Str("service", ls.serviceName).
			Str("module", ls.moduleName)
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

func (ls *LoggerService) Close() error {
	if ls == nil {
		return nil
	}

	for _, logger := range ls.childs {
		if err := logger.Close(); err != nil {
			return fmt.Errorf("logger.Close: %w", err)
		}
	}

	for _, logger := range ls.loggers {
		if closer, ok := logger.(io.Closer); ok {
			if err := closer.Close(); err != nil {
				return fmt.Errorf("%T.Close: %w", logger, err)
			}
		}
	}

	return nil
}
