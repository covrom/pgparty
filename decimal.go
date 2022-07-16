package pgparty

import (
	"bytes"
	"database/sql/driver"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/ericlagergren/decimal"
	"github.com/ericlagergren/decimal/math"
)

func init() {
	gob.Register(&Decimal{})
}

type Decimal152 Decimal

func (Decimal152) PostgresType() string {
	return "NUMERIC(15,2)"
}

type Decimal153 Decimal

func (Decimal153) PostgresType() string {
	return "NUMERIC(15,3)"
}

type Decimal192 Decimal

func (Decimal192) PostgresType() string {
	return "NUMERIC(19,2)"
}

type Decimal []byte

func (Decimal) PostgresTypeWithLenPrec(ln, prec int) string {
	return fmt.Sprintf("NUMERIC(%d,%d)", ln, prec)
}

func (Decimal) PostgresDefaultValue() string {
	return `'0.0'`
}

func (u Decimal) PostgresAllowNull() bool {
	return false
}

func (d Decimal) GetNumber() (*decimal.Big, error) {
	tmp := &decimal.Big{Context: decimal.Context128}

	if len(d) == 0 {
		return tmp, nil
	}

	err := tmp.UnmarshalText([]byte(d))

	return tmp, err
}

func (d *Decimal) SetNumber(dd *decimal.Big) {
	*d = []byte(dd.String())
}

func (d *Decimal) UnmarshalText(data []byte) error {
	*d = make([]byte, len(data))
	copy(*d, data)
	return nil
}

func (d Decimal) MarshalText() ([]byte, error) {
	return []byte(d), nil
}

func (d Decimal) String() string {
	return string(d)
}

func (d *Decimal) Scan(value interface{}) error {
	if value == nil {
		*d = Decimal("0")
		return nil
	}

	var dd Decimal

	switch val := value.(type) {
	case []byte:
		dd = make(Decimal, len(val))
		copy(dd, val)
	default:
		dd = Decimal([]byte(fmt.Sprint(value)))
	}

	_, err := (&dd).GetNumber()
	if err != nil {
		return err
	}
	*d = dd

	return nil
}

func (d Decimal) Value() (driver.Value, error) {
	if len(d) == 0 {
		return "0", nil
	}
	return string(d), nil
}

func (d Decimal) MarshalJSON() ([]byte, error) {
	if len(d) == 0 {
		return json.Marshal(nil)
	}
	return []byte(d), nil
}

func (d *Decimal) UnmarshalJSON(b []byte) error {
	if bytes.EqualFold(b, []byte("null")) {
		*d = []byte("0")
		return nil
	}
	tmp := &decimal.Big{Context: decimal.Context128}
	err := tmp.UnmarshalText(b)
	if err == nil {
		*d = make([]byte, len(b))
		copy(*d, b)
	}
	return err
}

// store.Converter interface, d must contain zero value before call
func (d *Decimal) ConvertFrom(v interface{}) error {
	if v == nil {
		return nil
	}
	switch vv := v.(type) {
	case Decimal:
		*d = vv
		return nil
	case *Decimal:
		*d = *vv
		return nil
	case NullDecimal:
		*d = vv.Decimal
		return nil
	case *NullDecimal:
		*d = vv.Decimal
		return nil
	}

	value := reflect.Indirect(reflect.ValueOf(v))
	switch value.Kind() {
	case
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint,
		reflect.Float32, reflect.Float64:
		*d = Decimal([]byte(fmt.Sprint(value.Interface())))
		return nil

	case reflect.String:
		*d = Decimal([]byte(value.String()))
		if _, err := d.GetNumber(); err != nil {
			return err
		}
		return nil

	case reflect.Slice:
		switch vv := value.Interface().(type) {
		case []byte:
			*d = Decimal(vv)
			if _, err := d.GetNumber(); err != nil {
				return err
			}
			return nil
		}
	}

	return fmt.Errorf("can't convert value of type %T to Decimal", value.Interface())
}

