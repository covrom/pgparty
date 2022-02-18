package pgparty

// Storable - интерфейс, которая должна реализовывать структура, которая должна храниться в базе.
type Storable interface {
	StoreName() string
}

// Schemable привязывает модель к нужной схеме, в основном используется для схемы store.AdminSchema
type Schemable interface {
	SchemaName() string
}
