package pgparty

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"log"
	"reflect"
)

var ErrStoreAlreadyExists = errors.New("store already exists in core")

// Store - хранилище типа sql, ElasticSearch и тд. Базовая реализация интерфейса Storer
type Store struct {
	name              string                      // имя хранилища
	modelDescriptions map[reflect.Type]*ModelDesc // характеристики хранилища
	queryReplacers    map[string]map[string]ReplaceEntry

	listeners map[reflect.Type][]interface{}
}

// Init инициализирует хранилище, присваивая ему имя, переданное в качестве аргумента этому методу.
func (s *Store) Init(name string) {
	s.name = name
	s.modelDescriptions = make(map[reflect.Type]*ModelDesc)
	s.queryReplacers = make(map[string]map[string]ReplaceEntry)
	s.listeners = make(map[reflect.Type][]interface{})
}

// Name возвращает имя хранилища.
func (s Store) Name() string {
	return s.name
}

// ModelDescriptions возвращает map характеристик хранилища типа ModelDescription.
func (s Store) ModelDescriptions() map[reflect.Type]*ModelDesc {
	return s.modelDescriptions
}

// QueryReplacers возвращает map["&Model"]map[Old]New - замены текста в запросе для всех моделей
func (s Store) QueryReplacers() map[string]map[string]ReplaceEntry {
	return s.queryReplacers
}

func (s Store) InitQueryReplacers() {
	for _, md := range s.modelDescriptions {
		mdrepls, rpls, err := s.ReplaceEntries(md)
		if err != nil {
			log.Fatal("InitQueryReplacers: ", err)
		}
		for _, mdrepl := range mdrepls {
			s.queryReplacers[mdrepl] = rpls
		}
	}
}

// GetModelDescription ищет объект типа ModelDescription по значению (по структуре), который передает в функцию.
// Метод возвращает объект типа ModelDescription и объект типа bool, если bool равняется true - нет ошибки, если false - есть ошибка.
func (s Store) GetModelDescription(model interface{}) (*ModelDesc, bool) {
	ret, ok := s.modelDescriptions[reflect.Indirect(reflect.ValueOf(model)).Type()]
	return ret, ok
}

// Получение описания модели из его reflect.Type
func (s Store) GetModelDescriptionByType(typ reflect.Type) (*ModelDesc, bool) {
	ret, ok := s.modelDescriptions[typ]
	return ret, ok
}

// Добавление описания модели в хранилище. Хранилище может дальше работать с типами моделей, которые были зарегистрированы.
func (s *Store) RegisterModels(sts ...Storable) error {
	for _, st := range sts {
		value := reflect.Indirect(reflect.ValueOf(st))
		if value.Kind() != reflect.Struct {
			return fmt.Errorf("only structs are supported: %s is not a struct", value.Type())
		}

		shn := ""
		if v, ok := st.(Schemable); ok {
			shn = v.SchemaName()
		}
		md, err := NewModelDescription(value.Type(), st.StoreName(), shn)
		if err != nil {
			return err
		}

		// if _, ok = s.modelDescriptions[md.ModelType()]; ok {
		// 	return ErrStoreAlreadyExists //fmt.Errorf("store with name '%s' already exists in core", name)
		// }

		s.modelDescriptions[md.ModelType()] = md
	}

	s.InitQueryReplacers()

	return nil
}

func (s *Store) RegisterListener(listener interface{}, models ...interface{}) error {
	var types []reflect.Type
	for _, model := range models {
		types = append(types, reflect.Indirect(reflect.ValueOf(model)).Type())
	}

	return s.RegisterListenerForType(listener, types...)
}

func (s *Store) RegisterListenerForType(listener interface{}, types ...reflect.Type) error {
	add := func(listeners []interface{}) []interface{} {
		for _, l := range listeners {
			if l == listener {
				return listeners
			}
		}
		return append(listeners, listener)
	}

	for _, typ := range types {
		md, ok := s.GetModelDescriptionByType(typ)
		if !ok {
			return fmt.Errorf("unknown model type %s", typ)
		}
		s.listeners[md.ModelType()] = add(s.listeners[md.ModelType()])
	}

	return nil
}

func ValidateValuer(field reflect.Value) interface{} {
	switch v := field.Interface().(type) {
	case UUIDv4:
		if !v.IsZero() {
			val, err := v.Value()
			if err == nil {
				return val
			}
		}
	case driver.Valuer:
		val, err := v.Value()
		if err == nil {
			return val
		}
	}
	return nil
}

func (s *Store) Close() {
	// log.Fatal("(s *Store) Close() unimplemented")
}
