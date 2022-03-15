package pgparty

import (
	"context"
	"fmt"
)

type Shard struct {
	ID    string
	Store *PgStore
}

type CtxShard struct{}

func WithShard(ctx context.Context, s Shard) context.Context {
	return context.WithValue(ctx, CtxShard{}, s)
}

func ShardFromContext(ctx context.Context) (Shard, error) {
	if v, ok := ctx.Value(CtxShard{}).(Shard); ok {
		return v, nil
	}
	return Shard{}, fmt.Errorf("context does not contain a shard")
}
