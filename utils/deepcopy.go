package utils

import (
	"fmt"
	"reflect"
)

func DeepCopyReflect(dst, src reflect.Value) {
	switch src.Kind() {
	case reflect.Interface:
		value := src.Elem()
		if !value.IsValid() {
			return
		}
		newValue := reflect.New(value.Type()).Elem()
		DeepCopyReflect(newValue, value)
		dst.Set(newValue)
	case reflect.Ptr:
		value := src.Elem()
		if !value.IsValid() {
			return
		}
		dst.Set(reflect.New(value.Type()))
		DeepCopyReflect(dst.Elem(), value)
	case reflect.Map:
		dst.Set(reflect.MakeMap(src.Type()))
		keys := src.MapKeys()
		for _, key := range keys {
			value := src.MapIndex(key)
			newValue := reflect.New(value.Type()).Elem()
			DeepCopyReflect(newValue, value)
			dst.SetMapIndex(key, newValue)
		}
	case reflect.Slice:
		dst.Set(reflect.MakeSlice(src.Type(), src.Len(), src.Cap()))
		for i := 0; i < src.Len(); i++ {
			DeepCopyReflect(dst.Index(i), src.Index(i))
		}
	case reflect.Struct:
		typeSrc := src.Type()
		for i := 0; i < src.NumField(); i++ {
			value := src.Field(i)
			tag := typeSrc.Field(i).Tag
			if value.CanSet() && tag.Get("deepcopy") != "-" {
				DeepCopyReflect(dst.Field(i), value)
			}
		}
	default:
		dst.Set(src)
	}
}

func DeepCopy(dst, src interface{}) error {
	typeDst := reflect.TypeOf(dst)
	typeSrc := reflect.TypeOf(src)
	if typeDst != typeSrc {
		return fmt.Errorf("DeepCopy: %s != %s", typeDst, typeSrc)
	}
	if typeSrc.Kind() != reflect.Ptr {
		return fmt.Errorf("DeepCopy: pass arguments by address")
	}

	valueDst := reflect.ValueOf(dst).Elem()
	valueSrc := reflect.ValueOf(src).Elem()
	if !valueDst.IsValid() || !valueSrc.IsValid() {
		return fmt.Errorf("DeepCopy: invalid arguments")
	}

	DeepCopyReflect(valueDst, valueSrc)
	return nil
}

func DeepClone(v interface{}) interface{} {
	dst := reflect.New(reflect.TypeOf(v)).Elem()
	DeepCopyReflect(dst, reflect.ValueOf(v))
	return dst.Interface()
}
