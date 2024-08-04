package pgparty

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"reflect"
	"runtime"

	"github.com/jackc/pgx/v5"
	"github.com/jmoiron/sqlx"
)

func (sr *PgStore) PrepareQuery(ctx context.Context, query string) (string, error) {
	shs, err := ShardsFromContext(ctx)
	if err != nil {
		return "", fmt.Errorf("PrepareQuery: %w", err)
	}

	repls, _, err := shs.AnalyzeAndReplaceQuery(sr, query)
	if err != nil {
		return "", err
	}

	res := Rebind(repls)

	if sr.trace {
		log.Println(res)
	}

	return res, nil
}

func Get[T any](ctx context.Context, query string, dest *T, args ...interface{}) error {
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

func GetByID[T Storable](ctx context.Context, dest *T, id interface{}) error {
	s, err := ShardFromContext(ctx)
	if err != nil {
		_, file, no, ok := runtime.Caller(1)
		if ok {
			log.Printf("Get error at %s line %d: %s", file, no, err)
		}
		return fmt.Errorf("Get: %w", err)
	}
	return s.Store.PrepGet(ctx, fmt.Sprintf("SELECT * FROM %s WHERE id = ?", (*new(T)).DatabaseName()), dest, id)
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
		args = append([]interface{}{pgx.QueryExecModeSimpleProtocol}, args...)
		log.Println("PrepGet QuerySimpleProtocol: ", q)
	}

	rv := reflect.ValueOf(dest)
	if rv.Kind() != reflect.Pointer {
		return fmt.Errorf("dest is not a pointer")
	}
	rv = rv.Elem()
	if !rv.IsValid() {
		return fmt.Errorf("dest is nil pointer")
	}
	rt := rv.Type()
	if reflect.PointerTo(rt).Implements(reflect.TypeOf((*RowScanner)(nil)).Elem()) {
		rows, err := sr.tx.QueryxContext(ctx, q, args...)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("PrepGet query %q error: %s", q, err)
		}
		if err != nil {
			return err
		}
		defer rows.Close()
		if !rows.Next() {
			if err := rows.Err(); err != nil {
				log.Printf("PrepGet query %q error: %s", q, err)
				return err
			}
			return sql.ErrNoRows
		}

		v := reflect.New(rt).Interface().(RowScanner)

		if err := v.Scan(rows, ""); err != nil {
			return err
		}
		rv.Set(reflect.ValueOf(v).Elem())

		err = nil
	} else {
		err = sr.tx.Unsafe().GetContext(ctx, dest, q, args...)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("PrepGet query %q error: %s", q, err)
		}
	}
	return err
}

func Select[T any](ctx context.Context, query string, dest *[]T, args ...interface{}) error {
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

type RowScanner interface {
	Scan(rows sqlx.ColScanner, prefix string) error
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
		args = append([]interface{}{pgx.QueryExecModeSimpleProtocol}, args...)
		log.Println("PrepSelect QuerySimpleProtocol: ", query)
	}

	if IsLoggingQuery(ctx) {
		log.Println(q)
	}

	rv := reflect.ValueOf(dest)
	if rv.Kind() != reflect.Pointer {
		return fmt.Errorf("dest is not a pointer")
	}
	rv = rv.Elem()
	if !rv.IsValid() || rv.Kind() != reflect.Slice {
		return fmt.Errorf("dest is not a pointer to slice")
	}
	rt := rv.Type().Elem()
	if reflect.PointerTo(rt).Implements(reflect.TypeOf((*RowScanner)(nil)).Elem()) {
		rows, err := sr.tx.QueryxContext(ctx, q, args...)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("PrepSelect query %q error: %s", q, err)
		}
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			v := reflect.New(rt).Interface().(RowScanner)
			err := v.Scan(rows, "")
			if err != nil {
				return err
			}
			rv.Set(reflect.Append(rv, reflect.ValueOf(v).Elem()))
		}
		if err := rows.Err(); err != nil {
			log.Printf("PrepSelect query %q error: %s", q, err)
			return err
		}
		err = nil
	} else {
		err = sr.tx.Unsafe().SelectContext(ctx, dest, q, args...)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("PrepSelect query %q error: %s", q, err)
		}
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
		args = append([]interface{}{pgx.QueryExecModeSimpleProtocol}, args...)
		log.Println("PrepQueryx QuerySimpleProtocol: ", query)
	}
	res, err := sr.tx.QueryxContext(ctx, q, args...)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("PrepQueryx query %q error: %s", q, err)
	}
	return res, err
}

func SelectCursorWalk[T any](ctx context.Context, cursorName, selectQuery string, destSlice *[]T, fetchSize int,
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
		args = append([]interface{}{pgx.QueryExecModeSimpleProtocol}, args...)
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
