package pgparty

import (
	"context"

	"github.com/covrom/pgparty/modelcols"
)

type MigrationRegistrator interface {
	Start(ctx context.Context, stx *PgStore, schema, migrname string) error
	CheckStarted(ctx context.Context, stx *PgStore, schema, migrname string) (bool, error)
	Stop(ctx context.Context, stx *PgStore, schema, migrname string) error
}

type MigrationProcessor interface {
	AfterCreateNewSchemaTable(ctx context.Context, stx *PgStore, md *ModelDesc, schema string) error
	AfterAlterModelError(ctx context.Context, err error, stx *PgStore, md *ModelDesc, sqsdb, sqsmd *modelcols.SQLModel, schema string) error
	AfterMigrate(ctx context.Context, stx *PgStore, reg MigrationRegistrator, sqsdb, sqsmd *modelcols.SQLModel, schema string) error
	AfterCommit(ctx context.Context, stx *PgStore) error
}
