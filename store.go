package pgparty

import (
	"log"
	"reflect"
)

type Store struct {
	modelDescriptions map[reflect.Type]*ModelDesc // характеристики хранилища
	queryReplacers    map[string]map[string]ReplaceEntry
}

// Init инициализирует хранилище, присваивая ему имя, переданное в качестве аргумента этому методу.
func (s *Store) Init() {
	s.modelDescriptions = make(map[reflect.Type]*ModelDesc)
	s.queryReplacers = make(map[string]map[string]ReplaceEntry)
}

// ModelDescriptions возвращает map характеристик хранилища типа ModelDescription.
func (s Store) ModelDescriptions() map[reflect.Type]*ModelDesc {
	return s.modelDescriptions
}

// QueryReplacers возвращает map["&Model"]map[Old]New - замены текста в запросе для всех моделей
func (s Store) QueryReplacers() map[string]map[string]ReplaceEntry {
	return s.queryReplacers
}

func (s Store) initQueryReplacers() {
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

		shn := ""
		if v, ok := st.(Schemable); ok {
			shn = v.SchemaName()
		}
		md, err := NewModelDescription(st, shn)
		if err != nil {
			return err
		}

		s.modelDescriptions[md.ModelType()] = md
	}

	s.initQueryReplacers()

	return nil
}

func (s *Store) Close() {
	// log.Fatal("(s *Store) Close() unimplemented")
}
