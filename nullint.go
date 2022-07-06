package pgparty

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
)

func init() {
	gob.Register(&NullInt64{})
}

type NullInt64 sql.NullInt64

func (NullInt64) PostgresType() string {
	return "BIGINT"
}

func (NullInt64) PostgresDefaultValue() string {
	return `0`
}

func (n NullInt64) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return json.Marshal(nil)
	}
	return json.Marshal(n.Int64)
}

func (n *NullInt64) UnmarshalJSON(b []byte) error {
	if bytes.EqualFold(b, []byte("null")) {
		n.Int64 = 0
		n.Valid = false
		return nil
	}
	tmp := int64(0)
	err := json.Unmarshal(b, &tmp)
	if err == nil {
		n.Int64 = tmp
		n.Valid = true
	}
	return err
}

func (n *NullInt64) Scan(value interface{}) error {
	return (*sql.NullInt64)(n).Scan(value)
}

func (n NullInt64) Value() (driver.Value, error) {
	return sql.NullInt64(n).Value()
}

type Int64 int64

func (Int64) PostgresType() string {
	return "BIGINT"
}

func (Int64) PostgresDefaultValue() string {
	return `0`
}

func (n Int64) MarshalJSON() ([]byte, error) {
	return json.Marshal(int64(n))
}

func (n *Int64) UnmarshalJSON(b []byte) error {
	if bytes.EqualFold(b, []byte("null")) {
		*n = 0
		return nil
	}
	tmp := int64(0)
	err := json.Unmarshal(b, &tmp)
	if err == nil {
		*n = Int64(tmp)
	}
	return err
}

func (n *Int64) Scan(value interface{}) error {
	tmp := &sql.NullInt64{}
	if err := tmp.Scan(value); err != nil {
		return err
	}
	if tmp.Valid {
		*n = Int64(tmp.Int64)
	} else {
		*n = 0
	}
	return nil
}

func (n Int64) Value() (driver.Value, error) {
	return (sql.NullInt64{Int64: int64(n), Valid: true}).Value()
}

func (n Int64) Int() int {
	return int(n)
}

func (n *Int64) SetInt(i int) {
	*n = Int64(i)
}

// store.Converter interface, n must contain zero value before call
func (f *Int64) ConvertFrom(v interface{}) error {
	if v == nil {
		return nil
	}
	switch vv := v.(type) {
	case int:
		*f = Int64(vv)
		return nil
	case *int:
		*f = Int64(*vv)
		return nil
	case NullInt64:
		*f = Int64(vv.Int64)
		return nil
	case *NullInt64:
		*f = Int64(vv.Int64)
		return nil
	}
	value := reflect.Indirect(reflect.ValueOf(v))
	switch value.Kind() {
	case
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		*f = Int64(value.Int())
		return nil

	case reflect.Float32, reflect.Float64:
		*f = Int64(value.Float())
		return nil
	case reflect.String:
		i, err := strconv.Atoi(value.String())
		*f = Int64(i)
		return err
	}

	return fmt.Errorf("can't convert value of type %T to Int64", value.Interface())
}

// store.Converter interface, n must contain zero value before call
func (n *NullInt64) ConvertFrom(v interface{}) error {
	if v == nil {
		return nil
	}
	switch vv := v.(type) {
	case int:
		*n = NullInt64{int64(vv), true}
		return nil
	case *int:
		*n = NullInt64{int64(*vv), true}
		return nil
	case NullInt64:
		*n = vv
		return nil
	case *NullInt64:
		*n = *vv
		return nil
	}
	value := reflect.Indirect(reflect.ValueOf(v))
	switch value.Kind() {
	case
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		*n = NullInt64{value.Int(), true}
		return nil

	case reflect.Float32, reflect.Float64:
		*n = NullInt64{int64(value.Float()), true}
		return nil
	case reflect.String:
		i, err := strconv.Atoi(value.String())
		*n = NullInt64{int64(i), true}
		return err
	}

	return fmt.Errorf("can't convert value of type %T to Int64", value.Interface())
}
