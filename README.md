# GoLogger

A flexible, multi-backend logging service for Go applications with support for console, file, and Sentry logging simultaneously.

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.18-blue)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

## Features

- 🎯 **Multiple Backends** - Console, File (with daily rotation), and Sentry support
- 🔄 **Unified Interface** - Single API for all logging backends
- 📅 **Daily Log Rotation** - Automatic daily rotation for file logger
- 🔗 **Error Chain Support** - Automatic error type extraction from wrapped errors
- 🧵 **Thread-Safe** - Concurrent-safe logging operations
- ⚡ **High Performance** - Built on top of the blazing-fast [zerolog](https://github.com/rs/zerolog) library
- 📊 **Structured Logging** - Rich, structured log events with context
- 🏗️ **Clean Architecture** - Well-separated concerns with interface-based design

## Installation

```bash
go get github.com/snakehunterr/gologger
```

## Quick start

```go
package main

import (
    "errors"
    "github.com/snakehunterr/gologger/logger"
)

func main() {
    // Create a new logger service
    ls := logger.NewLoggerService()

    // Add console logger for development
    if err := ls.WithConsoleLogger(&logger.ConsoleLoggerConfig{
        Level: "debug",
    }); err != nil {
        panic(err)
    }

    // Add file logger with daily rotation
    if err := ls.WithFileLogger(&logger.FileLoggerConfig{
        Level:  "info",
        LogDir: "./logs",
    }); err != nil {
        panic(err)
    }

    // Add Sentry logger for error tracking (optional)
    if err := ls.WithSentryLogger(&logger.SentryLoggerConfig{
        DSN:         "https://your-sentry-dsn@sentry.io/123",
        Level:       "error",
        AppVersion:  "1.0.0",
        Environment: "production",
    }); err != nil {
        panic(err)
    }

    // Log to all configured backends
    ls.Info().
        Str("service", "api").
        Str("version", "1.0.0").
        Msg("Application started successfully")

    // Log errors with automatic error type extraction
    if err := someOperation(); err != nil {
        ls.Error().Err(err).Msg("Operation failed")
    }

    // Don't forget to close when done
    defer ls.Close()
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
ls.WithConsoleLogger(&logger.ConsoleLoggerConfig{
    Level: "debug",
})
```

Output Format: HH:MM:SS LEVEL message key=value

### FileLogger

Writes logs to files with automatic daily rotation. Files are named using the format **YYYY-MM-DD.log**

```go
type FileLoggerConfig struct {
    Level  string `json:"level"`   // Log level
    LogDir string `json:"log_dir"` // Directory for log files (default: ".")
}
```

Example:

```go
ls.WithFileLogger(&logger.FileLoggerConfig{
    Level:  "info",
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
    DSN         string `json:"dsn"`          // Sentry DSN
    Level       string `json:"level"`        // Minimum log level
    AppVersion  string `json:"app_version"`  // Application version
    Environment string `json:"environment"`  // Environment name
}
```

Example:

```go
ls.WithSentryLogger(&logger.SentryLoggerConfig{
    DSN:         "https://your-dsn@sentry.io/project-id",
    Level:       "error",
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

## Usage guide

### Basic logging

```go
// Different log levels
ls.Trace().Msg("Trace message")
ls.Debug().Msg("Debug message")
ls.Info().Msg("Info message")
ls.Warn().Msg("Warning message")
ls.Error().Msg("Error message")
```

### Structured Logging

Add context to your logs with structured fields:

```go
ls.Info().
    Str("user_id", "123").
    Str("action", "login").
    Msg("User authentication")
```

### Formatted String Fields

```go
ls.Info().
    Strf("duration", "%dms", 150).
    Strf("user_agent", "Mozilla/5.0 (%s)", os).
    Msg("Request processed")
```

### Error Logging

The logger automatically extracts and includes error type information from the innermost error in the chain:

```go
// Original error chain: fmt.Errorf("context: %w", sql.ErrNoRows)
err := someFunction()
ls.Error().
    Err(err).                      // Automatically adds "error_type": "ErrNoRows"
    Str("query", "SELECT * FROM users").
    Msg("Database query failed")
```

### Formatted Messages and Errors

```go
// Using Msgf for formatted strings
ls.Info().Msgf("Processing order #%d for user %s", orderID, username)

// Using Errf for formatted error messages
ls.Error().Errf("Failed to process payment #%d: %v", paymentID, err)
```

### Chaining Events

Build up log events incrementally or chain them together:

```go
// Method chaining
ls.Info().
    Str("service", "api").
    Str("endpoint", "/users").
    Msg("Request completed")

// Building events in steps
event := ls.Info()
event = event.Str("service", "api")
event = event.Str("endpoint", "/users")
event.Msg("Request completed")
```

## Core Components

### Logger Interface

All loggers implement a common interface for consistency:

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

The main orchestrator that manages multiple logger instances and broadcasts log events to all configured backends simultaneously.

- WithConsoleLogger(config) - Configure console output
- WithFileLogger(config) - Configure file output with rotation
- WithSentryLogger(config) - Configure Sentry integration
- Trace() through Error() - Create log events at different levels
- Close() - Gracefully shutdown all loggers

**Important: Each logger type can only be configured once. Subsequent calls will be silently ignored.**

### LoggerEvents

A collection of \*zerolog.Event that broadcasts method calls to all configured loggers. Methods include:

- Msg(msg) - Finalize and send the message
- Msgf(format, args...) - Send formatted message
- Str(key, val) - Add a string field
- Strf(key, format, args...) - Add a formatted string field
- Err(err) - Add an error with automatic type extraction
- Errf(format, args...) - Add a formatted error

## Error Type Extraction

When using .Err(), the system:

1. Unwraps the error chain to find the innermost error
2. Extracts the type name using reflection
3. Adds both the full error chain and the innermost error type as structured fields

This makes it easy to filter and group errors by their root cause in log aggregation tools.

## Thread Safety

- File Logger: Uses mutex locks to protect file operations and rotation
- Console Logger: Inherently thread-safe (writes to stdout)
- Sentry Logger: Uses Sentry's built-in concurrency handling
- LoggerService: Thread-safe as long as underlying loggers are thread-safe

## Requirements

- Go 1.18 or higher
- Dependencies:
  - github.com/rs/zerolog - High-performance logging library
  - github.com/getsentry/sentry-go - Sentry error tracking

## Best Practices

Always close the logger service when your application shuts down:

```go
defer ls.Close()
```

Set appropriate log levels for each environment:

- Development: debug or trace
- Staging: info or debug
- Production: info or warn

Always use .Err() for errors to get automatic error type extraction:

```go
ls.Error().Err(err).Msg("Operation failed")
```

## Acknowledgments

- zerolog - The high-performance logging library this project is built on
- sentry-go - Official Sentry SDK for Go
