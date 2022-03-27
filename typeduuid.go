package pgparty

import (
	"bytes"
	"database/sql/driver"
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
		UUIDv4(id).String(),       // Use XID's string marshalling
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
