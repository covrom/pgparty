package pgparty

import (
	"database/sql"
	"sync"
)

type Shard struct {
	ID     string
	DB     *sql.DB
	Schema string
}

// TODO: Shard transaction and prep methods

type Shards struct {
	sync.RWMutex
	m map[string]Shard
}

func NewShards() *Shards {
	return &Shards{
		m: make(map[string]Shard),
	}
}

func (s *Shards) SetShard(id string, db *sql.DB, schema string) {
	s.Lock()
	defer s.Unlock()
	s.m[id] = Shard{id, db, schema}
}

func (s *Shards) ShardByID(id string) (Shard, bool) {
	s.RLock()
	defer s.RUnlock()
	ret, ok := s.m[id]
	return ret, ok
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
