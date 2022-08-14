package pgparty

import (
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type JsonErr struct {
	Err error `json:"_err"`
}

type JsonViewer[T Storable] interface {
	JsonView() JsonViewErr[T]
}

type SQLViewer[T Storable] interface {
	SQLView() SQLViewErr[T]
}

type JsonViewErr[T Storable] struct {
	Value *JsonView[T]
	Err   error `json:"_err,omitempty"`
}

func (v JsonViewErr[T]) MarshalJSON() ([]byte, error) {
	if v.Err != nil {
		return json.Marshal(JsonErr{Err: v.Err})
	}
	return json.Marshal(v.Value)
}

func (v *JsonViewErr[T]) UnmarshalJSON(b []byte) error {
	v.Value = &JsonView[T]{}
	v.Err = json.Unmarshal(b, v.Value)
	if v.Err != nil {
		v.Value = nil
		return v.Err
	}
	return nil
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

// prefix is table alias with point at end, or empty string
func (v *SQLViewErr[T]) Scan(rows sqlx.ColScanner, prefix string) error {
	if v == nil {
		return fmt.Errorf("must not nil")
	}
	v.Value = &SQLView[T]{}
	v.Err = v.Value.Scan(rows, prefix)
	return v.Err
}
