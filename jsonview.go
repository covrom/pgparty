package pgparty

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	jsoniter "github.com/json-iterator/go"
)

type aJsonViewer interface {
	aJsonView()
}

func IsJsonView(v interface{}) bool {
	_, ok := v.(aJsonViewer)
	return ok
}

type JsonViewer[T Storable] interface {
	JsonView() *JsonView[T]
}

type JsonView[T Storable] struct {
	V      T
	MD     *ModelDesc
	Filled []*FieldDescription
}

var _ aJsonViewer = &JsonView[Storable]{}

func (mo *JsonView[T]) aJsonView() {}

func (mo *JsonView[T]) Valid() bool {
	return len(mo.Filled) > 0 && mo.MD != nil
}

func NewJsonView[T Storable]() (*JsonView[T], error) {
	val := *(new(T))
	md, err := (MD[T]{Val: val}).MD()
	if err != nil {
		return nil, err
	}
	ret := &JsonView[T]{
		V:      val,
		MD:     md,
		Filled: nil,
	}
	return ret, nil
}

func (mo *JsonView[T]) IsFilled(structFieldNames ...string) bool {
	allfnd := true
	for _, fn := range structFieldNames {
		fnd := false
		for _, fd := range mo.Filled {
			if fd.FieldName == fn {
				fnd = true
			}
		}
		allfnd = allfnd && fnd
	}
	return len(structFieldNames) > 0 && allfnd
}

func (mo *JsonView[T]) SetFilled(structFieldNames ...string) error {
	for _, fn := range structFieldNames {
		if mo.IsFilled(fn) {
			continue
		}
		fd, err := mo.MD.ColumnByFieldName(fn)
		if err != nil {
			return err
		}
		mo.Filled = append(mo.Filled, fd)
	}
	return nil
}

func (mo *JsonView[T]) SetUnfilled(structFieldNames ...string) error {
	i := 0
	for _, fd := range mo.Filled {
		fnd := false
		for _, fn := range structFieldNames {
			if fd.FieldName == fn {
				fnd = true
				break
			}
		}
		if !fnd {
			mo.Filled[i] = fd
			i++
		}
	}
	for j := i; j < len(mo.Filled); j++ {
		mo.Filled[j] = nil
	}
	mo.Filled = mo.Filled[:i]

	return nil
}

func (mo *JsonView[T]) IsFullFilled() bool {
	allfilled := true
	mo.MD.WalkColumnPtrs(func(_ int, fd *FieldDescription) error {
		for _, fdf := range mo.Filled {
			if fdf == fd {
				return nil
			}
		}
		allfilled = false
		return errors.New("break")
	})
	return allfilled && mo.MD.ColumnPtrsCount() > 0
}

func (mo *JsonView[T]) SetFullFilled() {
	mo.Filled = make([]*FieldDescription, 0, mo.MD.ColumnPtrsCount())
	mo.MD.WalkColumnPtrs(func(_ int, fd *FieldDescription) error {
		mo.Filled = append(mo.Filled)
		return nil
	})
}

func (mo *JsonView[T]) MarshalJSON() ([]byte, error) {
	b := &bytes.Buffer{}
	b.Grow(len(mo.Filled) * 32)
	enc := json.NewEncoder(b)
	b.WriteByte('{')
	comma := false
	for _, fd := range mo.Filled {
		rv := reflect.ValueOf(mo.V).FieldByName(fd.FieldName)
		v := rv.Interface()
		if v == nil || fd.JsonName == "" {
			continue
		}

		if fd.JsonOmitEmpty && rv.IsZero() {
			continue
		}

		if comma {
			b.WriteByte(',')
		}
		b.WriteByte('"')
		b.WriteString(fd.JsonName)
		b.WriteByte('"')
		b.WriteByte(':')

		if err := enc.Encode(v); err != nil {
			return nil, fmt.Errorf("JsonView.MarshalJSON error: %w", err)
		}

		comma = true
	}
	b.WriteByte('}')
	res := b.Bytes()

	return res, nil
}

