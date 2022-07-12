package pgparty

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/covrom/pgparty/utils"
)

type mdMap struct {
	sync.RWMutex
	m map[reflect.Type]*ModelDesc
}

var mdRepo = mdMap{
	m: make(map[reflect.Type]*ModelDesc),
}

type MD[T Storable] struct {
	Val T
}

func (m MD[T]) MD() (*ModelDesc, error) {
	value := reflect.Indirect(reflect.ValueOf(m.Val))
	if value.Kind() != reflect.Struct {
		return nil, fmt.Errorf("only structs are supported: %s is not a struct", value.Type())
	}
	modelType := value.Type()
	mdRepo.RLock()
	if ret, ok := mdRepo.m[modelType]; ok {
		mdRepo.RUnlock()
		return ret, nil
	}
	mdRepo.RUnlock()

	mdRepo.Lock()
	defer mdRepo.Unlock()

	if ret, ok := mdRepo.m[modelType]; ok {
		return ret, nil
	}

	storeName := m.Val.StoreName()

	md := &ModelDesc{
		modelType: modelType,
		storeName: storeName,
	}

	if err := md.init(); err != nil {
		return nil, fmt.Errorf("init ModelDesc failed: %s", err)
	}

	mdRepo.m[modelType] = md
	return md, nil
}

type ModelDescriber interface {
	MD() (*ModelDesc, error)
}

func Register[T ModelDescriber](sh Shard, m T) error {
	md, err := m.MD()
	if err != nil {
		return fmt.Errorf("init ModelDesc failed: %w", err)
	}

	sh.Store.modelDescriptions[md.ModelType()] = md

	mdrepls, rpls, err := md.ReplaceEntries(sh.Store.Schema())
	if err != nil {
		return err
	}
	for _, mdrepl := range mdrepls {
		sh.Store.queryReplacers[mdrepl] = rpls
	}

	return nil
}

type ModelDesc struct {
	modelType reflect.Type
	storeName string

	idField        *FieldDescription
	createdAtField *FieldDescription
	updatedAtField *FieldDescription
	deletedAtField *FieldDescription

	columns           []FieldDescription
	columnPtrs        []*FieldDescription
	columnByFieldName map[string]*FieldDescription
	columnByJsonName  map[string]*FieldDescription
}

// Получение типа модели
func (md ModelDesc) ModelType() reflect.Type {
	return md.modelType
}

// Получение название типа модели
func (md ModelDesc) StoreName() string {
	return md.storeName
}

// Уникальные длинное имя
func (md ModelDesc) UniqName() string {
	return utils.GetUniqTypeName(md.modelType)
}

func (md *ModelDesc) IdField() *FieldDescription        { return md.idField }
func (md *ModelDesc) CreatedAtField() *FieldDescription { return md.createdAtField }
func (md *ModelDesc) UpdatedAtField() *FieldDescription { return md.updatedAtField }
func (md *ModelDesc) DeletedAtField() *FieldDescription { return md.deletedAtField }

func (md *ModelDesc) ColumnPtrsCount() int              { return len(md.columnPtrs) }
func (md *ModelDesc) ColumnPtr(i int) *FieldDescription { return md.columnPtrs[i] }
func (md *ModelDesc) WalkColumnPtrs(f func(i int, v *FieldDescription) error) error {
	for fdi := 0; fdi < md.ColumnPtrsCount(); fdi++ {
		fd := md.ColumnPtr(fdi)
		if err := f(fdi, fd); err != nil {
			return err
		}
	}
	return nil
}

// GetColumnByFieldName - get fd by struct field name
func (md ModelDesc) ColumnByFieldName(fieldName string) (*FieldDescription, error) {
	field, ok := md.columnByFieldName[fieldName]
	if !ok {
		return nil, fmt.Errorf("ColumnByFieldName no such field: %s.%s", md.modelType.Name(), fieldName)
	}
	return field, nil
}

