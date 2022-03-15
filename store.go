package pgparty

import (
	"reflect"
)

type (
	sqlPattern = string
)

type Store struct {
	modelDescriptions map[reflect.Type]*ModelDesc
	queryReplacers    map[sqlPattern]map[string]ReplaceEntry
}

func (s *Store) Init() {
	s.modelDescriptions = make(map[reflect.Type]*ModelDesc)
	s.queryReplacers = make(map[sqlPattern]map[string]ReplaceEntry)
}

func (s Store) ModelDescriptions() map[reflect.Type]*ModelDesc {
	return s.modelDescriptions
}

func (s Store) QueryReplacers() map[sqlPattern]map[string]ReplaceEntry {
	return s.queryReplacers
}

func (s Store) GetModelDescription(model Storable) (*ModelDesc, bool) {
	ret, ok := s.modelDescriptions[reflect.Indirect(reflect.ValueOf(model)).Type()]
	return ret, ok
}

// Получение описания модели из его reflect.Type
func (s Store) GetModelDescriptionByType(typ reflect.Type) (*ModelDesc, bool) {
	ret, ok := s.modelDescriptions[typ]
	return ret, ok
}

func (s *Store) Close() {
	// log.Fatal("(s *Store) Close() unimplemented")
}
