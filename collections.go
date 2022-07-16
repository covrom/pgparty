package pgparty

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type AnyObjectMap map[string]interface{}

func (AnyObjectMap) PostgresType() string {
	return "JSONB"
}

func (AnyObjectMap) PostgresDefaultValue() string {
	return `'{}'::jsonb`
}

func (u AnyObjectMap) PostgresAllowNull() bool {
	return false
}

func (f *AnyObjectMap) Scan(value interface{}) error {
	if value == nil {
		*f = make(map[string]interface{})
		return nil
	}

	if val, ok := value.([]byte); ok {
		return json.Unmarshal(val, f)
	}

	return nil
}

func (f AnyObjectMap) Value() (driver.Value, error) {
	rv, err := json.Marshal(f)
	return string(rv), err
}

func (f AnyObjectMap) MarshalJSON() ([]byte, error) {
	if f == nil {
		return []byte("{}"), nil
	}

	var v map[string]interface{} = f

	return json.Marshal(v)
}

func (f *AnyObjectMap) UnmarshalJSON(b []byte) error {
	if f == nil {
		return errors.New("json.RawMessage: UnmarshalJSON on nil pointer")
	} else if *f == nil {
		*f = make(AnyObjectMap)
	}

	v := make(map[string]interface{})
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	*f = v

	return nil
}

func (f AnyObjectMap) String() string {
	sb := make([]string, 0, len(f))
	for k, v := range f {
		sb = append(sb, fmt.Sprintf("%q = %v", k, v))
	}
	return strings.Join(sb, ", ")
}

func (f AnyObjectMap) ConvertTo(value interface{}) error {
	bval, err := json.Marshal(f)
	if err != nil {
		return err
	}
	return json.Unmarshal(bval, value)
}

type Uint64Array []uint64

func (Uint64Array) PostgresType() string {
	return "JSONB"
}

func (Uint64Array) PostgresDefaultValue() string {
	return `'[]'::jsonb`
}

func (u Uint64Array) PostgresAllowNull() bool {
	return false
}

func (f *Uint64Array) Scan(value interface{}) error {
	if value == nil {
		*f = make([]uint64, 0)
		return nil
	}

	if val, ok := value.([]byte); ok {
		return json.Unmarshal(val, f)
	}

	return nil
}

func (f Uint64Array) Value() (driver.Value, error) {
	return json.Marshal(f)
}

func (f Uint64Array) MarshalJSON() ([]byte, error) {
	if f == nil {
		return []byte("[]"), nil
	}

	var v []uint64 = f

	return json.Marshal(v)
}

func (f *Uint64Array) UnmarshalJSON(b []byte) error {
	if f == nil {
		return errors.New("uint64Array: UnmarshalJSON on nil pointer")
	}

	v := make([]uint64, 0)
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	*f = v

	return nil
}