// GetColumnsByFieldNames - get fd's by struct field name
func (md ModelDesc) ColumnsByFieldNames(fieldNames ...string) (res []*FieldDescription) {
	for _, fieldName := range fieldNames {
		field, ok := md.columnByFieldName[fieldName]
		if !ok {
			panic(fmt.Sprintf("ColumnsByFieldNames no such field: %s.%s", md.modelType.Name(), fieldName))
		}
		res = append(res, field)
	}
	return
}

func (md ModelDesc) ColumnByJsonName(jsonName string) (*FieldDescription, error) {
	field, ok := md.columnByJsonName[jsonName]
	if !ok {
		return nil, fmt.Errorf("ColumnByJsonName no such field: %s.%s", md.modelType.Name(), jsonName)
	}
	return field, nil
}

func (md *ModelDesc) init() error {
	columns := make([]FieldDescription, 0, md.modelType.NumField())
	columnByName := make(map[string]*FieldDescription)
	columnByJsonName := make(map[string]*FieldDescription)
	columnByFieldName := make(map[string]*FieldDescription)

	if err := fillColumns(md.modelType, &columns); err != nil {
		return err
	}

	// fill shortcuts
	for i := range columns {
		column := &columns[i]
		if _, ok := columnByName[column.Name]; ok {
			return fmt.Errorf("column name not uniq: '%s'", column.Name)
		}
		columnByName[column.Name] = column
		columnByFieldName[column.StructField.Name] = column
		if jsonName := utils.JsonFieldName(column.StructField); len(jsonName) > 0 {
			columnByJsonName[jsonName] = column
		} else {
			columnByJsonName[column.StructField.Name] = column
		}
	}

	md.columnPtrs = make([]*FieldDescription, len(columns))
	// should not be in previous loop as it can return
	for i := range columns {
		column := &columns[i]
		column.Idx = i
		md.columnPtrs[i] = column

		switch column.StructField.Name {
		case IDField:
			md.idField = column
		case CreatedAtField:
			md.createdAtField = column
		case UpdatedAtField:
			md.updatedAtField = column
		case DeletedAtField:
			md.deletedAtField = column
		}
	}

	md.columns = columns
	md.columnByFieldName = columnByFieldName
	md.columnByJsonName = columnByJsonName

	return nil
}

func NewModelDescription[T Storable](m T) (*ModelDesc, error) {
	value := reflect.Indirect(reflect.ValueOf(m))
	if value.Kind() != reflect.Struct {
		return nil, fmt.Errorf("only structs are supported: %s is not a struct", value.Type())
	}
	modelType := value.Type()
	storeName := m.StoreName()
	modelDescription := ModelDesc{
		modelType: modelType,
		storeName: storeName,
	}

	if err := modelDescription.init(); err != nil {
		return nil, fmt.Errorf("init ModelDesc failed: %s", err)
	}

	return &modelDescription, nil
}

type FieldDescriber interface {
	FD() *FieldDescription
}

func fillColumns(typ reflect.Type, columns *[]FieldDescription) error {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return fmt.Errorf("%s not a struct or a pointer to struct", typ.String())
	}

	for i := 0; i < typ.NumField(); i++ {
		structField := typ.Field(i)
		if len(structField.PkgPath) > 0 {
			continue
		}

		if structField.Anonymous {
			if err := fillColumns(structField.Type, columns); err != nil {
				return err
			}
			continue
		}

		ft := structField.Type
		if ft.Implements(reflect.TypeOf((*FieldDescriber)(nil)).Elem()) {
			v := reflect.New(ft).Elem().Interface().(FieldDescriber)
			*columns = append(*columns, *v.FD())
		} else if column := NewFieldDescription(structField); column != nil {
			*columns = append(*columns, *column)
		}
	}

	return nil
}

func (md ModelDesc) CreateSlicePtr() interface{} {
	slt := reflect.SliceOf(md.modelType)
	return reflect.New(slt).Interface()
}

func (md ModelDesc) CreateElemPtr() interface{} {
	return reflect.New(md.modelType).Interface()
}
