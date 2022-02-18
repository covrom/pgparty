package utils

import (
	"fmt"
	"reflect"
	"strings"
)

func GetFieldByName(typ reflect.Type, name string) (reflect.StructField, error) {
	for i := 0; i < typ.NumField(); i++ {
		structField := typ.Field(i)
		if !structField.Anonymous {
			continue
		}

		var subTyp reflect.Type

		switch structField.Type.Kind() {
		case reflect.Struct:
			subTyp = structField.Type
		case reflect.Ptr:
			elem := structField.Type.Elem()
			if elem.Kind() == reflect.Struct {
				subTyp = elem
			}
		}

		if subTyp != nil {
			if field, err := GetFieldByName(subTyp, name); err == nil {
				return field, nil
			}
		}
	}

	if field, ok := typ.FieldByName(name); ok {
		return field, nil
	}

	return reflect.StructField{}, fmt.Errorf("no '%s' field", name)
}

func GetFieldValueByName(val reflect.Value, name string) (reflect.Value, error) {
	val = reflect.Indirect(val)
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		if !typ.Field(i).Anonymous {
			continue
		}

		subVal := reflect.Indirect(val.Field(i))
		if subVal.Kind() != reflect.Struct {
			continue
		}

		if field, err := GetFieldValueByName(subVal, name); err == nil {
			return field, nil
		}
	}

	if _, ok := typ.FieldByName(name); ok {
		return val.FieldByName(name), nil
	}

	return reflect.Value{}, fmt.Errorf("no '%s' field", name)
}

func JsonFieldName(field reflect.StructField) string {
	jsonTag := strings.Split(field.Tag.Get("json"), ",")[0]
	if jsonTag == "-" {
		return ""
	} else if len(jsonTag) == 0 {
		return field.Name
	}
	return jsonTag
}

func JsonTagToFieldName(value reflect.Value) map[string]string {
	return JsonTagToFieldNameByType(reflect.Indirect(value).Type())
}

func JsonTagToFieldNameByType(valueType reflect.Type) map[string]string {
	ret := make(map[string]string)

	for i := 0; i < valueType.NumField(); i++ {
		field := valueType.Field(i)
		if jsonName := JsonFieldName(field); len(jsonName) > 0 {
			ret[jsonName] = field.Name
		}
	}

	return ret
}

func GetUniqTypeName(typ reflect.Type) string {
	return fmt.Sprintf("%s.%s", typ.PkgPath(), typ.Name())
}
