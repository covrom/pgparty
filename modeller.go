package pgparty

import "reflect"

type StructField struct {
	Name string
	Type      reflect.Type      // field type
	Tag       reflect.StructTag // field tag string
}

type Modeller interface {
	NumField() int
	Field(i int) StructField
}
