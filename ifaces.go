package pgparty

import "reflect"

// Storable is an interface that the model structure must implement
type Storable interface {
	DatabaseName() string
}

// Viewable is an interface that the view-model structure must implement
type Viewable interface {
	Storable
	ViewQuery() string
}

// MaterializedViewable is an interface that the materialized view-model structure must implement
type MaterializedViewable interface {
	Viewable
	MaterializedView() // not called, define with empty body
}

// Modeller defines model with fields.
// You can combine it with Viewable and MaterializedViewable interfaces.
type Modeller interface {
	ReflectType() reflect.Type
	TypeName() string
	DatabaseName() string
	Fields() []FieldDescription
	// optional Viewable
	// optional MaterializedViewable
}
