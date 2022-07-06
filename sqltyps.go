package pgparty

import (
	"fmt"
	"reflect"
)

const jsonType string = "jsonb"

var sqlTypesMap = map[reflect.Kind]string{
	reflect.Bool:    "BOOLEAN",
	reflect.Int:     "BIGINT",
	reflect.Int8:    "SMALLINT",
	reflect.Int16:   "SMALLINT",
	reflect.Int32:   "INT",
	reflect.Int64:   "BIGINT",
	reflect.Uint:    "BIGINT",
	reflect.Uint8:   "SMALLINT",
	reflect.Uint16:  "INT",
	reflect.Uint32:  "BIGINT",
	reflect.Uint64:  "BIGINT",
	reflect.Float32: "FLOAT4",
	reflect.Float64: "FLOAT8",
	// json
	reflect.Struct: jsonType,
	reflect.Slice:  jsonType,
	reflect.Map:    jsonType,
}

var defaultSQLKindValues = map[reflect.Kind]string{
	reflect.Bool:    "FALSE",
	reflect.Int:     "0",
	reflect.Int8:    "0",
	reflect.Int16:   "0",
	reflect.Int32:   "0",
	reflect.Int64:   "0",
	reflect.Uint:    "0",
	reflect.Uint8:   "0",
	reflect.Uint16:  "0",
	reflect.Uint32:  "0",
	reflect.Uint64:  "0",
	reflect.Float32: "0",
	reflect.Float64: "0",
	reflect.String:  `''`,
}

type PostgresTyper interface {
	PostgresType() string
}

func SQLType(ft reflect.Type, ln, prec int) string {
	deepft := ft
	for deepft.Kind() == reflect.Ptr {
		deepft = deepft.Elem()
	}
	if deepft.Implements(reflect.TypeOf((*PostgresTyper)(nil)).Elem()) {
		v := reflect.New(deepft).Elem().Interface().(PostgresTyper)
		return v.PostgresType()
	}
	if ft.Kind() == reflect.Slice {
		if ft.Elem().Kind() == reflect.Uint8 {
			// []byte, не более 16 Мб
			return "BYTEA"
		}
	} else if ft.Kind() == reflect.String {
		return fmt.Sprintf("VARCHAR(%d)", ln)
	}
	return sqlTypesMap[ft.Kind()]
}

type PostgresDefaultValuer interface {
	PostgresDefaultValue() string
}

func SQLDefaultValue(ft reflect.Type) string {
	var ret string
	deepft := ft
	for deepft.Kind() == reflect.Ptr {
		deepft = deepft.Elem()
	}
	if deepft.Implements(reflect.TypeOf((*PostgresDefaultValuer)(nil)).Elem()) {
		v := reflect.New(deepft).Elem().Interface().(PostgresDefaultValuer)
		return v.PostgresDefaultValue()
	}
	ret = defaultSQLKindValues[ft.Kind()]

	return ret
}