func (x *Decimal) ParseGoType(v interface{}) {
	switch vv := v.(type) {
	case int:
		*x = Decimal(decimal.New(int64(vv), 0).String())
	case int8:
		*x = Decimal(decimal.New(int64(vv), 0).String())
	case int16:
		*x = Decimal(decimal.New(int64(vv), 0).String())
	case int32:
		*x = Decimal(decimal.New(int64(vv), 0).String())
	case int64:
		*x = Decimal(decimal.New(vv, 0).String())
	case uint:
		*x = Decimal(decimal.New(int64(vv), 0).String())
	case uint8:
		*x = Decimal(decimal.New(int64(vv), 0).String())
	case uint16:
		*x = Decimal(decimal.New(int64(vv), 0).String())
	case uint32:
		*x = Decimal(decimal.New(int64(vv), 0).String())
	case uint64:
		num := &decimal.Big{Context: decimal.Context128}
		num.SetUint64(vv)
		*x = Decimal(num.String())
	case uintptr:
		num := &decimal.Big{Context: decimal.Context128}
		num.SetUint64(uint64(vv))
		*x = Decimal(num.String())
	case float32:
		num := &decimal.Big{Context: decimal.Context128}
		num.SetFloat64(float64(vv))
		*x = Decimal(num.String())
	case float64:
		num := &decimal.Big{Context: decimal.Context128}
		num.SetFloat64(vv)
		*x = Decimal(num.String())
	default:
		rv := reflect.Indirect(reflect.ValueOf(v))
		if rv.Kind() == reflect.Interface {
			rv = rv.Elem()
		}
		if rv.Kind() == reflect.Float32 || rv.Kind() == reflect.Float64 {
			num := &decimal.Big{Context: decimal.Context128}
			num.SetFloat64(rv.Float())
			*x = Decimal(num.String())
		} else {
			*x = Decimal(decimal.New(rv.Int(), 0).String())
		}
	}
}

func (x Decimal) Int() int64 {
	num, err := x.GetNumber()
	if err != nil {
		panic(err)
	}
	i, ok := num.Int64() // целая часть, без округления
	if !ok {
		return 0
	}
	return i
}

func (x Decimal) Uint() uint64 {
	num, err := x.GetNumber()
	if err != nil {
		panic(err)
	}
	i, ok := num.Uint64() // целая часть, без округления
	if !ok {
		return 0
	}
	return i
}

// до целого
func (x Decimal) RoundHalfUp() int64 {
	num, err := x.GetNumber()
	if err != nil {
		panic(err)
	}
	num.Context.RoundingMode = decimal.ToNearestAway
	i, ok := num.RoundToInt().Int64() // целая часть, округление вверх, если модуль>0.5
	if !ok {
		return 0
	}
	return i
}

// два знака после запятой 0.00
func (x Decimal) RoundHalfUpTo2Sign() Decimal {
	return x.RoundTo(2, decimal.ToNearestAway)
}

func (x Decimal) RoundTo(n int, m decimal.RoundingMode) Decimal {
	num, err := x.GetNumber()
	if err != nil {
		panic(err)
	}
	num.Context.RoundingMode = m
	num.Quantize(n)
	return Decimal(num.String())
}

func (x Decimal) Float() float64 {
	num, err := x.GetNumber()
	if err != nil {
		panic(err)
	}
	i, ok := num.Float64()
	if !ok {
		return i
	}
	return i
}

func (x Decimal) Bool() bool {
	num, err := x.GetNumber()
	if err != nil {
		panic(err)
	}
	return num.Cmp(&decimal.Big{Context: decimal.Context128}) != 0
}

func (x Decimal) Add(d2 Decimal) Decimal {
	xnum, err := x.GetNumber()
	if err != nil {
		panic(err)
	}
	d2num, err := d2.GetNumber()
	if err != nil {
		panic(err)
	}

	return Decimal(decimal.Context128.Add(&decimal.Big{Context: decimal.Context128}, xnum, d2num).String())
}

func (x Decimal) Sub(d2 Decimal) Decimal {
	xnum, err := x.GetNumber()
	if err != nil {
		panic(err)
	}
	d2num, err := d2.GetNumber()
	if err != nil {
		panic(err)
	}
	return Decimal(decimal.Context128.Sub(&decimal.Big{Context: decimal.Context128}, xnum, d2num).String())
}

