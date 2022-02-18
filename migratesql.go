package pgparty

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"

	"github.com/covrom/pgparty/modelcols"
)

func Field2SQLColumn(f FieldDescription) (modelcols.SQLColumn, modelcols.SQLIndexes, error) {
	fld := f.StructField
	ft := fld.Type
	// isPtr := false
	for ft.Kind() == reflect.Ptr {
		ft = fld.Type.Elem()
		// isPtr = true
	}

	sqc := modelcols.SQLColumn{
		// Table:      tname,
		ColName:    f.Name,
		NotNull:    !f.Nullable,
		PrimaryKey: f.PK,
	}

	var sqci modelcols.SQLIndexes

	// можно применять тэг "sql" для определения типа данных в sql
	if len(f.SQLTypeDef) > 0 {
		sqc.DataType = f.SQLTypeDef
	} else {
		sqt := SQLType(ft, f.Ln, f.Prec)
		if len(sqt) == 0 {
			return sqc, nil, fmt.Errorf("sql type not defined for type %v", ft)
		}
		sqc.DataType = sqt
	}

	if len(f.DefVal) > 0 {
		sqc.DefaultValue = f.DefVal
	} else if sqc.NotNull && !sqc.PrimaryKey {
		if dv, ok := defaultSQLValues[ft]; ok {
			sqc.DefaultValue = dv
		} else {
			sqc.DefaultValue = defaultSQLKindValues[ft.Kind()]
		}
	}

	for _, idx := range f.Indexes {
		if len(idx) > 0 {
			idxParts := strings.Split(idx, " ")
			sqi := modelcols.SQLIndex{
				Columns: []string{f.Name},
			}
			for i, pi := range idxParts {
				if i == 0 {
					sqi.Name = strings.ToLower(pi)
				} else {
					switch strings.ToLower(pi) {
					case "concurrently":
						sqi.Concurrently = true
					case "unique":
						sqi.IsUnique = true
					case "btree", "hash", "gist", "spgist", "gin", "brin":
						sqi.MethodName = pi
					default:
						if len(sqi.Options) > 0 {
							sqi.Options += " "
						}
						sqi.Options = sqi.Options + pi
					}
				}
			}
			sqci = append(sqci, sqi)
		}
	}

	for _, idx := range f.GinIndexes {
		if len(idx) > 0 {
			idxParts := strings.Split(idx, " ")
			sqi := modelcols.SQLIndex{
				Columns:    []string{f.Name},
				MethodName: "gin",
			}
			for i, pi := range idxParts {
				if i == 0 {
					sqi.Name = strings.ToLower(pi)
				} else {
					if len(sqi.Options) > 0 {
						sqi.Options += " "
					}
					sqi.Options = sqi.Options + pi
				}
			}
			sqci = append(sqci, sqi)
		}
	}

	for _, idx := range f.UniqIndexes {
		if len(idx) > 0 {
			idxParts := strings.Split(idx, " ")
			sqi := modelcols.SQLIndex{
				Columns:  []string{f.Name},
				IsUnique: true,
			}
			for i, pi := range idxParts {
				if i == 0 {
					sqi.Name = strings.ToLower(pi)
				} else {
					switch strings.ToLower(pi) {
					case "concurrently":
						sqi.Concurrently = true
					case "unique":
						sqi.IsUnique = true
					case "btree", "hash", "gist", "spgist", "gin", "brin":
						sqi.MethodName = pi
					default:
						if len(sqi.Options) > 0 {
							sqi.Options += " "
						}
						sqi.Options = sqi.Options + pi
					}
				}
			}
			sqci = append(sqci, sqi)
		}
	}

	return sqc, sqci, nil
}

func MD2SQLModel(md *ModelDesc) (*modelcols.SQLModel, error) {
	ret := &modelcols.SQLModel{
		Table: md.StoreName(),
	}
	sqs := make(modelcols.SQLColumns, 0, md.ColumnPtrsCount())
	sqis := make(modelcols.SQLIndexes, 0)

	for fdIdx := 0; fdIdx < md.ColumnPtrsCount(); fdIdx++ {
		f := md.ColumnPtr(fdIdx)
		if !f.IsStored() {
			continue
		}

		sqc, sqi, err := Field2SQLColumn(*f)
		if err != nil {
			return nil, err
		}
		sqs = append(sqs, sqc)
		for _, idx := range sqi {
			fnd := false
			for i, exidx := range sqis {
				if strings.EqualFold(exidx.Name, idx.Name) {
					exidx.Columns = append(exidx.Columns, idx.Columns...)
					sqis[i] = exidx
					fnd = true
					break
				}
			}
			if !fnd {
				sqis = append(sqis, idx)
			}
		}
	}

	if len(sqs) == 0 {
		return nil, fmt.Errorf("sql fields not found in type %v", md.ModelType())
	}

	sort.Slice(sqs, func(i, j int) bool {
		return sqs[i].ColName < sqs[j].ColName
	})

	sort.Slice(sqis, func(i, j int) bool {
		return sqis[i].Name < sqis[j].Name
	})

	for _, idx := range sqis {
		sort.Slice(idx.Columns, func(i, j int) bool {
			return idx.Columns[i] < idx.Columns[j]
		})
	}

	ret.Columns = sqs
	ret.Indexes = sqis
	return ret, nil
}

