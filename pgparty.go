package pgparty

import (
	"context"
	"fmt"
	"log"
	"path"
	"runtime"
	"strings"

	"github.com/jmoiron/sqlx"
)

func WithTx(ctx context.Context, f func(context.Context) error) error {
	s, err := ShardFromContext(ctx)
	if err != nil {
		return fmt.Errorf("WithTx: %w", err)
	}
	return s.Store.WithTx(ctx, func(stx *PgStore) error {
		ctxTx := WithShard(ctx, Shard{s.ID, stx})
		return f(ctxTx)
	})
}

func WithTxInShard(ctx context.Context, shardId string, f func(context.Context) error) error {
	olds, err := ShardFromContext(ctx)
	if err == nil && olds.ID != shardId && olds.Store.tx != nil {
		return fmt.Errorf("WithTxInShard %q: transaction in the other shard was initiated before: %q", shardId, olds.ID)
	}

	if err == nil && olds.ID == shardId {
		return WithTx(ctx, f)
	}

	ctx, err = SelectShard(ctx, shardId)
	if err != nil {
		return fmt.Errorf("WithTxInShard: %w", err)
	}
	return WithTx(ctx, f)
}

type PgStore struct {
	Store

	db     *sqlx.DB
	tx     *sqlx.Tx
	schema string

	trace bool
}

func NewPgStore(db *sqlx.DB, schema string) *PgStore {
	if schema == "" {
		panic("schema is empty")
	}
	ret := &PgStore{
		db:     db,
		schema: schema,
	}

	ret.Init()

	return ret
}

func (sr *PgStore) Close() {
	sr.db.Close()
}

func (sr PgStore) String() string {
	return sr.db.DriverName()
}

func (sr PgStore) Schema() string {
	return sr.schema
}

// SetUnsafe sets a version of Tx which will silently succeed to scan when
// columns in the SQL result have no fields in the destination struct.
func (sr *PgStore) SetUnsafe() {
	sr.tx = sr.tx.Unsafe()
}

func (sr PgStore) WithTx(ctx context.Context, f func(storeCopy *PgStore) error) (err error) {
	// если вызов внутри store с существующей транзакцией - выполняем внутри нее
	if sr.tx != nil {
		if e := f(&sr); e != nil {
			return e
		}
		return nil
	}

	// иначе создаем и выполняем транзакцию
	return sr.WithBeginTx(ctx, f)
}

func IdentifyPanic() string {
	var name, file string
	var line int
	var pc [16]uintptr

	n := runtime.Callers(3, pc[:])
	cnt := 0
	for _, pc := range pc[:n] {
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		file, line = fn.FileLine(pc)
		fnname := fn.Name()
		if strings.HasPrefix(fnname, "runtime.") {
			continue
		}
		fnname = fmt.Sprintf("%s(%s:%d)", path.Base(fnname), file, line)
		if name == "" {
			name = fnname
		} else {
			name = name + ", " + fnname
		}
		cnt++
		if cnt == 5 {
			break
		}
	}
	if name != "" {
		return name
	}

	return fmt.Sprintf("pc:%x", pc[:n])
}

func (sr PgStore) WithBeginTx(ctx context.Context, f func(storeCopy *PgStore) error) (err error) {
	var newTx *sqlx.Tx
	if tx, e := sr.db.BeginTxx(ctx, nil); e != nil {
		return e
	} else {
		newTx = tx
	}

	nst := sr
	nst.tx = newTx

	commit := false
	defer func() {
		if r := recover(); r != nil || !commit {
			if r != nil {
				log.Printf("!!! TRANSACTION PANIC !!! : %s\n%s", r, IdentifyPanic())
			}
			if e := newTx.Rollback(); e != nil {
				err = e
			} else if r != nil {
				err = fmt.Errorf("transaction panic: %s", r)
			}
		} else if commit {
			if e := newTx.Commit(); e != nil {
				err = e
			}
		}
	}()

	nstp := &nst

	sch := sr.schema
	q := fmt.Sprintf(`SELECT set_config('search_path', '%s', true)`, sch)
	if IsLoggingQuery(ctx) {
		log.Println(q)
	}
	if rows, e := nstp.tx.QueryxContext(ctx, q); e != nil {
		return e
	} else {
		_ = rows.Close()
	}

	if e := f(nstp); e != nil {
		return e
	}

	commit = true
	return nil
}

func (sr PgStore) Begin(ctx context.Context) (*PgStore, error) {
	var err error
	if sr.tx == nil {
		sr.tx, err = sr.db.BeginTxx(ctx, nil)
	}
	return &sr, err
}

func (sr PgStore) Commit() error {
	if sr.tx == nil {
		return ErrorNoTransaction{}
	}
	return sr.tx.Commit()
}

func (sr PgStore) Rollback() error {
	if sr.tx == nil {
		return ErrorNoTransaction{}
	}
	return sr.tx.Rollback()
}

func (sr PgStore) Tx() *sqlx.Tx {
	return sr.tx
}
