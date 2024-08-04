package pgparty

import "reflect"

type Modeller interface {
	ReflectType() reflect.Type
	TypeName() string
	DatabaseName() string
	NumField() int
	Field(i int) *FieldDescription
	WalkFields(f func(i int, fd *FieldDescription) error) error
}
