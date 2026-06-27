package logger

import "github.com/rs/zerolog"

type Logger interface {
	Trace() *zerolog.Event
	Debug() *zerolog.Event
	Info() *zerolog.Event
	Warn() *zerolog.Event
	Err(err error) *zerolog.Event
	Errf(format string, args ...any) *zerolog.Event
	Error() *zerolog.Event
	Close() error
}
