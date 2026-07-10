# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`gologger` (module `github.com/snakehunterr/gologger`) is a Go library — not a service — that provides a unified logging interface over multiple backends (console, file, Sentry, OpenObserve) built on top of `zerolog`. There is no `main` package; consumers `go get` this module.

## Commands

```bash
go build ./...        # compile everything
go vet ./...           # static checks
go test ./...          # run all tests
go test ./... -run Test_LoggerServiceNoNilPanic -v   # run a single test
gofmt -l .              # check formatting (no dedicated lint config in repo)
```

There is no Makefile, CI workflow, or linter config in this repo — the commands above are the full toolchain.

## Architecture

### Package layout

- Root package `gologger` — public API (`LoggerService`, `LoggerEvents`, context helpers, re-exported config/level types).
- `loggers/` — one file per backend (`console_logger.go`, `file_logger.go`, `sentry_logger.go`, `open_observe_logger.go`), each implementing the `Logger` interface (`logger_interface.go`): `Trace/Debug/Info/Warn/Error() *zerolog.Event`, `GetLevel()`, `Close()`.
- `types/` — plain data structs shipped as structured log fields (`UserContext`, `TraceContext`, `FiberContext`, `Stacktrace`, `LoggerMessage`) plus converters to Sentry/OTel shapes. `types.go` at the root re-exports these as type aliases so consumers don't need to import the `types` subpackage directly.

### The broadcast pattern

`LoggerService` (`logger_service.go`) holds a slice of configured backends (`ls.loggers []Logger`, capacity 4: console/file/sentry/openobserve). Each `With*Logger(config)` method constructs and stores exactly one backend of that kind — **calling `With*Logger` again on a service that already has that backend replaces it** (closes the old one first), it isn't additive across calls in the way one might expect from a multi-backend API.

Calling `logger.Info()` (etc.) fans out to every configured backend via `callToLoggers`, producing a `LoggerEvents` — a `[]*zerolog.Event`, one per backend. Every method on `LoggerEvents` (`Str`, `Err`, `Ctx`, `Stack`, `FiberCtx`, `StatusCode`, ...) loops over that slice and applies itself to each event, then `Msg`/`Msgf` finalizes and flushes all of them. This is how one `logger.Error().Err(err).Msg("...")` call reaches console + file + Sentry + OpenObserve simultaneously. When adding a new chainable method to `LoggerEvents`, follow the same nil-safe loop-and-reassign pattern (`le[i] = event.XXX(...)`) used by the existing methods.

### Nil-safety as a design requirement

Every exported method on `*LoggerService` and `LoggerEvents` explicitly handles a nil receiver (see the `if ls == nil` / `if le == nil` guards throughout `logger_service.go` and `logger_events.go`, and the dedicated test `Test_LoggerServiceNoNilPanic` in `logger_events_test.go`). This is intentional: call chains like `logger.Error().Err(err).Ctx(ctx).Msg(...)` must never panic even if `logger` itself is `nil` (e.g. an unconfigured/optional dependency in a consuming app). Preserve this guarantee when touching either file — new chain methods need the same nil check at the top.

### Module hierarchy (`WithModuleName` / `NewChild`)

`WithModuleName(name)` derives a child `LoggerService` that shares the parent's already-configured backend instances (console/file/sentry are shared pointers; OpenObserve too) but tags events with a different `ModuleName`. `NewChild(name)` is similar but re-creates a fresh `OpenObserveLogger` from the parent's config (new OTel providers) rather than sharing the instance. Children are tracked in `ls.childs` so `Close()` on the root cascades to every derived child.

### Context propagation

`ctx_values.go` defines two unexported `context.Context` keys (`userContextKey`, `traceContextKey`) with `With*Context`/`*ContextFromContext` accessor pairs. `LoggerEvents.Ctx(ctx)` reads both off the context and attaches them as structured fields; if no explicit `TraceContext` was set, it falls back to pulling the OTel span from `ctx` via `trace.SpanContextFromContext`. `LoggerEvents.FiberCtx(*fiber.Ctx)` separately captures HTTP request metadata (method, path, IP, proxy headers) — this is a snapshot taken eagerly (`types.NewFiberCtx`), which matters because `*fiber.Ctx` is recycled by fasthttp after the handler returns.

### Backend-specific writers

Sentry and OpenObserve backends don't call their SDKs directly from log calls — instead they configure a `zerolog.Logger` whose `io.Writer` is a custom type (`SentryWriter`, `openObserveWriter`) that receives the already-serialized JSON log line, unmarshals it back into `types.LoggerMessage`, and translates it into the target SDK's native event/record shape (Sentry event/exception, OTel `otellog.Record`). When adding fields to `LoggerMessage`, update the corresponding `To*` converter methods on the `types` structs so the new field actually reaches Sentry/OTel.

### Error type extraction

`LoggerEvents.Err(err)` walks the error chain with `errors.Unwrap` to the innermost error and reflects on its type name, attaching it as the `error_type` field — this lets log aggregation group errors by root cause regardless of how many `fmt.Errorf("...: %w", err)` wrappers were added along the way.

### Config validation

Every `*Config` type in `loggers/` has a `Validate()` method that parses the string `Level` into a `zerolog.Level` and checks required fields (DSN, Environment, etc.), called internally by the corresponding `New*Logger` constructor. Configs are otherwise plain structs — there's no separate builder/options pattern.
