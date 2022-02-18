package pgparty

import (
	"fmt"
	"strings"
)

// создавет строку вида json_build_object('jsonFieldName', prefixStructFieldName ...)
// если берутся поля от модели, то нужно использовать префикс ":".
// если от именованной таблицы tabl, то нужно использвать префикс "tabl.:"
// если с префиксом модели, то нужно использвать префикс ":ModelName."
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
				if fd.StructField.Name == v {
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
			fmt.Fprintf(sb, `'%s',%s%s`, fd.JsonName, prefix, fd.StructField.Name)
			needComma = true
		}
	}
	fmt.Fprint(sb, ")")
	return sb.String()
}
