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

	schmd := schema + "." + md.StoreName()

	rpls[mdrepl] = ReplaceEntry{schmd, schema, md.ModelType()}
	rpls[mdprefix+".*"] = ReplaceEntry{schmd + ".*", schema, nil}

	for fdi := 0; fdi < md.ColumnPtrsCount(); fdi++ {
		fd := md.ColumnPtr(fdi)
		if fd.Skip {
			continue
		}
		rpls[":"+fd.StructField.Name] = ReplaceEntry{fd.Name, "", fd.StructField.Type}
		rpls[mdprefix+"."+fd.StructField.Name] = ReplaceEntry{schmd + "." + fd.Name, "", fd.StructField.Type}
		rpls[mdprefix+".json."+fd.StructField.Name] = ReplaceEntry{"'" + fd.JsonName + "'", "", fd.StructField.Type}
	}
	return mdrepls, rpls, nil
}
