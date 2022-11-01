package modelcols

import (
	"database/sql/driver"
	"encoding/json"
	"strings"
)

type SQLModel struct {
	Table          string     `json:"table"`
	Columns        SQLColumns `json:"cols,omitempty"`
	Indexes        SQLIndexes `json:"idxs,omitempty"`
	ViewQuery      string     `json:"viewQuery,omitempty"`
	IsView         bool       `json:"isView,omitempty"`
	IsMaterialized bool       `json:"isMaterialized,omitempty"`
}

func (f SQLModel) String() string {
	b, _ := json.Marshal(f)
	return string(b)
}

func (f *SQLModel) Scan(value interface{}) error {
	if value == nil {
		*f = SQLModel{}
		return nil
	}

	if val, ok := value.([]byte); ok {
		if err := json.Unmarshal(val, f); err != nil {
			return err
		}
	}

	return nil
}

func (f SQLModel) Value() (driver.Value, error) {
	rv, err := json.Marshal(f)
	return string(rv), err
}

func (m SQLModel) AllIndexLowerNames() map[string]SQLIndex {
	ret := make(map[string]SQLIndex)
	for _, v := range m.Indexes {
		inm := strings.ToLower(m.Table + v.Name)
		ret[inm] = v
	}
	return ret
}

func (from *SQLModel) Equal(to *SQLModel) bool {
	if len(from.Columns) != len(to.Columns) ||
		len(from.Indexes) != len(to.Indexes) {
		return false
	}

	res := strings.EqualFold(from.Table, to.Table)
	if !res {
		return false
	}

	for _, v1 := range from.Columns {
		fnd := false
		for _, v2 := range to.Columns {
			if strings.EqualFold(v1.ColName, v2.ColName) {
				fnd = true
				res = res && v1.Equal(v2)
				if !res {
					return false
				}
				break
			}
		}
		if !fnd {
			return false
		}
	}

	for _, v1 := range from.Indexes {
		fnd := false
		for _, v2 := range to.Indexes {
			if strings.EqualFold(v1.Name, v2.Name) {
				fnd = true
				res = res && v1.Equal(v2)
				if !res {
					return false
				}
				break
			}
		}
		if !fnd {
			return false
		}
	}

	return res
}
