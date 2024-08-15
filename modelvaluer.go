package pgparty

import (
	"context"
	"fmt"
	"reflect"

	"github.com/covrom/pgparty/utils"
)

type ModelValuer interface {
	FieldID() any
	FieldValue(fd *FieldDescription) (any, error)
	SetValue(fd *FieldDescription, v any) error
}

// get field value by struct field name
// defval must be Model{}.Field
func Field[T Storable, F any](ctx context.Context, modelItem T, defval F, fieldName string) (F, error) {
	s, err := ShardFromContext(ctx)
	if err != nil {
		return defval, fmt.Errorf("FieldT: %w", err)
	}
	v, err := s.Store.Field(modelItem, fieldName)
	if err != nil {
		return defval, fmt.Errorf("FieldT: %w", err)
	}
	return v.(F), nil
}

// get field value by struct field name
func (sr *PgStore) Field(modelItem Storable, fieldName string) (any, error) {
	sn := sr.Schema()
	md, ok := sr.GetModelDescription(modelItem)
	if !ok {
		return nil, fmt.Errorf("Field error: cant't get model description for %T in schema %q", modelItem, sn)
	}
	fd, err := md.ColumnByFieldName(fieldName)
	if err != nil {
		return nil, err
	}
	return sr.FieldByFD(modelItem, fd)
}

func (sr *PgStore) FieldByFD(modelItem Storable, fd *FieldDescription) (any, error) {
	if fd == nil {
		return nil, fmt.Errorf("FieldByFD: fd is nil")
	}
	if mv, ok := modelItem.(ModelValuer); ok {
		return mv.FieldValue(fd)
	}
	fv, err := utils.GetFieldValueByName(reflect.Indirect(reflect.ValueOf(modelItem)), fd.FieldName)
	if err != nil {
		return nil, err
	}
	return fv.Interface(), nil
}
