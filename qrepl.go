package pgparty

import (
	"fmt"
	"reflect"
)

type ReplaceEntry struct {
	To     string
	Schema string
	Typ    reflect.Type
}

// ReplaceEntries создает map[Old]New - замены текста в запросе для нужной модели
func (md *ModelDesc) ReplaceEntries(schema string) ([]string, map[string]ReplaceEntry, error) {
	rpls := make(map[string]ReplaceEntry)

	if len(md.ModelType().Name()) == 0 {
		return nil, rpls, fmt.Errorf("model has not exported type name")
	}

	mdrepl := "&" + md.ModelType().Name()
	mdrepls := []string{mdrepl}

	mdprefix := ":" + md.ModelType().Name()

	schmd := schema + "." + md.DatabaseName()

	rpls[mdrepl] = ReplaceEntry{schmd, schema, md.ModelType()}
	rpls[mdprefix+".*"] = ReplaceEntry{schmd + ".*", schema, nil}

	for fdi := 0; fdi < md.ColumnPtrsCount(); fdi++ {
		fd := md.ColumnPtr(fdi)
		if fd.Skip {
			continue
		}
		rpls[":"+fd.FieldName] = ReplaceEntry{fd.DatabaseName, "", fd.ElemType}
		rpls[mdprefix+"."+fd.FieldName] = ReplaceEntry{schmd + "." + fd.DatabaseName, "", fd.ElemType}
		rpls[mdprefix+".json."+fd.FieldName] = ReplaceEntry{"'" + fd.JsonName + "'", "", fd.ElemType}
	}
	return mdrepls, rpls, nil
}
