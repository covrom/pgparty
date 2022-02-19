package pgparty

import (
	"reflect"
	"sort"
)

type (
	schemaName = string
	sqlPattern = string
)

type Store struct {
	modelDescriptions map[schemaName]map[reflect.Type]*ModelDesc
	queryReplacers    map[sqlPattern]map[string]ReplaceEntry
}

func (s *Store) Init() {
	s.modelDescriptions = make(map[schemaName]map[reflect.Type]*ModelDesc)
	s.queryReplacers = make(map[sqlPattern]map[string]ReplaceEntry)
}

func (s Store) AllSchemas() []string {
	ret := make([]string, len(s.modelDescriptions))
	for k := range s.modelDescriptions {
		ret = append(ret, k)
	}
	sort.Strings(ret)
	return ret
}

func (s Store) ModelDescriptions(schema string) map[reflect.Type]*ModelDesc {
	return s.modelDescriptions[schema]
}

func (s Store) QueryReplacers() map[string]map[string]ReplaceEntry {
	return s.queryReplacers
}

func (s Store) GetModelDescription(schema string, model Storable) (*ModelDesc, bool) {
	if mds, ok := s.modelDescriptions[schema]; ok {
		ret, ok := mds[reflect.Indirect(reflect.ValueOf(model)).Type()]
		return ret, ok
	}
	return nil, false
}

// Получение описания модели из его reflect.Type
func (s Store) GetModelDescriptionByType(schema string, typ reflect.Type) (*ModelDesc, bool) {
	if mds, ok := s.modelDescriptions[schema]; ok {
		ret, ok := mds[typ]
		return ret, ok
	}
	return nil, false
}

func (s *Store) Close() {
	// log.Fatal("(s *Store) Close() unimplemented")
}
