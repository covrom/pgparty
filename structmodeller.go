package pgparty

import (
	"fmt"
	"reflect"

	"github.com/jmoiron/sqlx"
)

type StructModel[T any] struct {
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

func (s StructModel[T]) TypeName() TypeName {
	return TypeName(s.ReflectType().Name())
}

func (s StructModel[T]) DatabaseName() string {
	type dnamer interface {
		DatabaseName() string
	}
	if dn, ok := any(s.M).(dnamer); ok {
		return dn.DatabaseName()
	}
	return sqlx.NameMapper(s.ReflectType().Name())
}

func (s StructModel[T]) ViewQuery() string {
	_, _, viewQuery := viewAttrs(s.M)
	return viewQuery
}

func (s StructModel[T]) MaterializedView() bool {
	_, isMaterialized, _ := viewAttrs(s.M)
	return isMaterialized
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

		frv := rv.FieldByName(structField.Name)

		if structField.Anonymous {
			if m, ok := frv.Interface().(Modeller); ok {
				mfds := m.Fields()
				*columns = append(*columns, mfds...)
			} else {
				structFields(frv, columns)
			}
			continue
		}

		if v, ok := frv.Interface().(FieldDescriber); ok {
			*columns = append(*columns, *v.FD())
		} else if column := NewFDByStructField(structField); column != nil {
			*columns = append(*columns, *column)
		}
	}
}
