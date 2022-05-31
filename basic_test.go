package pgparty_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/covrom/pgparty"
	log "github.com/sirupsen/logrus"
)

type BasicModel struct {
	ID       pgparty.UUID[BasicModel]      `json:"id"`
	Data     pgparty.NullJsonB             `json:"data"`
	AppXID   pgparty.XID[pgparty.AppXID]   `json:"appId" unikey:"appidx" key:"traceappidx"`
	TraceXID pgparty.XID[pgparty.TraceXID] `json:"traceId" key:"traceappidx"`
}

func (BasicModel) StoreName() string { return "basic_models" }

func (BasicModel) UUIDPrefix() string { return "basic_model_" }

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
	// CREATE TABLE shard1.basic_models (app_xid CHAR(20) NOT NULL,data jsonb,id UUID NOT NULL,trace_xid CHAR(20) NOT NULL,PRIMARY KEY (id))
	// CREATE UNIQUE INDEX basic_modelsappidx ON shard1.basic_models(app_xid )
	// CREATE INDEX basic_modelstraceappidx ON shard1.basic_models(app_xid, trace_xid )
	// INSERT INTO shard1._config (table_name,storej)
	// 	VALUES($1,$2) ON CONFLICT(table_name) DO
	// 	UPDATE SET storej=excluded.storej

	// $1 =  basic_models ,
	// $2 =  {"table":"basic_models","cols":[
	// {"ColName":"app_xid","DataType":"CHAR(20)","DefaultValue":"","NotNull":true,"PrimaryKey":false},
	// {"ColName":"data","DataType":"jsonb","DefaultValue":"","NotNull":false,"PrimaryKey":false},
	// {"ColName":"id","DataType":"UUID","DefaultValue":"","NotNull":true,"PrimaryKey":true},
	// {"ColName":"trace_xid","DataType":"CHAR(20)","DefaultValue":"","NotNull":true,"PrimaryKey":false}],
	// "idxs":[
	// {"name":"appidx","isUnique":true,"columns":["app_xid"]},
	// {"name":"traceappidx","columns":["app_xid","trace_xid"]}]}

	// future migrations use this '_config' table for building differencies as ALTER DDL queries

	// create a model element
	el := BasicModel{
		ID: pgparty.NewUUID[BasicModel](),
		Data: *pgparty.NewNullJsonB(map[string]any{
			"field1": "string data",
			"field2": 1344,
			"field3": pgparty.NowUTC(),
		}),
		AppXID:   pgparty.NewXID[pgparty.AppXID](),
		TraceXID: pgparty.NewXID[pgparty.TraceXID](),
	}

	// replace it in database by id
	if err := pgparty.WithTxInShard(ctx, shard.ID, func(ctx context.Context) error {
		return pgparty.Replace[BasicModel](ctx, el)
	}); err != nil {
		t.Errorf("pgparty.Replace error: %s", err)
		return
	}
	// this produces sql queries:
	// INSERT INTO shard1.basic_models (id,data,app_xid,trace_xid) VALUES($1,$2,$3,$4) ON CONFLICT(id) DO UPDATE SET (data,app_xid,trace_xid)=(excluded.data,excluded.app_xid,excluded.trace_xid)

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

	fld, err := pgparty.Field(pgparty.WithShard(ctx, shard), els[0], "ID", BasicModel{}.ID)
	if err != nil {
		t.Errorf("pgparty.FieldT error: %s", err)
		return
	}
	if fld != el.ID {
		t.Errorf("pgparty.FieldT error: fld != el.ID: %s != %s", fld, el.ID)
		return
	}

	dm1, _ := els[0].Data.MarshalJSON()
	dm2, _ := el.Data.MarshalJSON()
	if !bytes.Equal(dm1, dm2) {
		t.Errorf("pgparty.Select error: !bytes.Equal(dm1, dm2): %q != %q", string(dm1), string(dm2))
		return
	}

	mval, err := json.Marshal(els[0])
	if err != nil {
		t.Errorf("json.Marshal(els[0]) error: %s", err)
		return
	}
	mraw, err := json.Marshal(el)
	if err != nil {
		t.Errorf("json.Marshal(el) error: %s", err)
		return
	}
	if !bytes.Equal(mval, mraw) {
		t.Errorf("pgparty.Select error: !bytes.Equal(dm1, dmraw): %q != %q", string(mval), string(mraw))
		return
	}

	jst := pgparty.UUIDJsonTyped[BasicModel](el.ID)
	jstb, err := json.Marshal(jst)
	if err != nil {
		t.Errorf("json.Marshal(jst) error: %s", err)
		return
	}
	jstb2 := []byte(fmt.Sprintf(`"%s"`, el.ID.String()))
	if !bytes.Equal(jstb, jstb2) {
		t.Errorf("jstb not bytes.Equal: %s != %s", string(jstb), string(jstb2))
		return
	}

	jv := pgparty.UUIDJsonTyped[BasicModel]{}
	if err := (&jv).UnmarshalJSON(jstb); err != nil {
		t.Errorf("(&jv).UnmarshalJSON(jstb) error: %s", err)
		return
	}
	if jv.UUID != el.ID.UUID {
		t.Errorf("jv.UUID != el.ID.UUID: %s != %s", jv.UUID.String(), el.ID.UUID.String())
		return
	}

	jelb, err := json.Marshal(el.ID.UUID)
	if err != nil {
		t.Errorf("json.Marshal(el.ID.UUID) error: %s", err)
		return
	}
	if err := (&jv).UnmarshalJSON(jelb); err != nil {
		t.Errorf("(&jv).UnmarshalJSON(jelb) error: %s", err)
		return
	}
	if jv.UUID != el.ID.UUID {
		t.Errorf("jv.UUID != el.ID.UUID: %s != %s", jv.UUID.String(), el.ID.UUID.String())
		return
	}
}
