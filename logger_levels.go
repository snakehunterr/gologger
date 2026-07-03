package gologger

import "github.com/rs/zerolog"

type LoggerLevel = string

var (
	LevelTrace = zerolog.LevelTraceValue
	LevelDebug = zerolog.LevelDebugValue
	LevelInfo  = zerolog.LevelInfoValue
	LevelWarn  = zerolog.LevelWarnValue
	LevelError = zerolog.LevelErrorValue
)
