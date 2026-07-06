package types

import (
	"strconv"
	"time"

	otellog "go.opentelemetry.io/otel/log"
)

type LoggerMessage struct {
	Time         time.Time     `json:"time"`
	Message      string        `json:"message"`
	Service      string        `json:"service"`
	Module       string        `json:"module"`
	Level        string        `json:"level"`
	Error        string        `json:"error"`
	ErrorType    string        `json:"error_type"`
	UserContext  *UserContext  `json:"user_context"`
	TraceContext *TraceContext `json:"trace_context"`
	StatusCode   *int          `json:"status_code"`
	FiberCtx     *FiberContext `json:"fiber_context"`
}

func (m *LoggerMessage) ToSentryTags() (tags map[string]string, ok bool) {
	if m == nil {
		return nil, false
	}

	tags = map[string]string{
		"service": m.Service,
		"module":  m.Module,
		"level":   m.Level,
	}

	if m.StatusCode != nil {
		tags["status_code"] = strconv.Itoa(*m.StatusCode)
	}

	return tags, true
}

func (m *LoggerMessage) ToOTELAttributes() []otellog.KeyValue {
	if m == nil {
		return nil
	}

	attrs := []otellog.KeyValue{
		otellog.String("service_name", m.Service),
		otellog.String("module_name", m.Module),
	}

	if m.StatusCode != nil {
		attrs = append(
			attrs,
			otellog.Int("status_code", *m.StatusCode),
		)
	}

	return attrs
}
