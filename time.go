package pgparty

import (
	"bytes"
	"database/sql/driver"
	"encoding/gob"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/covrom/rustime"
)

func init() {
	gob.Register(&Time{})
}

type Time time.Time

// Date()
const ISOTimeFormat = `"2006-01-02T15:04:05.999Z07:00"`

// date
const BrowserDateFormat = `"2006-01-02"`

// datetime-local
const BrowserDateTimeFormat = `"2006-01-02T15:04"`

func (u Time) PostgresType() string {
	return "TIMESTAMPTZ"
}

func (u Time) PostgresDefaultValue() string {
	return `'epoch'`
}

func (u Time) PostgresAllowNull() bool {
	return false
}

func (t Time) MarshalJSON() ([]byte, error) {
	tt := time.Time(t)
	if y := tt.Year(); y < 0 || y >= 10000 {
		return nil, errors.New("Timestamp.MarshalJSON: year outside of range [0,9999]")
	}
	if tt.IsZero() {
		return []byte("null"), nil
	}
	var fm string
	// if tt.Hour() == 0 && tt.Minute() == 0 && tt.Second() == 0 {
	// 	fm = BrowserDateFormat
	// 	tt = tt.UTC()
	// } else if tt.Nanosecond() == 0 {
	// 	fm = BrowserDateTimeFormat
	// 	tt = tt.UTC()
	// } else {
	fm = ISOTimeFormat
	tt = tt.UTC()
	// }

	b := make([]byte, 0, len(fm))
	b = tt.AppendFormat(b, fm)
	return b, nil
}

func (t *Time) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte(`""`)) || bytes.EqualFold(b, []byte(`null`)) {
		*t = Time(time.Time{})
		return nil
	}
	tt, err := time.ParseInLocation(ISOTimeFormat, string(b), time.UTC)
	if err == nil {
		*t = Time(tt)
	} else {
		tt, err = time.ParseInLocation(BrowserDateTimeFormat, string(b), time.UTC)
		if err == nil {
			*t = Time(tt)
		} else {
			tt, err = time.ParseInLocation(BrowserDateFormat, string(b), time.UTC)
			if err == nil {
				*t = Time(tt)
			}
		}
	}
	return err
}

func (t Time) MarshalBinary() ([]byte, error) {
	return time.Time(t).MarshalBinary()
}

func (t *Time) UnmarshalBinary(data []byte) error {
	return (*time.Time)(t).UnmarshalBinary(data)
}

func (t Time) GobEncode() ([]byte, error) {
	return time.Time(t).GobEncode()
}

func (t *Time) GobDecode(data []byte) error {
	return (*time.Time)(t).GobDecode(data)
}

func (t Time) String() string {
	return time.Time(t).String()
}

func (t Time) Time() time.Time {
	return time.Time(t)
}

func (t Time) Round(d time.Duration) Time {
	return Time(t.Time().Round(d))
}

// д (d) - день месяца (цифрами) без лидирующего нуля
// дд (dd) - день месяца (цифрами) с лидирующим нулем
// ддд (ddd) - краткое название дня недели
// дддд (dddd) - полное название дня недели
// М (M) - номер месяца (цифрами) без лидирующего нуля
// ММ (MM) - номер месяца (цифрами) с лидирующим нулем
// МММ (MMM) - краткое название месяца
// ММММ (MMMM) - полное название месяца
// К (Q) - номер квартала в году
// г (y) - номер года без века и лидирующего нуля
// гг (yy) - номер года без века с лидирующим нулем
// гггг (yyyy) - номер года с веком
// ч (h) - час в 24 часовом варианте без лидирующих нулей
// чч (hh) - час в 24 часовом варианте с лидирующим нулем
// м (m) - минута без лидирующего нуля
// мм (mm) - минута с лидирующим нулем
// с (s) - секунда без лидирующего нуля
// сс (ss) - секунда с лидирующим нулем
// ссс (sss) - миллисекунда с лидирующим нулем
func (t Time) FormatRus(f string) string {
	return rustime.FormatTimeRu(t.Time(), f)
}

// AssignToустанавливает значение целевой переменной с учетом ее типа.
// Поддерживаются типы: time.Time, Time, NullTime и указатели на них
func (t Time) AssignTo(v reflect.Value) {
	v = reflect.Indirect(v)
	switch v.Type() {
	case reflect.TypeOf(Time{}):
		v.Set(reflect.ValueOf(t))
	case reflect.TypeOf(time.Time{}):
		v.Set(reflect.ValueOf(time.Time(t)))
	case reflect.TypeOf(NullTime{}):
		v.Set(reflect.ValueOf(NullTime{t, t != Time{}}))
	}
}

// Scan implements the Scanner interface.
func (t *Time) Scan(value interface{}) (err error) {
	if value == nil {
		*t = Time{}
		return
	}
	switch v := value.(type) {
	case time.Time:
		*t = Time(v)
		return
	case []byte:
		tt, err := parseDateTime(string(v), time.UTC)
		*t = Time(tt)
		return err
	case string:
		tt, err := parseDateTime(v, time.UTC)
		*t = Time(tt)
		return err
	}

	return fmt.Errorf("Can't convert %T to time.Time", value)
}

// Value implements the driver Valuer interface.
func (t Time) Value() (driver.Value, error) {
	return time.Time(t), nil
}

func NowUTC() Time {
	return Time(time.Now().UTC())
}

// далее копия из драйвера sql, для других баз возможно надо делать другие
const timeFormat = "2006-01-02 15:04:05.999999"

func parseDateTime(str string, loc *time.Location) (t time.Time, err error) {
	base := "0000-00-00 00:00:00.0000000"
	switch len(str) {
	case 10, 19, 21, 22, 23, 24, 25, 26: // up to "YYYY-MM-DD HH:MM:SS.MMMMMM"
		if str == base[:len(str)] {
			return
		}
		t, err = time.Parse(timeFormat[:len(str)], str)
	default:
		err = fmt.Errorf("invalid time string: %s", str)
		return
	}

	// Adjust location
	if err == nil && loc != time.UTC {
		y, mo, d := t.Date()
		h, mi, s := t.Clock()
		t, err = time.Date(y, mo, d, h, mi, s, t.Nanosecond(), loc), nil
	}

	return
}

// store.Converter interface, t must contain zero value before call
func (t *Time) ConvertFrom(v interface{}) error {
	if v == nil {
		return nil
	}
	switch vv := v.(type) {
	case Time:
		*t = vv
		return nil
	case *Time:
		*t = *vv
		return nil
	case NullTime:
		*t = vv.Time
		return nil
	case *NullTime:
		*t = vv.Time
		return nil
	}

	value := reflect.Indirect(reflect.ValueOf(v))
	switch value.Kind() {
	case
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		*t = Time(time.Unix(value.Int(), 0))
		return nil

	case reflect.Float32, reflect.Float64:
		*t = Time(time.Unix(int64(value.Float()), 0))
		return nil

	case reflect.String:
		if tt, err := time.Parse(time.RFC3339, value.String()); err != nil {
			return err
		} else {
			*t = Time(tt)
			return nil
		}

	case reflect.Slice:
		if vv, ok := value.Interface().([]byte); ok {
			if tt, err := time.Parse(time.RFC3339, string(vv)); err != nil {
				return err
			} else {
				*t = Time(tt)
				return nil
			}
		}

	default:
		switch v := value.Interface().(type) {
		case NullTime:
			if v.Valid {
				*t = v.Time
				return nil
			}
			return nil

		case time.Time:
			*t = Time(v)
			return nil
		}
	}

	return fmt.Errorf("can't convert value of type %T to Time", value.Interface())
}
