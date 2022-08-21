package pgparty

import (
	"bytes"
	"database/sql/driver"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"

	"github.com/btcsuite/btcutil/base58"
	"github.com/google/uuid"
)

type UUIDv4 struct {
	uuid.UUID
}

func init() {
	gob.Register(&UUIDv4{})
	gob.Register(&UUID58{})
}

func (UUIDv4) PostgresType() string {
	return "UUID"
}

func (UUIDv4) PostgresDefaultValue() string {
	var empty UUIDv4
	return fmt.Sprintf(`'%s'`, empty)
}

func (UUIDv4) PostgresAllowNull() bool {
	return false
}

func (u UUIDv4) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.UUID)
}

func (u *UUIDv4) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || bytes.EqualFold(b, []byte("null")) || bytes.EqualFold(b, []byte(`""`)) {
		*u = UUIDv4{}
		return nil
	}
	return json.Unmarshal(b, &u.UUID)
}

func (u UUIDv4) Value() (driver.Value, error) {
	return []byte(u.UUID[:]), nil
}

func (u *UUIDv4) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		return nil

	case string:
		// if an empty UUID comes from a table, we return a null UUID
		if src == "" {
			return nil
		}

		// see Parse for required string format
		uu, err := uuid.Parse(src)
		if err != nil {
			return fmt.Errorf("Scan: %v", err)
		}
		u.UUID = uu

	case []byte:
		// if an empty UUID comes from a table, we return a null UUID
		if len(src) == 0 {
			return nil
		}
		if bytes.EqualFold(src, []byte("null")) {
			u.UUID = uuid.UUID{}
			return nil
		}
		// assumes a simple slice of bytes if 16 bytes
		// otherwise attempts to parse
		if len(src) != 16 {
			// see Parse for required string format
			uu, err := uuid.ParseBytes(src)
			if err != nil {
				return fmt.Errorf("Scan: %v", err)
			}
			u.UUID = uu
		} else {
			copy(u.UUID[:], src)
		}

	default:
		return fmt.Errorf("Scan: unable to scan type %T into UUID", src)
	}

	return nil
}

func (u UUIDv4) IsZero() bool {
	return binary.BigEndian.Uint64(u.UUID[0:8]) == 0 && binary.BigEndian.Uint64(u.UUID[8:16]) == 0
}

func (u UUIDv4) String() string {
	return u.UUID.String()
}

func (u UUIDv4) GobEncode() ([]byte, error) {
	return u.MarshalBinary()
}

func (u *UUIDv4) GobDecode(data []byte) error {
	return (&(u.UUID)).UnmarshalBinary(data)
}

func UUIDNewV4() UUIDv4 {
	return UUIDv4{uuid.New()}
}

func UUIDFromString(s string) (UUIDv4, error) {
	id, err := uuid.Parse(s)
	return UUIDv4{id}, err
}

func UUIDMustFromString(s string) UUIDv4 {
	id := uuid.MustParse(s)
	return UUIDv4{id}
}

// store.Converter interface, u must contain zero value before call
func (u *UUIDv4) ConvertFrom(v interface{}) error {
	if v == nil {
		return nil
	}
	switch vv := v.(type) {
	case UUIDv4:
		*u = vv
		return nil
	case *UUIDv4:
		*u = *vv
		return nil
	}
	if err := u.Scan(v); err != nil {
		return err
	}
	return nil
}

type UUID58 UUIDv4

func (UUID58) PostgresType() string {
	return "UUID"
}

func (UUID58) PostgresDefaultValue() string {
	var empty UUIDv4
	return fmt.Sprintf(`'%s'`, empty)
}

func (UUID58) PostgresAllowNull() bool {
	return false
}

func (u UUID58) MarshalJSON() ([]byte, error) {
	u58 := base58.Encode(u.UUID[:])
	return json.Marshal(u58)
}

func (u *UUID58) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || bytes.EqualFold(b, []byte("null")) || bytes.EqualFold(b, []byte(`""`)) {
		*u = UUID58{}
		return nil
	}
	var u58 string
	if err := json.Unmarshal(b, &u58); err != nil {
		return err
	}
	if uid, err := uuid.FromBytes(base58.Decode(u58)); err != nil {
		return err
	} else {
		u.UUID = uid
	}
	return nil
}

func (u UUID58) Value() (driver.Value, error) {
	return []byte(u.UUID[:]), nil
}

func (u *UUID58) Scan(src interface{}) error {
	return ((*UUIDv4)(u)).Scan(src)
}

func (u UUID58) IsZero() bool {
	return UUIDv4(u).IsZero()
}

func (u UUID58) String() string {
	return u.UUID.String()
}

func (u UUID58) GobEncode() ([]byte, error) {
	return u.MarshalBinary()
}

func (u *UUID58) GobDecode(data []byte) error {
	return (&(u.UUID)).UnmarshalBinary(data)
}

func UUIDNew58() UUID58 {
	return UUID58{uuid.New()}
}

func UUID58FromString(s string) (UUID58, error) {
	id, err := uuid.Parse(s)
	return UUID58{id}, err
}

func UUID58MustFromString(s string) UUID58 {
	id := uuid.MustParse(s)
	return UUID58{id}
}

// store.Converter interface, u must contain zero value before call
func (u *UUID58) ConvertFrom(v interface{}) error {
	if v == nil {
		return nil
	}
	switch vv := v.(type) {
	case UUID58:
		*u = vv
		return nil
	case *UUID58:
		*u = *vv
		return nil
	case UUIDv4:
		*u = UUID58(vv)
		return nil
	case *UUIDv4:
		*u = UUID58(*vv)
		return nil
	}

	if err := u.Scan(v); err != nil {
		return err
	}
	return nil
}
