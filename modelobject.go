package pgparty

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"

	jsoniter "github.com/json-iterator/go"
)

type ModelObject struct {
	vals []any
	md   *ModelDesc
}

func NewModelObject(md *ModelDesc) *ModelObject {
	vals := make([]any, md.ColumnPtrsCount())
	md.WalkColumnPtrs(func(i int, fd *FieldDescription) (e error) {
		vals[i] = reflect.New(fd.ElemType).Elem().Interface()
		return
	})
	return &ModelObject{
		vals: vals,
		md:   md,
	}
}

func ModelObjectFrom[T Modeller](ctx context.Context, modelItem T) (*ModelObject, error) {
	s, err := ShardFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("ModelObjectFrom: %w", err)
	}
	return s.Store.ModelObjectFrom(modelItem)
}

func (sr *PgStore) ModelObjectFrom(modelItem Modeller) (*ModelObject, error) {
	sn := sr.Schema()
	md, ok := sr.GetModelDescription(modelItem)
	if !ok {
		return nil, fmt.Errorf("ModelObjectFrom error: cant't get model description for %T in schema %q", modelItem, sn)
	}
	vals := make([]any, md.ColumnPtrsCount())
	err := md.WalkColumnPtrs(func(i int, fd *FieldDescription) (e error) {
		vals[i], e = sr.FieldByFD(modelItem, fd)
		return
	})
	if err != nil {
		return nil, err
	}
	return &ModelObject{
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

func (m *ModelObject) SetFieldID(id any) error {
	if m.md.IdField() == nil {
		return fmt.Errorf("id field not found")
	}
	return m.SetValue(m.md.IdField(), id)
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
	if v == nil {
		m.vals[fd.Idx] = v
	} else if reflect.TypeOf(v).AssignableTo(fd.ElemType) {
		m.vals[fd.Idx] = v
	} else if reflect.TypeOf(v).ConvertibleTo(fd.ElemType) {
		m.vals[fd.Idx] = reflect.ValueOf(v).Convert(fd.ElemType).Interface()
	} else {
		return fmt.Errorf("uncompatible type %T with field type %s", reflect.TypeOf(v), fd.ElemType.String())
	}

	return nil
}

func (m *ModelObject) FieldValueByName(fn string) (any, error) {
	fd, err := m.md.ColumnByFieldName(fn)
	if err != nil {
		return nil, err
	}
	return m.vals[fd.Idx], nil
}

func (m *ModelObject) SetValueByName(fn string, v any) error {
	fd, err := m.md.ColumnByFieldName(fn)
	if err != nil {
		return err
	}
	m.vals[fd.Idx] = v
	return nil
}

func (m *ModelObject) String() string {
	b, _ := json.Marshal(m)
	return string(b)
}

func (m *ModelObject) MD() (*ModelDesc, error) {
	return m.md, nil
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
		m.vals[fd.Idx] = reflect.Indirect(reflect.ValueOf(tempv)).Interface()
		return true
	})

	return nil
}

func (m *ModelObject) DBData() (cols []string, vals []any) {
	ln := m.md.ColumnPtrsCount()
	cols = make([]string, 0, ln)
	vals = make([]any, 0, ln)
	for fdi, v := range m.vals {
		fd := m.md.ColumnPtr(fdi)
		if fd.Skip {
			continue
		}
		cols = append(cols, fd.DatabaseName)
		vals = append(vals, v)
	}
	return
}

// json and jsonb value
func (m *ModelObject) Value() (driver.Value, error) {
	b := &bytes.Buffer{}
	b.Grow(m.md.ColumnPtrsCount() * 32)
	enc := json.NewEncoder(b)
	b.WriteByte('{')
	comma := false
	for fdi, v := range m.vals {
		fd := m.md.ColumnPtr(fdi)
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
			return nil, fmt.Errorf("ModelObjectJson.Value error: %w", err)
		}

		comma = true
	}
	b.WriteByte('}')
	res := b.Bytes()

	return res, nil
}

// sql database method for json_agg(expression)
func (m *ModelObject) Scan(value interface{}) error {
	if m.md == nil {
		return fmt.Errorf("ModelObject: model description is empty")
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
		m.Clear()
		return nil
	}

	iter := jsoniter.ParseBytes(jsoniter.ConfigCompatibleWithStandardLibrary, b)
	if iter.WhatIsNext() != jsoniter.ObjectValue {
		return fmt.Errorf("json must contain an object: %s", string(b))
	}

	iter.ReadObjectCB(func(it *jsoniter.Iterator, k string) bool {
		fd, err := m.md.ColumnByDatabaseName(k)
		if err != nil || fd.Skip {
			return true
		}
		tempv := reflect.New(fd.ElemType).Interface()
		it.ReadVal(tempv)
		m.vals[fd.Idx] = reflect.Indirect(reflect.ValueOf(tempv)).Interface()
		return true
	})

	return nil
}
