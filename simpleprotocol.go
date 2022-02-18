package pgparty

import (
	"context"
)

type preferSimpleProtocol struct{}

func WithSimpleProtocol(ctx context.Context) context.Context {
	return context.WithValue(ctx, preferSimpleProtocol{}, true)
}

func IsSimpleProtocol(ctx context.Context) bool {
	if spctx, ok := ctx.Value(preferSimpleProtocol{}).(bool); ok {
		return spctx
	}
	return false
}
