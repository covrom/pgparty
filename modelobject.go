package pgparty

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/covrom/pgparty/utils"
)

// get field value by struct field name
// defval must be Model{}.Field
func Field[T Storable, F any](ctx context.Context, modelItem T, fieldName string, defval F) (F, error) {
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
func (sr *PgStore) Field(modelItem Storable, fieldName string) (interface{}, error) {
	sn := sr.Schema()
	md, ok := sr.GetModelDescription(modelItem)
	if !ok {
		return nil, fmt.Errorf("Field error: cant't get model description for %T in schema %q", modelItem, sn)
	}
	fd, err := md.ColumnByFieldName(fieldName)
	if err != nil {
		return nil, err
	}
	fv, err := utils.GetFieldValueByName(reflect.Indirect(reflect.ValueOf(modelItem)), fd.StructField.Name)
	if err != nil {
		return nil, err
	}
	return fv.Interface(), nil
}

var uoPool = sync.Pool{}

func getMOSlice(c int) []ModelObject {
	sl := uoPool.Get()
	if sl != nil {
		vsl := sl.([]ModelObject)
		if cap(vsl) >= c {
			return vsl
		}
	}
	return make([]ModelObject, 0, c)
}

func putMOSlice(sl []ModelObject) {
	if sl == nil {
		return
	}
	uoPool.Put(sl[:0])
}

type ModelObject struct {
	elem    Storable
	idField interface{}
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
	idf, err := sr.Field(modelItem, IDField)
	if err != nil {
		return ModelObject{}, err
	}
	return ModelObject{
		elem:    modelItem,
		idField: idf,
		md:      md,
	}, nil
}

type UniqueObjects struct {
	objs []ModelObject
	uniq map[interface{}]int
}

func NewUniqueObjects() *UniqueObjects {
	return &UniqueObjects{
		objs: getMOSlice(16),
		uniq: make(map[interface{}]int),
	}
}

func (uo *UniqueObjects) Object(id interface{}) (ModelObject, bool) {
	if i, ok := uo.uniq[id]; ok {
		return uo.objs[i], true
	}
	return ModelObject{}, false
}

func (uo *UniqueObjects) Objects() []ModelObject {
	return uo.objs
}

func (uo *UniqueObjects) CopyObjects() (res []ModelObject) {
	res = make([]ModelObject, len(uo.objs))
	copy(res, uo.objs)
	return
}

func (uo *UniqueObjects) AddObject(data ModelObject) error {
	id := data.idField
	if id == nil {
		return fmt.Errorf("UniqueObjects.AddObject: data doesn't have id value")
	}
	i := len(uo.objs)
	uo.objs = append(uo.objs, data)
	uo.uniq[id] = i
	return nil
}

func (uo *UniqueObjects) Close() {
	putMOSlice(uo.objs)
	uo.objs = nil
	uo.uniq = nil
}

func (uo *UniqueObjects) Reset() {
	for i := range uo.objs {
		uo.objs[i] = ModelObject{}
	}
	uo.objs = uo.objs[:0]
	for k := range uo.uniq {
		delete(uo.uniq, k)
	}
}
