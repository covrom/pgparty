package pgparty_test

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/covrom/pgparty"
	log "github.com/sirupsen/logrus"
)

type Test struct {
	ID          pgparty.UUIDv4 `json:"id"`
	Name        string         `json:"name"`
	Description pgparty.String `json:"desc"`
	Subs        []SubTest      `json:"subs"`
}

func (Test) DatabaseName() string { return "test" }

type SubTest struct {
	ID   pgparty.UUIDv4 `json:"id"`
	Name string         `json:"name"`
}

func TestJsonViewUnmarshalJSON(t *testing.T) {
	el := Test{
		ID:          pgparty.UUIDNewV4(),
		Name:        "simple name",
		Description: "second name",
		Subs: []SubTest{
			{
				ID:   pgparty.UUIDNewV4(),
				Name: "sub1",
			},
			{
				ID:   pgparty.UUIDNewV4(),
				Name: "sub2",
			},
		},
	}
	jv, err := pgparty.NewJsonView[pgparty.StructModel[Test]]()
	if err != nil {
		t.Error(err)
		return
	}
	b, err := json.Marshal(el)
	if err != nil {
		t.Error(err)
		return
	}
	if err := json.Unmarshal(b, jv); err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(jv.V, el) {
		t.Logf("%+v", jv.V)
		t.Error()
	}
}

type BasicModelJsonViewDB struct {
	DBData *pgparty.JsonView[pgparty.StructModel[BasicModel]] `db:"jsv"`
}

func TestJsonViewBasicUsage(t *testing.T) {
	if db == nil {
		log.Fatal("run TestMain before")
	}

	shs, ctx := pgparty.NewShards(pgparty.WithLoggingQuery(context.Background()))

	shard := shs.SetShard("shard1", db, "shard1")

	if err := pgparty.Register(shard, pgparty.MD[pgparty.StructModel[BasicModel]]{}); err != nil {
		t.Errorf("pgparty.Register error: %s", err)
		return
	}

	if err := shard.Migrate(ctx, nil); err != nil {
		t.Errorf("shard.Migrate error: %s", err)
		return
	}

	el := pgparty.StructModel[BasicModel]{M: BasicModel{
		ID: pgparty.NewUUID[BasicModel](),
		Data: *pgparty.NewNullJsonB(map[string]any{
			"field1": "string data",
			"field2": 1344,
			"field3": pgparty.NowUTC(),
		}),
		AppXID:   pgparty.NewXID[pgparty.AppXID](),
		TraceXID: pgparty.NewXID[pgparty.TraceXID](),
	}}

	if err := pgparty.WithTxInShard(ctx, shard.ID, func(ctx context.Context) error {
		return pgparty.Replace[pgparty.StructModel[BasicModel]](ctx, el)
	}); err != nil {
		t.Errorf("pgparty.Replace error: %s", err)
		return
	}

	jv := BasicModelJsonViewDB{}

	if err := pgparty.WithTxInShard(ctx, shard.ID, func(ctx context.Context) error {
		return pgparty.Get[BasicModelJsonViewDB](ctx, `
		select jsonb_build_object(':Data',:Data, ':AppXID',:AppXID) as jsv from &BasicModel where :ID = ?
		`, &jv, el.M.ID)
	}); err != nil {
		t.Errorf("pgparty.Get error: %s", err)
		return
	}

	jvjson, err := json.Marshal(jv.DBData)
	if err != nil {
		t.Errorf("json.Marshal error: %s", err)
		return
	}

	md, _ := (pgparty.MD[pgparty.StructModel[BasicModel]]{}).MD()
	fds := md.ColumnsByFieldNames("Data", "AppXID")
	je := BasicModelJsonViewDB{
		DBData: &pgparty.JsonView[pgparty.StructModel[BasicModel]]{
			V: pgparty.StructModel[BasicModel]{M: BasicModel{
				Data:   el.M.Data,
				AppXID: el.M.AppXID,
			}},
			MD:     md,
			Filled: fds,
		},
	}
	jejson, err := json.Marshal(je.DBData)
	if err != nil {
		t.Errorf("json.Marshal error: %s", err)
		return
	}
	if !bytes.Equal(jejson, jvjson) {
		t.Errorf("json not equal:\n%s\n%s", string(jejson), string(jvjson))
	}

	jv = BasicModelJsonViewDB{}

	if err := pgparty.WithTxInShard(ctx, shard.ID, func(ctx context.Context) error {
		return pgparty.Get[BasicModelJsonViewDB](ctx, `
		select ?::jsonb as jsv
		`, &jv, je.DBData)
	}); err != nil {
		t.Errorf("pgparty.Get error: %s", err)
		return
	}

	jvjson, err = json.Marshal(jv.DBData)
	if err != nil {
		t.Errorf("json.Marshal error: %s", err)
		return
	}

	jejson, err = json.Marshal(je.DBData)
	if err != nil {
		t.Errorf("json.Marshal error: %s", err)
		return
	}

	if !bytes.Equal(jejson, jvjson) {
		t.Errorf("json not equal:\n%s\n%s", string(jejson), string(jvjson))
	}

	jv = BasicModelJsonViewDB{}

	if err := pgparty.WithTxInShard(ctx, shard.ID, func(ctx context.Context) error {
		return pgparty.Get[BasicModelJsonViewDB](ctx, `
		select ? as jsv
		`, &jv, je.DBData)
	}); err != nil {
		t.Errorf("pgparty.Get error: %s", err)
		return
	}

	jvjson, err = json.Marshal(jv.DBData)
	if err != nil {
		t.Errorf("json.Marshal error: %s", err)
		return
	}

	jejson, err = json.Marshal(je.DBData)
	if err != nil {
		t.Errorf("json.Marshal error: %s", err)
		return
	}

	if !bytes.Equal(jejson, jvjson) {
		t.Errorf("json not equal:\n%s\n%s", string(jejson), string(jvjson))
	}
}
