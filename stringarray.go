package pgparty

import (
	"database/sql/driver"
	"encoding/gob"
	"encoding/json"
	"errors"
)

func init() {
	gob.Register(&StringArray{})
}

type StringArray []string

func (f *StringArray) Scan(value interface{}) error {
	if value == nil {
		*f = make([]string, 0)
		return nil
	}

	switch val := value.(type) {
	case []byte:
		return json.Unmarshal(val, f)
	}

	return nil
}

func (f StringArray) Value() (driver.Value, error) {
	return json.Marshal(f)
}

func (f StringArray) MarshalJSON() ([]byte, error) {
	if f == nil {
		return []byte("[]"), nil
	}

	var v []string = f

	return json.Marshal(v)
}

func (f *StringArray) UnmarshalJSON(b []byte) error {
	if f == nil {
		return errors.New("StringArray: UnmarshalJSON on nil pointer")
	}

	v := make([]string, 0)
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	*f = v

	return nil
}
