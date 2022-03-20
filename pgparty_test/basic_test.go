package pgparty_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/covrom/pgparty"
	log "github.com/sirupsen/logrus"
)

type BasicModel struct {
	ID   pgparty.UUIDv4    `json:"id"`
	Data pgparty.NullJsonB `json:"data"`
}

func (BasicModel) StoreName() string { return "basic_models" }

func TestBasicUsage(t *testing.T) {
	if db == nil {
		log.Fatal("run TestMain before")
	}

	// create shards repository with only one database connect
	shs, ctx := pgparty.NewShards(pgparty.WithLoggingQuery(context.Background()))

	shard := shs.SetShard("shard1", db, "shard1")

	// register models in shard
	if err := pgparty.Register(shard, pgparty.MD[BasicModel]{}); err != nil {
		t.Errorf("pgparty.Register error: %s", err)
		return
	}

	// migrate models in database to current structure
	if err := shard.Migrate(ctx, nil); err != nil {
		t.Errorf("shard.Migrate error: %s", err)
		return
	}
	// this produces sql queries:
	// CREATE TABLE shard1.basic_models (data jsonb,id UUID NOT NULL,PRIMARY KEY (id))
	// INSERT INTO shard1._config (table_name,storej)
	// 	VALUES($1,$2) ON CONFLICT(table_name) DO
	// 	UPDATE SET storej=excluded.storej

	// $1 =  basic_models ,
	// $2 =  {"table":"basic_models","cols":[{"ColName":"data","DataType":"jsonb","DefaultValue":"","NotNull":false,"PrimaryKey":false},
	// {"ColName":"id","DataType":"UUID","DefaultValue":"","NotNull":true,"PrimaryKey":true}]}

	// future migrations use this '_config' table for building differencies as ALTER DDL queries

	// create a model element
	el := BasicModel{
		ID: pgparty.NewV4(),
		Data: *pgparty.NewNullJsonB(map[string]any{
			"field1": "string data",
			"field2": 1344,
			"field3": pgparty.NowUTC(),
		}),
	}

	// replace it in database by id
	if err := pgparty.WithTxInShard(ctx, shard.ID, func(ctx context.Context) error {
		return pgparty.Replace[BasicModel](ctx, el)
	}); err != nil {
		t.Errorf("pgparty.Replace error: %s", err)
		return
	}
	// this produces sql queries:
	// INSERT INTO shard1.basic_models (id,data) VALUES($1,$2) ON CONFLICT(id) DO UPDATE SET data=excluded.data

	// select stored data from model table
	var els []BasicModel
	if err := shard.WithTx(ctx, func(ctx context.Context) error {
		return pgparty.Select[BasicModel](ctx, `SELECT * FROM &BasicModel`, &els) // &BasicModel - model named by golang struct type name
	}); err != nil {
		t.Errorf("pgparty.Select error: %s", err)
		return
	}
	// this produces sql queries:
	// SELECT * FROM shard1.basic_models

	if els[0].ID != el.ID {
		t.Errorf("pgparty.Select error: els[0].ID != el.ID: %s != %s", els[0].ID, el.ID)
		return
	}
	if els[0].Data.Valid != el.Data.Valid {
		t.Errorf("pgparty.Select error: els[0].Data.Valid != el.Data.Valid: %v != %v", els[0].Data.Valid, el.Data.Valid)
		return
	}
	dm1, _ := els[0].Data.MarshalJSON()
	dm2, _ := el.Data.MarshalJSON()
	if !bytes.Equal(dm1, dm2) {
		t.Errorf("pgparty.Select error: !bytes.Equal(dm1, dm2): %q != %q", string(dm1), string(dm2))
		return
	}
}
