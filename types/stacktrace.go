package types

import (
	"strings"

	"github.com/getsentry/sentry-go"
)

type StackFrame struct {
	Function string `json:"function"`
	Module   string `json:"module"`
	Filename string `json:"filename"`
	Lineno   int    `json:"lineno"`
}

type Stacktrace struct {
	Frames []StackFrame `json:"frames"`
}

// Generate SentryStackTrace
func (s *Stacktrace) ToSentryStacktrace() (*sentry.Stacktrace, bool) {
	if s == nil || len(s.Frames) == 0 {
		return nil, false
	}

	frames := make([]sentry.Frame, len(s.Frames))
	for i, f := range s.Frames {
		frames[i] = sentry.Frame{
			Function: f.Function,
			Module:   f.Module,
			Filename: f.Filename,
			Lineno:   f.Lineno,
			// FIXME: same output is if InApp: true
			InApp: !strings.Contains(f.Filename, "/pkg/mod/") && !strings.HasPrefix(f.Function, "runtime."),
		}
	}

	return &sentry.Stacktrace{Frames: frames}, true
}
