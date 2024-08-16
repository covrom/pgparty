package pgparty

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
)

type RowsGenerator interface {
	QueryRows() (*sqlx.Rows, error)
}

type ModelQueryEngine[T Modeller, R RowsGenerator] struct {
	result []JsonViewer[T]
	err    error
	done   bool
}

func (q *ModelQueryEngine[T, R]) ResponseList(rowsGen R) ([]JsonViewer[T], error) {
	if q.done {
		if q.err != nil {
			return nil, q.err
		}
		return q.result, nil
	}
	q.done = true
	q.result = make([]JsonViewer[T], 0, 100)

	rows, err := rowsGen.QueryRows()

	if err != nil && err != sql.ErrNoRows {
		q.err = err
		return nil, q.err
	}
	defer rows.Close()

	for rows.Next() {
		v := &SQLView[T]{}
		q.err = v.Scan(rows, "")
		if q.err != nil {
			return nil, q.err
		}
		q.result = append(q.result, v)
	}
	if err := rows.Err(); err != nil {
		q.err = err
	}
	if q.err != nil {
		return nil, q.err
	}
	return q.result, nil
}
