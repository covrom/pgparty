package pgparty

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rs/xid"
)

type XIDType interface {
	XIDPrefix() string
}

type XID[T XIDType] xid.ID

func NewXID[T XIDType]() XID[T] { return XID[T](xid.New()) }

func NilXID[T XIDType]() XID[T] { return XID[T](xid.NilID()) }

func ParseXID[T XIDType](s string) (XID[T], error) {
	var id xid.ID
	var rt T
	if err := (&id).UnmarshalText([]byte(strings.TrimPrefix(s, rt.XIDPrefix()))); err != nil {
		return NilXID[T](), err
	}
	return XID[T](id), nil
}

func (id XID[T]) String() string {
	var resourceType T // create the default value for the resource type

	return fmt.Sprintf(
		"%s%s",
		resourceType.XIDPrefix(), // Extract the "prefix" we want from the resource type
		xid.ID(id).String(),      // Use XID's string marshalling
	)
}

func (XID[T]) PostgresType() string {
	return "CHAR(20)"
}

func (u XID[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(xid.ID(u))
}

func (u *XID[T]) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || bytes.EqualFold(b, []byte("null")) || bytes.EqualFold(b, []byte(`""`)) {
		*u = XID[T](xid.NilID())
		return nil
	}
	v := (*xid.ID)(u)
	return json.Unmarshal(b, v)
}

func (u XID[T]) Value() (driver.Value, error) {
	return xid.ID(u).Value()
}

func (u *XID[T]) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		return nil

	case string:
		// if an empty UUID comes from a table, we return a null UUID
		if src == "" {
			return nil
		}

		v := (*xid.ID)(u)
		// see Parse for required string format
		return v.Scan([]byte(src))
	case []byte:
		// if an empty UUID comes from a table, we return a null UUID
		if len(src) == 0 {
			return nil
		}
		if bytes.EqualFold(src, []byte("null")) {
			*(*xid.ID)(u) = xid.ID{}
			return nil
		}
		v := (*xid.ID)(u)
		return v.Scan(src)
	default:
		return fmt.Errorf("Scan: unable to scan type %T into XID", src)
	}
}

type XIDJsonTyped[T XIDType] XID[T]

func (u XIDJsonTyped[T]) MarshalJSON() ([]byte, error) {
	res, err := json.Marshal(XID[T](u))
	if err != nil {
		return res, err
	}
	var resourceType T
	prefix := resourceType.XIDPrefix()
	ret := make([]byte, len(prefix)+len(res))
	ret[0] = res[0]
	copy(ret[1:], prefix)
	copy(ret[1+len(prefix):], res[1:])
	return ret, nil
}

func (u *XIDJsonTyped[T]) UnmarshalJSON(b []byte) error {
	var resourceType T
	prefix := resourceType.XIDPrefix()

	if len(b) > len(prefix)+1 && bytes.Equal([]byte(prefix), b[1:1+len(prefix)]) {
		copy(b[1:], b[1+len(prefix):])
		b = b[:len(b)-len(prefix)]
	}

	if len(b) == 0 || bytes.EqualFold(b, []byte("null")) || bytes.EqualFold(b, []byte(`""`)) {
		*u = XIDJsonTyped[T](XID[T]{})
		return nil
	}
	v := (*XID[T])(u)
	return json.Unmarshal(b, v)
}

type AppXID struct{}

func (u AppXID) XIDPrefix() string { return "app_" }

type TraceXID struct{}

func (a TraceXID) XIDPrefix() string { return "trace_" }

type SomeXID struct{}

func (u SomeXID) XIDPrefix() string { return "" }
