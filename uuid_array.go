package pgparty

import (
	"database/sql/driver"
	"encoding/gob"
	"encoding/json"
	"errors"
)

func init() {
	gob.Register(&UUIDv4Array{})
}

type UUIDv4Array []UUIDv4

func (a *UUIDv4Array) Raw() []UUIDv4 {
	return *a
}

func (a *UUIDv4Array) Scan(value interface{}) error {
	if value == nil {
		*a = make([]UUIDv4, 0)
		return nil
	}

	if val, ok := value.([]byte); ok {
		return json.Unmarshal(val, a)
	}

	return nil
}

func (a UUIDv4Array) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a UUIDv4Array) MarshalJSON() ([]byte, error) {
	if a == nil {
		return []byte("[]"), nil
	}

	var v []UUIDv4 = a

	return json.Marshal(v)
}

func (a *UUIDv4Array) UnmarshalJSON(b []byte) error {
	if a == nil {
		return errors.New("UUIDArray: UnmarshalJSON on nil pointer")
	}

	v := make([]UUIDv4, 0)
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	*a = v

	return nil
}
