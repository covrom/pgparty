package pgparty

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx/reflectx"
)

// In expands slice values in args, returning the modified query string
// and a new arg list that can be executed by a database. The `query` should
// use the `?` or `$n` bindVar.  The return value uses the `?` bindVar.
func In(query string, args ...interface{}) (string, []interface{}, error) {
	// argMeta stores reflect.Value and length for slices and
	// the value itself for non-slice arguments
	type argMeta struct {
		v      reflect.Value
		i      interface{}
		length int
		from   int
	}

	var flatArgsCount int
	var anySlices bool

	meta := make([]argMeta, len(args))

	for i, arg := range args {
		if a, ok := arg.(driver.Valuer); ok {
			aVal := reflect.ValueOf(a)
			switch aVal.Kind() {
			case reflect.Ptr:
				if aVal.IsNil() {
					arg = nil
				} else {
					arg, _ = a.Value()
				}
			default:
				arg, _ = a.Value()
			}
		}

		isSlice := false
		v := reflect.ValueOf(arg)
		if arg != nil {
			t := reflectx.Deref(v.Type())
			// []byte is a driver.Value type so it should not be expanded
			isSlice = t.Kind() == reflect.Slice && t != reflect.TypeOf([]byte{})
		}
		if isSlice {
			vlen := v.Len()
			meta[i].length = vlen
			meta[i].v = v

			anySlices = true
			meta[i].from = flatArgsCount + 1
			flatArgsCount += vlen

			if vlen == 0 {
				return "", nil, fmt.Errorf("empty slice passed to 'in' query %q : %#v", query, arg)
			}
		} else {
			meta[i].i = arg
			meta[i].from = flatArgsCount + 1
			flatArgsCount++
		}
	}

	// don't do any parsing if there aren't any slices;  note that this means
	// some errors that we might have caught below will not be returned.
	if !anySlices {
		return query, args, nil
	}

	newArgs := make([]interface{}, flatArgsCount)
	buf := make([]byte, 0, len(query)+3*flatArgsCount)

	var arg, offset int

	for i := strings.IndexAny(query, "?$"); i != -1; i = strings.IndexAny(query, "?$") {
		if arg >= len(meta) {
			// if an argument wasn't passed, lets return an error;  this is
			// not actually how database/sql Exec/Query works, but since we are
			// creating an argument list programmatically, we want to be able
			// to catch these programmer errors earlier.
			return "", nil, errors.New("number of bindVars exceeds arguments")
		}
		offset = 0

		var argM argMeta
		if query[i] == '?' {
			if i+1 < len(query) && query[i+1] == '?' {
				// skip ??
				buf = append(buf, query[:i+2]...)
				query = query[i+2:]
				continue
			}
			argM = meta[arg]
			arg++
		} else {
			numa := 0
			for j := i + 1; j < len(query); j++ {
				if c := query[j]; c >= '0' && c <= '9' {
					numa = 10*numa + int(c-'0')
					offset++
				} else {
					break
				}
			}
			if numa > len(meta) {
				// if an argument wasn't passed, lets return an error;  this is
				// not actually how database/sql Exec/Query works, but since we are
				// creating an argument list programmatically, we want to be able
				// to catch these programmer errors earlier.
				return "", nil, fmt.Errorf("argument number '$%d' out of range", numa)
			}
			argM = meta[numa-1]
		}

		// write everything up to and including our ? character
		buf = append(buf, query[:i]...)
		buf = append(buf, '$')
		buf = strconv.AppendInt(buf, int64(argM.from), 10)

		if argM.length > 0 {
			for si := 1; si < argM.length; si++ {
				buf = append(buf, ',', '$')
				buf = strconv.AppendInt(buf, int64(argM.from+si), 10)
			}
			putReflectSlice(newArgs, argM.v, argM.length, argM.from-1)
		} else {
			// not a slice
			newArgs[argM.from-1] = argM.i
		}

		// slice the query and reset the offset. this avoids some bookkeeping for
		// the write after the loop
		query = query[offset+i+1:]
	}

	buf = append(buf, query...)

	// if arg < len(meta) {
	// 	return "", nil, errors.New("number of bindVars less than number arguments")
	// }

	return string(buf), newArgs, nil
}

func putReflectSlice(args []interface{}, v reflect.Value, vlen int, toidx int) {
	switch val := v.Interface().(type) {
	case []interface{}:
		for i := range val {
			args[i+toidx] = val[i]
		}
	case []int:
		for i := range val {
			args[i+toidx] = val[i]
		}
	case []string:
		for i := range val {
			args[i+toidx] = val[i]
		}
	default:
		for si := 0; si < vlen; si++ {
			args[si+toidx] = v.Index(si).Interface()
		}
	}
}
