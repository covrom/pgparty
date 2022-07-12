package pgparty

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

type JsonMasked[T Storable] struct {
	V      T
	MD     *ModelDesc
	Filled []*FieldDescription
}

func (mo *JsonMasked[T]) Valid() bool {
	return len(mo.Filled) > 0 && mo.MD != nil
}

func NewJsonMasked[T Storable]() (*JsonMasked[T], error) {
	val := *(new(T))
	md, err := (MD[T]{Val: val}).MD()
	if err != nil {
		return nil, err
	}
	ret := &JsonMasked[T]{
		V:      val,
		MD:     md,
		Filled: nil,
	}
	return ret, nil
}

func (mo *JsonMasked[T]) IsFilled(structFieldNames ...string) bool {
	allfnd := true
	for _, fn := range structFieldNames {
		fnd := false
		for _, fd := range mo.Filled {
			if fd.StructField.Name == fn {
				fnd = true
			}
		}
		allfnd = allfnd && fnd
	}
	return len(structFieldNames) > 0 && allfnd
}

func (mo *JsonMasked[T]) IsFullFilled() bool {
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

func (mo *JsonMasked[T]) SetFullFilled() {
	mo.Filled = make([]*FieldDescription, 0, mo.MD.ColumnPtrsCount())
	mo.MD.WalkColumnPtrs(func(_ int, fd *FieldDescription) error {
		mo.Filled = append(mo.Filled)
		return nil
	})
}

func (mo *JsonMasked[T]) MarshalJSON() ([]byte, error) {
	b := &bytes.Buffer{}
	b.Grow(len(mo.Filled) * 32)
	enc := json.NewEncoder(b)
	b.WriteByte('{')
	comma := false
	for _, fd := range mo.Filled {
		rv := reflect.ValueOf(mo.V).FieldByName(fd.StructField.Name)
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
			return nil, fmt.Errorf("ModelObject.MarshalJSON error: %w", err)
		}

		comma = true
	}
	b.WriteByte('}')
	res := b.Bytes()

	return res, nil
}

func (mo *JsonMasked[T]) UnmarshalJSON(b []byte) error {
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
	data := make(map[string]interface{})
	err = json.Unmarshal(b, &data)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &mo.V)
	if err != nil {
		return err
	}
	morv := reflect.Indirect(reflect.ValueOf(mo))
	mo.Filled = make([]*FieldDescription, 0, len(data))
	for k := range data {
		fd, err := mo.MD.ColumnByJsonName(k)
		if err != nil {
			continue
		}

		if fd.JsonSkip {
			newv := reflect.Zero(fd.StructField.Type)
			morv.FieldByName(fd.StructField.Name).Set(newv)
			continue
		}

		mo.Filled = append(mo.Filled, fd)
	}

	return nil
}
