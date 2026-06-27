package loggers

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
)

type ConsoleLoggerConfig struct {
	Level string `json:"level"`
	level zerolog.Level
}

func (c *ConsoleLoggerConfig) Validate() error {
	if c == nil {
		return ConfigValidateError("*ConsoleLoggerConfig is nil")
	}
	var err error
	if c.level, err = zerolog.ParseLevel(c.Level); err != nil {
		return ConfigValidateError(fmt.Sprintf("zerolog.ParseLevel: %s", err))
	}

	return nil
}

type ConsoleLogger struct {
	logger zerolog.Logger
}

func NewConsoleLogger(config *ConsoleLoggerConfig) (*ConsoleLogger, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config.Validate: %w", err)
	}

	cl := &ConsoleLogger{}

	cl.logger = zerolog.New(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.TimeFormat = "15:04:05"
		w.Out = os.Stdout
	})).
		Level(config.level).
		With().
		Timestamp().
		Logger()

	return cl, nil
}

func (l *ConsoleLogger) Trace() *zerolog.Event {
	return l.logger.Trace()
}

func (l *ConsoleLogger) Debug() *zerolog.Event {
	return l.logger.Debug()
}

func (l *ConsoleLogger) Info() *zerolog.Event {
	return l.logger.Info()
}

func (l *ConsoleLogger) Warn() *zerolog.Event {
	return l.logger.Warn()
}

func (l *ConsoleLogger) Error() *zerolog.Event {
	return l.logger.Error()
}

func (l *ConsoleLogger) Close() error {
	return nil
}
