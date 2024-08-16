package pgparty

import (
	"context"
	"fmt"
	"sync"
)

type TypeName string

type mdMap struct {
	sync.RWMutex
	m map[TypeName]*ModelDesc
}

var mdRepo = mdMap{
	m: make(map[TypeName]*ModelDesc),
}

type MD[T Modeller] struct {
	Val T
}

func (m MD[T]) MD() (*ModelDesc, error) {
	modelType := m.Val.TypeName()
	mdRepo.RLock()
	if ret, ok := mdRepo.m[modelType]; ok {
		mdRepo.RUnlock()
		return ret, nil
	}
	mdRepo.RUnlock()

	mdRepo.Lock()
	defer mdRepo.Unlock()

	if ret, ok := mdRepo.m[modelType]; ok {
		return ret, nil
	}

	modelDescription, err := NewModelDescription(m.Val)
	if err != nil {
		return nil, err
	}

	mdRepo.m[modelType] = modelDescription
	return modelDescription, nil
}

type ModelDescriber interface {
	MD() (*ModelDesc, error)
}

func Register[T ModelDescriber](sh Shard, m T) error {
	md, err := m.MD()
	if err != nil {
		return fmt.Errorf("init ModelDesc failed: %w", err)
	}

	sh.Store.modelDescriptions[md.TypeName()] = md

	mdrepls, rpls, err := md.ReplaceEntries(sh.Store.Schema())
	if err != nil {
		return err
	}
	for _, mdrepl := range mdrepls {
		sh.Store.queryReplacers[mdrepl] = rpls
	}

	return nil
}

type ModelDesc struct {
	m         Modeller
	typeName  TypeName
	storeName string

	idField        *FieldDescription
	createdAtField *FieldDescription
	updatedAtField *FieldDescription
	deletedAtField *FieldDescription

	columns           []FieldDescription
	columnPtrs        []*FieldDescription
	allFDs            map[*FieldDescription]struct{}
	columnByName      map[string]*FieldDescription // by database name
	columnByFieldName map[string]*FieldDescription // by struct field name
	columnByJsonName  map[string]*FieldDescription // by json name

	viewQuery      string
	isView         bool
	isMaterialized bool
}

func (md ModelDesc) Modeller() Modeller {
	return md.m
}

func (md ModelDesc) DatabaseName() string {
	return md.storeName
}

func (md ModelDesc) TypeName() TypeName {
	return md.typeName
}

func (md *ModelDesc) IdField() *FieldDescription        { return md.idField }
func (md *ModelDesc) CreatedAtField() *FieldDescription { return md.createdAtField }
func (md *ModelDesc) UpdatedAtField() *FieldDescription { return md.updatedAtField }
func (md *ModelDesc) DeletedAtField() *FieldDescription { return md.deletedAtField }

func (md *ModelDesc) ColumnPtrsCount() int              { return len(md.columnPtrs) }
func (md *ModelDesc) ColumnPtr(i int) *FieldDescription { return md.columnPtrs[i] }
func (md *ModelDesc) WalkColumnPtrs(f func(i int, v *FieldDescription) error) error {
	for fdi := 0; fdi < md.ColumnPtrsCount(); fdi++ {
		fd := md.ColumnPtr(fdi)
		if err := f(fdi, fd); err != nil {
			return err
		}
	}
	return nil
}

func (md ModelDesc) IsView() bool {
	return md.isView
}

func (md ModelDesc) IsMaterialized() bool {
	return md.isMaterialized
}

func (md ModelDesc) ViewQuery(ctx context.Context, sr *PgStore) (string, error) {
	return sr.PrepareQuery(ctx, md.viewQuery)
}

func viewAttrs(m any) (isView, isMaterialized bool, viewQuery string) {
	var v Viewable
	var vm MaterializedViewable

	v, isView = m.(Viewable)
	vm, isMaterialized = m.(MaterializedViewable)

	if isView {
		viewQuery = v.ViewQuery()
		isView = viewQuery != ""
	}

	if isMaterialized {
		isMaterialized = vm.MaterializedView()
	}

	isMaterialized = isMaterialized && isView

	return
}

