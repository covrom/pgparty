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

var defaultSQLValues = map[reflect.Type]string{
	reflect.TypeOf(Time{}):        `'epoch'`,
	reflect.TypeOf(Decimal{}):     `'0.0'`,
	reflect.TypeOf(Text("")):      ``,
	reflect.TypeOf(StringArray{}): `'[]'::jsonb`,
	reflect.TypeOf(UUIDv4Array{}): `'[]'::jsonb`,
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

func SQLType(ft reflect.Type, ln, prec int) string {
	if ft.Kind() == reflect.Slice {
		if ft.Elem().Kind() == reflect.Uint8 {
			if ft == reflect.TypeOf(Decimal{}) {
				// для decimal длина не может быть больше 50 знаков
				if ln > 50 {
					ln = 15
				}
				return fmt.Sprintf("NUMERIC(%d,%d)", ln, prec)
			}
			// []byte, не более 16 Мб
			return "BYTEA"
		}
	} else if ft.Kind() == reflect.Struct {
		if ft == reflect.TypeOf(UUIDv4{}) {
			return "UUID"
		} else if ft == reflect.TypeOf(UUIDv4Array{}) {
			return jsonType
		} else if ft == reflect.TypeOf(NullTime{}) {
			return "TIMESTAMPTZ"
		} else if ft == reflect.TypeOf(Time{}) {
			return "TIMESTAMPTZ"
		} else if ft == reflect.TypeOf(NullBool{}) {
			return "BOOLEAN"
		} else if ft == reflect.TypeOf(NullFloat64{}) {
			return "FLOAT8"
		} else if ft == reflect.TypeOf(NullInt64{}) {
			return "BIGINT"
		} else if ft == reflect.TypeOf(BigSerial{}) {
			return "BIGSERIAL"
		} else if ft == reflect.TypeOf(NullString{}) {
			return fmt.Sprintf("VARCHAR(%d)", ln)
		} else if ft == reflect.TypeOf(NullText{}) {
			return "TEXT"
		} else if ft == reflect.TypeOf(NullJsonB{}) {
			return jsonType
		} else if ft == reflect.TypeOf(JsonB{}) {
			return jsonType
		} else if ft == reflect.TypeOf(StringArray{}) {
			return jsonType
		} else if ft == reflect.TypeOf(NullDecimal{}) {
			if ln > 50 {
				ln = 15
			}
			return fmt.Sprintf("NUMERIC(%d,%d)", ln, prec)
		}
	} else if ft.Kind() == reflect.String {
		if ft == reflect.TypeOf(Text("")) {
			return "TEXT"
		}
		return fmt.Sprintf("VARCHAR(%d)", ln)
	}
	return sqlTypesMap[ft.Kind()]
}
