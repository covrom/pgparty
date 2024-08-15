package pgparty

import (
	"context"
	"fmt"
)

type ModelObject struct {
	elem    Storable
	idField any
	md      *ModelDesc
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
	idf, err := sr.FieldByFD(modelItem, md.IdField())
	if err != nil {
		return ModelObject{}, err
	}
	return ModelObject{
		elem:    modelItem,
		idField: idf,
		md:      md,
	}, nil
}
