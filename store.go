package pgparty

import (
	"reflect"
)

type (
	sqlPattern = string
)

type Store struct {
	modelDescriptions map[TypeName]*ModelDesc
	queryReplacers    map[sqlPattern]map[string]ReplaceEntry
}

func (s *Store) Init() {
	s.modelDescriptions = make(map[TypeName]*ModelDesc)
	s.queryReplacers = make(map[sqlPattern]map[string]ReplaceEntry)
}

func (s Store) ModelDescriptions() map[TypeName]*ModelDesc {
	return s.modelDescriptions
}

func (s Store) QueryReplacers() map[sqlPattern]map[string]ReplaceEntry {
	return s.queryReplacers
}

func (s Store) GetModelDescription(model Modeller) (*ModelDesc, bool) {
	ret, ok := s.modelDescriptions[model.TypeName()]
	return ret, ok
}

// Получение описания модели из его reflect.Type
func (s Store) GetModelDescriptionByType(typ reflect.Type) (*ModelDesc, bool) {
	v := reflect.New(typ).Elem().Interface()
	vv, ok := v.(Modeller)
	if !ok {
		return nil, false
	}
	ret, ok := s.modelDescriptions[vv.TypeName()]
	return ret, ok
}

func (s *Store) Close() {
	// log.Fatal("(s *Store) Close() unimplemented")
}
