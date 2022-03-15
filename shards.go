package pgparty

import (
	"sync"

	"github.com/jmoiron/sqlx"
)

type Shards struct {
	sync.RWMutex
	m map[string]Shard
}

func NewShards() *Shards {
	return &Shards{
		m: make(map[string]Shard),
	}
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