func (x Decimal) Mul(d2 Decimal) Decimal {
	xnum, err := x.GetNumber()
	if err != nil {
		panic(err)
	}
	d2num, err := d2.GetNumber()
	if err != nil {
		panic(err)
	}
	return Decimal(decimal.Context128.Mul(&decimal.Big{Context: decimal.Context128}, xnum, d2num).String())
}

func (x Decimal) Pow(d2 Decimal) Decimal {
	xnum, err := x.GetNumber()
	if err != nil {
		panic(err)
	}
	d2num, err := d2.GetNumber()
	if err != nil {
		panic(err)
	}
	return Decimal(math.Pow(&decimal.Big{Context: decimal.Context128}, xnum, d2num).String())
}

func (x Decimal) Div(d2 Decimal) Decimal {
	xnum, err := x.GetNumber()
	if err != nil {
		panic(err)
	}
	d2num, err := d2.GetNumber()
	if err != nil {
		panic(err)
	}
	return Decimal(decimal.Context128.Quo(&decimal.Big{Context: decimal.Context128}, xnum, d2num).String())
}

func (x Decimal) Mod(d2 Decimal) Decimal {
	xnum, err := x.GetNumber()
	if err != nil {
		panic(err)
	}
	d2num, err := d2.GetNumber()
	if err != nil {
		panic(err)
	}
	_, tmp := decimal.Context128.QuoRem(&decimal.Big{Context: decimal.Context128}, xnum, d2num, &decimal.Big{Context: decimal.Context128})
	return Decimal(tmp.String())
}

func (x Decimal) Equal(d2 Decimal) bool {
	xnum, err := x.GetNumber()
	if err != nil {
		panic(err)
	}
	d2num, err := d2.GetNumber()
	if err != nil {
		panic(err)
	}
	return xnum.Cmp(d2num) == 0
}

func (x Decimal) Less(d2 Decimal) bool {
	xnum, err := x.GetNumber()
	if err != nil {
		panic(err)
	}
	d2num, err := d2.GetNumber()
	if err != nil {
		panic(err)
	}
	// x <  d2
	return xnum.Cmp(d2num) < 0
}

func (x Decimal) LessOrEqual(d2 Decimal) bool {
	xnum, err := x.GetNumber()
	if err != nil {
		panic(err)
	}
	d2num, err := d2.GetNumber()
	if err != nil {
		panic(err)
	}
	// x <=  d2
	return xnum.Cmp(d2num) <= 0
}

func NewDecimalFromInt64(x int64) Decimal {
	return Decimal(decimal.New(x, 0).String())
}

func NewDecimalFromUint64(x uint64) Decimal {
	dn := &decimal.Big{Context: decimal.Context128}
	dn.SetUint64(x)
	return Decimal(dn.String())
}

func NewDecimalFromFloat64(x float64) Decimal {
	dn := &decimal.Big{Context: decimal.Context128}
	dn.SetFloat64(x)
	return Decimal(dn.String())
}

func NewDecimalFromInt(x int) Decimal {
	return NewDecimalFromInt64(int64(x))
}

func (x Decimal) Negative() Decimal {
	num, err := x.GetNumber()
	if err != nil {
		panic(err)
	}
	return Decimal((&decimal.Big{Context: decimal.Context128}).Neg(num).String())
}

// consts

func Zero() Decimal {
	return Decimal((&decimal.Big{Context: decimal.Context128}).String())
}

func One() Decimal {
	return NewDecimalFromInt64(1)
}

func NegOne() Decimal {
	return NewDecimalFromInt64(-1)
}

func NewDecimalFromAny(v interface{}) (*Decimal, error) {
	d := &Decimal{}
	if err := d.ConvertFrom(v); err != nil {
		return nil, err
	}
	return d, nil
}

func Float64ToDecimal(v float64) Decimal {
	num := &decimal.Big{Context: decimal.Context128}
	num.SetFloat64(v)
	return Decimal(num.String())
}
