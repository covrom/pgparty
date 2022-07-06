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
	gob.Register(&NullBool{})
}

type NullBool sql.NullBool

func (NullBool) PostgresType() string {
	return "BOOLEAN"
}

func (NullBool) PostgresDefaultValue() string {
	return `FALSE`
}

func (n NullBool) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return json.Marshal(nil)
	}
	return json.Marshal(n.Bool)
}

func (n *NullBool) UnmarshalJSON(b []byte) error {
	if bytes.EqualFold(b, []byte("null")) {
		n.Bool = false
		n.Valid = false
		return nil
	}
	tmp := false
	err := json.Unmarshal(b, &tmp)
	if err == nil {
		n.Bool = tmp
		n.Valid = true
	}
	return err
}

func (n *NullBool) Scan(value interface{}) error {
	return (*sql.NullBool)(n).Scan(value)
}

func (n NullBool) Value() (driver.Value, error) {
	return sql.NullBool(n).Value()
}

type Bool bool

func (Bool) PostgresType() string {
	return "BOOLEAN"
}

func (Bool) PostgresDefaultValue() string {
	return `FALSE`
}

func (n Bool) MarshalJSON() ([]byte, error) {
	return json.Marshal(bool(n))
}

func (n *Bool) UnmarshalJSON(b []byte) error {
	if bytes.EqualFold(b, []byte("null")) {
		*n = false
		return nil
	}
	var tmp bool
	err := json.Unmarshal(b, &tmp)
	if err == nil {
		*n = Bool(tmp)
	}
	return err
}

func (n *Bool) Scan(value interface{}) error {
	tmp := &sql.NullBool{}
	if err := tmp.Scan(value); err != nil {
		return err
	}
	if tmp.Valid {
		*n = Bool(tmp.Bool)
	} else {
		*n = false
	}
	return nil
}

func (n Bool) Value() (driver.Value, error) {
	return (sql.NullBool{Bool: bool(n), Valid: true}).Value()
}

func (n Bool) Bool() bool {
	return bool(n)
}

func (n *Bool) SetBool(b bool) {
	*n = Bool(b)
}

// store.Converter interface, n must contain zero value before call
func (b *Bool) ConvertFrom(v interface{}) error {
	if v == nil {
		return nil
	}
	switch vv := v.(type) {
	case bool:
		*b = Bool(vv)
		return nil
	case *bool:
		*b = Bool(*vv)
		return nil
	case NullBool:
		*b = Bool(vv.Bool)
		return nil
	case *NullBool:
		*b = Bool(vv.Bool)
		return nil
	}
	value := reflect.Indirect(reflect.ValueOf(v))
	switch value.Kind() {
	case
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		*b = value.Int() != 0
		return nil
	case reflect.Float32, reflect.Float64:
		*b = value.Float() != 0
		return nil
	case reflect.String:
		bb, err := strconv.ParseBool(value.String())
		if err != nil {
			return fmt.Errorf("can't convert string '%s' to bool: %s", value.String(), err)
		}
		*b = Bool(bb)
		return nil
	}

	return fmt.Errorf("can't convert value of type %T to bool", value.Interface())
}

// store.Converter interface, n must contain zero value before call
func (n *NullBool) ConvertFrom(v interface{}) error {
	if v == nil {
		return nil
	}
	switch vv := v.(type) {
	case bool:
		*n = NullBool{vv, true}
		return nil
	case *bool:
		*n = NullBool{*vv, true}
		return nil
	case NullBool:
		*n = vv
		return nil
	case *NullBool:
		*n = *vv
		return nil
	}
	var b Bool
	if err := (&b).ConvertFrom(v); err != nil {
		return err
	}
	*n = NullBool{bool(b), true}
	return nil
}
