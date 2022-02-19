package pgparty

import (
	"fmt"
	"reflect"

	"github.com/covrom/pgparty/utils"
)

func Register[T Storable](st *Store, schema string, m T) error {
	value := reflect.Indirect(reflect.ValueOf(m))
	if value.Kind() != reflect.Struct {
		return fmt.Errorf("only structs are supported: %s is not a struct", value.Type())
	}
	modelType := value.Type()
	storeName := m.StoreName()

	md := &ModelDesc{
		modelType: modelType,
		storeName: storeName,
		schema:    schema,
	}

	if err := md.init(); err != nil {
		return fmt.Errorf("init ModelDesc failed: %s", err)
	}

	if mds, ok := st.modelDescriptions[md.schema]; ok {
		mds[md.ModelType()] = md
	} else {
		st.modelDescriptions[md.schema] = make(map[reflect.Type]*ModelDesc)
		st.modelDescriptions[md.schema][md.ModelType()] = md
	}

	mdrepls, rpls, err := ReplaceEntries(md)
	if err != nil {
		return err
	}
	for _, mdrepl := range mdrepls {
		st.queryReplacers[mdrepl] = rpls // FIXME: split by schema
	}

	return nil
}

type ModelDesc struct {
	modelType reflect.Type
	storeName string
	schema    string

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

func (md *ModelDesc) Schema() string                    { return md.schema }
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

// GetColumnByFieldName - найти описание поля по его имени
func (md ModelDesc) ColumnByFieldName(fieldName string) (*FieldDescription, error) {
	field, ok := md.columnByFieldName[fieldName]
	if !ok {
		return nil, fmt.Errorf("ColumnByFieldName no such field: %s.%s", md.modelType.Name(), fieldName)
	}
	return field, nil
}

// GetColumnsByFieldNames - найти описание полей по именам.
// Если поле не найдено, вызывается panic
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

func NewModelDescription[T Storable](m T, schema string) (*ModelDesc, error) {
	value := reflect.Indirect(reflect.ValueOf(m))
	if value.Kind() != reflect.Struct {
		return nil, fmt.Errorf("only structs are supported: %s is not a struct", value.Type())
	}
	modelType := value.Type()
	storeName := m.StoreName()
	modelDescription := ModelDesc{
		modelType: modelType,
		storeName: storeName,
		schema:    schema,
	}

	if err := modelDescription.init(); err != nil {
		return nil, fmt.Errorf("init ModelDesc failed: %s", err)
	}

	return &modelDescription, nil
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

		if column := NewFieldDescription(structField); column != nil {
			*columns = append(*columns, *column)
		}
	}

	return nil
}

func (md ModelDesc) CreateSlicePtr() interface{} {
	slt := reflect.SliceOf(md.modelType)
	return reflect.New(slt).Interface()
}
