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
	gob.Register(&BigSerial{})
}

type BigSerial sql.NullInt64

func (BigSerial) PostgresType() string {
	return "BIGSERIAL"
}

func (BigSerial) PostgresDefaultValue() string {
	return ``
}

func (u BigSerial) PostgresAllowNull() bool {
	return true
}

func (n BigSerial) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return json.Marshal(nil)
	}
	return json.Marshal(n.Int64)
}

func (n *BigSerial) UnmarshalJSON(b []byte) error {
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

func (n *BigSerial) Scan(value interface{}) error {
	return (*sql.NullInt64)(n).Scan(value)
}

func (n BigSerial) Value() (driver.Value, error) {
	return sql.NullInt64(n).Value()
}

// store.Converter interface, n must contain zero value before call
func (n *BigSerial) ConvertFrom(v interface{}) error {
	if v == nil {
		return nil
	}
	switch vv := v.(type) {
	case int:
		*n = BigSerial{int64(vv), true}
		return nil
	case *int:
		*n = BigSerial{int64(*vv), true}
		return nil
	case BigSerial:
		*n = vv
		return nil
	case *BigSerial:
		*n = *vv
		return nil
	}
	value := reflect.Indirect(reflect.ValueOf(v))
	switch value.Kind() {
	case
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		*n = BigSerial{value.Int(), true}
		return nil

	case reflect.Float32, reflect.Float64:
		*n = BigSerial{int64(value.Float()), true}
		return nil
	case reflect.String:
		i, err := strconv.Atoi(value.String())
		*n = BigSerial{int64(i), true}
		return err
	}

	return fmt.Errorf("can't convert value of type %T to BigSerial", value.Interface())
}
