package pgparty

import (
	"context"
	"fmt"
	"sync"

	"github.com/jmoiron/sqlx"
)

type Shards struct {
	sync.RWMutex
	m map[string]Shard
}

func NewShards(ctx context.Context) (*Shards, context.Context) {
	shs := &Shards{
		m: make(map[string]Shard),
	}
	return shs, WithShards(ctx, shs)
}

func SetShard(ctx context.Context, id string, db *sqlx.DB, schema string) (Shard, error) {
	shs, err := ShardsFromContext(ctx)
	if err != nil {
		return Shard{}, err
	}
	return shs.SetShard(id, db, schema), nil
}

func ShardByID(ctx context.Context, id string) (Shard, error) {
	shs, err := ShardsFromContext(ctx)
	if err != nil {
		return Shard{}, err
	}
	ret, ok := shs.ShardByID(id)
	if ok {
		return ret, nil
	}
	return Shard{}, fmt.Errorf("shard %q not found", id)
}

func SelectShard(ctx context.Context, id string) (context.Context, error) {
	s, err := ShardByID(ctx, id)
	if err != nil {
		return ctx, err
	}
	return WithShard(ctx, s), nil
}

func DeleteShard(ctx context.Context, id string) error {
	shs, err := ShardsFromContext(ctx)
	if err != nil {
		return err
	}
	shs.DeleteShard(id)
	return nil
}

func WalkShards(ctx context.Context, f func(Shard) error) error {
	shs, err := ShardsFromContext(ctx)
	if err != nil {
		return err
	}
	return shs.Walk(f)
}

func (s *Shards) SetShard(id string, db *sqlx.DB, schema string) Shard {
	s.Lock()
	defer s.Unlock()
	sh := Shard{id, NewPgStore(db, schema)}
	s.m[id] = sh
	return sh
}

func (s *Shards) ShardByID(id string) (Shard, bool) {
	s.RLock()
	defer s.RUnlock()
	ret, ok := s.m[id]
	return ret, ok
}

func (s *Shards) DeleteShard(id string) {
	s.Lock()
	defer s.Unlock()
	delete(s.m, id)
}

func (s *Shards) Walk(f func(Shard) error) error {
	s.RLock()
	defer s.RUnlock()
	for _, v := range s.m {
		if err := f(v); err != nil {
			return err
		}
	}
	return nil
}

type CtxShards struct{}

func WithShards(ctx context.Context, s *Shards) context.Context {
	return context.WithValue(ctx, CtxShards{}, s)
}

func ShardsFromContext(ctx context.Context) (*Shards, error) {
	if v, ok := ctx.Value(CtxShards{}).(*Shards); ok {
		return v, nil
	}
	return nil, fmt.Errorf("context does not contain shards")
}
