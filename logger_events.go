package logger

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/rs/zerolog"
)

type LoggerEvents []*zerolog.Event

func (le LoggerEvents) Msg(msg string) {
	for _, event := range le {
		if event == nil {
			continue
		}
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

func (le LoggerEvents) Strf(key, format string, args ...any) LoggerEvents {
	return le.Str(key, fmt.Sprintf(format, args))
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

func (le LoggerEvents) Errf(format string, args ...any) LoggerEvents {
	return le.Err(fmt.Errorf(format, args...))
}
