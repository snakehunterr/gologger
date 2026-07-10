package gologger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"go.uber.org/zap/zapcore"
)

func TestNewLoggerService(t *testing.T) {
	ls := NewLoggerService("svc")

	if ls.ServiceName != "svc" {
		t.Fatalf("ServiceName = %q, want %q", ls.ServiceName, "svc")
	}
	if len(ls.loggers) != 0 {
		t.Fatalf("loggers = %d, want 0", len(ls.loggers))
	}
}

func TestWithConsoleLogger(t *testing.T) {
	ls := NewLoggerService("svc")

	if err := ls.WithConsoleLogger(&ConsoleLoggerConfig{Level: LevelDebug}); err != nil {
		t.Fatalf("WithConsoleLogger: %v", err)
	}
	if ls.consoleLogger == nil {
		t.Fatal("consoleLogger is nil")
	}
	if len(ls.loggers) != 1 {
		t.Fatalf("loggers = %d, want 1", len(ls.loggers))
	}
}

func TestWithConsoleLogger_InvalidLevel(t *testing.T) {
	ls := NewLoggerService("svc")

	if err := ls.WithConsoleLogger(&ConsoleLoggerConfig{Level: "not-a-level"}); err == nil {
		t.Fatal("expected error for invalid level, got nil")
	}
}

func TestWithConsoleLogger_ReplacesExisting(t *testing.T) {
	ls := NewLoggerService("svc")

	if err := ls.WithConsoleLogger(&ConsoleLoggerConfig{Level: LevelDebug}); err != nil {
		t.Fatalf("WithConsoleLogger (1st): %v", err)
	}
	first := ls.consoleLogger

	if err := ls.WithConsoleLogger(&ConsoleLoggerConfig{Level: LevelInfo}); err != nil {
		t.Fatalf("WithConsoleLogger (2nd): %v", err)
	}

	if ls.consoleLogger == first {
		t.Fatal("expected a new consoleLogger instance after re-configuring")
	}
	if len(ls.loggers) != 1 {
		t.Fatalf("loggers = %d, want 1 (replace, not append)", len(ls.loggers))
	}
}

func TestWithFileLogger(t *testing.T) {
	dir := t.TempDir()
	ls := NewLoggerService("svc")

	if err := ls.WithFileLogger(&FileLoggerConfig{Level: LevelInfo, LogDir: dir}); err != nil {
		t.Fatalf("WithFileLogger: %v", err)
	}

	ls.Info().Msg("hello")

	if err := ls.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("log files = %d, want 1", len(entries))
	}

	content, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(content), `"message":"hello"`) {
		t.Fatalf("log content missing message: %s", content)
	}
}

func TestWithSentryLogger_InvalidConfig(t *testing.T) {
	ls := NewLoggerService("svc")

	if err := ls.WithSentryLogger(&SentryLoggerConfig{}); err == nil {
		t.Fatal("expected error for missing DSN, got nil")
	}
}

func TestWithOpenObserveLogger_InvalidConfig(t *testing.T) {
	ls := NewLoggerService("svc")

	if err := ls.WithOpenObserveLogger(&OpenObserveLoggerConfig{}); err == nil {
		t.Fatal("expected error for missing collector endpoint, got nil")
	}
}

func TestWithLoggers_NilReceiver(t *testing.T) {
	var ls *LoggerService

	if err := ls.WithConsoleLogger(&ConsoleLoggerConfig{Level: LevelInfo}); err == nil {
		t.Fatal("WithConsoleLogger: expected error on nil receiver")
	}
	if err := ls.WithFileLogger(&FileLoggerConfig{Level: LevelInfo}); err == nil {
		t.Fatal("WithFileLogger: expected error on nil receiver")
	}
	if err := ls.WithSentryLogger(&SentryLoggerConfig{}); err == nil {
		t.Fatal("WithSentryLogger: expected error on nil receiver")
	}
	if err := ls.WithOpenObserveLogger(&OpenObserveLoggerConfig{}); err == nil {
		t.Fatal("WithOpenObserveLogger: expected error on nil receiver")
	}
}

func TestWithModuleName(t *testing.T) {
	parent := NewLoggerService("svc")
	if err := parent.WithConsoleLogger(&ConsoleLoggerConfig{Level: LevelInfo}); err != nil {
		t.Fatalf("WithConsoleLogger: %v", err)
	}

	child := parent.WithModuleName("mod")

	if child.ServiceName != "svc" {
		t.Fatalf("child.ServiceName = %q, want %q", child.ServiceName, "svc")
	}
	if child.ModuleName != "mod" {
		t.Fatalf("child.ModuleName = %q, want %q", child.ModuleName, "mod")
	}
	if child.consoleLogger != parent.consoleLogger {
		t.Fatal("expected child to share parent's consoleLogger instance")
	}
	if len(parent.childs) != 1 {
		t.Fatalf("parent.childs = %d, want 1", len(parent.childs))
	}
}

func TestWithModuleName_NilReceiver(t *testing.T) {
	var ls *LoggerService

	if child := ls.WithModuleName("mod"); child != nil {
		t.Fatal("expected nil child from nil receiver")
	}
}

func TestNewChild(t *testing.T) {
	parent := NewLoggerService("svc")
	if err := parent.WithConsoleLogger(&ConsoleLoggerConfig{Level: LevelInfo}); err != nil {
		t.Fatalf("WithConsoleLogger: %v", err)
	}

	child, err := parent.NewChild("child-svc")
	if err != nil {
		t.Fatalf("NewChild: %v", err)
	}

	if child.ServiceName != "child-svc" {
		t.Fatalf("child.ServiceName = %q, want %q", child.ServiceName, "child-svc")
	}
	if child.consoleLogger != parent.consoleLogger {
		t.Fatal("expected child to share parent's consoleLogger instance")
	}
	if len(parent.childs) != 1 {
		t.Fatalf("parent.childs = %d, want 1", len(parent.childs))
	}
}

