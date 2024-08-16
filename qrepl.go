package pgparty

import (
	"fmt"
)

type ReplaceEntry string

// ReplaceEntries создает map[Old]New - замены текста в запросе для нужной модели
func (md *ModelDesc) ReplaceEntries(schema string) ([]string, map[string]ReplaceEntry, error) {
	rpls := make(map[string]ReplaceEntry)

	if len(md.TypeName()) == 0 {
		return nil, rpls, fmt.Errorf("model has not exported type name")
	}

	mdrepl := "&" + string(md.TypeName())
	mdrepls := []string{mdrepl}

	mdprefix := ":" + string(md.TypeName())

	schmd := schema + "." + md.DatabaseName()

	rpls[mdrepl] = ReplaceEntry(schmd)
	rpls[mdprefix+".*"] = ReplaceEntry(schmd + ".*")

	for fdi := 0; fdi < md.ColumnPtrsCount(); fdi++ {
		fd := md.ColumnPtr(fdi)
		if fd.Skip {
			continue
		}
		rpls[":"+fd.FieldName] = ReplaceEntry(fd.DatabaseName)
		rpls[mdprefix+"."+fd.FieldName] = ReplaceEntry(schmd + "." + fd.DatabaseName)
		rpls[mdprefix+".json."+fd.FieldName] = ReplaceEntry("'" + fd.JsonName + "'")
	}
	return mdrepls, rpls, nil
}
