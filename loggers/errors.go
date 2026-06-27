package loggers

type LoggerError string

func (err LoggerError) Error() string {
	return string(err)
}

const (
	ErrFileLoggerLogFileIsNil = LoggerError("*FileLogger.file is nil")
)

type ConfigValidateError string

func (err ConfigValidateError) Error() string {
	return string(err)
}