func SQLCreateTable(pt *PatchTable, md *ModelDesc, schema string) error {
	sqs, err := MD2SQLModel(md)
	if err != nil {
		return err
	}
	SQLCreateTableWithColumns(pt, sqs)
	return nil
}

func SQLCreateTableWithColumns(pt *PatchTable, sqs *modelcols.SQLModel) {
	pt.AddCreateTablePatch(
		PatchCreateTable{
			Schema: pt.Schema,
			Table:  pt.Name,
			Cols:   sqs.Columns,
		},
	)
	for _, idx := range sqs.Indexes {
		pt.AddCreateIndexPatch(
			PatchCreateIndex{
				Schema: pt.Schema,
				Table:  pt.Name,
				Index:  idx,
			},
		)
	}
}

func SQLAlterTable(schema, tname string, last, to *modelcols.SQLModel, dbcolinfos []DBColInfo, dbidxs DBIndexDefs) []string {
	patchTable := &PatchTable{
		Schema: schema,
		Name:   tname,
	}

	// перебираем колонки
	for _, col := range to.Columns {
		fnd, dbfnd := false, false

		// подбираем данные такой же колонки из схемы в БД
		dbcol := modelcols.SQLColumn{}
		for _, d := range last.Columns {
			if strings.EqualFold(col.ColName, d.ColName) {
				fnd = true
				dbcol = d
				break
			}
		}

		dbcolinfo := DBColInfo{}
		if !fnd {
			// если не нашли в схеме, смотрим на базу
			for _, d := range dbcolinfos {
				if strings.EqualFold(col.ColName, d.Name) {
					dbfnd = true
					dbcolinfo = d
					log.Printf("found DBColInfo: %+v", d)
					break
				}
			}
		}

		if !fnd && !dbfnd {
			// нет в схеме БД или в самой БД - добавляем колонку
			patchTable.AddColumnPatch(PatchAddColumn{
				Col: col,
			})
		}

		if dbfnd {
			// это колонка из бд
			// имитация наличия целевой колонки в схеме, для сравнения
			dbcol.ColName = col.ColName
			dbcol.DefaultValue = col.DefaultValue
			dbcol.PrimaryKey = col.PrimaryKey
			dbcol.NotNull = strings.EqualFold(dbcolinfo.IsNullable, "NO")
			dbcol.DataType = strings.ToUpper(dbcolinfo.Type)
			if dbcol.DataType == "VARCHAR" && dbcolinfo.CharLen != nil {
				dbcol.DataType = fmt.Sprintf("VARCHAR(%d)", *dbcolinfo.CharLen)
			}
			if dbcol.DataType == "JSONB" {
				dbcol.DataType = jsonType
			}
			if dbcol.DataType == "BOOL" {
				dbcol.DataType = "BOOLEAN"
			}
			fnd = true
		}

		if fnd {
			// есть колонка в схеме из БД (или ее имитация) - сравниваем и делаем патчи

			if col.PrimaryKey != dbcol.PrimaryKey {
				// TODO: подумать, как тут менять PK, просто так наверное не выйдет
				// пока считаем что это исключительный случай
				panic("Нельзя менять Primary Key!")
			}

			// если было с null, а стало not null - апдейтим к новому DefaultValue
			if col.NotNull && !dbcol.NotNull && len(col.DefaultValue) > 0 {
				patchTable.AddUpdateNullsPatch(PatchUpdateNulls{
					Schema: schema,
					Table:  tname,
					Col:    col,
				})
			}

			// изменился тип колонки
			if !strings.EqualFold(col.DataType, dbcol.DataType) {
				patchTable.AddColumnPatch(PatchAlterColumnType{
					Col: col,
				})
			}

			// сменился not null
			if col.NotNull != dbcol.NotNull {
				patchTable.AddColumnPatch(PatchAlterColumnNullable{
					Col: col,
				})
			}

			// сменилось дефолтное значение
			if col.DefaultValue != dbcol.DefaultValue {
				patchTable.AddColumnPatch(PatchAlterColumnDefVal{
					Col: col,
				})
			}
		}
	}

	// сравниваем индексы схемы из БД
	// если нет в схеме БД или есть в схеме но нет в БД (или отличаются колонки) - пересоздаем
	knownidxs := make(map[string]bool, len(to.Indexes))
	for _, idxto := range to.Indexes {
		idxlast, oklast := last.Indexes.FindByName(idxto.Name)
		dbidx, okdb := dbidxs.FindByName(patchTable.Name + idxto.Name)
		if okdb {
			knownidxs[strings.ToLower(patchTable.Name+idxto.Name)] = true
		}
		if oklast {
			if okdb {
				// есть в схеме и в базе - пересоздаем если отличаются
				if !(idxto.Equal(idxlast) &&
					IndexEqualDBIndex(patchTable.Name, idxto, dbidx)) {
					// пересоздаем
					patchTable.AddDropIndexPatch(PatchDropIndex{
						Schema: schema,
						Table:  tname,
						Index:  idxto.Name,
					})
					patchTable.AddCreateIndexPatch(PatchCreateIndex{
						Schema: schema,
						Table:  tname,
						Index:  idxto,
					})
				}
			} else {
				// есть в схеме, нет в базе
				patchTable.AddCreateIndexPatch(PatchCreateIndex{
					Schema: schema,
					Table:  tname,
					Index:  idxto,
				})
			}
		} else {
			if okdb {
				// нет в схеме, есть в базе - пересоздаем, т.к. он нужен в новом виде
				patchTable.AddDropIndexPatch(PatchDropIndex{
					Schema: schema,
					Table:  tname,
					Index:  idxto.Name,
				})
				patchTable.AddCreateIndexPatch(PatchCreateIndex{
					Schema: schema,
					Table:  tname,
					Index:  idxto,
				})
			} else {
				// нигде нет - создаем
				patchTable.AddCreateIndexPatch(PatchCreateIndex{
					Schema: schema,
					Table:  tname,
					Index:  idxto,
				})
			}
		}
	}

	// удаляем неактуальные индексы
	for _, idx := range dbidxs {
		if !knownidxs[strings.ToLower(idx.Name)] {
			patchTable.AddDropIndexPatch(PatchDropIndex{
				Schema: schema,
				Force:  true,
				Index:  idx.Name,
			})
		}
	}

	return patchTable.Queries()
}

