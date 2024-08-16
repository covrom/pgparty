package pgparty

// Modeller defines model with fields.
// You can combine it with Viewable and MaterializedViewable interfaces.
type Modeller interface {
	TypeName() TypeName
	DatabaseName() string
	Fields() []FieldDescription
	// optional Viewable
	// optional MaterializedViewable
}

// Viewable is an interface that the view-model structure must implement
type Viewable interface {
	Modeller
	ViewQuery() string
}

// MaterializedViewable is an interface that the materialized view-model structure must implement
type MaterializedViewable interface {
	Viewable
	MaterializedView() bool
}