func TestNewChild_NilReceiver(t *testing.T) {
	var ls *LoggerService

	child, err := ls.NewChild("x")
	if child != nil || err != nil {
		t.Fatalf("NewChild on nil receiver = (%v, %v), want (nil, nil)", child, err)
	}
}

func TestClose_CascadesToChildren(t *testing.T) {
	dir := t.TempDir()
	parent := NewLoggerService("svc")
	if err := parent.WithFileLogger(&FileLoggerConfig{Level: LevelInfo, LogDir: dir}); err != nil {
		t.Fatalf("WithFileLogger: %v", err)
	}
	child := parent.WithModuleName("mod")

	if err := parent.Close(); err != nil {
		t.Fatalf("parent.Close: %v", err)
	}

	// child shares the same (now-closed) fileLogger; closing again must be a no-op, not an error.
	if err := child.Close(); err != nil {
		t.Fatalf("child.Close after parent closed: %v", err)
	}
}

func TestClose_NilReceiver(t *testing.T) {
	var ls *LoggerService

	if err := ls.Close(); err != nil {
		t.Fatalf("Close on nil receiver: %v", err)
	}
}

func TestGetMinZerologLevel(t *testing.T) {
	ls := NewLoggerService("svc")

	if lvl := ls.GetMinZerologLevel(); lvl != zerolog.Disabled {
		t.Fatalf("GetMinZerologLevel with no backends = %v, want Disabled", lvl)
	}

	if err := ls.WithConsoleLogger(&ConsoleLoggerConfig{Level: LevelWarn}); err != nil {
		t.Fatalf("WithConsoleLogger: %v", err)
	}
	if err := ls.WithFileLogger(&FileLoggerConfig{Level: LevelDebug, LogDir: t.TempDir()}); err != nil {
		t.Fatalf("WithFileLogger: %v", err)
	}
	defer ls.Close()

	if lvl := ls.GetMinZerologLevel(); lvl != zerolog.DebugLevel {
		t.Fatalf("GetMinZerologLevel = %v, want DebugLevel (lowest of warn/debug)", lvl)
	}
}

func TestGetMinZerologLevel_NilReceiver(t *testing.T) {
	var ls *LoggerService

	if lvl := ls.GetMinZerologLevel(); lvl != zerolog.Disabled {
		t.Fatalf("GetMinZerologLevel on nil receiver = %v, want Disabled", lvl)
	}
}

func TestGetMinZapLevel(t *testing.T) {
	ls := NewLoggerService("svc")
	if err := ls.WithConsoleLogger(&ConsoleLoggerConfig{Level: LevelWarn}); err != nil {
		t.Fatalf("WithConsoleLogger: %v", err)
	}

	if lvl := ls.GetMinZapLevel(); lvl != zapcore.WarnLevel {
		t.Fatalf("GetMinZapLevel = %v, want WarnLevel", lvl)
	}
}

func TestGetModuleName(t *testing.T) {
	ls := NewLoggerService("svc")
	child := ls.WithModuleName("mod")

	if got := child.GetModuleName(); got != "mod" {
		t.Fatalf("GetModuleName = %q, want %q", got, "mod")
	}

	var nilLs *LoggerService
	if got := nilLs.GetModuleName(); got != "" {
		t.Fatalf("GetModuleName on nil receiver = %q, want empty", got)
	}
}

func TestTracerMeter_NoOpWhenUnconfigured(t *testing.T) {
	ls := NewLoggerService("svc")

	if ls.Tracer() == nil {
		t.Fatal("Tracer() returned nil, want no-op tracer")
	}
	if ls.Meter() == nil {
		t.Fatal("Meter() returned nil, want no-op meter")
	}

	var nilLs *LoggerService
	if nilLs.Tracer() == nil {
		t.Fatal("Tracer() on nil receiver returned nil, want no-op tracer")
	}
	if nilLs.Meter() == nil {
		t.Fatal("Meter() on nil receiver returned nil, want no-op meter")
	}
}

func TestEventLevels_NilReceiver(t *testing.T) {
	var ls *LoggerService

	if ls.Trace() != nil {
		t.Fatal("Trace() on nil receiver should return nil LoggerEvents")
	}
	if ls.Debug() != nil {
		t.Fatal("Debug() on nil receiver should return nil LoggerEvents")
	}
	if ls.Info() != nil {
		t.Fatal("Info() on nil receiver should return nil LoggerEvents")
	}
	if ls.Warn() != nil {
		t.Fatal("Warn() on nil receiver should return nil LoggerEvents")
	}
	if ls.Error() != nil {
		t.Fatal("Error() on nil receiver should return nil LoggerEvents")
	}
}

func TestEventBroadcast_TagsAllBackends(t *testing.T) {
	dir := t.TempDir()
	ls := NewLoggerService("svc")
	if err := ls.WithFileLogger(&FileLoggerConfig{Level: LevelInfo, LogDir: dir}); err != nil {
		t.Fatalf("WithFileLogger: %v", err)
	}

	child := ls.WithModuleName("mymodule")
	child.Info().Str("key", "val").Msg("hello world")

	if err := ls.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	for _, want := range []string{
		`"service":"svc"`,
		`"module":"mymodule"`,
		`"key":"val"`,
		`"message":"hello world"`,
		`"caller_name"`,
	} {
		if !strings.Contains(string(content), want) {
			t.Fatalf("log content missing %s: %s", want, content)
		}
	}
}
