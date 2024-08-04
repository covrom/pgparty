package pgparty

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

type MdjsField struct {
	Name   string            `json:"name"`
	DBName string            `json:"dbname"`
	Type   reflect.Type      `json:"-"`
	Desc   *FieldDescription `json:"-"`
}

type Mdjs struct {
	ModelName string      `json:"model"`
	Md        *ModelDesc  `json:"-"`
	Fields    []MdjsField `json:"fields"`
}

func ModelsAndFields(ctx context.Context, st *PgStore) ([]Mdjs, error) {
	ret := make([]Mdjs, 0, 10)
	mds := st.ModelDescriptions()
	for _, md := range mds {
		sn := st.Schema()
		m := Mdjs{
			ModelName: "&" + md.ModelType().Name(),
			Md:        md,
		}

		m.Fields = append(m.Fields, MdjsField{":" + md.ModelType().Name() + ".*", "", nil, nil})

		for i := 0; i < md.ColumnPtrsCount(); i++ {
			fd := md.ColumnPtr(i)
			if fd.Skip {
				continue
			}
			m.Fields = append(m.Fields,
				MdjsField{":" + fd.FieldName, fd.DatabaseName, fd.ElemType, fd},
				MdjsField{":" + md.ModelType().Name() + "." + fd.FieldName, fd.DatabaseName, fd.ElemType, fd})
		}

		dbcols := make([]string, 0, 10)
		if err := st.WithTx(ctx, func(stx *PgStore) error {
			return stx.Tx().SelectContext(ctx, &dbcols,
				`SELECT column_name FROM information_schema.columns 
				WHERE table_name = '`+md.DatabaseName()+`' AND table_schema = '`+sn+`'`)
		}); err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("select column names: %s", err)
		}

		for _, dbc := range dbcols {
			if len(dbc) == 0 {
				continue
			}
			fnd := false
			for _, f := range m.Fields {
				if dbc == f.DBName {
					fnd = true
					break
				}
			}
			if !fnd {
				m.Fields = append(m.Fields, MdjsField{"-" + dbc, dbc, reflect.TypeOf(nil), nil})
			}
		}

		sort.Slice(m.Fields, func(i, j int) bool {
			return strings.Compare(m.Fields[i].Name, m.Fields[j].Name) < 0
		})

		ret = append(ret, m)
	}
	sort.Slice(ret, func(i, j int) bool {
		return strings.Compare(ret[i].ModelName, ret[j].ModelName) < 0
	})

	return ret, nil
}
