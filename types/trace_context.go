package types

import (
	"encoding/hex"
	"fmt"

	"github.com/getsentry/sentry-go"
	"go.opentelemetry.io/otel/trace"
)

type TraceContext struct {
	TraceID string `json:"trace_id"`
	SpanID  string `json:"span_id"`
}

func (c *TraceContext) ToSentryPropagationContext() (ctx *sentry.PropagationContext, ok bool) {
	if c == nil {
		return nil, false
	}
	if c.TraceID == "" || c.SpanID == "" {
		return nil, false
	}

	traceID, err := traceIDFromHex(c.TraceID)
	if err != nil {
		return nil, false
	}

	spanID, err := spanIDFromHex(c.SpanID)
	if err != nil {
		return nil, false
	}

	return &sentry.PropagationContext{
		TraceID: traceID,
		SpanID:  spanID,
	}, true
}

func (c *TraceContext) ToSpanContextConfig() (cfg *trace.SpanContextConfig, ok bool) {
	if c == nil {
		return nil, false
	}

	if c.TraceID == "" || c.SpanID == "" {
		return nil, false
	}

	traceID, err := trace.TraceIDFromHex(c.TraceID)
	if err != nil {
		return nil, false
	}

	spanID, err := trace.SpanIDFromHex(c.SpanID)
	if err != nil {
		return nil, false
	}

	return &trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	}, true
}

// traceIDFromHex decodes a 32-character hex string (as produced by
// OTel's trace.TraceID.String()) into a sentry.TraceID.
func traceIDFromHex(s string) (sentry.TraceID, error) {
	var id sentry.TraceID
	b, err := hex.DecodeString(s)
	if err != nil {
		return id, fmt.Errorf("hex.DecodeString: %w", err)
	}
	if len(b) != len(id) {
		return id, fmt.Errorf("invalid trace id length: got %d bytes, want %d", len(b), len(id))
	}
	copy(id[:], b)
	return id, nil
}

// spanIDFromHex decodes a 16-character hex string (as produced by
// OTel's trace.SpanID.String()) into a sentry.SpanID.
func spanIDFromHex(s string) (sentry.SpanID, error) {
	var id sentry.SpanID
	b, err := hex.DecodeString(s)
	if err != nil {
		return id, fmt.Errorf("hex.DecodeString: %w", err)
	}
	if len(b) != len(id) {
		return id, fmt.Errorf("invalid span id length: got %d bytes, want %d", len(b), len(id))
	}
	copy(id[:], b)
	return id, nil
}
