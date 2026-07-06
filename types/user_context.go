package types

import (
	"strconv"
	"strings"

	"github.com/getsentry/sentry-go"
	otellog "go.opentelemetry.io/otel/log"
)

type UserContext struct {
	ID        int64    `json:"user_id"`
	IPAddress string   `json:"user_ip_address"`
	Roles     []string `json:"user_roles"`
}

func (c *UserContext) ToSentryUser() (user *sentry.User, ok bool) {
	if c == nil {
		return nil, false
	}

	return &sentry.User{
		ID:        strconv.FormatInt(c.ID, 10),
		IPAddress: c.IPAddress,
		Data: map[string]string{
			"roles": strings.Join(c.Roles, "/"),
		},
	}, true
}

func (c *UserContext) ToOTELAttributes() []otellog.KeyValue {
	if c == nil {
		return nil
	}

	return []otellog.KeyValue{
		otellog.Int64("user.id", c.ID),
		otellog.String("user.ip_address", c.IPAddress),
		otellog.String("user.roles", strings.Join(c.Roles, "/")),
	}
}
