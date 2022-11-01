package pgparty

// Storable is an interface that the model structure must implement
type Storable interface {
	StoreName() string
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
