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

// Model example
type BasicModel struct {
	ID       pgparty.UUID[BasicModel]      `json:"id"`
	Data     pgparty.NullJsonB             `json:"data"`
	AppXID   pgparty.XID[pgparty.AppXID]   `json:"appId" unikey:"appidx" key:"traceappidx"`
	TraceXID pgparty.XID[pgparty.TraceXID] `json:"traceId" key:"traceappidx"`
}

func (BasicModel) DatabaseName() string { return "basic_models" }

func (BasicModel) UUIDPrefix() string { return "basic_model_" }

// View example
type BasicView struct {
	ID       pgparty.UUID[BasicModel]      `json:"id"`
	AppXID   pgparty.XID[pgparty.AppXID]   `json:"appId"`
	TraceXID pgparty.XID[pgparty.TraceXID] `json:"traceId"`
}

func (BasicView) DatabaseName() string { return "basic_views" }
func (BasicView) ViewQuery() string {
	return `SELECT
		:ID, :AppXID, :TraceXID
		FROM &BasicModel`
}

// Test model and view example
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
	if err := pgparty.Register(shard, pgparty.MD[BasicView]{}); err != nil {
		t.Errorf("pgparty.Register error: %s", err)
		return
	}

	// migrate models in database to current structure
	if err := shard.Migrate(ctx, nil); err != nil {
		t.Errorf("shard.Migrate error: %s", err)
		return
	}
	// this produces sql queries:
	// CREATE TABLE shard1.basic_models (app_xid VARCHAR(20) NOT NULL DEFAULT '00000000000000000000',data JSONB,id UUID NOT NULL,trace_xid VARCHAR(20) NOT NULL DEFAULT '00000000000000000000',PRIMARY KEY (id))
	// CREATE UNIQUE INDEX basic_modelsappidx ON shard1.basic_models(app_xid )
	// CREATE INDEX basic_modelstraceappidx ON shard1.basic_models(app_xid, trace_xid )
	// INSERT INTO shard1._config (table_name,storej)
	// 	VALUES($1,$2) ON CONFLICT(table_name) DO
	// 	UPDATE SET storej=excluded.storej

	// $1 =  basic_models ,
	// $2 =  {"table":"basic_models","cols":[
	// {"ColName":"app_xid","DataType":"VARCHAR(20)","DefaultValue":"'00000000000000000000'","NotNull":true,"PrimaryKey":false},
	// {"ColName":"data","DataType":"JSONB","DefaultValue":"","NotNull":false,"PrimaryKey":false},
	// {"ColName":"id","DataType":"UUID","DefaultValue":"","NotNull":true,"PrimaryKey":true},
	// {"ColName":"trace_xid","DataType":"VARCHAR(20)","DefaultValue":"'00000000000000000000'","NotNull":true,"PrimaryKey":false}
	// ],"idxs":[
	// {"name":"appidx","isUnique":true,"columns":["app_xid"]},
	// {"name":"traceappidx","columns":["app_xid","trace_xid"]}]}

	// CREATE OR REPLACE VIEW shard1.basic_views AS SELECT  id, app_xid, trace_xid  FROM shard1.basic_models
	// INSERT INTO shard1._config (table_name,storej)
	// VALUES($1,$2) ON CONFLICT(table_name) DO
	// UPDATE SET storej=excluded.storej , $1 =  basic_views , $2 =  {"table":"basic_views","cols":[
	// {"ColName":"app_xid","DataType":"VARCHAR(20)","DefaultValue":"'00000000000000000000'","NotNull":true,"PrimaryKey":false},
	// {"ColName":"id","DataType":"UUID","DefaultValue":"","NotNull":true,"PrimaryKey":true},
	// {"ColName":"trace_xid","DataType":"VARCHAR(20)","DefaultValue":"'00000000000000000000'","NotNull":true,"PrimaryKey":false}
	// ],"viewQuery":"SELECT  id, app_xid, trace_xid  FROM shard1.basic_models","isView":true}

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

	// select stored data from view
	var vels []BasicView
	if err := shard.WithTx(ctx, func(ctx context.Context) error {
		return pgparty.Select[BasicView](ctx, `SELECT * FROM &BasicView`, &vels)
	}); err != nil {
		t.Errorf("pgparty.Select error: %s", err)
		return
	}
	// this produces sql queries:
	// SELECT * FROM shard1.basic_views

	if els[0].ID != el.ID {
		t.Errorf("pgparty.Select error: els[0].ID != el.ID: %s != %s", els[0].ID, el.ID)
		return
	}

	if vels[0].ID != el.ID {
		t.Errorf("pgparty.Select error: vels[0].ID != el.ID: %s != %s", vels[0].ID, el.ID)
		return
	}

	if els[0].Data.Valid != el.Data.Valid {
		t.Errorf("pgparty.Select error: els[0].Data.Valid != el.Data.Valid: %v != %v", els[0].Data.Valid, el.Data.Valid)
		return
	}

	fld, err := pgparty.Field(pgparty.WithShard(ctx, shard), els[0], BasicModel{}.ID, "ID")
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

	var svels []pgparty.SQLViewErr[BasicModel]
	if err := shard.WithTx(ctx, func(ctx context.Context) error {
		return pgparty.Select(ctx, `SELECT :ID,:AppXID FROM &BasicModel`, &svels)
	}); err != nil {
		t.Errorf("pgparty.Select error: %s", err)
		return
	}

	if len(svels) == 0 {
		t.Errorf("len(svels) == 0")
		return
	}

	if svels[0].Value == nil {
		t.Errorf("svels[0].Value == nil")
		return
	}

	if svels[0].Value.V.ID != el.ID {
		t.Errorf("pgparty.Select error: svels[0].ID != el.ID: %s != %s", svels[0].Value.V.ID, el.ID)
		return
	}

	if svels[0].Value.V.AppXID != el.AppXID {
		t.Errorf("pgparty.Select error: svels[0].Value.V.AppXID != el.AppXID: %v != %v",
			svels[0].Value.V.AppXID, el.AppXID)
		return
	}

	if len(svels[0].Value.Filled) != 2 {
		t.Errorf("length svels[0].Value.Filled != 2")
		return
	}

	jvel := svels[0].JsonView()
	if len(jvel.Value.Filled) != 2 {
		t.Errorf("length jvel.Value.Filled != 2")
		return
	}

	jvb, err := json.Marshal(jvel)
	if err != nil {
		t.Errorf("jvel marshal error: %s", err)
	}

	jvbe := pgparty.JsonViewErr[BasicModel]{}
	if err := json.Unmarshal(jvb, (&jvbe)); err != nil {
		t.Errorf("json.Unmarshal(jvb,(&jvbe)) error: %s", err)
		return
	}
	if jvel.Value.V.ID != jvbe.Value.V.ID {
		t.Errorf("jvel.Value.V.ID != jvbe.Value.V.ID")
		return
	}

	var svel pgparty.SQLViewErr[BasicModel]
	if err := shard.WithTx(ctx, func(ctx context.Context) error {
		return pgparty.Get(ctx, `SELECT :ID,:AppXID FROM &BasicModel`, &svel)
	}); err != nil {
		t.Errorf("pgparty.Get error: %s", err)
		return
	}

	if svel.Value == nil {
		t.Errorf("svel.Value == nil")
		return
	}

	if svel.Value.V.ID != el.ID {
		t.Errorf("pgparty.Get error: svel.ID != el.ID: %s != %s", svel.Value.V.ID, el.ID)
		return
	}

	if svel.Value.V.AppXID != el.AppXID {
		t.Errorf("pgparty.Get error: svel.Value.V.AppXID != el.AppXID: %v != %v",
			svel.Value.V.AppXID, el.AppXID)
		return
	}

	if len(svel.Value.Filled) != 2 {
		t.Errorf("length svel.Value.Filled != 2")
		return
	}

	jvel = svel.JsonView()
	if len(jvel.Value.Filled) != 2 {
		t.Errorf("length jvel.Value.Filled != 2")
		return
	}

	jvb, err = json.Marshal(jvel)
	if err != nil {
		t.Errorf("jvel marshal error: %s", err)
	}

	jvbe = pgparty.JsonViewErr[BasicModel]{}
	if err := json.Unmarshal(jvb, (&jvbe)); err != nil {
		t.Errorf("json.Unmarshal(jvb,(&jvbe)) error: %s", err)
		return
	}
	if jvel.Value.V.ID != jvbe.Value.V.ID {
		t.Errorf("jvel.Value.V.ID != jvbe.Value.V.ID")
		return
	}
}
