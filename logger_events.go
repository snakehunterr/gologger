package gologger

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/snakehunterr/gologger/types"
	"go.opentelemetry.io/otel/trace"
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
	return le.Str(key, fmt.Sprintf(format, args...))
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

func (le LoggerEvents) Ctx(ctx context.Context) LoggerEvents {
	le = le.userCtx(ctx)
	return le.traceCtx(ctx)
}

func (le LoggerEvents) userCtx(ctx context.Context) LoggerEvents {
	if ctx == nil {
		return le
	}

	if uc, ok := UserContextFromContext(ctx); ok {
		for i, event := range le {
			if event == nil {
				continue
			}
			le[i] = event.Interface("user_context", uc)
		}
	}

	return le
}

func (le LoggerEvents) traceCtx(ctx context.Context) LoggerEvents {
	if ctx == nil {
		return le
	}

	tc, ok := TraceContextFromContext(ctx)
	if !ok {
		span := trace.SpanContextFromContext(ctx)
		if span.IsValid() {
			tc = types.TraceContext{
				TraceID: span.TraceID().String(),
				SpanID:  span.SpanID().String(),
			}
			ok = true
		}
	}
	if ok {
		for i, event := range le {
			if event == nil {
				continue
			}
			le[i] = event.Interface("trace_context", tc)
		}
	}

	return le
}

func (le LoggerEvents) FiberCtx(ctx *fiber.Ctx) LoggerEvents {
	for i, event := range le {
		if event == nil {
			continue
		}

		le[i] = event.Interface("fiber_context", types.NewFiberCtx(ctx))
	}

	return le
}

// captureStackFrames capture stack traces from skip to maxFrames
func captureStackFrames(skip, maxFrames int) []types.StackFrame {
	pcs := make([]uintptr, maxFrames)
	n := runtime.Callers(skip, pcs)
	if n == 0 {
		return nil
	}

	framesIter := runtime.CallersFrames(pcs[:n])

	var collected []types.StackFrame
	for {
		frame, more := framesIter.Next()
		collected = append(collected, types.StackFrame{
			Function: frame.Function,
			Filename: frame.File,
			Lineno:   frame.Line,
		})
		if !more {
			break
		}
	}

	for i, j := 0, len(collected)-1; i < j; i, j = i+1, j-1 {
		collected[i], collected[j] = collected[j], collected[i]
	}

	return collected
}

// Stack capture callers stack trace
func (le LoggerEvents) Stack() LoggerEvents {
	frames := captureStackFrames(3, 32)
	if len(frames) == 0 {
		return le
	}

	st := types.Stacktrace{Frames: frames}

	for i, event := range le {
		if event == nil {
			continue
		}
		le[i] = event.Interface("stack_trace", st)
	}

	return le
}

func (le LoggerEvents) StatusCode(code int) LoggerEvents {
	for i, event := range le {
		if event == nil {
			continue
		}

		le[i] = event.Int("status_code", code)
	}

	return le
}
