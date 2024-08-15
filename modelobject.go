package pgparty

import (
	"context"
	"fmt"
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
