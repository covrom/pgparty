package modelcols

import (
	"strings"
)

type SQLColumn struct {
	ColName      string
	DataType     string
	DefaultValue string
	NotNull      bool
	PrimaryKey   bool
}

func (sqc SQLColumn) Equal(cto SQLColumn) bool {
	return strings.EqualFold(sqc.ColName, cto.ColName) &&
		strings.EqualFold(sqc.DataType, cto.DataType) &&
		sqc.DefaultValue == cto.DefaultValue &&
		sqc.NotNull == cto.NotNull &&
		sqc.PrimaryKey == cto.PrimaryKey
}

type SQLColumns []SQLColumn

func (sqs SQLColumns) FindColumnByName(name string) (SQLColumn, bool) {
	for _, v := range sqs {
		if strings.EqualFold(v.ColName, name) {
			return v, true
		}
	}
	return SQLColumn{}, false
}
