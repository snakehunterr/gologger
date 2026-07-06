package gologger

import (
	"context"

	"github.com/snakehunterr/gologger/types"
)

// ctxKey — приватный тип ключа, чтобы избежать коллизий с другими
// пакетами, кладущими значения в тот же context.Context.
type ctxKey int

const (
	userContextKey ctxKey = iota
	traceContextKey
)

// WithUserContext возвращает новый context.Context, несущий uc.
// Используйте это в middleware/хендлере, где известен пользователь:
//
//	ctx = gologger.WithUserContext(ctx, types.UserContext{
//		UserID:    userID,
//		UserRoles: roles,
//	})
func WithUserContext(ctx context.Context, uc types.UserContext) context.Context {
	return context.WithValue(ctx, userContextKey, uc)
}

// UserContextFromContext достаёт UserContext, ранее положенный через
// WithUserContext. ok == false, если значение отсутствует.
func UserContextFromContext(ctx context.Context) (types.UserContext, bool) {
	uc, ok := ctx.Value(userContextKey).(types.UserContext)
	return uc, ok
}

// WithTraceContext позволяет явно задать TraceContext, минуя OTel-спан —
// например, если trace_id пришёл из внешнего заголовка (traceparent),
// а не из локально запущенного span'а.
func WithTraceContext(ctx context.Context, tc types.TraceContext) context.Context {
	return context.WithValue(ctx, traceContextKey, tc)
}

// TraceContextFromContext достаёт TraceContext, ранее положенный через
// WithTraceContext.
func TraceContextFromContext(ctx context.Context) (types.TraceContext, bool) {
	tc, ok := ctx.Value(traceContextKey).(types.TraceContext)
	return tc, ok
}