func (mo *JsonView[T]) UnmarshalJSON(b []byte) error {
	mo.V = *(new(T))
	md, err := (MD[T]{Val: mo.V}).MD()
	if err != nil {
		return err
	}
	mo.MD = md
	mo.Filled = nil
	if bytes.EqualFold(b, []byte("null")) {
		mo.MD = nil
		return nil
	}
	if mo.MD == nil {
		return fmt.Errorf("model description not found for %T", mo.V)
	}

	iter := jsoniter.ParseBytes(jsoniter.ConfigCompatibleWithStandardLibrary, b)
	if iter.WhatIsNext() != jsoniter.ObjectValue {
		return fmt.Errorf("json must contain an object")
	}

	morv := reflect.ValueOf(&mo.V)
	mo.Filled = make([]*FieldDescription, 0, len(mo.MD.columns))

	iter.ReadObjectCB(func(it *jsoniter.Iterator, k string) bool {
		fd, err := mo.MD.ColumnByJsonName(k)
		if err != nil {
			return true
		}

		if fd.JsonSkip {
			newv := reflect.Zero(fd.ElemType)
			morv.Elem().FieldByName(fd.FieldName).Set(newv)
			return true
		}

		mo.Filled = append(mo.Filled, fd)
		f := morv.Elem().FieldByName(fd.FieldName).Addr().Interface()
		it.ReadVal(f)

		return true
	})

	return nil
}

// sql database json/jsonb value
func (mo *JsonView[T]) Value() (driver.Value, error) {
	b := &bytes.Buffer{}
	b.Grow(len(mo.Filled) * 32)
	enc := json.NewEncoder(b)
	b.WriteByte('{')
	comma := false
	for _, fd := range mo.Filled {
		rv := reflect.ValueOf(mo.V).FieldByName(fd.FieldName)
		v := rv.Interface()
		if fd.Skip {
			continue
		}

		if comma {
			b.WriteByte(',')
		}
		b.WriteByte('"')
		b.WriteString(fd.DatabaseName)
		b.WriteByte('"')
		b.WriteByte(':')

		if err := enc.Encode(v); err != nil {
			return nil, fmt.Errorf("JsonView.Value error: %w", err)
		}

		comma = true
	}
	b.WriteByte('}')
	res := b.Bytes()

	return res, nil
}

// sql database json/jsonb value
func (mo *JsonView[T]) Scan(value interface{}) error {
	mo.V = *(new(T))
	md, err := (MD[T]{Val: mo.V}).MD()
	if err != nil {
		return err
	}
	mo.MD = md
	mo.Filled = nil
	if mo.MD == nil {
		return fmt.Errorf("model description not found for %T", mo.V)
	}

	if value == nil {
		return nil
	}

	var b []byte

	switch bb := value.(type) {
	case []byte:
		b = bb
	case string:
		b = []byte(bb)
	default:
		return fmt.Errorf("unsupported database data type %T, needs []byte", value)
	}

	if bytes.EqualFold(b, []byte("null")) {
		mo.MD = nil
		return nil
	}
	if mo.MD == nil {
		return fmt.Errorf("model description not found for %T", mo.V)
	}

	iter := jsoniter.ParseBytes(jsoniter.ConfigCompatibleWithStandardLibrary, b)
	if iter.WhatIsNext() != jsoniter.ObjectValue {
		return fmt.Errorf("json must contain an object: %s", string(b))
	}

	morv := reflect.ValueOf(&mo.V)
	mo.Filled = make([]*FieldDescription, 0, len(mo.MD.columns))

	iter.ReadObjectCB(func(it *jsoniter.Iterator, k string) bool {
		fd, ok := mo.MD.columnByName[k]
		if !ok {
			return true
		}

		if fd.Skip {
			newv := reflect.Zero(fd.ElemType)
			morv.Elem().FieldByName(fd.FieldName).Set(newv)
			return true
		}

		mo.Filled = append(mo.Filled, fd)
		f := morv.Elem().FieldByName(fd.FieldName).Addr().Interface()
		if IsJsonView(f) &&
			it.WhatIsNext() == jsoniter.ObjectValue {
			if err := f.(sql.Scanner).Scan(it.SkipAndReturnBytes()); err != nil {
				it.ReportError("sql.Scan", err.Error())
				return false
			}
		} else {
			it.ReadVal(f)
		}
		return true
	})

	return nil
}

func (jv *JsonView[T]) SQLView() *SQLView[T] {
	sv := &SQLView[T]{
		V:  jv.V,
		MD: jv.MD,
	}
	if len(jv.Filled) > 0 {
		sv.Filled = make([]*FieldDescription, len(jv.Filled))
		copy(sv.Filled, jv.Filled)
	}
	return sv
}
