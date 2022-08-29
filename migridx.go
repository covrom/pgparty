package pgparty

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/covrom/pgparty/modelcols"
)

type DBIndexDef struct {
	Name   string      `db:"indname"`
	Table  string      `db:"tablename"`
	Schema string      `db:"nspname"`
	Fields StringArray `db:"indkey_names"`
}

func (d DBIndexDef) String() string {
	return fmt.Sprintf("%s (%s)", d.Name, strings.Join(d.Fields, ", "))
}

type DBIndexDefs []DBIndexDef

func (idxs DBIndexDefs) FindByName(n string) (DBIndexDef, bool) {
	for _, idx := range idxs {
		if strings.EqualFold(idx.Name, n) {
			return idx, true
		}
	}
	return DBIndexDef{}, false
}

func CurrentSchemaIndexes(ctx context.Context, tablename string) (DBIndexDefs, error) {
	s, err := ShardFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("CurrentSchemaIndexes: %w", err)
	}
	stx := s.Store
	if stx == nil || stx.tx == nil {
		return nil, fmt.Errorf("context must contains store transaction")
	}
	// mdsn := stx.Schema()
	// tablename = mdsn + "." + tablename

	var idxs []DBIndexDef

	q := `select
			i.relname as indname,
			idx.indrelid::regclass::text tablename,
			to_jsonb(array(
			select
				pg_get_indexdef(idx.indexrelid,
				k + 1,
				true)
			from
				generate_subscripts(idx.indkey, 1) as k
			order by
				k
			)) as indkey_names,
			ns.nspname nspname
		from
			pg_index as idx
		join pg_class as i
		on
			i.oid = idx.indexrelid
		join pg_am as am
		on
			i.relam = am.oid
		join pg_namespace as ns
		on
			ns.oid = i.relnamespace
		where 
		not idx.indisprimary
		and idx.indrelid::regclass::text = $1
		order by i.relname`

	// log.Printf("tablename = %s", tablename)
	if err := stx.tx.SelectContext(ctx, &idxs, q, tablename); err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return idxs, nil
}

func IndexesEqualToDBIndexes(sqs *modelcols.SQLModel, dbidxs DBIndexDefs) bool {
	ins := sqs.AllIndexLowerNames()
	if len(ins) != len(dbidxs) {
		return false
	}
	for _, dbidx := range dbidxs {
		sqfds, ok := ins[strings.ToLower(dbidx.Name)]
		if !ok {
			return false
		}
		// сравним филды
		if !IndexEqualDBIndex(sqs.Table, sqfds, dbidx) {
			return false
		}
	}
	return true
}

func IndexEqualDBIndex(tname string, idx modelcols.SQLIndex, dbidx DBIndexDef) bool {
	inm := strings.ToLower(tname + idx.Name)
	return strings.EqualFold(inm, dbidx.Name) &&
		modelcols.ColumnsEqual(dbidx.Fields, idx.Columns)
}
