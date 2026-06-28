package gologger

import (
	"fmt"
	"io"

	"github.com/snakehunterr/gologger/loggers"

	"github.com/rs/zerolog"
)

type LoggerService struct {
	consoleLogger     *loggers.ConsoleLogger
	fileLogger        *loggers.FileLogger
	sentryLogger      *loggers.SentryLogger
	openObserveLogger *loggers.OpenObserveLogger
	loggers           []Logger
}

func NewLoggerService() *LoggerService {
	return &LoggerService{}
}

func (ls *LoggerService) WithConsoleLogger(config *ConsoleLoggerConfig) error {
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
	events := make([]*zerolog.Event, len(ls.loggers))

	for i, logger := range ls.loggers {
		events[i] = fn(logger)
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
	for _, logger := range ls.loggers {
		if closer, ok := logger.(io.Closer); ok {
			if err := closer.Close(); err != nil {
				return fmt.Errorf("%T.Close: %w", logger, err)
			}
		}
	}

	return nil
}
