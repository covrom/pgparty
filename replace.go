package pgparty

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/covrom/pgparty/utils"
)

// Replace сохраняет новый или существующий элемент модели, не вызывая никаких callbacks
func (sr *PgStore) Replace(ctx context.Context, modelItem interface{}) error {
	// ctx = WithLoggingQuery(ctx)

	return sr.WithTx(ctx, func(srx *PgStore) error {
		md, ok := sr.GetModelDescription(modelItem)
		if !ok {
			return fmt.Errorf("cant't get model description for %T", modelItem)
		}

		cols := make([]string, 0, md.ColumnPtrsCount())
		vals := make([]interface{}, 0, md.ColumnPtrsCount())

		for i := 0; i < md.ColumnPtrsCount(); i++ {
			fd := md.ColumnPtr(i)
			if fd.SkipReplace || !fd.IsStored() {
				continue
			}
			fv, err := utils.GetFieldValueByName(reflect.Indirect(reflect.ValueOf(modelItem)), fd.StructField.Name)
			if err != nil {
				return err
			}
			cols = append(cols, fd.Name)
			vals = append(vals, fv.Interface())
		}

		fillers := strings.Join(strings.Split(strings.Repeat("?", len(vals)), ""), ",")

		replQuery := ""

		updkeys := make([]string, 0, len(cols))
		exclkeys := make([]string, 0, len(cols))
		for _, k := range cols {
			if k == md.IdField().Name {
				continue
			}
			updkeys = append(updkeys, k)
			exclkeys = append(exclkeys, "excluded."+k)
		}
		mdsn := md.StoreName()
		if md.Schema() != "" {
			mdsn = md.Schema() + "." + md.StoreName()
		}
		if len(updkeys) == 1 {
			replQuery = fmt.Sprintf(`INSERT INTO %s (%s) VALUES(%s) ON CONFLICT(%s) DO UPDATE SET %s=%s`,
				mdsn, strings.Join(cols, ","), fillers,
				md.IdField().Name, strings.Join(updkeys, ","), strings.Join(exclkeys, ","))
		} else if len(updkeys) > 0 {
			replQuery = fmt.Sprintf(`INSERT INTO %s (%s) VALUES(%s) ON CONFLICT(%s) DO UPDATE SET (%s)=(%s)`,
				mdsn, strings.Join(cols, ","), fillers,
				md.IdField().Name, strings.Join(updkeys, ","), strings.Join(exclkeys, ","))
		} else {
			replQuery = fmt.Sprintf(`INSERT INTO %s (%s) VALUES(%s) ON CONFLICT(%s) DO NOTHING`,
				mdsn, strings.Join(cols, ","), fillers,
				md.IdField().Name)
		}

		if IsLoggingQuery(ctx) {
			log.Println("REPLACE QUERY WITH VALUES: ", replQuery, vals)
		}

		if _, err := sr.PrepExec(ctx, replQuery, vals...); err != nil {
			return err
		}
		return nil
	})
}
