package pgparty

import (
	"fmt"
	"sort"
	"strings"

	"github.com/covrom/pgparty/modelcols"
)

type PatchTable struct {
	Schema string
	Name   string

	DropIndexes   []fmt.Stringer
	UpdateNulls   []fmt.Stringer
	AlterCols     []fmt.Stringer
	CreateTables  []fmt.Stringer
	CreateIndexes []fmt.Stringer
}

func (pt PatchTable) Queries() []string {
	ret := make([]string, 0)
	if len(pt.DropIndexes) > 0 {
		for _, c := range pt.DropIndexes {
			ret = append(ret, c.String())
		}
	}
	if len(pt.UpdateNulls) > 0 {
		for _, c := range pt.UpdateNulls {
			ret = append(ret, c.String())
		}
	}
	if len(pt.AlterCols) > 0 {
		cs := make([]string, 0, len(pt.AlterCols))
		for _, c := range pt.AlterCols {
			cs = append(cs, c.String())
		}
		ret = append(ret, fmt.Sprintf("ALTER TABLE %s.%s %s", pt.Schema, pt.Name, strings.Join(cs, ", ")))
	}
	if len(pt.CreateTables) > 0 {
		for _, c := range pt.CreateTables {
			ret = append(ret, c.String())
		}
	}
	if len(pt.CreateIndexes) > 0 {
		for _, c := range pt.CreateIndexes {
			ret = append(ret, c.String())
		}
	}
	return ret
}

func (pt *PatchTable) AddColumnPatch(cp fmt.Stringer) {
	pt.AlterCols = append(pt.AlterCols, cp)
}

func (pt *PatchTable) AddDropIndexPatch(cp fmt.Stringer) {
	pt.DropIndexes = append(pt.DropIndexes, cp)
}

func (pt *PatchTable) AddUpdateNullsPatch(cp fmt.Stringer) {
	pt.UpdateNulls = append(pt.UpdateNulls, cp)
}

func (pt *PatchTable) AddCreateTablePatch(cp fmt.Stringer) {
	pt.CreateTables = append(pt.CreateTables, cp)
}

func (pt *PatchTable) AddCreateIndexPatch(cp fmt.Stringer) {
	pt.CreateIndexes = append(pt.CreateIndexes, cp)
}

type PatchAddColumn struct {
	Col modelcols.SQLColumn
}

func (c PatchAddColumn) String() string {
	snull := ""
	if c.Col.NotNull {
		snull = " NOT NULL"
	}
	sdef := ""
	if len(c.Col.DefaultValue) > 0 && !c.Col.PrimaryKey {
		sdef = " DEFAULT " + c.Col.DefaultValue
	}
	return fmt.Sprintf("ADD COLUMN %s %s%s%s", c.Col.ColName, c.Col.DataType, snull, sdef)
}

type PatchAlterColumnType struct {
	Col modelcols.SQLColumn
}

func (c PatchAlterColumnType) String() string {
	return fmt.Sprintf("ALTER COLUMN %s TYPE %s", c.Col.ColName, c.Col.DataType)
}

type PatchAlterColumnNullable struct {
	Col modelcols.SQLColumn
}

func (c PatchAlterColumnNullable) String() string {
	if c.Col.NotNull {
		return fmt.Sprintf("ALTER COLUMN %s SET NOT NULL", c.Col.ColName)
	}
	return fmt.Sprintf("ALTER COLUMN %s DROP NOT NULL", c.Col.ColName)
}

type PatchAlterColumnDefVal struct {
	Col modelcols.SQLColumn
}

func (c PatchAlterColumnDefVal) String() string {
	if len(c.Col.DefaultValue) > 0 && !c.Col.PrimaryKey {
		return fmt.Sprintf("ALTER COLUMN %s SET DEFAULT %s", c.Col.ColName, c.Col.DefaultValue)
	}
	return fmt.Sprintf("ALTER COLUMN %s DROP DEFAULT", c.Col.ColName)
}

type PatchDropIndex struct {
	Schema string
	Table  string
	Index  string
	Force  bool
}

func (c PatchDropIndex) String() string {
	if c.Force {
		return fmt.Sprintf("DROP INDEX %s.%s%s", c.Schema, c.Table, c.Index)
	}
	return fmt.Sprintf("DROP INDEX IF EXISTS %s.%s%s", c.Schema, c.Table, c.Index)
}

type PatchUpdateNulls struct {
	Schema string
	Table  string
	Col    modelcols.SQLColumn
}

