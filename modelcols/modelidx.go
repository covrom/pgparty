package modelcols

import "strings"

type SQLIndex struct {
	Name         string   `json:"name"`
	IsUnique     bool     `json:"isUnique,omitempty"`
	MethodName   string   `json:"methodName,omitempty"`
	Columns      []string `json:"columns"`
	Options      string   `json:"options,omitempty"` // jsonb_path_ops или (title NULLS FIRST)
	Concurrently bool     `json:"concurrently,omitempty"`
	With         string   `json:"with,omitempty"`
	Where        string   `json:"where,omitempty"`
}

type SQLIndexes []SQLIndex

func (idxs SQLIndexes) FindByName(n string) (SQLIndex, bool) {
	for _, idx := range idxs {
		if strings.EqualFold(idx.Name, n) {
			return idx, true
		}
	}
	return SQLIndex{}, false
}

func (idx SQLIndex) Equal(to SQLIndex) bool {
	return strings.EqualFold(idx.Name, to.Name) &&
		idx.IsUnique == to.IsUnique &&
		strings.EqualFold(idx.MethodName, to.MethodName) &&
		ColumnsEqual(idx.Columns, to.Columns) &&
		strings.EqualFold(idx.Options, to.Options) &&
		idx.Concurrently == to.Concurrently &&
		strings.EqualFold(idx.With, to.With) &&
		strings.EqualFold(idx.Where, to.Where)
}

func ColumnsEqual(cols1, cols2 []string) bool {
	if len(cols1) != len(cols2) {
		return false
	}
	for _, c1 := range cols1 {
		fnd := false
		for _, c2 := range cols2 {
			if strings.EqualFold(c1, c2) {
				fnd = true
				break
			}
		}
		if !fnd {
			return false
		}
	}
	return true
}
