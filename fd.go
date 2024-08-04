package pgparty

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/covrom/pgparty/utils"
)

// Description of field in struct
type FieldDescription struct {
	Idx             int          // index in ModelDescription.Columns
	FieldName       string       // struct field name
	ElemType        reflect.Type // type
	DatabaseName    string       // database name
	JsonName        string       // json name
	Ln              int          // length for string and numbers
	Prec            int          // precision length for float/decimals
	SQLTypeDef      string       // raw postgres type definition
	DefVal          string       // default postgres value
	Indexes         []string     // btree index names
	GinIndexes      []string     // gin index names
	UniqIndexes     []string     // unique btree index names
	Nullable        bool         // null is allowed
	Skip            bool         // database skip
	SkipReplace     bool         // ignore on upsert
	FullTextEnabled bool         // enable fulltext search
	PK              bool         // is primary key field
	JsonSkip        bool         // skip in json
	JsonOmitEmpty   bool         // omit empty on json marshal
	IsID            bool         // ID field of model
	IsCreatedAt     bool         // CreatedAt field of model
	IsUpdatedAt     bool         // UpdatedAt field of model
	IsDeletedAt     bool         // DeletedAt field of model
}

func NewFDByStructField(structField reflect.StructField) *FieldDescription {
	fieldName := structField.Name

	name := ToSnakeCase2(fieldName)
	dbtag := structField.Tag.Get(TagDBName)
	if len(dbtag) > 0 {
		if dbtag == "-" {
			name = ""
		} else {
			name = dbtag
		}
	}

	elemType := structField.Type
	switch elemType.Kind() {
	case reflect.Ptr, reflect.Slice:
		elemType = elemType.Elem()
	}

	var fullTextEnabled bool

	switch strings.ToLower(structField.Tag.Get(TagFullText)) {
	case "true", "enabled", "yes", "on", "1":
		fullTextEnabled = true
	}

	column := FieldDescription{
		FieldName:       structField.Name,
		ElemType:        elemType,
		DatabaseName:    name,
		JsonName:        utils.JsonFieldName(structField),
		Skip:            name == "" || structField.Tag.Get(TagStore) == "-",
		SkipReplace:     structField.Type == reflect.TypeOf(BigSerial{}),
		Nullable:        SQLAllowNull(structField.Type),
		FullTextEnabled: fullTextEnabled,
		PK:              structField.Name == IDField,
		JsonOmitEmpty:   strings.Contains(structField.Tag.Get("json"), ",omitempty"),
		JsonSkip:        structField.Tag.Get("json") == "-",
	}

	switch strings.ToLower(structField.Tag.Get("nullable")) {
	case "true", "enabled", "yes", "on", "1":
		column.Nullable = true
	case "false", "disabled", "no", "off", "0":
		column.Nullable = false
	}

	if v, ok := structField.Tag.Lookup(TagPK); ok {
		column.PK = v != "-"
	}

	// len for decimal
	column.Ln = 19
	if lns, ok := structField.Tag.Lookup(TagLen); ok && len(lns) > 0 {
		if ln, err := strconv.Atoi(lns); err == nil {
			column.Ln = ln
		}
	}

	// prec for decimal
	column.Prec = 6
	if precs, ok := structField.Tag.Lookup(TagPrec); ok && len(precs) > 0 {
		if prec, err := strconv.Atoi(precs); err == nil {
			column.Prec = prec
		}
	}

	if tname, ok := structField.Tag.Lookup(TagSql); ok && len(tname) > 0 {
		column.SQLTypeDef = tname
	}

	if dv, ok := structField.Tag.Lookup(TagDefVal); ok && len(dv) > 0 {
		column.DefVal = dv
	}

	if indexes, ok := structField.Tag.Lookup(TagKey); ok && len(indexes) > 0 {
		column.Indexes = strings.Split(indexes, ",")
	}

	if ginIndex, ok := structField.Tag.Lookup(TagGinKey); ok && len(ginIndex) > 0 {
		column.GinIndexes = strings.Split(ginIndex, ",")
	}

	if uniIndex, ok := structField.Tag.Lookup(TagUniqueKey); ok && len(uniIndex) > 0 {
		column.UniqIndexes = strings.Split(uniIndex, ",")
	}

	switch column.FieldName {
	case IDField:
		column.IsID = true
	case CreatedAtField:
		column.IsCreatedAt = true
	case UpdatedAtField:
		column.IsUpdatedAt = true
	case DeletedAtField:
		column.IsDeletedAt = true
	}

	return &column
}

// Вывод FieldDescription в виде сткроки
func (fd FieldDescription) String() string {
	ret := fd.FieldName

	if fd.Skip {
		return "- " + ret + " [skip]"
	}

	nullable := ""
	if fd.Nullable {
		nullable = "*"
	}

	ret = fmt.Sprintf("%s%s (db: %s)", nullable, ret, fd.DatabaseName)

	return ret
}

// Говорит о том, сохраняется ли текущее поле в хранилище или нет
func (fd *FieldDescription) IsStored() bool {
	return !fd.Skip
}

func (fd *FieldDescription) MarshalJSON() ([]byte, error) {
	if fd == nil {
		return []byte("null"), nil
	}
	return []byte(fd.JsonName), nil
}

func (fd *FieldDescription) MarshalText() (text []byte, err error) {
	if fd == nil {
		return []byte("null"), nil
	}
	return []byte(fd.JsonName), nil
}

type Valuer interface {
	Value() (driver.Value, error)
	PostgresType() string
}
type Scanner[T any] interface {
	*T
	Scan(src any) error
}

type FD[T Valuer, PT Scanner[T]] struct {
	V     T
	Valid bool
	fd    *FieldDescription
}

func (fdt *FD[T, PT]) FD() *FieldDescription {
	if fdt.fd != nil {
		return fdt.fd
	}
	panic(fmt.Sprintf("%T - fd not initialized", fdt))
}

func NewStructFieldFD[T Valuer, PT Scanner[T]](structField reflect.StructField) *FD[T, PT] {
	return &FD[T, PT]{
		fd:    NewFDByStructField(structField),
		V:     *(new(T)),
		Valid: false,
	}
}

func (n *FD[T, PT]) Scan(value any) error {
	if value == nil {
		n.V, n.Valid = *(new(T)), false
		return nil
	}
	d := PT(new(T))
	err := d.Scan(value)
	if err != nil {
		return err
	}
	n.Valid = true
	n.V = *(d)
	return nil
}

func (n FD[T, PT]) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.V.Value()
}

func (n FD[T, PT]) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return json.Marshal(nil)
	}
	return json.Marshal(n.V)
}

func (n *FD[T, PT]) UnmarshalJSON(b []byte) error {
	if bytes.EqualFold(b, []byte("null")) {
		n.V, n.Valid = *(new(T)), false
		return nil
	}
	err := json.Unmarshal(b, &n.V)
	n.Valid = (err == nil)
	return err
}

func (n FD[T, PT]) PostgresType() string {
	return (*(new(T))).PostgresType()
}
