package loggers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

type SentryLoggerConfig struct {
	DSN         string `json:"dsn"`
	Level       string `json:"level"`
	level       zerolog.Level
	AppVersion  string `json:"app_version"`
	Environment string `json:"environment"`
}

func (c *SentryLoggerConfig) Validate() error {
	if c == nil {
		return ConfigValidateError("*SentryLoggerConfig is nil")
	}
	if c.DSN == "" {
		return ConfigValidateError("c.DSN is empty")
	}
	var err error
	if c.level, err = zerolog.ParseLevel(c.Level); err != nil {
		return ConfigValidateError(fmt.Sprintf("zerolog.ParseLevel: %s", err))
	}
	if c.AppVersion == "" {
		return ConfigValidateError("c.AppVersion is empty")
	}
	if c.Environment == "" {
		return ConfigValidateError("c.Environment is empty")
	}

	return nil
}

type SentryWriter struct {
	levelMap map[string]sentry.Level
}

func NewSentryWriter() *SentryWriter {
	levelMap := map[string]sentry.Level{
		zerolog.LevelTraceValue: sentry.LevelDebug,
		zerolog.LevelDebugValue: sentry.LevelDebug,
		zerolog.LevelInfoValue:  sentry.LevelInfo,
		zerolog.LevelWarnValue:  sentry.LevelWarning,
		zerolog.LevelErrorValue: sentry.LevelError,
		zerolog.LevelFatalValue: sentry.LevelFatal,
	}

	return &SentryWriter{
		levelMap: levelMap,
	}
}

func (w *SentryWriter) Write(p []byte) (int, error) {
	var msg LoggerMessage
	if err := json.Unmarshal(p, &msg); err != nil {
		return 0, fmt.Errorf("json.Unmarshal: %w", err)
	}

	var err error
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetTag("module", msg.Module)
		scope.SetTag("service", msg.Service)
		scope.SetTag("level", msg.Level)
		scope.SetTag("logger", "sentry-logger")

		scope.SetContext("event_context", msg.ToSentryContext())

		if msg.Level == zerolog.LevelErrorValue {
			if e := w.captureError(&msg); e != nil {
				err = fmt.Errorf("captureError: %w", e)
			}
		} else {
			if e := w.captureEvent(&msg); e != nil {
				err = fmt.Errorf("captureEvent: %w", e)
			}
		}
	})

	if err != nil {
		return 0, err
	}

	return len(p), nil
}

func (w *SentryWriter) captureEvent(msg *LoggerMessage) error {
	event := sentry.NewEvent()

	event.Message = msg.Message

	sentry.CaptureEvent(event)

	return nil
}

func (w *SentryWriter) captureError(msg *LoggerMessage) error {
	event := sentry.NewEvent()

	event.Message = msg.Message

	event.Exception = []sentry.Exception{
		{
			Type:   msg.ErrorType,
			Value:  msg.Message,
			Module: msg.Module,
		},
	}

	sentry.CaptureEvent(event)

	return nil
}

type SentryLogger struct {
	config *SentryLoggerConfig
	writer *SentryWriter
	logger zerolog.Logger
	closed bool
}

func NewSentryLogger(config *SentryLoggerConfig) (*SentryLogger, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config.Validate: %w", err)
	}

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:         config.DSN,
		Environment: config.Environment,
	}); err != nil {
		return nil, fmt.Errorf("sentry.Init: %w", err)
	}

	logger := &SentryLogger{
		config: config,
		writer: NewSentryWriter(),
	}

	logger.logger = zerolog.New(logger.writer).
		Level(config.level).
		With().
		Timestamp().
		Logger()

	return logger, nil
}

func (l *SentryLogger) Trace() *zerolog.Event {
	return l.logger.Trace()
}

func (l *SentryLogger) Debug() *zerolog.Event {
	return l.logger.Debug()
}

func (l *SentryLogger) Info() *zerolog.Event {
	return l.logger.Info()
}

func (l *SentryLogger) Warn() *zerolog.Event {
	return l.logger.Warn()
}

func (l *SentryLogger) Error() *zerolog.Event {
	return l.logger.Error()
}

func (l *SentryLogger) GetLevel() zerolog.Level {
	return l.logger.GetLevel()
}

func (l *SentryLogger) Close() error {
	if l.closed {
		return nil
	}

	if !sentry.Flush(5 * time.Second) {
		return fmt.Errorf("sentry.Flush returned false")
	}

	l.closed = true
	return nil
}
