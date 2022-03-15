package pgparty

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"reflect"
	"runtime"

	"github.com/jackc/pgx/v4"
	"github.com/jmoiron/sqlx"
)

// PrepareQuery подготавливает запрос, в котором могут быть указаны имена моделей из языка го &ModelName или полей :FieldName, :ModelName.FieldName
// Имя таблицы many-2-many связи подставляется по шаблону &ModelName.Many2ManyFieldName, в ней два id-поля по шаблону :ModelName1_id и :ModelName2_id
// Prepare всегда делается на уровне базы, поэтому, нужно всегдла явно затаскивать его в контекст нужной транзакции через tx.Stmtx(psql)
// Можно вызывать Prepare сколько угодно раз, но реально рассчитываться он будет один раз, в остальные разы будет браться из кэша.
func (sr *PgStore) PrepareQuery(ctx context.Context, query string) (string, error) {
	s, err := ShardFromContext(ctx)
	if err != nil {
		return "", fmt.Errorf("PrepareQuery: %w", err)
	}
	schema := s.Store.Schema()

	repls, _, err := sr.AnalyzeAndReplaceQuery(query, schema)
	if err != nil {
		return "", err
	}

	res := Rebind(repls)

	if sr.trace {
		log.Println(res)
	}

	return res, nil
}

func Get[T Storable](ctx context.Context, query string, dest *T, args ...interface{}) error {
	s, err := ShardFromContext(ctx)
	if err != nil {
		_, file, no, ok := runtime.Caller(1)
		if ok {
			log.Printf("Get error at %s line %d: %s", file, no, err)
		}
		return fmt.Errorf("Get: %w", err)
	}
	return s.Store.PrepGet(ctx, query, dest, args...)
}

func (sr *PgStore) PrepGet(ctx context.Context, query string, dest interface{}, args ...interface{}) error {
	if sr.tx == nil {
		return sr.WithTx(ctx, func(stx *PgStore) error {
			return stx.PrepGet(ctx, query, dest, args...)
		})
	}

	var err error
	query, args, err = In(query, args...)
	if err != nil {
		return err
	}
	q, err := sr.PrepareQuery(ctx, query)
	if err != nil {
		return err
	}
	if IsLoggingQuery(ctx) {
		log.Println(q)
	}
	if IsSimpleProtocol(ctx) {
		args = append([]interface{}{pgx.QuerySimpleProtocol(true)}, args...)
		log.Println("PrepGet QuerySimpleProtocol: ", q)
	}

	err = sr.tx.Unsafe().GetContext(ctx, dest, q, args...)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("PrepGet query %q error: %s", q, err)
	}
	return err
}

func Select[T Storable](ctx context.Context, query string, dest *[]T, args ...interface{}) error {
	s, err := ShardFromContext(ctx)
	if err != nil {
		_, file, no, ok := runtime.Caller(1)
		if ok {
			log.Printf("PrepSelect error at %s line %d: %s", file, no, err)
		}
		return fmt.Errorf("PrepSelect: %w", err)
	}
	return s.Store.PrepSelect(ctx, query, dest, args...)
}

func (sr *PgStore) PrepSelect(ctx context.Context, query string, dest interface{}, args ...interface{}) error {
	if sr.tx == nil {
		return sr.WithTx(ctx, func(stx *PgStore) error {
			return stx.PrepSelect(ctx, query, dest, args...)
		})
	}
	var err error
	query, args, err = In(query, args...)
	if err != nil {
		return err
	}
	q, err := sr.PrepareQuery(ctx, query)
	if err != nil {
		return err
	}

	if IsSimpleProtocol(ctx) {
		args = append([]interface{}{pgx.QuerySimpleProtocol(true)}, args...)
		log.Println("PrepSelect QuerySimpleProtocol: ", query)
	}

	if IsLoggingQuery(ctx) {
		log.Println(q)
	}

	err = sr.tx.Unsafe().SelectContext(ctx, dest, q, args...)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("PrepSelect query %q error: %s", q, err)
	}
	return err
}

func Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	s, err := ShardFromContext(ctx)
	if err != nil {
		_, file, no, ok := runtime.Caller(1)
		if ok {
			log.Printf("PrepExec error at %s line %d: %s", file, no, err)
		}
		return nil, fmt.Errorf("PrepExec: %w", err)
	}
	return s.Store.PrepExec(ctx, query, args...)
}

