package loggers

import (
	"time"

	"github.com/getsentry/sentry-go"
	otellog "go.opentelemetry.io/otel/log"
)

type LoggerMessage struct {
	Time      time.Time `json:"time"`
	Message   string    `json:"message"`
	Service   string    `json:"service"`
	Module    string    `json:"module"`
	Level     string    `json:"level"`
	Error     string    `json:"error"`
	ErrorType string    `json:"error_type"`
}

func (m *LoggerMessage) ToSentryContext() sentry.Context {
	ctx := sentry.Context{}

	if m == nil {
		return ctx
	}

	ctx["service"] = m.Service
	ctx["module"] = m.Module
	ctx["message"] = m.Message
	ctx["level"] = m.Level

	return ctx
}

func (m *LoggerMessage) ToOTELAttrubutes() []otellog.KeyValue {
	if m == nil {
		return nil
	}

	return []otellog.KeyValue{
		otellog.String("service_name", m.Service),
		otellog.String("module", m.Module),
	}
}
