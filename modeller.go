package pgparty

import (
	"fmt"
	"reflect"

	"github.com/covrom/pgparty/utils"
)

type StructModel struct {
	M Storable
}

func (s StructModel) ReflectType() reflect.Type {
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

func (s StructModel) TypeName() string {
	return utils.GetUniqTypeName(s.ReflectType())
}

func (s StructModel) DatabaseName() string {
	return s.M.DatabaseName()
}

func (s StructModel) Fields() []FieldDescription {
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
