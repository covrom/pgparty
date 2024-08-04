package pgparty

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"runtime"
	"strings"

	"github.com/covrom/pgparty/utils"
)

func Replace[T Storable](ctx context.Context, modelItem T, skipFields ...string) error {
	s, err := ShardFromContext(ctx)
	if err != nil {
		_, file, no, ok := runtime.Caller(1)
		if ok {
			log.Printf("Replace error at %s line %d: %s", file, no, err)
		}
		return fmt.Errorf("Replace: %w", err)
	}
	return s.Store.Replace(ctx, modelItem, skipFields...)
}

// Replace is "insert or update" operation using ID field as key
func (sr *PgStore) Replace(ctx context.Context, modelItem Storable, skipFields ...string) error {
	// ctx = WithLoggingQuery(ctx)
	s, err := ShardFromContext(ctx)
	if err != nil {
		return fmt.Errorf("Replace: %w", err)
	}

	sn := s.Store.Schema()

	return sr.WithTx(ctx, func(srx *PgStore) error {
		md, ok := srx.GetModelDescription(modelItem)
		if !ok {
			return fmt.Errorf("Replace error: cant't get model description for %T in schema %q", modelItem, sn)
		}

		cols := make([]string, 0, md.ColumnPtrsCount())
		vals := make([]interface{}, 0, md.ColumnPtrsCount())

		for i := 0; i < md.ColumnPtrsCount(); i++ {
			fd := md.ColumnPtr(i)
			if fd.SkipReplace || !fd.IsStored() {
				continue
			}
			fnd := false
			for _, skf := range skipFields {
				if skf == fd.FieldName {
					fnd = true
					break
				}
			}
			if fnd {
				continue
			}
			fv, err := utils.GetFieldValueByName(reflect.Indirect(reflect.ValueOf(modelItem)), fd.FieldName)
			if err != nil {
				return err
			}
			cols = append(cols, fd.DatabaseName)
			vals = append(vals, fv.Interface())
		}

		fillers := strings.Join(strings.Split(strings.Repeat("?", len(vals)), ""), ",")

		replQuery := ""

		updkeys := make([]string, 0, len(cols))
		exclkeys := make([]string, 0, len(cols))
		for _, k := range cols {
			if k == md.IdField().DatabaseName {
				continue
			}
			if crf := md.CreatedAtField(); crf != nil && crf.DatabaseName == k {
				continue
			}
			updkeys = append(updkeys, k)
			exclkeys = append(exclkeys, "excluded."+k)
		}
		mdsn := sn + "." + md.DatabaseName()
		if len(updkeys) == 1 {
			replQuery = fmt.Sprintf(`INSERT INTO %s (%s) VALUES(%s) ON CONFLICT(%s) DO UPDATE SET %s=%s`,
				mdsn, strings.Join(cols, ","), fillers,
				md.IdField().DatabaseName, strings.Join(updkeys, ","), strings.Join(exclkeys, ","))
		} else if len(updkeys) > 0 {
			replQuery = fmt.Sprintf(`INSERT INTO %s (%s) VALUES(%s) ON CONFLICT(%s) DO UPDATE SET (%s)=(%s)`,
				mdsn, strings.Join(cols, ","), fillers,
				md.IdField().DatabaseName, strings.Join(updkeys, ","), strings.Join(exclkeys, ","))
		} else {
			replQuery = fmt.Sprintf(`INSERT INTO %s (%s) VALUES(%s) ON CONFLICT(%s) DO NOTHING`,
				mdsn, strings.Join(cols, ","), fillers,
				md.IdField().DatabaseName)
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
