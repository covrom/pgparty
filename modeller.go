package pgparty

type Modeller interface {
	NumField() int
	Field(i int) *FieldDescription
}
