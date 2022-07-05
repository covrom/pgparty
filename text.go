package pgparty

import (
	"database/sql/driver"
	"encoding/gob"
	"encoding/json"
	"fmt"
)

func init() {
	t := Text("")
	gob.Register(&t)
}

type Text string

func (u Text) PostgresType() string {
	return "TEXT"
}

func (u Text) Value() (driver.Value, error) {
	return []byte(u), nil
}

func (u *Text) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		return nil

	case string:
		if src == "" {
			return nil
		}
		*u = Text(src)
	case []byte:
		if len(src) == 0 {
			return nil
		}
		*u = Text(src)
	}

	return fmt.Errorf("Scan: unable to scan type %T into Text", src)
}

func (s *Text) ConvertFrom(v interface{}) error {
	if v == nil {
		return nil
	}
	switch vv := v.(type) {
	case string:
		*s = Text(vv)
		return nil
	case *string:
		*s = Text(*vv)
		return nil
	case NullString:
		*s = Text(vv.String)
		return nil
	case *NullString:
		*s = Text(vv.String)
		return nil
	case []byte:
		*s = Text(vv)
		return nil
	case *[]byte:
		*s = Text(*vv)
	case Text:
		*s = vv
		return nil
	case *Text:
		*s = *vv
		return nil
	case NullText:
		*s = Text(vv.String)
		return nil
	case *NullText:
		*s = Text(vv.String)
		return nil
	}
	jsdata, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("can't convert value of type %T to json-string: %s", v, err)
	}
	*s = Text(jsdata)
	return nil
}
