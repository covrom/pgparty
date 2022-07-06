package pgparty

import (
	"bytes"
	"database/sql/driver"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"time"
)

func init() {
	gob.Register(&NullTime{})
}

type NullTime struct {
	Time  Time
	Valid bool // Valid is true if Time is not NULL
}

func (u NullTime) PostgresType() string {
	return "TIMESTAMPTZ"
}

func (u NullTime) PostgresDefaultValue() string {
	return `'epoch'`
}

func (nt *NullTime) Scan(value interface{}) (err error) {
	if value == nil {
		nt.Time, nt.Valid = Time{}, false
		return
	}

	switch v := value.(type) {
	case time.Time:
		nt.Time, nt.Valid = Time(v), true
		return
	case Time:
		nt.Time, nt.Valid = v, true
		return
	case []byte:
		tt, err := parseDateTime(string(v), time.UTC)
		nt.Time = Time(tt)
		nt.Valid = (err == nil)
		return err
	case string:
		tt, err := parseDateTime(v, time.UTC)
		nt.Time = Time(tt)
		nt.Valid = (err == nil)
		return err
	}

	nt.Valid = false
	return fmt.Errorf("Can't convert %T to time.Time", value)
}

// Value implements the driver Valuer interface.
func (nt NullTime) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	return nt.Time.Value()
}

func (n NullTime) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return json.Marshal(nil)
	}
	return json.Marshal(n.Time)
}

func (n *NullTime) UnmarshalJSON(b []byte) error {
	if bytes.EqualFold(b, []byte("null")) {
		n.Time = Time{}
		n.Valid = false
		return nil
	}
	err := json.Unmarshal(b, &n.Time)
	n.Valid = (err == nil)
	return err
}

// store.Converter interface, t must contain zero value before call
func (t *NullTime) ConvertFrom(v interface{}) error {
	if v == nil {
		return nil
	}
	switch vv := v.(type) {
	case Time:
		*t = NullTime{vv, true}
		return nil
	case *Time:
		*t = NullTime{*vv, true}
		return nil
	case NullTime:
		*t = vv
		return nil
	case *NullTime:
		*t = *vv
		return nil
	}
	tt := Time{}
	if err := (&tt).ConvertFrom(v); err != nil {
		return err
	}
	*t = NullTime{tt, true}
	return nil
}

func (nt NullTime) String() string {
	if !nt.Valid {
		return ""
	}
	return nt.Time.String()
}
