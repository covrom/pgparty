package pgparty

import (
	"database/sql/driver"
	"encoding/gob"
	"encoding/json"
	"fmt"
)

func init() {
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})
	gob.Register(&JsonB{})
}

type JsonB struct {
	// Val содержит указатель на значение
	// После анмаршала (из базы/json) тип его всегда становится map[string]interface{} или другим, как указано в описании к json.Unmarshal
	// вне зависимости от того, что там было до этого
	Val interface{}
}

func NewJsonB(val interface{}) *JsonB {
	return &JsonB{
		Val: val,
	}
}

func (JsonB) PostgresType() string {
	return "JSONB"
}

func (JsonB) PostgresDefaultValue() string {
	return `'{}'::jsonb`
}

func (u JsonB) PostgresAllowNull() bool {
	return false
}

func (n *JsonB) Scan(value interface{}) error {
	if value == nil {
		n.Val = nil
		return nil
	}
	switch val := value.(type) {
	case []byte:
		return json.Unmarshal(val, n)
	}
	return fmt.Errorf("unsupported database data type %T, needs []byte", value)
}

func (n JsonB) ConvertTo(value interface{}) error {
	bval, err := json.Marshal(n)
	if err != nil {
		return err
	}
	return json.Unmarshal(bval, value)
}

// Value implements the driver Valuer interface.
func (n JsonB) Value() (driver.Value, error) {
	j, err := json.Marshal(n.Val)
	return string(j), err
}

func (n JsonB) MarshalJSON() ([]byte, error) {
	return json.Marshal(n.Val)
}

func (n *JsonB) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &(n.Val))
}

// store.Converter interface, n must contain zero value before call
func (n *JsonB) ConvertFrom(v interface{}) error {
	if v == nil {
		n.Val = nil
		return nil
	}
	switch vv := v.(type) {
	case JsonB:
		*n = vv
		return nil
	case *JsonB:
		*n = *vv
		return nil
	case NullJsonB:
		*n = JsonB{vv.Val}
		return nil
	case *NullJsonB:
		*n = JsonB{vv.Val}
		return nil
	case []byte:
		return json.Unmarshal(vv, n)
	case *[]byte:
		return json.Unmarshal(*vv, n)
	}

	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonBytes, n)
}
