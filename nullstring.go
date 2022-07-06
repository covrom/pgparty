package pgparty

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/gob"
	"encoding/json"
	"fmt"
)

func init() {
	gob.Register(&NullString{})
}

type NullString sql.NullString

func (NullString) PostgresType() string {
	return "VARCHAR"
}

func (NullString) PostgresDefaultValue() string {
	return `''`
}

func (n NullString) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return json.Marshal(nil)
	}
	return json.Marshal(n.String)
}

func (n *NullString) UnmarshalJSON(b []byte) error {
	if bytes.EqualFold(b, []byte("null")) {
		n.String = ""
		n.Valid = false
		return nil
	}
	tmp := ""
	err := json.Unmarshal(b, &tmp)
	if err == nil {
		n.String = tmp
		n.Valid = true
	}
	return err
}

func (n *NullString) Scan(value interface{}) error {
	return (*sql.NullString)(n).Scan(value)
}

func (n NullString) Value() (driver.Value, error) {
	return sql.NullString(n).Value()
}

type String string

func (String) PostgresType() string {
	return "VARCHAR"
}

func (String) PostgresDefaultValue() string {
	return `''`
}

func (n String) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(n))
}

func (n *String) UnmarshalJSON(b []byte) error {
	if bytes.EqualFold(b, []byte("null")) {
		*n = ""
		return nil
	}
	var tmp string
	err := json.Unmarshal(b, &tmp)
	if err == nil {
		*n = String(tmp)
	}
	return err
}

func (n *String) Scan(value interface{}) error {
	tmp := &sql.NullString{}
	if err := tmp.Scan(value); err != nil {
		return err
	}
	if tmp.Valid {
		*n = String(tmp.String)
	} else {
		*n = ""
	}
	return nil
}

func (n String) Value() (driver.Value, error) {
	return (sql.NullString{String: string(n), Valid: true}).Value()
}

func (n String) String() string {
	return string(n)
}

func (n *String) SetString(s string) {
	*n = String(s)
}

// store.Converter interface, n must contain zero value before call
func (s *String) ConvertFrom(v interface{}) error {
	if v == nil {
		return nil
	}
	switch vv := v.(type) {
	case string:
		*s = String(vv)
		return nil
	case *string:
		*s = String(*vv)
		return nil
	case NullString:
		*s = String(vv.String)
		return nil
	case *NullString:
		*s = String(vv.String)
		return nil
	case []byte:
		*s = String(vv)
		return nil
	case *[]byte:
		*s = String(*vv)
	case Text:
		*s = String(vv)
		return nil
	case *Text:
		*s = String(*vv)
		return nil
	case NullText:
		*s = String(vv.String)
		return nil
	case *NullText:
		*s = String(vv.String)
		return nil
	}
	jsdata, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("can't convert value of type %T to json-string: %s", v, err)
	}
	*s = String(jsdata)
	return nil
}

// store.Converter interface, n must contain zero value before call
func (n *NullString) ConvertFrom(v interface{}) error {
	if v == nil {
		return nil
	}
	switch vv := v.(type) {
	case string:
		*n = NullString{vv, true}
		return nil
	case *string:
		*n = NullString{*vv, true}
		return nil
	case NullString:
		*n = vv
		return nil
	case *NullString:
		*n = *vv
		return nil
	case []byte:
		*n = NullString{string(vv), true}
		return nil
	case *[]byte:
		*n = NullString{string(*vv), true}
	case Text:
		*n = NullString{string(vv), true}
		return nil
	case *Text:
		*n = NullString{string(*vv), true}
		return nil
	case NullText:
		*n = NullString{vv.String, true}
		return nil
	case *NullText:
		*n = NullString{vv.String, true}
		return nil
	}
	var s String
	if err := (&s).ConvertFrom(v); err != nil {
		return err
	}
	*n = NullString{string(s), true}
	return nil
}
