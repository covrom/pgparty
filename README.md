# pgparty
Postgres database/sql access layer with Go generics 1.18+.

You can easily create go structures as models with tag descriptions for tables, fields and indexes in postgres schemas.

`pgparty` implements it in the database, with automatic migration between model descriptions in your service and database. 

We use a special table with model descriptions for migration.

## Basic usage

Create shards repository with only one shard with one database connect
```go
shs, ctx := pgparty.NewShards(pgparty.WithLoggingQuery(context.Background()))

db, err = pgparty.InitDB(pgparty.DatabaseDSN{Postgres: databaseUrl})
if err != nil {
    return err
}

shard := shs.SetShard("shard1", db, "shard1")
```

One shard contains one postgres schema. Its automatically created if not exist.

Define some models, that implements `Storable` interface:
```go
type BasicModel struct {
	ID       pgparty.UUID[BasicModel]      `json:"id"`
	Data     pgparty.NullJsonB             `json:"data"`
	AppXID   pgparty.XID[pgparty.AppXID]   `json:"appId"`
	TraceXID pgparty.XID[pgparty.TraceXID] `json:"traceId"`
}

func (BasicModel) DatabaseName() string { return "basic_models" }
func (BasicModel) UUIDPrefix() string { return "basic_model_" }
```

Now, register some models in shard:
```go
if err := pgparty.Register(shard, pgparty.MD[BasicModel]{}); err != nil {
    t.Errorf("pgparty.Register error: %s", err)
    return
}
```

With each shard we can migrate models in database to current structure. At any time.
```go
if err := shard.Migrate(ctx, nil); err != nil {
    t.Errorf("shard.Migrate error: %s", err)
    return
}
```

This produces sql queries:
```sql
CREATE TABLE shard1.basic_models (app_xid CHAR(20) NOT NULL,data jsonb,id UUID NOT NULL,trace_xid CHAR(20) NOT NULL,PRIMARY KEY (id))
CREATE UNIQUE INDEX basic_modelsappidx ON shard1.basic_models(app_xid )
CREATE INDEX basic_modelstraceappidx ON shard1.basic_models(app_xid, trace_xid )
INSERT INTO shard1._config (table_name,storej) VALUES($1,$2) ON CONFLICT(table_name) DO UPDATE SET storej=excluded.storej
```
where $1 =  `basic_models` ,
$2 =  
```json
{
    "table":"basic_models",
    "cols":[
        {"ColName":"app_xid","DataType":"CHAR(20)","DefaultValue":"","NotNull":true,"PrimaryKey":false},
        {"ColName":"data","DataType":"jsonb","DefaultValue":"","NotNull":false,"PrimaryKey":false},
        {"ColName":"id","DataType":"UUID","DefaultValue":"","NotNull":true,"PrimaryKey":true},
        {"ColName":"trace_xid","DataType":"CHAR(20)","DefaultValue":"","NotNull":true,"PrimaryKey":false}
        ],
    "idxs":[
        {"name":"appidx","isUnique":true,"columns":["app_xid"]},
        {"name":"traceappidx","columns":["app_xid","trace_xid"]}
        ]
}
```

Future migrations use this '_config' table for building differencies as ALTER DDL queries.

Next time, create a model element
```go
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
```

Then replace it in database by id
```go
if err := pgparty.WithTxInShard(ctx, shard.ID, func(ctx context.Context) error {
    return pgparty.Replace[BasicModel](ctx, el)
}); err != nil {
    t.Errorf("pgparty.Replace error: %s", err)
    return
}
```

This produces sql queries:
```sql
INSERT INTO shard1.basic_models (id,data,app_xid,trace_xid) VALUES($1,$2,$3,$4) 
ON CONFLICT(id) DO UPDATE SET (data,app_xid,trace_xid)=(excluded.data,excluded.app_xid,excluded.trace_xid)
```

Now, select written data from database:
```go
var els []BasicModel
if err := shard.WithTx(ctx, func(ctx context.Context) error {
    return 
        pgparty.Select[BasicModel](ctx, 
            `SELECT * FROM &BasicModel`, // &BasicModel - model named by golang struct type name
        &els)
}); err != nil {
    t.Errorf("pgparty.Select error: %s", err)
    return
}
```

This produces sql queries:
```sql
SELECT * FROM shard1.basic_models
```