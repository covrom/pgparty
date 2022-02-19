package pgparty

import (
	"context"
	"database/sql"
	"fmt"
	"log"
)

func (sr *PgStore) Migrate(ctx context.Context, schemaForCommon string, mProcessor MigrationProcessor) error {
	if e := sr.WithTx(ctx, func(stx *PgStore) error {
		tx := stx.tx
		ctxTx := WithPgStore(ctx, stx)
		for _, mdsn := range stx.AllSchemas() {
			mds := stx.ModelDescriptions(mdsn)
			// 	if _, err := tx.ExecContext(ctxTx, `DROP SCHEMA IF EXISTS public`); err != nil {
			// 		log.Println(err)
			// 	}
			for _, md := range mds {
				// FIXME: remove common schema, must be only target schema

				// если схема для всех - пустая, и это модель без схемы (общая), то пропускаем
				// таким образом, будут созданы все глобальные модели
				if schemaForCommon == "" && mdsn == "" {
					continue
				}
				// пропускаем все глобальные модели, если указана схема для общих
				if schemaForCommon != "" && mdsn != "" {
					continue
				}
				if mdsn == "" {
					// подставляем общую схему для общих моделей (без схемы)
					mdsn = schemaForCommon
				}

				// здесь мы просто созраняем имя схемы в контексте
				// поскольку транзакция запущена общая на все схемы, то здесь не следует ожидать,
				// что запросы выполнятся в верной схеме из этого контекста,
				// нужно явно во все запросы добавлять имя схемы
				ctxMdTx := WithCurrentSchema(ctxTx, mdsn)

				// убедимся, что есть схема
				if err := EnsureModelSchema(ctxMdTx, md); err != nil {
					return err
				}

				dbidxs, err := CurrentSchemaIndexes(ctxMdTx, md.StoreName())
				if err != nil {
					return fmt.Errorf("Migrate CurrentSchemaIndexes error: %w", err)
				}

				log.Printf("db table %s have indexes: %s", mdsn+"."+md.StoreName(), dbidxs)

				// грузим конфиг схемы
				dbconf := &DbConfigTable{}
				if err := dbconf.LoadTable(ctxMdTx, md.StoreName()); err != nil {
					return err
				}

				sqsmd, err := MD2SQLModel(md)
				if err != nil {
					return err
				}

				if dbconf.IsEmpty() {
					// пустая - создаем
					if err := SQLCreateModelWithColumns(ctxMdTx, md, sqsmd); err != nil {
						return err
					}
					if mProcessor != nil {
						if err := mProcessor.AfterCreateNewSchemaTable(ctxMdTx, stx, md, mdsn); err != nil {
							return err
						}
					}
					continue
				}

				sqsdb := dbconf.Storej

				if !(sqsdb.Equal(sqsmd) && IndexesEqualToDBIndexes(sqsmd, dbidxs)) {
					// модифицируем таблицу
					err := SQLAlterModel(ctxMdTx, md, dbidxs, sqsdb, sqsmd)
					if err != nil {
						if mProcessor != nil {
							if err2 := mProcessor.AfterAlterModelError(ctxMdTx, err, stx, md, sqsdb, sqsmd, mdsn); err2 != nil {
								return err2
							}
						}
						return err
					}
				}

				// миграции
				if _, err := tx.ExecContext(ctxMdTx,
					`CREATE TABLE IF NOT EXISTS `+mdsn+`._migrations (name VARCHAR(250) NOT NULL, PRIMARY KEY (name))`); err != nil {
					return err
				}
				if mProcessor != nil {
					if err := mProcessor.AfterMigrate(ctxMdTx, stx, sr, sqsdb, sqsmd, mdsn); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}); e != nil {
		return e
	}

	if mProcessor != nil {
		if err := mProcessor.AfterCommit(ctx, sr); err != nil {
			return err
		}
	}

	return nil
}

func (sr *PgStore) Start(ctx context.Context, stx *PgStore, schema, migrname string) error {
	if _, err := stx.Tx().ExecContext(ctx,
		`INSERT INTO `+schema+`._migrations (name) VALUES('`+migrname+`')`); err != nil {
		return fmt.Errorf("can't insert into _migrations: %w", err)
	}
	return nil
}

func (sr *PgStore) CheckStarted(ctx context.Context, stx *PgStore, schema, migrname string) (bool, error) {
	ok := false
	if rows, err := stx.Tx().QueryxContext(ctx,
		`SELECT name FROM `+schema+`._migrations WHERE name = '`+migrname+`'`); err != nil && err != sql.ErrNoRows {
		return false, fmt.Errorf("can't get migration %q: %w", migrname, err)
	} else {
		if rows.Next() {
			ok = true
		}
		rows.Close()
	}
	return ok, nil
}

func (sr *PgStore) Stop(ctx context.Context, stx *PgStore, schema, migrname string) error {
	if _, err := stx.Tx().ExecContext(ctx,
		`DELETE FROM `+schema+`._migrations WHERE name = '`+migrname+`'`); err != nil {
		return fmt.Errorf("can't delete from _migrations: %w", err)
	}
	return nil
}
