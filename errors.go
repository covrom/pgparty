package pgparty

import "reflect"

// Ошибка транзакции
type ErrorNoTransaction struct{}

func (e ErrorNoTransaction) Error() string {
	return "transaction is not started"
}

// Ошибка "не найден"
type ErrorNotFound struct {
	ID      interface{}
	Type    reflect.Type
	Message string
}
