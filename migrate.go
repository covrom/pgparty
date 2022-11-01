package pgparty

import (
	"context"
	"database/sql"
	"fmt"
	"log"
)

func (sr *PgStore) Migrate(ctx context.Context, mProcessor MigrationProcessor) error {
	shard, err := ShardFromContext(ctx)
	if err != nil {
		return fmt.Errorf("Migrate: %w", err)
	}
	if e := sr.WithTx(ctx, func(stx *PgStore) error {
		tx := stx.tx
		ctxTx := WithShard(ctx, Shard{shard.ID, stx})
		mdsn := stx.Schema()
		mds := stx.ModelDescriptions()
		// 	if _, err := tx.ExecContext(ctxTx, `DROP SCHEMA IF EXISTS public`); err != nil {
		// 		log.Println(err)
		// 	}
		for _, md := range mds {
			// здесь мы просто сохраняем имя схемы в контексте
			// поскольку транзакция запущена общая на все схемы, то здесь не следует ожидать,
			// что запросы выполнятся в верной схеме из этого контекста,
			// нужно явно во все запросы добавлять имя схемы

			// убедимся, что есть схема
			if err := EnsureModelSchema(ctxTx, md); err != nil {
				return err
			}

			dbidxs, err := CurrentSchemaIndexes(ctxTx, md.StoreName())
			if err != nil {
				return fmt.Errorf("Migrate CurrentSchemaIndexes error: %w", err)
			}

			log.Printf("db table %s have indexes: %s", mdsn+"."+md.StoreName(), dbidxs)

			// грузим конфиг схемы
			dbconf := &DbConfigTable{}
			if err := dbconf.LoadTable(ctxTx, md.StoreName()); err != nil {
				return err
			}

			sqsmd, err := stx.MD2SQLModel(ctxTx, md)
			if err != nil {
				return err
			}

			if dbconf.IsEmpty() {
				// пустая - создаем
				if err := SQLCreateModelWithColumns(ctxTx, md, sqsmd); err != nil {
					return err
				}
				if mProcessor != nil {
					if err := mProcessor.AfterCreateNewSchemaTable(ctxTx, stx, md, mdsn); err != nil {
						return err
					}
				}
				continue
			}

			sqsdb := dbconf.Storej

			if !(sqsdb.Equal(sqsmd) && IndexesEqualToDBIndexes(sqsmd, dbidxs)) {
				// модифицируем таблицу
				err := SQLAlterModel(ctxTx, md, dbidxs, sqsdb, sqsmd)
				if err != nil {
					if mProcessor != nil {
						if err2 := mProcessor.AfterAlterModelError(ctxTx, err, stx, md, sqsdb, sqsmd, mdsn); err2 != nil {
							return err2
						}
					}
					return err
				}
			}

			// миграции
			if _, err := tx.ExecContext(ctxTx,
				`CREATE TABLE IF NOT EXISTS `+mdsn+`._migrations (name VARCHAR(250) NOT NULL, PRIMARY KEY (name))`); err != nil {
				return err
			}
			if mProcessor != nil {
				if err := mProcessor.AfterMigrate(ctxTx, stx, sr, sqsdb, sqsmd, mdsn); err != nil {
					return err
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
