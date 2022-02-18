package pgparty

import (
	"bytes"
	"database/sql/driver"
	"encoding/gob"
	"encoding/json"
	"fmt"
)

func init() {
	gob.Register(&NullJsonB{})
}

type NullJsonB struct {
	// Val содержит указатель на значение
	// После анмаршала (из базы/json) тип его всегда становится map[string]interface{} или другим, как указано в описании к json.Unmarshal
	// вне зависимости от того, что там было до этого
	Val   interface{}
	Valid bool // Valid is true if Time is not NULL
}

func NewNullJsonB(val interface{}) *NullJsonB {
	return &NullJsonB{
		Val:   val,
		Valid: val != nil,
	}
}

func (n *NullJsonB) Scan(value interface{}) error {
	if value == nil {
		n.Val, n.Valid = nil, false
		return nil
	}

	switch val := value.(type) {
	case []byte:
		err := json.Unmarshal(val, n)
		n.Valid = (err == nil)
		return err
	}

	return fmt.Errorf("unsupported database data type %T, needs []byte", value)
}

func (n NullJsonB) ConvertTo(value interface{}) error {
	bval, err := json.Marshal(n)
	if err != nil {
		return err
	}
	return json.Unmarshal(bval, value)
}

// Value implements the driver Valuer interface.
func (n NullJsonB) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	j, err := json.Marshal(n.Val)
	return string(j), err
}

func (n NullJsonB) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return json.Marshal(nil)
	}
	return json.Marshal(n.Val)
}

func (n *NullJsonB) UnmarshalJSON(b []byte) error {
	if bytes.EqualFold(b, []byte("null")) {
		n.Val, n.Valid = nil, false
		return nil
	}
	err := json.Unmarshal(b, &(n.Val))
	n.Valid = (err == nil)
	return err
}

func (n *NullJsonB) ConvertFrom(v interface{}) error {
	if v == nil {
		n.Val, n.Valid = nil, false
		return nil
	}
	switch vv := v.(type) {
	case JsonB:
		*n = NullJsonB{vv, true}
		return nil
	case *JsonB:
		*n = NullJsonB{*vv, true}
		return nil
	case NullJsonB:
		*n = vv
		return nil
	case *NullJsonB:
		*n = *vv
		return nil
	case []byte:
		err := json.Unmarshal(vv, n)
		n.Valid = (err == nil)
		return err
	case *[]byte:
		err := json.Unmarshal(*vv, n)
		n.Valid = (err == nil)
		return err
	}
	bval, err := json.Marshal(v)
	if err != nil {
		return err
	}
	err = json.Unmarshal(bval, n)
	n.Valid = (err == nil)
	return err
}
