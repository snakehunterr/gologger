package logger

import (
	"errors"
	"fmt"
	"io"
	"reflect"

	"github.com/snakehunterr/gologger/loggers"

	"github.com/rs/zerolog"
)

type LoggerEvents []*zerolog.Event

func (le LoggerEvents) Msg(msg string) {
	for _, event := range le {
		event.Msg(msg)
	}
}

func (le LoggerEvents) Msgf(format string, args ...any) {
	le.Msg(fmt.Sprintf(format, args...))
}

func (le LoggerEvents) Str(key, val string) LoggerEvents {
	for _, event := range le {
		if event == nil {
			continue
		}
		*event = *event.Str(key, val)
	}

	return le
}

func (le LoggerEvents) Err(err error) LoggerEvents {
	inner := err

	for {
		unwrapped := errors.Unwrap(inner)

		if unwrapped == nil {
			break
		}

		inner = unwrapped
	}

	rftype := reflect.TypeOf(inner)

	var errorType string
	if rftype.Kind() == reflect.Ptr {
		errorType = rftype.Elem().Name()
	} else {
		errorType = rftype.Name()
	}

	for _, event := range le {
		if event == nil {
			continue
		}
		*event = *event.Err(err).Str("error_type", errorType)
	}

	return le
}

type LoggerService struct {
	consoleLogger *loggers.ConsoleLogger
	fileLogger    *loggers.FileLogger
	sentryLogger  *loggers.SentryLogger
	loggers       []Logger
}

func NewLoggerService() *LoggerService {
	return &LoggerService{}
}

type ConsoleLoggerConfig = loggers.ConsoleLoggerConfig

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

type FileLoggerConfig = loggers.FileLoggerConfig

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

type SentryLoggerConfig = loggers.SentryLoggerConfig

func (ls *LoggerService) WithSentryLogger(config *SentryLoggerConfig) error {
	if ls.fileLogger != nil {
		return nil
	}

	var err error
	if ls.sentryLogger, err = loggers.NewSentryLogger(config); err != nil {
		return fmt.Errorf("loggers.NewSentryLogger: %w", err)
	}

	ls.loggers = append(ls.loggers, ls.sentryLogger)

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