func SQLCreateModelWithColumns(ctx context.Context, md *ModelDesc, sqs *modelcols.SQLModel) error {
	stx := PgStoreFromContext(ctx)
	sn, ok := CurrentSchemaFromContext(ctx)
	if !ok || stx == nil || stx.tx == nil {
		return fmt.Errorf("context must contains store transaction and schema")
	}

	pt := &PatchTable{
		Schema: sn,
		Name:   md.StoreName(),
	}

	SQLCreateTableWithColumns(pt, sqs)

	qsqls := pt.Queries()

	log.Println(strings.Join(qsqls, "\n"))

	for _, qsql := range qsqls {
		_, err := stx.tx.ExecContext(ctx, qsql)
		if err != nil {
			return err
		}
	}

	return SaveModelConfig(ctx, md)
}

func SQLAlterModel(ctx context.Context, md *ModelDesc, mddbidxs DBIndexDefs, last, to *modelcols.SQLModel) error {
	stx := PgStoreFromContext(ctx)
	sn, ok := CurrentSchemaFromContext(ctx)
	if !ok || stx == nil || stx.tx == nil {
		return fmt.Errorf("context must contains store transaction and schema")
	}

	colinfos, err := DBColumnsInfo(ctx, stx.tx, sn, md.StoreName())
	if err != nil {
		return fmt.Errorf("SQLAlterModel DBColumnsInfo error: %w", err)
	}

	qsqls := SQLAlterTable(sn, md.StoreName(), last, to, colinfos, mddbidxs)

	log.Println(strings.Join(qsqls, "\n"))

	for _, qsql := range qsqls {
		_, err = stx.tx.ExecContext(ctx, qsql)
		if err != nil {
			return fmt.Errorf("SQLAlterModel ExecContext1 error: %w", err)
		}
	}

	return SaveModelConfig(ctx, md)
}
