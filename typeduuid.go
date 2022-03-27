package pgparty

import (
	"bytes"
	"database/sql/driver"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
)

type UUIDType interface {
	UUIDPrefix() string
}

type AppUUID struct{}

func (u AppUUID) IDPrefix() string { return "app_" }

type TraceUUID struct{}

func (a TraceUUID) IDPrefix() string { return "trace_" }

type SomeUUID struct{}

func (u SomeUUID) IDPrefix() string { return "" }

type UUID[T UUIDType] UUIDv4

func NewUUID[T UUIDType]() UUID[T] { return UUID[T](UUIDNewV4()) }

func NilUUID[T UUIDType]() UUID[T] { return UUID[T](UUIDv4{}) }

func ParseUUID[T UUIDType](s string) (UUID[T], error) {
	var id UUIDv4
	var rt T
	if err := (&id).UnmarshalText([]byte(strings.TrimPrefix(s, rt.UUIDPrefix()))); err != nil {
		return NilUUID[T](), err
	}
	return UUID[T](id), nil
}

func (id UUID[T]) String() string {
	var resourceType T // create the default value for the resource type

	return fmt.Sprintf(
		"%s%s",
		resourceType.UUIDPrefix(), // Extract the "prefix" we want from the resource type
		UUIDv4(id).String(),       // Use ID's string marshalling
	)
}

func (UUID[T]) PostgresType() string {
	return "UUID"
}

func (u UUID[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(UUIDv4(u))
}

func (u *UUID[T]) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || bytes.EqualFold(b, []byte("null")) || bytes.EqualFold(b, []byte(`""`)) {
		*u = UUID[T](UUIDv4{})
		return nil
	}
	v := (*UUIDv4)(u)
	return json.Unmarshal(b, v)
}

func (u UUID[T]) Value() (driver.Value, error) {
	return UUIDv4(u).Value()
}

func (u *UUID[T]) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		return nil

	case string:
		// if an empty UUID comes from a table, we return a null UUID
		if src == "" {
			return nil
		}

		v := (*UUIDv4)(u)
		// see Parse for required string format
		return v.Scan([]byte(src))
	case []byte:
		// if an empty UUID comes from a table, we return a null UUID
		if len(src) == 0 {
			return nil
		}
		if bytes.EqualFold(src, []byte("null")) {
			*(*UUIDv4)(u) = UUIDv4{}
			return nil
		}
		v := (*UUIDv4)(u)
		return v.Scan(src)
	default:
		return fmt.Errorf("Scan: unable to scan type %T into UUID", src)
	}
}

func (u UUID[T]) IsZero() bool {
	return binary.BigEndian.Uint64(u.UUID[0:8]) == 0 && binary.BigEndian.Uint64(u.UUID[8:16]) == 0
}

func (u *UUID[T]) ConvertFrom(v interface{}) error {
	if v == nil {
		return nil
	}
	switch vv := v.(type) {
	case UUIDv4:
		*u = UUID[T](vv)
		return nil
	case *UUIDv4:
		*u = *(*UUID[T])(vv)
		return nil
	case UUID[T]:
		*u = vv
		return nil
	case *UUID[T]:
		*u = *vv
		return nil
	}
	if err := u.Scan(v); err != nil {
		return err
	}
	return nil
}

type UUIDJsonTyped[T UUIDType] UUID[T]

func (u UUIDJsonTyped[T]) MarshalJSON() ([]byte, error) {
	res, err := json.Marshal(UUIDv4(u))
	if err != nil {
		return res, err
	}
	var resourceType T
	prefix := resourceType.UUIDPrefix()
	ret := make([]byte, len(prefix)+len(res))
	ret[0] = res[0]
	copy(ret[1:], prefix)
	copy(ret[1+len(prefix):], res[1:])
	return ret, nil
}

func (u *UUIDJsonTyped[T]) UnmarshalJSON(b []byte) error {
	var resourceType T
	prefix := resourceType.UUIDPrefix()

	if len(b) > len(prefix)+1 && bytes.Equal([]byte(prefix), b[1:1+len(prefix)]) {
		copy(b[1:], b[1+len(prefix):])
		b = b[:len(b)-len(prefix)]
	}

	if len(b) == 0 || bytes.EqualFold(b, []byte("null")) || bytes.EqualFold(b, []byte(`""`)) {
		*u = UUIDJsonTyped[T](UUIDv4{})
		return nil
	}
	v := (*UUIDv4)(u)
	return json.Unmarshal(b, v)
}
