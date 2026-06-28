package loggers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"github.com/rs/zerolog"
)

// ── Config ─────────────────────────────────────────────────────────────────

type OpenObserveLoggerConfig struct {
	// Level is the minimum log level to send (trace/debug/info/warn/error).
	Level string `json:"level"`
	level zerolog.Level

	// CollectorEndpoint is the OTel Collector address, e.g. "localhost:4318".
	// Logs are sent to http://<CollectorEndpoint>/v1/logs.
	// The collector is responsible for forwarding to OpenObserve with the
	// correct stream-name header.
	CollectorEndpoint string `json:"collector_endpoint"`

	// ServiceName is attached to every log record as a resource attribute.
	ServiceName string `json:"service_name"`
}

func (c *OpenObserveLoggerConfig) Validate() error {
	if c == nil {
		return ConfigValidateError("*OpenObserveLoggerConfig is nil")
	}
	var err error
	if c.level, err = zerolog.ParseLevel(c.Level); err != nil {
		return ConfigValidateError(fmt.Sprintf("zerolog.ParseLevel: %s", err))
	}
	if c.CollectorEndpoint == "" {
		return ConfigValidateError("c.CollectorEndpoint is empty")
	}
	if c.ServiceName == "" {
		return ConfigValidateError("c.ServiceName is empty")
	}
	return nil
}

// ── Writer ─────────────────────────────────────────────────────────────────

// openObserveWriter implements io.Writer.
// zerolog writes a JSON line; we parse it and emit an otellog.Record.
type openObserveWriter struct {
	otelLogger otellog.Logger

	// zerolog level → OTel severity
	severityMap map[string]otellog.Severity
}

func newOpenObserveWriter(otelLogger otellog.Logger) *openObserveWriter {
	return &openObserveWriter{
		otelLogger: otelLogger,
		severityMap: map[string]otellog.Severity{
			zerolog.LevelTraceValue: otellog.SeverityTrace,
			zerolog.LevelDebugValue: otellog.SeverityDebug,
			zerolog.LevelInfoValue:  otellog.SeverityInfo,
			zerolog.LevelWarnValue:  otellog.SeverityWarn,
			zerolog.LevelErrorValue: otellog.SeverityError,
			zerolog.LevelFatalValue: otellog.SeverityFatal,
		},
	}
}

func (w *openObserveWriter) Write(p []byte) (int, error) {
	// Parse the JSON line zerolog produced
	var fields map[string]any
	if err := json.Unmarshal(p, &fields); err != nil {
		return 0, fmt.Errorf("json.Unmarshal: %w", err)
	}

	var r otellog.Record

	// Timestamp
	if ts, ok := fields[zerolog.TimestampFieldName].(string); ok {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			r.SetTimestamp(t)
		}
	} else {
		r.SetTimestamp(time.Now())
	}

	// Severity from zerolog level field
	if lvl, ok := fields[zerolog.LevelFieldName].(string); ok {
		if sev, ok := w.severityMap[lvl]; ok {
			r.SetSeverity(sev)
			r.SetSeverityText(lvl)
		}
	}

	// Body from zerolog message field
	if msg, ok := fields[zerolog.MessageFieldName].(string); ok {
		r.SetBody(otellog.StringValue(msg))
	}

	// All other fields become OTel log attributes
	attrs := make([]otellog.KeyValue, 0, len(fields))
	for k, v := range fields {
		if k == zerolog.TimestampFieldName ||
			k == zerolog.LevelFieldName ||
			k == zerolog.MessageFieldName {
			continue
		}
		switch val := v.(type) {
		case string:
			attrs = append(attrs, otellog.String(k, val))
		case float64:
			attrs = append(attrs, otellog.Float64(k, val))
		case bool:
			attrs = append(attrs, otellog.Bool(k, val))
		default:
			attrs = append(attrs, otellog.String(k, fmt.Sprintf("%v", val)))
		}
	}
	r.AddAttributes(attrs...)

	w.otelLogger.Emit(context.Background(), r)
	return len(p), nil
}

// ── Logger ─────────────────────────────────────────────────────────────────

type OpenObserveLogger struct {
	config         *OpenObserveLoggerConfig
	loggerProvider *sdklog.LoggerProvider
	logger         zerolog.Logger
}

func NewOpenObserveLogger(config *OpenObserveLoggerConfig) (*OpenObserveLogger, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config.Validate: %w", err)
	}

	ctx := context.Background()

	// OTLP HTTP log exporter → OTel Collector
	exporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(config.CollectorEndpoint),
		otlploghttp.WithURLPath("/v1/logs"),
		otlploghttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("otlploghttp.New: %w", err)
	}

	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)

	otelLogger := loggerProvider.Logger(config.ServiceName)

	l := &OpenObserveLogger{
		config:         config,
		loggerProvider: loggerProvider,
	}

	l.logger = zerolog.New(newOpenObserveWriter(otelLogger)).
		Level(config.level).
		With().
		Timestamp().
		Logger()

	return l, nil
}

func (l *OpenObserveLogger) Trace() *zerolog.Event { return l.logger.Trace() }
func (l *OpenObserveLogger) Debug() *zerolog.Event { return l.logger.Debug() }
func (l *OpenObserveLogger) Info() *zerolog.Event  { return l.logger.Info() }
func (l *OpenObserveLogger) Warn() *zerolog.Event  { return l.logger.Warn() }
func (l *OpenObserveLogger) Error() *zerolog.Event { return l.logger.Error() }

// Close flushes all pending log records to the collector.
func (l *OpenObserveLogger) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := l.loggerProvider.Shutdown(ctx); err != nil {
		return fmt.Errorf("loggerProvider.Shutdown: %w", err)
	}
	return nil
}
