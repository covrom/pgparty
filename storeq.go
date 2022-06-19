package pgparty

import (
	"context"
	"fmt"
)

// TODO: https://github.com/0x1000000/SqExpress
// https://itnext.io/filtering-by-dynamic-attributes-90ada3504361

func (s *PgStore) Select(ctx context.Context) *PgSelect {
	return &PgSelect{
		st: s,
	}
}

type PgSelect struct {
	st   *PgStore
	cols []string
	from string
}

func (ps *PgSelect) Field(model Storable, defval any, fieldName string) *PgSelect {
	md, ok := ps.st.GetModelDescription(model)
	if !ok {
		panic(fmt.Sprintf("model %T is not registered", model))
	}
	fd, err := md.ColumnByFieldName(fieldName)
	if err != nil {
		panic(err.Error())
	}

	ps.cols = append(ps.cols, fmt.Sprintf("%s.%s.%s", ps.st.schema, md.StoreName(), fd.Name))

	return ps
}

func (ps *PgSelect) From(model Storable) *PgSelect {
	md, ok := ps.st.GetModelDescription(model)
	if !ok {
		panic(fmt.Sprintf("model %T is not registered", model))
	}
	ps.from = fmt.Sprintf("%s.%s", ps.st.schema, md.StoreName())
	return ps
}
