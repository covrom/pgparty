package pgparty

import (
	"strconv"
	"strings"
)

// Rebind a query from the default bindtype (QUESTION) to the target bindtype.
// Escaping: ?? translate to ?
func Rebind(query string) string {
	// Add space enough for 10 params before we have to allocate
	rqb := make([]byte, 0, len(query)+10)

	var i int
	var j int64

	for i = strings.Index(query, "?"); i != -1; i = strings.Index(query, "?") {
		if i+1 < len(query) {
			si1 := query[i+1]
			if si1 == '?' {
				rqb = append(rqb, query[:i+1]...)
				query = query[i+2:]
				continue
			}
			if si1 == '|' || si1 == '&' {
				rqb = append(rqb, query[:i+2]...)
				query = query[i+2:]
				continue
			}
		}
		rqb = append(rqb, query[:i]...)
		rqb = append(rqb, '$')
		j++
		rqb = strconv.AppendInt(rqb, j, 10)
		query = query[i+1:]
	}

	return string(append(rqb, query...))
}
