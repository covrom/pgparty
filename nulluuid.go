package pgparty

import (
	"bytes"
	"database/sql/driver"
	"encoding/gob"
	"encoding/json"
	"fmt"
)

func init() {
	gob.Register(&NullUUIDv4{})
}

type NullUUIDv4 struct {
	ID    UUIDv4
	Valid bool
}

func (NullUUIDv4) PostgresType() string {
	return "UUID"
}

func (NullUUIDv4) PostgresDefaultValue() string {
	var empty UUIDv4
	return fmt.Sprintf(`'%s'`, empty)
}

func (u NullUUIDv4) PostgresAllowNull() bool {
	return true
}

func (n NullUUIDv4) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return json.Marshal(nil)
	}
	return json.Marshal(n.ID)
}

func (n *NullUUIDv4) UnmarshalJSON(b []byte) error {
	if bytes.EqualFold(b, []byte("null")) {
		n.ID = UUIDv4{}
		n.Valid = false
		return nil
	}
	tmp := UUIDv4{}
	err := json.Unmarshal(b, &tmp)
	if err == nil {
		n.ID = tmp
		n.Valid = true
	}
	return err
}

func (n *NullUUIDv4) Scan(value interface{}) error {
	if value == nil {
		n.ID, n.Valid = UUIDv4{}, false
		return nil
	}
	if err := (&n.ID).Scan(value); err != nil {
		return err
	}
	n.Valid = true
	return nil
}

func (n NullUUIDv4) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.ID.Value()
}
