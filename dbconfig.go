package pgparty

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sort"

	"github.com/covrom/pgparty/modelcols"
	"github.com/jmoiron/sqlx"
)

type DbConfigTable struct {
	TableName string              `db:"table_name"`
	Storej    *modelcols.SQLModel `db:"storej"`
}

func DbConfigTableFromModel(md *ModelDesc) (*DbConfigTable, error) {
	sqs, err := MD2SQLModel(md)
	if err != nil {
		return nil, err
	}
	ret := &DbConfigTable{
		TableName: md.StoreName(),
		Storej:    sqs,
	}
	return ret, nil
}

func (c *DbConfigTable) LoadTable(ctx context.Context, table string) error {
	s, err := ShardFromContext(ctx)
	if err != nil {
		return fmt.Errorf("LoadTable: %w", err)
	}
	stx := s.Store
	if stx == nil || stx.tx == nil {
		return fmt.Errorf("context must contains store transaction")
	}
	sn := stx.Schema()
	c.TableName = table
	c.Storej = &modelcols.SQLModel{}
	if err := stx.tx.GetContext(ctx, c, `SELECT table_name,storej from `+sn+`._config WHERE table_name = $1`,
		table); err != nil && err != sql.ErrNoRows {
		return err
	}
	sqsdb := *c.Storej
	sort.Slice(sqsdb.Columns, func(i, j int) bool {
		return sqsdb.Columns[i].ColName < sqsdb.Columns[j].ColName
	})

	sort.Slice(sqsdb.Indexes, func(i, j int) bool {
		return sqsdb.Indexes[i].Name < sqsdb.Indexes[j].Name
	})

	for _, idx := range sqsdb.Indexes {
		sort.Slice(idx.Columns, func(i, j int) bool {
			return idx.Columns[i] < idx.Columns[j]
		})
	}
	return nil
}

func (c DbConfigTable) SaveTable(ctx context.Context) error {
	s, err := ShardFromContext(ctx)
	if err != nil {
		return fmt.Errorf("SaveTable: %w", err)
	}
	stx := s.Store
	if stx == nil || stx.tx == nil {
		return fmt.Errorf("context must contains store transaction")
	}
	sn := stx.Schema()

	q := fmt.Sprintf(`INSERT INTO %s._config (table_name,storej) 
	VALUES($1,$2) ON CONFLICT(table_name) DO
	UPDATE SET storej=excluded.storej`, sn)

	log.Println(q, ", $1 = ", c.TableName, ", $2 = ", c.Storej)

	_, err = stx.tx.ExecContext(ctx, q, c.TableName, c.Storej)
	if err != nil {
		return fmt.Errorf("SaveModelConfig error: %w", err)
	}
	return nil
}

func (c *DbConfigTable) IsEmpty() bool {
	return c == nil || c.Storej == nil || (len(c.Storej.Columns) == 0 && len(c.Storej.Indexes) == 0)
}

type DbConfig []DbConfigTable

func NewDbConfig() *DbConfig {
	r := make(DbConfig, 0, 100)
	return &r
}

func (c *DbConfig) LoadAll(ctx context.Context, tx *sqlx.Tx, schema string) error {
	return tx.SelectContext(ctx, c, `SELECT table_name,storej from `+schema+`._config`)
}

func DBColumnsInfo(ctx context.Context, tx *sqlx.Tx, schema, tname string) ([]DBColInfo, error) {
	q := fmt.Sprintf(SQL_ColumnsInfo, tname, schema)
	ret := make([]DBColInfo, 0)
	err := tx.SelectContext(ctx, &ret, q)
	return ret, err
}

const SQL_ColumnsInfo = `select
column_name,
udt_name,
is_nullable,
character_maximum_length,
numeric_precision,
numeric_precision_radix
from
information_schema.columns
where
table_name = '%s'
and table_schema = '%s'`

type DBColInfo struct {
	Name       string `db:"column_name"`
	Type       string `db:"udt_name"`
	IsNullable string `db:"is_nullable"`
	CharLen    *int   `db:"character_maximum_length"`
	NumLen     *int   `db:"numeric_precision"`
	NumPrec    *int   `db:"numeric_precision_radix"`
}
