package loggers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/rs/zerolog"
	"github.com/snakehunterr/gologger/types"
)

type OpenObserveLoggerConfig struct {
	Level             string `json:"level"`
	level             zerolog.Level
	CollectorEndpoint string `json:"collector_endpoint"`
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
	return nil
}

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
	var msg types.LoggerMessage
	if err := json.Unmarshal(p, &msg); err != nil {
		return 0, fmt.Errorf("json.Unmarshal: %w", err)
	}

	var r otellog.Record

	r.SetTimestamp(msg.Time)

	if sev, ok := w.severityMap[msg.Level]; ok {
		r.SetSeverity(sev)
		r.SetSeverityText(msg.Level)
	}

	r.SetBody(otellog.StringValue(msg.Message))

	// All other fields become OTel log attributes
	r.AddAttributes(msg.ToOTELAttributes()...)
	r.AddAttributes(msg.UserContext.ToOTELAttributes()...)
	r.AddAttributes(msg.FiberCtx.ToOTELAttributes()...)

	ctx := context.Background()
	if cfg, ok := msg.TraceContext.ToSpanContextConfig(); ok {
		ctx = trace.ContextWithSpanContext(
			ctx,
			trace.NewSpanContext(*cfg),
		)
	}

	w.otelLogger.Emit(ctx, r)
	return len(p), nil
}

type OpenObserveLogger struct {
	serviceName string

	config *OpenObserveLoggerConfig

	resource *resource.Resource

	loggerProvider *sdklog.LoggerProvider
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider

	logger zerolog.Logger
	tracer trace.Tracer
	meter  metric.Meter

	closed bool
}

func NewOpenObserveLogger(serviceName string, config *OpenObserveLoggerConfig) (*OpenObserveLogger, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config.Validate: %w", err)
	}

	ctx := context.Background()

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("resource.New: %w", err)
	}

	l := &OpenObserveLogger{
		serviceName: serviceName,
		config:      config,
		resource:    res,
	}

	if err := l.setupLogs(ctx); err != nil {
		return nil, fmt.Errorf("setupLogs: %w", err)
	}

	if err := l.setupTraces(ctx); err != nil {
		return nil, fmt.Errorf("setupTraces: %w", err)
	}

	if err := l.setupMetrics(ctx); err != nil {
		return nil, fmt.Errorf("setupMetrics: %w", err)
	}

	return l, nil
}

func (l *OpenObserveLogger) setupLogs(ctx context.Context) error {
	// OTLP HTTP log exporter → OTel Collector
	exporter, err := otlploghttp.New(
		ctx,
		otlploghttp.WithEndpoint(l.config.CollectorEndpoint),
		otlploghttp.WithURLPath("/v1/logs"),
		otlploghttp.WithInsecure(),
	)
	if err != nil {
		return fmt.Errorf("otlploghttp.New: %w", err)
	}

	l.loggerProvider = sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(l.resource),
	)

	otelLogger := l.loggerProvider.Logger(l.serviceName)

	l.logger = zerolog.New(newOpenObserveWriter(otelLogger)).
		Level(l.config.level).
		With().
		Timestamp().
		Logger()

	return nil
}

func (l *OpenObserveLogger) setupTraces(ctx context.Context) error {
	// OTLP HTTP trace exporter → OTel Collector
	exporter, err := otlptracehttp.New(
		ctx,
		otlptracehttp.WithEndpoint(l.config.CollectorEndpoint),
		otlptracehttp.WithURLPath("/v1/traces"),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return fmt.Errorf("otlptracehttp.New: %w", err)
	}

	l.tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(l.resource),
	)

	l.tracer = l.tracerProvider.Tracer(l.serviceName)

	return nil
}

func (l *OpenObserveLogger) setupMetrics(ctx context.Context) error {
	// OTLP HTTP metric exporter → OTel Collector
	exporter, err := otlpmetrichttp.New(
		ctx,
		otlpmetrichttp.WithEndpoint(l.config.CollectorEndpoint),
		otlpmetrichttp.WithURLPath("/v1/metrics"),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		return fmt.Errorf("otlpmetrichttp.New: %w", err)
	}

	l.meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
		sdkmetric.WithResource(l.resource),
	)

	l.meter = l.meterProvider.Meter(l.serviceName)

	return nil
}

// Tracer returns the OTel tracer, for starting spans:
//
//	ctx, span := logger.Tracer().Start(ctx, "operation-name")
//	defer span.End()
func (l *OpenObserveLogger) Tracer() trace.Tracer {
	return l.tracer
}

// Meter returns the OTel meter, for creating instruments:
//
//	counter, err := logger.Meter().Int64Counter("requests_total")
func (l *OpenObserveLogger) Meter() metric.Meter {
	return l.meter
}

func (l *OpenObserveLogger) Trace() *zerolog.Event { return l.logger.Trace() }
func (l *OpenObserveLogger) Debug() *zerolog.Event { return l.logger.Debug() }
func (l *OpenObserveLogger) Info() *zerolog.Event  { return l.logger.Info() }
func (l *OpenObserveLogger) Warn() *zerolog.Event  { return l.logger.Warn() }
func (l *OpenObserveLogger) Error() *zerolog.Event { return l.logger.Error() }

func (l *OpenObserveLogger) GetLevel() zerolog.Level {
	return l.logger.GetLevel()
}

// Close flushes all pending log records to the collector.
func (l *OpenObserveLogger) Close() error {
	if l.closed {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := l.loggerProvider.Shutdown(ctx); err != nil {
		return fmt.Errorf("loggerProvider.Shutdown: %w", err)
	}

	if err := l.tracerProvider.Shutdown(ctx); err != nil {
		return fmt.Errorf("tracerProvider.Shutdown: %w", err)
	}

	if err := l.meterProvider.Shutdown(ctx); err != nil {
		return fmt.Errorf("meterProvider.Shutdown: %w", err)
	}

	l.closed = true
	return nil
}

func (l *OpenObserveLogger) GetConfig() *OpenObserveLoggerConfig {
	if l == nil {
		return nil
	}

	return l.config
}
