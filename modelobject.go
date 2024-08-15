package pgparty

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	jsoniter "github.com/json-iterator/go"
)

type ModelObject struct {
	vals []any
	md   *ModelDesc
}

func ModelObjectFrom[T Storable](ctx context.Context, modelItem T) (ModelObject, error) {
	s, err := ShardFromContext(ctx)
	if err != nil {
		return ModelObject{}, fmt.Errorf("ModelObjectFrom: %w", err)
	}
	return s.Store.ModelObjectFrom(modelItem)
}

func (sr *PgStore) ModelObjectFrom(modelItem Storable) (ModelObject, error) {
	sn := sr.Schema()
	md, ok := sr.GetModelDescription(modelItem)
	if !ok {
		return ModelObject{}, fmt.Errorf("ModelObjectFrom error: cant't get model description for %T in schema %q", modelItem, sn)
	}
	vals := make([]any, md.ColumnPtrsCount())
	err := md.WalkColumnPtrs(func(i int, fd *FieldDescription) (e error) {
		vals[i], e = sr.FieldByFD(modelItem, fd)
		return
	})
	if err != nil {
		return ModelObject{}, err
	}
	return ModelObject{
		vals: vals,
		md:   md,
	}, nil
}

func (m *ModelObject) FieldID() any {
	if m.md.IdField() == nil {
		return nil
	}
	return m.vals[m.md.IdField().Idx]
}

func (m *ModelObject) FieldValue(fd *FieldDescription) (any, error) {
	if fd == nil {
		return nil, nil
	}
	if _, ok := m.md.allFDs[fd]; !ok {
		return nil, fmt.Errorf("FieldValue: fd is not exists in model description")
	}
	return m.vals[fd.Idx], nil
}

func (m *ModelObject) SetValue(fd *FieldDescription, v any) error {
	if fd == nil {
		return nil
	}
	if _, ok := m.md.allFDs[fd]; !ok {
		return fmt.Errorf("SetValue: fd is not exists in model description")
	}
	m.vals[fd.Idx] = v
	return nil
}

func (m *ModelObject) String() string {
	b, _ := json.Marshal(m)
	return string(b)
}

func (m *ModelObject) MD() *ModelDesc {
	return m.md
}

func (m *ModelObject) Clear() {
	for i := range m.vals {
		m.vals[i] = nil
	}
}

func (m *ModelObject) MarshalJSON() ([]byte, error) {
	b := &bytes.Buffer{}
	b.Grow(len(m.vals) * 32)

	enc := json.NewEncoder(b)

	b.WriteByte('{')
	comma := false

	for fdi, v := range m.vals {
		fd := m.md.ColumnPtr(fdi)
		if fd.JsonName == "" || fd.JsonSkip {
			continue
		}

		if fd.JsonOmitEmpty && reflect.ValueOf(v).IsZero() {
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
			return nil, fmt.Errorf("ModelObject enc.Encode error: %w", err)
		}

		comma = true
	}
	b.WriteByte('}')
	res := b.Bytes()

	return res, nil
}

func (m *ModelObject) UnmarshalJSON(b []byte) error {
	if bytes.EqualFold(b, []byte("null")) {
		m.Clear()
		return nil
	}

	iter := jsoniter.ParseBytes(jsoniter.ConfigCompatibleWithStandardLibrary, b)
	if iter.WhatIsNext() != jsoniter.ObjectValue {
		return fmt.Errorf("json must contain an object: %s", string(b))
	}

	iter.ReadObjectCB(func(it *jsoniter.Iterator, k string) bool {
		fd, err := m.md.ColumnByJsonName(k)
		if (err != nil) || fd.JsonSkip {
			return true
		}
		tempv := reflect.New(fd.ElemType).Interface()
		it.ReadVal(tempv)
		m.vals[fd.Idx] = tempv
		return true
	})

	return nil
}
