package pgparty

import "context"

type logQuery struct{}

func WithLoggingQuery(ctx context.Context) context.Context {
	return context.WithValue(ctx, logQuery{}, true)
}

func IsLoggingQuery(ctx context.Context) bool {
	if spctx, ok := ctx.Value(logQuery{}).(bool); ok {
		return spctx
	}
	return false
}