// GetColumnByFieldName - get fd by struct field name
func (md ModelDesc) ColumnByFieldName(fieldName string) (*FieldDescription, error) {
	field, ok := md.columnByFieldName[fieldName]
	if !ok {
		return nil, fmt.Errorf("ColumnByFieldName no such field: %s.%s", md.TypeName(), fieldName)
	}
	return field, nil
}

// GetColumnsByFieldNames - get fd's by struct field name
func (md ModelDesc) ColumnsByFieldNames(fieldNames ...string) (res []*FieldDescription) {
	for _, fieldName := range fieldNames {
		field, ok := md.columnByFieldName[fieldName]
		if !ok {
			panic(fmt.Sprintf("ColumnsByFieldNames no such field: %s.%s", md.TypeName(), fieldName))
		}
		res = append(res, field)
	}
	return
}

func (md ModelDesc) ColumnByJsonName(jsonName string) (*FieldDescription, error) {
	field, ok := md.columnByJsonName[jsonName]
	if !ok {
		return nil, fmt.Errorf("ColumnByJsonName no such field: %s.%s", md.TypeName(), jsonName)
	}
	return field, nil
}

func (md ModelDesc) ColumnByDatabaseName(storeName string) (*FieldDescription, error) {
	field, ok := md.columnByName[storeName]
	if !ok {
		return nil, fmt.Errorf("ColumnByStoreName no such field: %s.%s", md.TypeName(), storeName)
	}
	return field, nil
}

func (md *ModelDesc) Init(m Modeller) error {
	md.typeName = m.TypeName()
	md.storeName = m.DatabaseName()

	columns := m.Fields()
	columnByName := make(map[string]*FieldDescription)
	columnByJsonName := make(map[string]*FieldDescription)
	columnByFieldName := make(map[string]*FieldDescription)

	md.isView, md.isMaterialized, md.viewQuery = viewAttrs(m)

	// fill shortcuts
	for i := range columns {
		column := &columns[i]
		if _, ok := columnByFieldName[column.FieldName]; ok {
			return fmt.Errorf("column name not uniq: '%s'", column.FieldName)
		}
		columnByName[column.DatabaseName] = column
		columnByFieldName[column.FieldName] = column
		if jsonName := column.JsonName; len(jsonName) > 0 {
			columnByJsonName[jsonName] = column
		} else {
			columnByJsonName[column.FieldName] = column
		}
	}

	md.columnPtrs = make([]*FieldDescription, len(columns))
	md.allFDs = make(map[*FieldDescription]struct{}, len(columns))
	// should not be in the previous loop, because there should be no changes if an error is returned above
	for i := range columns {
		column := &columns[i]
		column.Idx = i
		md.columnPtrs[i] = column
		md.allFDs[column] = struct{}{}

		switch {
		case column.IsID:
			md.idField = column
		case column.IsCreatedAt:
			md.createdAtField = column
		case column.IsUpdatedAt:
			md.updatedAtField = column
		case column.IsDeletedAt:
			md.deletedAtField = column
		}
	}

	md.columns = columns
	md.columnByFieldName = columnByFieldName
	md.columnByJsonName = columnByJsonName
	md.columnByName = columnByName

	return nil
}

func NewStructModelDescription[T any](m T) (*ModelDesc, error) {
	modelDescription := ModelDesc{}
	md := StructModel[T]{M: m}
	if err := modelDescription.Init(md); err != nil {
		return nil, fmt.Errorf("init ModelDesc failed: %s", err)
	}

	return &modelDescription, nil
}

// m can be Storable if struct or Modeller if custom type
func NewModelDescription[T Modeller](m T) (*ModelDesc, error) {
	modelDescription := ModelDesc{}

	if err := modelDescription.Init(m); err != nil {
		return nil, fmt.Errorf("init ModelDesc failed: %s", err)
	}

	return &modelDescription, nil
}

type FieldDescriber interface {
	FD() *FieldDescription
}
