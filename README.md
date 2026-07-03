# GoLogger

A flexible, multi-backend logging service for Go applications with support for console, file, Sentry, and OpenObserve (logs + traces + metrics) simultaneously — all behind one unified interface.

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.18-blue)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

## Features

- 🎯 **Multiple Backends** - Console, File (with daily rotation), Sentry, and OpenObserve support
- 🔄 **Unified Interface** - Single API for all logging backends
- 📅 **Daily Log Rotation** - Automatic daily rotation for file logger
- 🔗 **Error Chain Support** - Automatic error type extraction from wrapped errors
- 🧵 **Thread-Safe** - Concurrent-safe logging operations
- ⚡ **High Performance** - Built on top of the blazing-fast [zerolog](https://github.com/rs/zerolog) library
- 📊 **Structured Logging** - Rich, structured log events with context
- 🔭 **Distributed Tracing** - OpenTelemetry spans via OpenObserve, with trace-aware log correlation
- 📈 **Metrics** - OpenTelemetry counters/histograms/gauges via OpenObserve
- 🌳 **Module Hierarchy** - Derive scoped child loggers (`WithModuleName`) that inherit all configured backends
- 🏗️ **Clean Architecture** - Well-separated concerns with interface-based design

## Installation

```bash
go get -u github.com/snakehunterr/gologger
```

## Quick start

```go
package main

import (
 "database/sql"

 "github.com/snakehunterr/gologger"
)

func main() {
 // Create a new logger service: (serviceName, moduleName)
 logger := gologger.NewLoggerService("my-service", "main")
 defer func() {
  if err := logger.Close(); err != nil {
   panic(err)
  }
 }()

 // Add console logger for development
 if err := logger.WithConsoleLogger(&gologger.ConsoleLoggerConfig{
  Level: gologger.LevelDebug,
 }); err != nil {
  panic(err)
 }

 // Add file logger with daily rotation
 if err := logger.WithFileLogger(&gologger.FileLoggerConfig{
  Level:  gologger.LevelInfo,
  LogDir: "./logs",
 }); err != nil {
  panic(err)
 }

 // Add Sentry logger for error tracking (optional)
 if err := logger.WithSentryLogger(&gologger.SentryLoggerConfig{
  DSN:         "https://your-sentry-dsn@sentry.io/123",
  Level:       gologger.LevelError,
  AppVersion:  "1.0.0",
  Environment: "production",
 }); err != nil {
  panic(err)
 }

 // Add OpenObserve for centralized logs, traces, and metrics (optional)
 if err := logger.WithOpenObserveLogger(&gologger.OpenObserveLoggerConfig{
  Level:             gologger.LevelInfo,
  CollectorEndpoint: "localhost:4318", // your OTel Collector OTLP/HTTP endpoint
  ServiceName:       "my-service",
 }); err != nil {
  panic(err)
 }

 // Log to all configured backends at once
 logger.Info().
  Str("action", "startup").
  Msg("application started successfully")

 // Log errors with automatic error type extraction
 if err := sql.ErrNoRows; err != nil {
  logger.Error().Err(err).Msg("operation failed")
 }
}
```

## Configuration

### ConsoleLogger

Outputs formatted logs to stdout with colored output and custom time format.

```go
type ConsoleLoggerConfig struct {
 Level string `json:"level"` // "trace", "debug", "info", "warn", "error", "fatal", "panic"
}
```

Example:

```go
logger.WithConsoleLogger(&gologger.ConsoleLoggerConfig{
 Level: gologger.LevelDebug,
})
```

Output format: `HH:MM:SS LEVEL message key=value`

### FileLogger

Writes logs to files with automatic daily rotation. Files are named using the format **YYYY-MM-DD.log**.

```go
type FileLoggerConfig struct {
 Level  string `json:"level"`   // Log level
 LogDir string `json:"log_dir"` // Directory for log files (default: ".")
}
```

Example:

```go
logger.WithFileLogger(&gologger.FileLoggerConfig{
 Level:  gologger.LevelInfo,
 LogDir: "./logs",
})
```

Features:

- Automatic daily file rotation at midnight
- Creates log directory if it doesn't exist
- Appends to existing log files
- Thread-safe file operations with mutex protection

### SentryLogger

Sends error and event data to Sentry for monitoring and alerting.

```go
type SentryLoggerConfig struct {
 DSN         string `json:"dsn"`         // Sentry DSN
 Level       string `json:"level"`       // Minimum log level
 AppVersion  string `json:"app_version"` // Application version
 Environment string `json:"environment"` // Environment name
}
```

Example:

```go
logger.WithSentryLogger(&gologger.SentryLoggerConfig{
 DSN:         "https://your-dsn@sentry.io/project-id",
 Level:       gologger.LevelError,
 AppVersion:  "1.0.0",
 Environment: "production",
})
```

Features:

- Maps zerolog levels to Sentry severity levels
- Captures error type and context
- Includes service and module tags
- Different handling for errors vs regular events
- Structured context for better Sentry grouping

### OpenObserveLogger

Ships logs, traces, and metrics to [OpenObserve](https://openobserve.ai/) via an OpenTelemetry Collector (OTLP/HTTP). One backend covers all three signal types.

```go
type OpenObserveLoggerConfig struct {
 Level             string `json:"level"`              // Minimum log level
 CollectorEndpoint string `json:"collector_endpoint"` // e.g. "localhost:4318"
 ServiceName       string `json:"service_name"`       // Used as the OTel resource service.name
}
```

Example:

```go
logger.WithOpenObserveLogger(&gologger.OpenObserveLoggerConfig{
 Level:             gologger.LevelInfo,
 CollectorEndpoint: "localhost:4318",
 ServiceName:       "my-service",
})
```

Features:

- Logs are shipped as OTel log records over `/v1/logs`, with `service`/`module` attached as attributes
- Traces and metrics are shipped over `/v1/traces` and `/v1/metrics` respectively
- All three signals share one `service.name` OTel resource, so they group correctly under the same service in OpenObserve
- Exposes the underlying `Tracer()` and `Meter()` for manual instrumentation
- `Close()` flushes and shuts down all three OTel providers (logs, traces, metrics)

> **Collector required:** `gologger` talks OTLP/HTTP to a collector, not directly to OpenObserve. See [Setting up the OTel Collector](#setting-up-the-otel-collector) below.

## Usage guide

### Basic logging

```go
logger.Trace().Msg("trace message")
logger.Debug().Msg("debug message")
logger.Info().Msg("info message")
logger.Warn().Msg("warning message")
logger.Error().Msg("error message")
```

### Structured logging

Add context to your logs with structured fields:

```go
logger.Info().
 Str("user_id", "123").
 Str("action", "login").
 Msg("user authentication")
```

### Formatted string fields

```go
logger.Info().
 Strf("duration", "%dms", 150).
 Strf("user_agent", "Mozilla/5.0 (%s)", os).
 Msg("request processed")
```

### Error logging

The logger automatically extracts and includes error type information from the innermost error in the chain:

```go
// Original error chain: fmt.Errorf("context: %w", sql.ErrNoRows)
err := someFunction()
logger.Error().
 Err(err). // automatically adds "error_type": "ErrNoRows"
 Str("query", "SELECT * FROM users").
 Msg("database query failed")
```

### Formatted messages and errors

```go
// Using Msgf for formatted strings
logger.Info().Msgf("processing order #%d for user %s", orderID, username)

// Using Errf for formatted error messages
logger.Error().Errf("failed to process payment #%d: %v", paymentID, err)
```

### Chaining events

```go
// Method chaining
logger.Info().
 Str("service", "api").
 Str("endpoint", "/users").
 Msg("request completed")

// Building events in steps
event := logger.Info()
event = event.Str("service", "api")
event = event.Str("endpoint", "/users")
event.Msg("request completed")
```

### Scoping loggers by module

`WithModuleName` derives a child `LoggerService` that shares the same configured backends (console, file, Sentry, OpenObserve) but tags every event with a different `module`. This is the recommended way to scope logging per package, per handler, or per request.

```go
logger := gologger.NewLoggerService("my-service", "main")
// ... configure backends on logger ...

dbLogger := logger.WithModuleName("database")
dbLogger.Info().Msg("connected to database") // module=database

authLogger := logger.WithModuleName("auth")
authLogger.Error().Err(err).Msg("token validation failed") // module=auth
```

`logger.Close()` also closes every child derived via `WithModuleName`, so you only need one `defer` at the top level.

### Distributed tracing

If `WithOpenObserveLogger` is configured, `Tracer()` returns a standard [OpenTelemetry](https://opentelemetry.io/) `trace.Tracer` for starting spans. If OpenObserve isn't configured, `Tracer()` returns a safe no-op tracer, so call sites never need a nil check.

```go
ctx, span := logger.Tracer().Start(ctx, "process-order")
defer span.End()

span.SetAttributes(attribute.String("order.id", orderID))

if err != nil {
 span.RecordError(err)
}
```

Pass the returned `ctx` down into anything you call inside the span — that's what builds the parent/child waterfall you see in OpenObserve.

### Trace-aware logging

Use the `*Ctx` variants of the logging methods to automatically stamp `trace_id`/`span_id` from the active span onto every emitted log event. This lets you jump from a log line straight to its trace in OpenObserve.

```go
logger.InfoCtx(ctx).Msg("order processed")
logger.ErrorCtx(ctx).Err(err).Msg("order processing failed")
```

Available for every level: `TraceCtx`, `DebugCtx`, `InfoCtx`, `WarnCtx`, `ErrorCtx`. If `ctx` carries no active span, these behave identically to their non-`Ctx` counterparts.

### Metrics

`Meter()` returns a standard OpenTelemetry `metric.Meter`. Create instruments once (e.g. at startup or lazily-cached) and reuse them — don't recreate an instrument on every call.

```go
requestsCounter, err := logger.Meter().Int64Counter("requests_total")
if err != nil {
 logger.Error().Err(err).Msg("failed to create counter")
}

// later, per request:
requestsCounter.Add(ctx, 1, metric.WithAttributes(
 attribute.String("status", "success"),
))
```

Like `Tracer()`, `Meter()` returns a no-op meter if OpenObserve isn't configured.

### Using with Fiber (or any web framework)

`gologger` is framework-agnostic and built on `context.Context`. With [Fiber](https://gofiber.io/), bridge `*fiber.Ctx` to `context.Context` via `c.UserContext()` in a middleware, so every handler gets request-scoped tracing and log correlation for free:

```go
func LoggerMiddleware(base *gologger.LoggerService) fiber.Handler {
 return func(c *fiber.Ctx) error {
  ctx, span := base.Tracer().Start(c.UserContext(), c.Route().Name)
  defer span.End()

  c.SetUserContext(ctx)
  c.Locals("logger", base.WithModuleName(c.Route().Name))

  return c.Next()
 }
}

func Handler(c *fiber.Ctx) error {
 logger := c.Locals("logger").(*gologger.LoggerService)
 ctx := c.UserContext()

 logger.InfoCtx(ctx).Msg("handling request")
 // ...
 return c.JSON(fiber.Map{"ok": true})
}
```

> Always use `c.UserContext()`, never `*fiber.Ctx` itself, for anything passed into tracing or downstream calls — `*fiber.Ctx` is recycled by fasthttp after the handler returns.

### Setting up the OTel Collector

`OpenObserveLogger` ships data to a local [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/), which forwards it to OpenObserve. A minimal collector config:

```yaml
receivers:
  otlp:
    protocols:
      http:
        endpoint: "0.0.0.0:4318"
processors:
  batch:
    timeout: 5s
exporters:
  otlphttp/openobserve_logs:
    endpoint: "http://openobserve:5080/api/default"
    headers:
      Authorization: "Basic <base64 user:pass>"
      stream-name: "my_service_logs"
    tls:
      insecure: true
  otlphttp/openobserve_traces:
    endpoint: "http://openobserve:5080/api/default"
    headers:
      Authorization: "Basic <base64 user:pass>"
      stream-name: "my_service_traces"
    tls:
      insecure: true
  otlphttp/openobserve_metrics:
    endpoint: "http://openobserve:5080/api/default"
    headers:
      Authorization: "Basic <base64 user:pass>"
      stream-name: "my_service_metrics"
    tls:
      insecure: true
service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlphttp/openobserve_logs]
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlphttp/openobserve_traces]
    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlphttp/openobserve_metrics]
```

> Use **separate streams per signal type**. Logs, traces, and metrics have incompatible schemas — routing them into the same OpenObserve stream can cause ingestion conflicts.

## Core Components

### Log levels

`gologger` re-exports zerolog's level values as its own constants, so consumers don't need to import `github.com/rs/zerolog` just to configure a level:

```go
type LoggerLevel = string

var (
 LevelTrace = "trace"
 LevelDebug = "debug"
 LevelInfo  = "info"
 LevelWarn  = "warn"
 LevelError = "error"
)
```

Use them anywhere a `Level` field is expected:

```go
logger.WithConsoleLogger(&gologger.ConsoleLoggerConfig{
 Level: gologger.LevelDebug,
})
```

Since `LoggerLevel` is a plain string alias, raw string literals (`"debug"`, `"info"`, etc.) work identically — the constants just give you compile-time-friendly names and IDE autocomplete.

### Logger interface

Every backend implements a common interface for consistency:

```go
type Logger interface {
 Trace() *zerolog.Event
 Debug() *zerolog.Event
 Info() *zerolog.Event
 Warn() *zerolog.Event
 Error() *zerolog.Event
 Close() error
}
```

### LoggerService

The main orchestrator that manages multiple logger backends and broadcasts log events to all configured backends simultaneously.

- `NewLoggerService(serviceName, moduleName)` - Create a new service, tagged with a service and module name
- `WithConsoleLogger(config)` - Configure console output
- `WithFileLogger(config)` - Configure file output with rotation
- `WithSentryLogger(config)` - Configure Sentry integration
- `WithOpenObserveLogger(config)` - Configure OpenObserve logs/traces/metrics
- `WithModuleName(name)` - Derive a child logger scoped to a different module, sharing the same backends
- `Trace()` through `Error()` - Create log events at different levels, broadcast to all backends
- `TraceCtx(ctx)` through `ErrorCtx(ctx)` - Same, but also stamp `trace_id`/`span_id` from the active span
- `Tracer()` - Get the OTel tracer for manual span creation (no-op if OpenObserve isn't configured)
- `Meter()` - Get the OTel meter for manual instrument creation (no-op if OpenObserve isn't configured)
- `Close()` - Gracefully shut down this service and all of its child loggers

**Important: Each backend type can only be configured once per `LoggerService`. Subsequent calls to the same `With*Logger` method are silently ignored.**

### LoggerEvents

A collection of `*zerolog.Event` (one per configured backend) that broadcasts method calls to all of them. Methods include:

- `Msg(msg)` - Finalize and send the message
- `Msgf(format, args...)` - Send a formatted message
- `Str(key, val)` - Add a string field
- `Strf(key, format, args...)` - Add a formatted string field
- `Err(err)` - Add an error with automatic type extraction
- `Errf(format, args...)` - Add a formatted error

## Error type extraction

When using `.Err()`, the system:

1. Unwraps the error chain to find the innermost error
2. Extracts the type name using reflection
3. Adds both the error and the innermost error type as structured fields

This makes it easy to filter and group errors by their root cause in log aggregation tools.

## Thread safety

- **Console Logger**: Inherently thread-safe (writes to stdout)
- **File Logger**: Uses mutex locks to protect file operations and rotation
- **Sentry Logger**: Uses Sentry's built-in concurrency handling
- **OpenObserve Logger**: Uses OpenTelemetry SDK's built-in batching/concurrency handling
- **LoggerService**: Thread-safe as long as the underlying configured loggers are thread-safe

## Requirements

- Go 1.18 or higher
- Dependencies:
  - `github.com/rs/zerolog` - High-performance logging library
  - `github.com/getsentry/sentry-go` - Sentry error tracking
  - `go.opentelemetry.io/otel` and related OTLP/HTTP exporter, SDK log/trace/metric packages - OpenObserve logs, traces, metrics

## Best practices

Always close the logger service when your application shuts down:

```go
defer logger.Close()
```

Set appropriate log levels for each environment, using `gologger.Level*` constants (or the equivalent raw strings):

- Development: `gologger.LevelTrace` or `gologger.LevelDebug`
- Staging: `gologger.LevelDebug` or `gologger.LevelInfo`
- Production: `gologger.LevelInfo` or `gologger.LevelWarn`

Always use `.Err()` for errors to get automatic error type extraction:

```go
logger.Error().Err(err).Msg("operation failed")
```

Prefer `WithModuleName` over creating separate `LoggerService` instances when you want per-package/per-request scoping — it reuses the already-configured backends instead of re-establishing new connections.

When tracing is enabled, prefer the `*Ctx` logging methods (`InfoCtx`, `ErrorCtx`, etc.) inside traced code paths so logs and traces stay correlated in OpenObserve.

Register a global OTel error handler so failures in the OpenObserve pipeline (auth issues, network problems, etc.) don't fail silently:

```go
otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
 log.Printf("otel error: %v\n", err)
}))
```

## Acknowledgments

- [zerolog](https://github.com/rs/zerolog) - The high-performance logging library this project is built on
- [sentry-go](https://github.com/getsentry/sentry-go) - Official Sentry SDK for Go
- [OpenTelemetry Go SDK](https://github.com/open-telemetry/opentelemetry-go) - Tracing, metrics, and logs instrumentation
- [OpenObserve](https://openobserve.ai/) - Observability backend for logs, traces, and metrics
