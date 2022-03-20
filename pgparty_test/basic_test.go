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

	shs, ctx := pgparty.NewShards(context.Background())

	shard := shs.SetShard("shard1", db, "shard1")

	if err := pgparty.Register(shard, pgparty.MD[BasicModel]{}); err != nil {
		t.Errorf("pgparty.Register error: %s", err)
		return
	}

	if err := shard.Migrate(ctx, nil); err != nil {
		t.Errorf("shard.Migrate error: %s", err)
		return
	}

	el := BasicModel{
		ID: pgparty.NewV4(),
		Data: *pgparty.NewNullJsonB(map[string]any{
			"field1": "string data",
			"field2": 1344,
			"field3": pgparty.NowUTC(),
		}),
	}

	if err := pgparty.WithTxInShard(ctx, shard.ID, func(ctx context.Context) error {
		return pgparty.Replace[BasicModel](ctx, el)
	}); err != nil {
		t.Errorf("pgparty.Replace error: %s", err)
		return
	}

	var els []BasicModel
	if err := shard.WithTx(ctx, func(ctx context.Context) error {
		return pgparty.Select[BasicModel](ctx, `SELECT * FROM &BasicModel`, &els)
	}); err != nil {
		t.Errorf("pgparty.Select error: %s", err)
		return
	}

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
