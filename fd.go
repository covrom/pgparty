package pgparty

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/covrom/pgparty/utils"
)

// FieldDescription - универсальная структура, используемая во многих моделях и структурах.
type FieldDescription struct {
	Idx             int                 // индекс в слайсе ModelDescription.Columns
	StructField     reflect.StructField // поле с базовыми характеристиками поля структуры
	ElemType        reflect.Type        // тип элемента, который характеризует структура
	Name            string              // store name (имя в хранилище)
	JsonName        string
	Nullable        bool
	Skip            bool
	SkipReplace     bool // игнорится только при реплейсе
	FullTextEnabled bool
}

func NewFieldDescription(structField reflect.StructField) *FieldDescription {
	fieldName := structField.Name

	name := ToSnakeCase2(fieldName)
	dbtag := structField.Tag.Get(TagDBName)
	if len(dbtag) > 0 {
		if dbtag == "-" {
			return nil
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
		switch strings.ToLower(nlbl) {
		case "true", "enabled", "yes", "on", "1":
			column.Nullable = true
		case "false", "disabled", "no", "off", "0":
			column.Nullable = false
		}
	}

	column.Skip = structField.Tag.Get(TagStore) == "-"
	return &column
}

// Вывод FieldDescription в виде сткроки
func (fd FieldDescription) String() string {
	ret := fd.StructField.Name

	if fd.Skip {
		return "- " + ret + " [skip]"
	}

	nullable := ""
	if fd.Nullable {
		nullable = "*"
	}

	ret = fmt.Sprintf("%s%s (store: %s)", nullable, ret, fd.Name)

	return ret
}

// Говорит о том, сохраняется ли текущее поле в хранилище или нет
func (fd *FieldDescription) IsStored() bool {
	return !fd.Skip
}

func (fd *FieldDescription) MarshalJSON() ([]byte, error) {
	if fd == nil {
		return []byte("null"), nil
	}
	return []byte(fd.JsonName), nil
}

func (fd *FieldDescription) MarshalText() (text []byte, err error) {
	if fd == nil {
		return []byte("null"), nil
	}
	return []byte(fd.JsonName), nil
}
