package pgparty

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/gob"
	"encoding/json"
)

func init() {
	gob.Register(&NullText{})
}

type NullText sql.NullString

func (u NullText) PostgresType() string {
	return "TEXT"
}

func (n NullText) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return json.Marshal(nil)
	}
	return json.Marshal(n.String)
}

func (n *NullText) UnmarshalJSON(b []byte) error {
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

func (n *NullText) Scan(value interface{}) error {
	return (*sql.NullString)(n).Scan(value)
}

func (n NullText) Value() (driver.Value, error) {
	return sql.NullString(n).Value()
}

// store.Converter interface, n must contain zero value before call
func (n *NullText) ConvertFrom(v interface{}) error {
	if v == nil {
		return nil
	}
	switch vv := v.(type) {
	case string:
		*n = NullText{vv, true}
		return nil
	case *string:
		*n = NullText{*vv, true}
		return nil
	case NullText:
		*n = vv
		return nil
	case *NullText:
		*n = *vv
		return nil
	case []byte:
		*n = NullText{string(vv), true}
		return nil
	case *[]byte:
		*n = NullText{string(*vv), true}
	case Text:
		*n = NullText{string(vv), true}
		return nil
	case *Text:
		*n = NullText{string(*vv), true}
		return nil
	case NullString:
		*n = NullText{vv.String, true}
		return nil
	case *NullString:
		*n = NullText{vv.String, true}
		return nil
	}
	var s Text
	if err := (&s).ConvertFrom(v); err != nil {
		return err
	}
	*n = NullText{string(s), true}
	return nil
}
