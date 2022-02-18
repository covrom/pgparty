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
