package loggers

import (
	"fmt"
	"os"
	"sync"
	"time"

	gopath "path/filepath"

	"github.com/rs/zerolog"
)

type FileLoggerConfig struct {
	Level  string `json:"level"`
	level  zerolog.Level
	LogDir string `json:"log_dir"`
}

func (c *FileLoggerConfig) Validate() error {
	if c == nil {
		return ConfigValidateError("*FileLoggerConfig is nil")
	}
	var err error
	if c.level, err = zerolog.ParseLevel(c.Level); err != nil {
		return ConfigValidateError(fmt.Sprintf("zerolog.ParseLevel: %s", err))
	}
	if c.LogDir == "" {
		c.LogDir = "."
	}

	return nil
}

type FileLogger struct {
	config   *FileLoggerConfig
	file     *os.File
	logger   zerolog.Logger
	currDate time.Time
	mu       sync.Mutex
	closed   bool
}

func NewFileLogger(config *FileLoggerConfig) (*FileLogger, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config.Validate(): %w", err)
	}

	logger := &FileLogger{
		config: config,
	}

	logger.currDate = logger.now()

	if err := logger.createFile(); err != nil {
		return nil, fmt.Errorf("createFile: %w", err)
	}

	if err := logger.newLogger(); err != nil {
		return nil, fmt.Errorf("newLogger: %w", err)
	}

	return logger, nil
}

func (l *FileLogger) createFile() error {
	filename := fmt.Sprintf("%s.log", l.currDate.Format("2006-01-02"))
	filepath := gopath.Join(l.config.LogDir, filename)
	dirname := gopath.Dir(filepath)
	if _, err := os.Stat(dirname); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("os.Stat: %w", err)
		}

		if err := os.MkdirAll(dirname, 0o755); err != nil {
			return fmt.Errorf("os.MkdirAll: %w", err)
		}
	}

	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("os.OpenFile: %w", err)
	}

	l.file = file
	return nil
}

func (l *FileLogger) newLogger() error {
	if l.file == nil {
		return ErrFileLoggerLogFileIsNil
	}

	l.logger = zerolog.New(l.file).
		Level(l.config.level).
		With().
		Timestamp().
		Logger()

	return nil
}

func (l *FileLogger) now() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

func (l *FileLogger) callToZeroLogger(fn func() *zerolog.Event) *zerolog.Event {
	if l.closed {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	currDate := l.now()
	if !currDate.Equal(l.currDate) {
		l.currDate = currDate

		if err := l.file.Close(); err != nil {
			// TODO:
		}

		if err := l.createFile(); err != nil {
			fmt.Fprintf(os.Stderr, "FileLogger rotation error: %v\n", err)
			return nil
		}

		if err := l.newLogger(); err != nil {
			fmt.Fprintf(os.Stderr, "FileLogger newLogger error: %v\n", err)
			return nil
		}

	}

	return fn()
}

func (l *FileLogger) Trace() *zerolog.Event {
	return l.callToZeroLogger(func() *zerolog.Event {
		return l.logger.Trace()
	})
}

func (l *FileLogger) Debug() *zerolog.Event {
	return l.callToZeroLogger(func() *zerolog.Event {
		return l.logger.Debug()
	})
}

func (l *FileLogger) Info() *zerolog.Event {
	return l.callToZeroLogger(func() *zerolog.Event {
		return l.logger.Info()
	})
}

func (l *FileLogger) Warn() *zerolog.Event {
	return l.callToZeroLogger(func() *zerolog.Event {
		return l.logger.Warn()
	})
}

func (l *FileLogger) Error() *zerolog.Event {
	return l.callToZeroLogger(func() *zerolog.Event {
		return l.logger.Error()
	})
}

func (l *FileLogger) GetLevel() zerolog.Level {
	return l.logger.GetLevel()
}

func (l *FileLogger) Close() error {
	if l.closed {
		return nil
	}
	if l.file == nil {
		return nil
	}

	err := l.file.Close()
	l.file = nil

	l.closed = true
	return err
}
