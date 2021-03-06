package pgparty

import (
	"bytes"
	"database/sql/driver"
	"encoding/gob"
	"encoding/json"
	"fmt"

	"github.com/ericlagergren/decimal"
)

func init() {
	gob.Register(&NullDecimal{})
}

type NullDecimal152 NullDecimal

func (NullDecimal152) PostgresType() string {
	return "NUMERIC(15,2)"
}

type NullDecimal153 NullDecimal

func (NullDecimal153) PostgresType() string {
	return "NUMERIC(15,3)"
}

type NullDecimal192 NullDecimal

func (NullDecimal192) PostgresType() string {
	return "NUMERIC(19,2)"
}

// Nullable decimal, length 19 precision 6
type NullDecimal struct {
	Decimal Decimal
	Valid   bool // Valid is true if Decimal is not NULL
}

func (NullDecimal) PostgresTypeWithLenPrec(ln, prec int) string {
	return fmt.Sprintf("NUMERIC(%d,%d)", ln, prec)
}

func (NullDecimal) PostgresDefaultValue() string {
	return `'0.0'`
}

func (u NullDecimal) PostgresAllowNull() bool {
	return true
}

// Scan implements the Scanner interface.
func (n *NullDecimal) Scan(value interface{}) error {
	if value == nil {
		n.Decimal, n.Valid = Decimal{}, false
		return nil
	}
	d := Decimal{}
	err := (&d).Scan(value)
	if err != nil {
		return err
	}
	n.Valid = true
	n.Decimal = d
	return nil
}

// Value implements the driver Valuer interface.
func (n NullDecimal) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Decimal.Value()
}

func (n NullDecimal) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return json.Marshal(nil)
	}
	return json.Marshal(n.Decimal)
}

func (n *NullDecimal) UnmarshalJSON(b []byte) error {
	if bytes.EqualFold(b, []byte("null")) {
		n.Decimal = nil
		n.Valid = false
		return nil
	}
	tmp := &decimal.Big{Context: decimal.Context128}
	err := tmp.UnmarshalText(b)
	if err == nil {
		n.Decimal = make([]byte, len(b))
		copy(n.Decimal, b)
		n.Valid = true
	}
	return err
}

func (d *NullDecimal) ConvertFrom(v interface{}) error {
	if v == nil {
		return nil
	}
	switch vv := v.(type) {
	case Decimal:
		*d = NullDecimal{Decimal: vv, Valid: true}
		return nil
	case *Decimal:
		*d = NullDecimal{Decimal: *vv, Valid: true}
		return nil
	case NullDecimal:
		*d = vv
		return nil
	case *NullDecimal:
		*d = *vv
		return nil
	}
	t := Decimal{}
	if err := (&t).ConvertFrom(v); err != nil {
		return err
	}
	*d = NullDecimal{Decimal: t, Valid: true}
	return nil
}
