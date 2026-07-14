package types

import (
	"path"
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
			InApp:    inApp(f.Filename),
		}
	}

	return &sentry.Stacktrace{Frames: frames}, true
}

func inApp(s string) bool {
	if strings.Contains(s, "github.com") {
		return false
	}
	if path.Base(path.Dir(s)) == "runtime" {
		return false
	}

	return true
}
