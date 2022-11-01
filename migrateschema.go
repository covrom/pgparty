package pgparty

import (
	"context"
	"fmt"
)

func (sr *PgStore) SaveModelConfig(ctx context.Context, md *ModelDesc) error {
	c, err := sr.DbConfigTableFromModel(ctx, md)
	if err != nil {
		return err
	}
	return c.SaveTable(ctx)
}

func EnsureModelSchema(ctx context.Context, md *ModelDesc) error {
	s, err := ShardFromContext(ctx)
	if err != nil {
		return fmt.Errorf("EnsureModelSchema: %w", err)
	}
	stx := s.Store
	if stx == nil || stx.tx == nil {
		return fmt.Errorf("context must contains store transaction")
	}
	mdsn := stx.Schema()
	// убедимся, что есть схема
	if _, err := stx.tx.ExecContext(ctx, `CREATE SCHEMA IF NOT EXISTS `+mdsn); err != nil {
		return err
	}

	// убедимся, что есть конфиг-таблица в схеме
	if _, err := stx.tx.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS `+mdsn+`._config (
		table_name VARCHAR(250) NOT NULL,
		storej JSONB NULL,
		PRIMARY KEY (table_name)
	)`); err != nil {
		return err
	}
	return nil
}
