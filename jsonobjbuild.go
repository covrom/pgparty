package pgparty

import (
	"fmt"
	"strings"
)

// Creates json_build_object('jsonFieldName', prefixStructFieldName ...)
// prefix is ​​the table name ending with a period or something else
func (sr *Store) JSONBuildObjectSQL(md *ModelDesc, prefix string, onlyStructFieldNames ...string) string {
	sb := &strings.Builder{}
	fmt.Fprint(sb, "json_build_object(")
	needComma := false
	for fdi := 0; fdi < md.ColumnPtrsCount(); fdi++ {
		fd := md.ColumnPtr(fdi)
		if fd.Skip {
			continue
		}
		if len(onlyStructFieldNames) > 0 {
			fnd := false
			for _, v := range onlyStructFieldNames {
				if fd.FieldName == v {
					fnd = true
					break
				}
			}
			if !fnd {
				continue
			}
		}
		if needComma {
			sb.WriteByte(',')
		}
		// только хранимые поля
		if !fd.Skip {
			fmt.Fprintf(sb, `'%s',%s%s`, fd.DatabaseName, prefix, fd.DatabaseName)
			needComma = true
		}
	}
	fmt.Fprint(sb, ")")
	return sb.String()
}
