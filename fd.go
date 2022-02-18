package pgparty

import (
	"fmt"
	"reflect"
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
