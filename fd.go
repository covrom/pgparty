package pgparty

import (
	"fmt"
	"reflect"
	"strconv"
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
	Ln              int
	Prec            int
	SQLTypeDef      string
	DefVal          string
	Indexes         []string
	GinIndexes      []string
	UniqIndexes     []string
	Nullable        bool
	Skip            bool
	SkipReplace     bool // игнорится только при реплейсе
	FullTextEnabled bool
	PK              bool
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
		PK:              structField.Name == IDField,
	}

	switch strings.ToLower(structField.Tag.Get("nullable")) {
	case "true", "enabled", "yes", "on", "1":
		column.Nullable = true
	case "false", "disabled", "no", "off", "0":
		column.Nullable = false
	}

	column.Skip = structField.Tag.Get(TagStore) == "-"

	if v, ok := structField.Tag.Lookup(TagPK); ok {
		column.PK = v != "-"
	}

	// тэг len для строк и decimal
	column.Ln = 150
	if lns, ok := structField.Tag.Lookup(TagLen); ok && len(lns) > 0 {
		if ln, err := strconv.Atoi(lns); err == nil {
			column.Ln = ln
		}
	}

	// тэг prec для decimal
	column.Prec = 2
	if precs, ok := structField.Tag.Lookup(TagPrec); ok && len(precs) > 0 {
		if prec, err := strconv.Atoi(precs); err == nil {
			column.Prec = prec
		}
	}

	if tname, ok := structField.Tag.Lookup(TagSql); ok && len(tname) > 0 {
		column.SQLTypeDef = tname
	}

	if dv, ok := structField.Tag.Lookup(TagDefVal); ok && len(dv) > 0 {
		column.DefVal = dv
	}

	if indexes, ok := structField.Tag.Lookup(TagKey); ok && len(indexes) > 0 {
		column.Indexes = strings.Split(indexes, ",")
	}

	if ginIndex, ok := structField.Tag.Lookup(TagGinKey); ok && len(ginIndex) > 0 {
		column.GinIndexes = strings.Split(ginIndex, ",")
	}

	if uniIndex, ok := structField.Tag.Lookup(TagUniqueKey); ok && len(uniIndex) > 0 {
		column.UniqIndexes = strings.Split(uniIndex, ",")
	}

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
