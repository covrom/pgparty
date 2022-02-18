package pgparty

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/covrom/pgparty/utils"
)

// Класс, описывающий модель, которая используется в Store
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

	for i := range columns {
		column := &columns[i]

		// fill shortcuts
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

// Создает новую модель и заполняет поля
func NewModelDescription(modelType reflect.Type, storeName, schema string) (*ModelDesc, error) {
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
		return nil
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

		fieldName := structField.Name

		// только так, т.к. эта же функция будет использоваться для распознавания имен полей БД
		name := ToSnakeCase2(fieldName)
		dbtag := structField.Tag.Get(TagDBName)
		if len(dbtag) > 0 {
			if dbtag == "-" {
				continue
			}
			name = dbtag
		}

		elemType := structField.Type
		switch elemType.Kind() {
		case reflect.Ptr, reflect.Slice:
			elemType = elemType.Elem()
		}

		var fullTextEnabled bool

		switch strings.ToLower(structField.Tag.Get(TagFullText)) {
		case "true", "enabled", "yes", "on", "1":
			fullTextEnabled = true
		}

		column := FieldDescription{
			StructField: structField,
			ElemType:    elemType,
			Name:        name,
			JsonName:    utils.JsonFieldName(structField),
			Skip:        structField.Tag.Get(TagStore) == "-",
			SkipReplace: structField.Type == reflect.TypeOf(BigSerial{}),
			Nullable: structField.Type.Kind() == reflect.Ptr ||
				structField.Type == reflect.TypeOf(NullTime{}) ||
				structField.Type == reflect.TypeOf(NullDecimal{}) ||
				structField.Type == reflect.TypeOf(NullBool{}) ||
				structField.Type == reflect.TypeOf(NullFloat64{}) ||
				structField.Type == reflect.TypeOf(NullInt64{}) ||
				structField.Type == reflect.TypeOf(BigSerial{}) ||
				structField.Type == reflect.TypeOf(NullText{}) ||
				structField.Type == reflect.TypeOf(NullJsonB{}) ||
				structField.Type == reflect.TypeOf(NullString{}),
			FullTextEnabled: fullTextEnabled,
		}

		nlbl := structField.Tag.Get("nullable")
		if nlbl != "" {
			if _, err := fmt.Fscanf(strings.NewReader(nlbl), "%t", &column.Nullable); err != nil {
				return err
			}
		}

		column.Skip = structField.Tag.Get(TagStore) == "-"

		*columns = append(*columns, column)
	}

	return nil
}

func (md ModelDesc) CreateSlicePtr() interface{} {
	slt := reflect.SliceOf(md.modelType)
	return reflect.New(slt).Interface()
}