func (sr *PgStore) PrepExec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if sr.tx == nil {
		var res sql.Result
		err := sr.WithTx(ctx, func(stx *PgStore) error {
			var e error
			res, e = stx.PrepExec(ctx, query, args...)
			return e
		})
		return res, err
	}
	var err error
	query, args, err = In(query, args...)
	if err != nil {
		return nil, err
	}
	q, err := sr.PrepareQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	if IsLoggingQuery(ctx) {
		log.Println(q)
	}

	res, err := sr.tx.ExecContext(ctx, q, args...)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("PrepExec query %q error: %s", q, err)
	}

	return res, err
}

func Query(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	s, err := ShardFromContext(ctx)
	if err != nil {
		_, file, no, ok := runtime.Caller(1)
		if ok {
			log.Printf("PrepQueryx error at %s line %d: %s", file, no, err)
		}
		return nil, fmt.Errorf("PrepQueryx: %w", err)
	}
	return s.Store.PrepQueryx(ctx, query, args...)
}

func (sr *PgStore) PrepQueryx(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	if sr.tx == nil {
		return nil, errors.New("PrepQueryx outside a transaction not supported")
	}

	var err error
	query, args, err = In(query, args...)
	if err != nil {
		return nil, err
	}
	q, err := sr.PrepareQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	if IsLoggingQuery(ctx) {
		log.Println(q)
	}
	if IsSimpleProtocol(ctx) {
		args = append([]interface{}{pgx.QuerySimpleProtocol(true)}, args...)
		log.Println("PrepQueryx QuerySimpleProtocol: ", query)
	}
	res, err := sr.tx.QueryxContext(ctx, q, args...)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("PrepQueryx query %q error: %s", q, err)
	}
	return res, err
}

func SelectCursorWalk[T Storable](ctx context.Context, cursorName, selectQuery string, destSlice *[]T, fetchSize int,
	f func(destSlice interface{}) error, args ...interface{},
) error {
	s, err := ShardFromContext(ctx)
	if err != nil {
		_, file, no, ok := runtime.Caller(1)
		if ok {
			log.Printf("SelectCursorWalk error at %s line %d: %s", file, no, err)
		}
		return fmt.Errorf("SelectCursorWalk: %w", err)
	}
	return s.Store.PrepSelectCursorWalk(ctx, cursorName, selectQuery, destSlice, fetchSize, f, args...)
}

func (sr *PgStore) PrepSelectCursorWalk(ctx context.Context, cursorName, selectQuery string, destSlice interface{}, fetchSize int,
	f func(destSlice interface{}) error, args ...interface{},
) error {
	slt := reflect.TypeOf(destSlice)
	if slt.Kind() != reflect.Ptr || slt.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("destSlice must be a pointer to slice")
	}
	var err error
	selectQuery, args, err = In(selectQuery, args...)
	if err != nil {
		return err
	}
	q, err := sr.PrepareQuery(ctx, selectQuery)
	if err != nil {
		return err
	}
	if IsLoggingQuery(ctx) {
		log.Println(q)
	}
	if IsSimpleProtocol(ctx) {
		args = append([]interface{}{pgx.QuerySimpleProtocol(true)}, args...)
	}

	dstCursorQuery := fmt.Sprintf(`DECLARE %s CURSOR FOR %s`, cursorName, q)
	fetchQuery := fmt.Sprintf(`FETCH %d FROM %s`, fetchSize, cursorName)

	if sr.tx != nil {
		if _, e := sr.tx.ExecContext(ctx, dstCursorQuery); e != nil {
			return e
		}
		for {
			reflect.Indirect(reflect.ValueOf(destSlice)).SetLen(0)
			if e := sr.tx.Unsafe().SelectContext(ctx, destSlice, fetchQuery, args...); e != nil {
				return e
			}
			if reflect.Indirect(reflect.ValueOf(destSlice)).Len() == 0 {
				break
			}
			if e := f(destSlice); e != nil {
				return e
			}
		}
	} else {
		return errors.New("PrepSelectCursorWalk outside a transaction not supported")
	}

	return nil
}
