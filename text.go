package pgparty

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
)

func init() {
	t := Text("")
	gob.Register(&t)
}

type Text string

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