func (c PatchUpdateNulls) String() string {
	return fmt.Sprintf("UPDATE %s.%s SET %s = %s WHERE %s IS NULL",
		c.Schema, c.Table, c.Col.ColName, c.Col.DefaultValue, c.Col.ColName)
}

type PatchCreateIndex struct {
	Schema string
	Table  string
	Index  modelcols.SQLIndex
}

func (c PatchCreateIndex) String() string {
	sb := &strings.Builder{}
	fmt.Fprint(sb, "CREATE")
	if c.Index.IsUnique {
		fmt.Fprint(sb, " UNIQUE")
	}
	fmt.Fprint(sb, " INDEX")
	if c.Index.Concurrently {
		fmt.Fprint(sb, " CONCURRENTLY")
	}
	fmt.Fprintf(sb, " %s%s ON %s.%s",
		c.Table, c.Index.Name, c.Schema, c.Table)
	if len(c.Index.MethodName) > 0 {
		fmt.Fprint(sb, " USING ", c.Index.MethodName)
	}
	fmt.Fprintf(sb, "(%s %s)", strings.Join(c.Index.Columns, ", "), c.Index.Options)
	if len(c.Index.With) > 0 {
		fmt.Fprint(sb, " WITH ", c.Index.With)
	}
	if len(c.Index.Where) > 0 {
		fmt.Fprint(sb, " WHERE ", c.Index.Where)
	}
	return sb.String()
}

type PatchCreateTable struct {
	Schema string
	Table  string
	Cols   modelcols.SQLColumns
}

func (c PatchCreateTable) String() string {
	res := make([]string, len(c.Cols))
	pks := make([]string, 0, 1)
	for i, v := range c.Cols {
		snull := ""
		if v.NotNull {
			snull = " NOT NULL"
		}
		sdef := ""
		if len(v.DefaultValue) > 0 && !v.PrimaryKey {
			sdef = " DEFAULT " + v.DefaultValue
		}
		res[i] = fmt.Sprintf("%s %s%s%s", v.ColName, v.DataType, snull, sdef)
		if v.PrimaryKey {
			pks = append(pks, v.ColName)
		}
	}
	if len(pks) > 0 {
		sort.Strings(pks)
		res = append(res, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(pks, ",")))
	}
	return fmt.Sprintf("CREATE TABLE %s.%s (%s)", c.Schema, c.Table, strings.Join(res, ","))
}

type PatchView struct {
	Schema string
	Name   string

	DropViews     []fmt.Stringer
	CreateViews   []fmt.Stringer
	DropIndexes   []fmt.Stringer
	CreateIndexes []fmt.Stringer
}

func (pt *PatchView) AddDropViewPatch(cp fmt.Stringer) {
	pt.DropViews = append(pt.DropViews, cp)
}

func (pt *PatchView) AddCreateViewPatch(cp fmt.Stringer) {
	pt.CreateViews = append(pt.CreateViews, cp)
}

func (pt *PatchView) AddDropIndexPatch(cp fmt.Stringer) {
	pt.DropIndexes = append(pt.DropIndexes, cp)
}

func (pt *PatchView) AddCreateIndexPatch(cp fmt.Stringer) {
	pt.CreateIndexes = append(pt.CreateIndexes, cp)
}

func (pt PatchView) Queries() []string {
	ret := make([]string, 0)
	if len(pt.DropIndexes) > 0 {
		for _, c := range pt.DropIndexes {
			ret = append(ret, c.String())
		}
	}
	if len(pt.DropViews) > 0 {
		for _, c := range pt.DropViews {
			ret = append(ret, c.String())
		}
	}
	if len(pt.CreateViews) > 0 {
		for _, c := range pt.CreateViews {
			ret = append(ret, c.String())
		}
	}
	if len(pt.CreateIndexes) > 0 {
		for _, c := range pt.CreateIndexes {
			ret = append(ret, c.String())
		}
	}
	return ret
}

type PatchCreateView struct {
	Schema       string
	Table        string
	Query        string
	Materialized bool
}

func (c PatchCreateView) String() string {
	if c.Materialized {
		return fmt.Sprintf("CREATE MATERIALIZED VIEW %s.%s AS %s", c.Schema, c.Table, c.Query)
	}
	return fmt.Sprintf("CREATE OR REPLACE VIEW %s.%s AS %s", c.Schema, c.Table, c.Query)
}

type PatchDropView struct {
	Schema string
	Table  string
}

func (c PatchDropView) String() string {
	return fmt.Sprintf("DROP VIEW IF EXISTS %s.%s", c.Schema, c.Table)
}
