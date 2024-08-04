package pgparty

import (
	"fmt"
	"reflect"

	"github.com/covrom/pgparty/utils"
)

type Modeller interface {
	ReflectType() reflect.Type
	TypeName() string
	DatabaseName() string
	Fields() []FieldDescription
}

type StructModel[T Storable] struct {
	M T
}

func (s StructModel[T]) ReflectType() reflect.Type {
	_, typ := reflStructType(s.M)
	return typ
}

func reflStructType(m any) (reflect.Value, reflect.Type) {
	value := reflect.Indirect(reflect.ValueOf(m))
	if value.Kind() != reflect.Struct {
		panic(fmt.Sprintf("only structs are supported: %s is not a struct", value.Type()))
	}
	return value, value.Type()
}

func (s StructModel[T]) TypeName() string {
	return utils.GetUniqTypeName(s.ReflectType())
}

func (s StructModel[T]) DatabaseName() string {
	return s.M.StoreName()
}

func (s StructModel[T]) Fields() []FieldDescription {
	rv, typ := reflStructType(s.M)
	columns := make([]FieldDescription, 0, typ.NumField())
	structFields(rv, &columns)
	return columns
}

func structFields(rv reflect.Value, columns *[]FieldDescription) {
	typ := rv.Type()

	for i := 0; i < typ.NumField(); i++ {
		structField := typ.Field(i)
		if len(structField.PkgPath) > 0 {
			continue
		}

		ft := structField.Type
		frv := rv.FieldByName(structField.Name)

		if structField.Anonymous {
			if ft.Implements(reflect.TypeFor[Modeller]()) {
				m := frv.Interface().(Modeller) // reflect.New(ft).Elem().Interface().(Modeller)
				mfds := m.Fields()
				*columns = append(*columns, mfds...)
			} else {
				structFields(frv, columns)
			}
			continue
		}

		if ft.Implements(reflect.TypeFor[FieldDescriber]()) {
			v := frv.Interface().(FieldDescriber)
			*columns = append(*columns, *v.FD())
		} else if column := NewFieldDescription(structField); column != nil {
			*columns = append(*columns, *column)
		}
	}
}
