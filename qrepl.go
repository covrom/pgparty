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
func ReplaceEntries(md *ModelDesc) ([]string, map[string]ReplaceEntry, error) {
	rpls := make(map[string]ReplaceEntry)

	if len(md.ModelType().Name()) == 0 {
		return nil, rpls, fmt.Errorf("model has not exported type name")
	}

	mdrepl := "&" + md.ModelType().Name()
	mdrepls := []string{mdrepl}

	if md.Schema() == "" {
		rpls[mdrepl] = ReplaceEntry{md.StoreName(), "", md.ModelType()}
		// все поля модели
		rpls[":"+md.ModelType().Name()+".*"] = ReplaceEntry{md.StoreName() + ".*", "", nil}
	} else {
		rpls[mdrepl] = ReplaceEntry{md.Schema() + "." + md.StoreName(), md.Schema(), md.ModelType()}
		// все поля модели
		rpls[":"+md.ModelType().Name()+".*"] = ReplaceEntry{md.Schema() + "." + md.StoreName() + ".*", md.Schema(), nil}
	}

	// отдельные поля
	for fdi := 0; fdi < md.ColumnPtrsCount(); fdi++ {
		fd := md.ColumnPtr(fdi)
		if fd.Skip {
			continue
		}
		rpls[":"+fd.StructField.Name] = ReplaceEntry{fd.Name, "", fd.StructField.Type}
		if md.Schema() == "" {
			rpls[":"+md.ModelType().Name()+"."+fd.StructField.Name] = ReplaceEntry{md.StoreName() + "." + fd.Name, "", fd.StructField.Type}
		} else {
			rpls[":"+md.ModelType().Name()+"."+fd.StructField.Name] = ReplaceEntry{md.Schema() +
				"." + md.StoreName() + "." + fd.Name, "", fd.StructField.Type}
		}
		rpls[":"+md.ModelType().Name()+".json."+fd.StructField.Name] = ReplaceEntry{"'" + fd.JsonName + "'", "", fd.StructField.Type}
	}
	return mdrepls, rpls, nil
}
