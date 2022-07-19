package pgparty

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/jmoiron/sqlx"
)

type SQLView[T Storable] struct {
	V      T
	MD     *ModelDesc
	Filled []*FieldDescription
}

func (mo *SQLView[T]) Valid() bool {
	return len(mo.Filled) > 0 && mo.MD != nil
}

func NewSQLView[T Storable]() (*SQLView[T], error) {
	val := *(new(T))
	md, err := (MD[T]{Val: val}).MD()
	if err != nil {
		return nil, err
	}
	ret := &SQLView[T]{
		V:      val,
		MD:     md,
		Filled: nil,
	}
	return ret, nil
}

func (mo *SQLView[T]) IsFilled(structFieldNames ...string) bool {
	allfnd := true
	for _, fn := range structFieldNames {
		fnd := false
		for _, fd := range mo.Filled {
			if fd.StructField.Name == fn {
				fnd = true
			}
		}
		allfnd = allfnd && fnd
	}
	return len(structFieldNames) > 0 && allfnd
}

func (mo *SQLView[T]) SetFilled(structFieldNames ...string) error {
	for _, fn := range structFieldNames {
		if mo.IsFilled(fn) {
			continue
		}
		fd, err := mo.MD.ColumnByFieldName(fn)
		if err != nil {
			return err
		}
		mo.Filled = append(mo.Filled, fd)
	}
	return nil
}

func (mo *SQLView[T]) SetUnfilled(structFieldNames ...string) error {
	i := 0
	for _, fd := range mo.Filled {
		fnd := false
		for _, fn := range structFieldNames {
			if fd.StructField.Name == fn {
				fnd = true
				break
			}
		}
		if !fnd {
			mo.Filled[i] = fd
			i++
		}
	}
	for j := i; j < len(mo.Filled); j++ {
		mo.Filled[j] = nil
	}
	mo.Filled = mo.Filled[:i]

	return nil
}

func (mo *SQLView[T]) IsFullFilled() bool {
	allfilled := true
	mo.MD.WalkColumnPtrs(func(_ int, fd *FieldDescription) error {
		for _, fdf := range mo.Filled {
			if fdf == fd {
				return nil
			}
		}
		allfilled = false
		return errors.New("break")
	})
	return allfilled && mo.MD.ColumnPtrsCount() > 0
}

func (mo *SQLView[T]) SetFullFilled() {
	mo.Filled = make([]*FieldDescription, 0, mo.MD.ColumnPtrsCount())
	mo.MD.WalkColumnPtrs(func(_ int, fd *FieldDescription) error {
		mo.Filled = append(mo.Filled)
		return nil
	})
}

func (mo *SQLView[T]) Columns() []string {
	ret := make([]string, 0, len(mo.Filled))
	for _, fd := range mo.Filled {
		if fd.Skip {
			continue
		}
		ret = append(ret, fd.Name)
	}
	return ret
}

func (mo *SQLView[T]) Values() []interface{} {
	ret := make([]interface{}, 0, len(mo.Filled))
	for _, fd := range mo.Filled {
		if fd.Skip {
			continue
		}
		rv := reflect.ValueOf(mo.V).FieldByName(fd.StructField.Name)
		v := rv.Interface()
		ret = append(ret, v)
	}
	return ret
}

// prefix is table alias with point at end, or empty string
func (mo *SQLView[T]) Scan(rows sqlx.ColScanner, prefix string) error {
	mo.V = *(new(T))
	md, err := (MD[T]{Val: mo.V}).MD()
	if err != nil {
		return err
	}
	mo.MD = md
	mo.Filled = nil
	if mo.MD == nil {
		return fmt.Errorf("model description not found for %T", mo.V)
	}
	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	vals := make([]interface{}, len(cols))

	morv := reflect.ValueOf(&mo.V)
	mo.Filled = make([]*FieldDescription, 0, len(cols))

	for i, k := range cols {
		cn := k
		if prefix != "" {
			if !strings.HasPrefix(strings.ToLower(cn), strings.ToLower(prefix)) {
				vals[i] = new(interface{})
				continue
			}
			cn = cn[len(prefix):]
		}
		fd, ok := mo.MD.columnByName[cn]
		if !ok || fd.Skip || fd.StructField.Tag.Get(TagDBName) == "-" {
			vals[i] = new(interface{})
			continue
		}

		vals[i] = morv.Elem().FieldByName(fd.StructField.Name).Addr().Interface()

		mo.Filled = append(mo.Filled, fd)
	}

	return rows.Scan(vals...)
}

func (sv *SQLView[T]) JsonView() *JsonView[T] {
	jv := &JsonView[T]{
		V:  sv.V,
		MD: sv.MD,
	}
	if len(sv.Filled) > 0 {
		jv.Filled = make([]*FieldDescription, len(sv.Filled))
		copy(jv.Filled, sv.Filled)
	}
	return jv
}
