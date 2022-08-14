package pgparty

type JsonViewErr[T Storable] struct {
	Value *JsonView[T]
	Err   error `json:"_err,omitempty"`
}

func (v JsonViewErr[T]) SQLView() SQLViewErr[T] {
	return SQLViewErr[T]{
		Value: v.Value.SQLView(),
		Err:   v.Err,
	}
}

type SQLViewErr[T Storable] struct {
	Value *SQLView[T]
	Err   error `json:"_err,omitempty"`
}

func (v SQLViewErr[T]) JsonView() JsonViewErr[T] {
	return JsonViewErr[T]{
		Value: v.Value.JsonView(),
		Err:   v.Err,
	}
}
