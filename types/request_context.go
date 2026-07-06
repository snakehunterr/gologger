package types

import (
	"github.com/getsentry/sentry-go"
	"github.com/gofiber/fiber/v2"
	otellog "go.opentelemetry.io/otel/log"
)

// FiberContext собирает все поля HTTP-запроса, которые нужно приложить
// к логу/событию Sentry для диагностики ошибок API.
type FiberContext struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Query  string `json:"query"`

	IP        string `json:"ip"`
	Host      string `json:"host"`
	UserAgent string `json:"user_agent"`
	Referer   string `json:"referer"`
	Protocol  string `json:"protocol"`

	XForwardedFor         string `json:"x_forwarded_for"`
	XForwardedProto       string `json:"x_forwarded_proto"`
	XRealIP               string `json:"x_real_ip"`
	XRequestID            string `json:"x_request_id"`
	XOriginalForwardedFor string `json:"x_original_forwarded_for"`
}

func NewFiberCtx(c *fiber.Ctx) *FiberContext {
	if c == nil {
		return nil
	}

	return &FiberContext{
		Method: c.Method(),
		Path:   c.Path(),
		Query:  c.OriginalURL(),

		IP:        c.IP(),
		Host:      c.Hostname(),
		UserAgent: c.Get("User-Agent"),
		Referer:   c.Get("Referer"),
		Protocol:  c.Protocol(),

		XForwardedFor:         c.Get("X-Forwarded-For"),
		XForwardedProto:       c.Get("X-Forwarded-Proto"),
		XRealIP:               c.Get("X-Real-IP"),
		XRequestID:            c.Get("X-Request-ID"),
		XOriginalForwardedFor: c.Get("X-Original-Forwarded-For"),
	}
}

func (c *FiberContext) ToSentryTags() (tags map[string]string, ok bool) {
	if c == nil {
		return nil, false
	}

	return map[string]string{
		"method": c.Method,
		"path":   c.Path,
	}, true
}

func (c *FiberContext) ToSentryRequestInfoContext() (context sentry.Context, ok bool) {
	if c == nil {
		return nil, false
	}

	return sentry.Context{
		"query":      c.Query,
		"host":       c.Host,
		"user_agent": c.UserAgent,
		"referer":    c.Referer,
		"protocol":   c.Protocol,
		"ip":         c.IP,
	}, true
}

func (c *FiberContext) ToSentryProxyInfoContext() (context sentry.Context, ok bool) {
	if c == nil {
		return nil, false
	}

	return sentry.Context{
		"X-Forwarded-For":          c.XForwardedFor,
		"X-Forwarded-Proto":        c.XForwardedProto,
		"X-Real-IP":                c.XRealIP,
		"X-Request-ID":             c.XRequestID,
		"X-Original-Forwarded-For": c.XOriginalForwardedFor,
	}, true
}

func (c *FiberContext) ToOTELAttributes() []otellog.KeyValue {
	if c == nil {
		return nil
	}

	attrs := []otellog.KeyValue{
		otellog.String("request.method", c.Method),
		otellog.String("request.path", c.Path),
		otellog.String("request.query", c.Query),
		otellog.String("request.host", c.Host),
		otellog.String("request.user_agent", c.UserAgent),
		otellog.String("request.referer", c.Referer),
		otellog.String("request.protocol", c.Protocol),
		otellog.String("request.ip", c.IP),
		otellog.String("request.proxy.x_forwarded_for", c.XForwardedFor),
		otellog.String("request.proxy.x_forwarded_proto", c.XForwardedProto),
		otellog.String("request.proxy.x_real_ip", c.XRealIP),
		otellog.String("request.proxy.x_request_id", c.XRequestID),
		otellog.String("request.proxy.x_original_forwarded_for", c.XOriginalForwardedFor),
	}

	return attrs
}
