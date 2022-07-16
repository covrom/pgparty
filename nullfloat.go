package pgparty

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"reflect"
)

func init() {
	gob.Register(&NullFloat64{})
}

type NullFloat64 sql.NullFloat64

func (NullFloat64) PostgresType() string {
	return "FLOAT8"
}

func (NullFloat64) PostgresDefaultValue() string {
	return `0`
}

func (u NullFloat64) PostgresAllowNull() bool {
	return true
}

func (n NullFloat64) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return json.Marshal(nil)
	}
	return json.Marshal(n.Float64)
}

func (n *NullFloat64) UnmarshalJSON(b []byte) error {
	if bytes.EqualFold(b, []byte("null")) {
		n.Float64 = 0
		n.Valid = false
		return nil
	}
	tmp := float64(0)
	err := json.Unmarshal(b, &tmp)
	if err == nil {
		n.Float64 = tmp
		n.Valid = true
	}
	return err
}

func (n *NullFloat64) Scan(value interface{}) error {
	return (*sql.NullFloat64)(n).Scan(value)
}

func (n NullFloat64) Value() (driver.Value, error) {
	return sql.NullFloat64(n).Value()
}

type Float64 float64

func (Float64) PostgresType() string {
	return "FLOAT8"
}

func (Float64) PostgresDefaultValue() string {
	return `0`
}

func (Float64) PostgresAllowNull() bool {
	return false
}

func (n Float64) MarshalJSON() ([]byte, error) {
	return json.Marshal(float64(n))
}

func (n *Float64) UnmarshalJSON(b []byte) error {
	if bytes.EqualFold(b, []byte("null")) {
		*n = 0
		return nil
	}
	tmp := float64(0)
	err := json.Unmarshal(b, &tmp)
	if err == nil {
		*n = Float64(tmp)
	}
	return err
}

func (n *Float64) Scan(value interface{}) error {
	tmp := &sql.NullFloat64{}
	if err := tmp.Scan(value); err != nil {
		return err
	}
	if tmp.Valid {
		*n = Float64(tmp.Float64)
	} else {
		*n = 0
	}
	return nil
}

func (n Float64) Value() (driver.Value, error) {
	return (sql.NullFloat64{Float64: float64(n), Valid: true}).Value()
}

func (n Float64) Float64() float64 {
	return float64(n)
}

func (n *Float64) SetFloat64(f float64) {
	*n = Float64(f)
}

// store.Converter interface, n must contain zero value before call
func (f *Float64) ConvertFrom(v interface{}) error {
	if v == nil {
		return nil
	}
	switch vv := v.(type) {
	case float64:
		*f = Float64(vv)
		return nil
	case *float64:
		*f = Float64(*vv)
		return nil
	case float32:
		*f = Float64(vv)
		return nil
	case *float32:
		*f = Float64(*vv)
		return nil
	case NullFloat64:
		*f = Float64(vv.Float64)
		return nil
	case *NullFloat64:
		*f = Float64(vv.Float64)
		return nil
	}
	value := reflect.Indirect(reflect.ValueOf(v))
	switch value.Kind() {
	case
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		*f = Float64(value.Int())
		return nil

	case reflect.Float32, reflect.Float64:
		*f = Float64(value.Float())
		return nil
	}

	return fmt.Errorf("can't convert value of type %T to Float64", value.Interface())
}

// store.Converter interface, n must contain zero value before call
func (n *NullFloat64) ConvertFrom(v interface{}) error {
	if v == nil {
		return nil
	}
	switch vv := v.(type) {
	case float64:
		*n = NullFloat64{vv, true}
		return nil
	case *float64:
		*n = NullFloat64{*vv, true}
		return nil
	case float32:
		*n = NullFloat64{float64(vv), true}
		return nil
	case *float32:
		*n = NullFloat64{float64(*vv), true}
		return nil
	case NullFloat64:
		*n = vv
		return nil
	case *NullFloat64:
		*n = *vv
		return nil
	}
	value := reflect.Indirect(reflect.ValueOf(v))
	switch value.Kind() {
	case
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		*n = NullFloat64{float64(value.Int()), true}
		return nil

	case reflect.Float32, reflect.Float64:
		*n = NullFloat64{value.Float(), true}
		return nil
	}

	return fmt.Errorf("can't convert value of type %T to NullFloat64", value.Interface())
}
