# AGENTS.md

AI agent instructions for the pgparty codebase.

## Project Overview

pgparty is a PostgreSQL access layer for Go 1.18+ with generics. Core capabilities:
- **Automatic schema migration** — Go structs → DDL, tracked in `_config` table per schema
- **Sharding** — one shard = one PostgreSQL schema + one `*sqlx.DB`
- **Query replacement** — `&ModelName` → schema-qualified table name at execution time
- **Field replacement** — `:FieldName` → column name at execution time
- **NestJS CRUD** — `crud/` package implements nestjsx/crud query params protocol

## Commands

```bash
go test ./...                    # requires Docker (PostgreSQL 13 via dockertest)
go test -v -run TestName         # specific test
go build ./...                   # build
go vet ./...                     # vet
```

## Architecture

### Data Flow

```
Go struct (Modeller)
  → StructModel[T] → Fields() → []FieldDescription
  → ModelDesc (cached in mdRepo, registered per-shard)
  → MD2SQLModel() → *modelcols.SQLModel
  → PatchTable/PatchView → DDL queries
  → _config table (JSONB)
```

### Key Types and Their Files

| Type | File | Responsibility |
|------|------|----------------|
| `Shards`, `Shard` | `shards.go`, `shard.go` | Shard registry, context keys |
| `PgStore` | `pgparty.go` | Per-shard store, transaction lifecycle |
| `Store` | `store.go` | Model descriptions + query replacers maps |
| `ModelDesc` | `md.go` | Model metadata, field indexes |
| `FieldDescription` | `mdflds.go` | Single field metadata (tags, types) |
| `ModelObject` | `modelobject.go` | Dynamic row container, `RowScanner` |
| `JsonView[T]` | `jsonview.go` | Partial-select view with filled-fields tracking |
| `SQLViewErr[T]` | `viewerr.go` | JsonView wrapper with error handling |
| `UniqueObjects` | `modelobjects.go` | Deduplicated ModelObject collection (sync.Pool) |
| `StructModel[T]` | `structmodeller.go` | Auto Modeller from any struct |
| `MD[T]` | `md.go` | ModelDescriber — produces ModelDesc |
| `PatchTable`, `PatchView` | `migrpatch.go` | DDL patch builder |
| `DbConfigTable` | `dbconfig.go` | `_config` table CRUD |
| `RequestQuery` | `crud/parser.go` | NestJS query parser |
| `RequestQueryBuilder` | `crud/builder.go` | NestJS query builder |

### Context Keys

Three context keys propagate state through the call chain:
- `CtxShards{}` → `*Shards` (registry)
- `CtxShard{}` → `Shard` (current shard)
- `logQuery{}` → `bool` (enable query logging)

All database functions extract shard from context. If missing, they return an error.

## Critical Rules

### 1. Tests Use Real PostgreSQL — Never Mock the Database

`main_test.go` spins up PostgreSQL 13 via dockertest. All integration tests use the global `db *sqlx.DB`. If you add new test files, use the existing `db` variable — do not create new containers.

### 2. Null Types Must Implement a Fixed Interface

Every nullable/custom SQL type must implement:

```go
PostgresType() string              // e.g. "BIGINT", "UUID", "VARCHAR(20)"
PostgresDefaultValue() string      // e.g. "0", "'00000000000000000000'"
PostgresAllowNull() bool           // true/false
sql.Scanner                        // Scan(src interface{}) error
driver.Valuer                      // Value() (driver.Value, error)
json.Marshaler                     // MarshalJSON() ([]byte, error)
json.Unmarshaler                   // UnmarshalJSON(b []byte) error
gob.GobEncoder, gob.GobDecoder     // for serialization
```

Register in `init()`: `gob.Register(&TypeName{})`.

### 3. Transaction Pattern

- `WithTx()` — if transaction already exists, reuse it; otherwise begin new one
- `WithTxInShard()` — switch to another shard's transaction (errors if current shard has active tx)
- Panics inside `WithBeginTx` are caught — transaction rolls back automatically
- Cross-shard transactions are **not** supported

### 4. Query Replacement Syntax

| Syntax | Replaces To | Example |
|--------|-------------|---------|
| `&ModelName` | `schema.table_name` | `&BasicModel` → `shard1.basic_models` |
| `&CURRSCHEMA.&Model` | current shard schema | `&CURRSCHEMA.&Foo` → `shard1.foo` |
| `:FieldName` | column name | `:ID` → `id` |
| `:Model.*` | all columns | `:BasicModel.*` → `id,app_xid,...` |
| `?` | positional param (rebounds to `$1`) | `WHERE id=?` → `WHERE id=$1` |

### 5. Struct Tags for Model Mapping

| Tag | Purpose | Example |
|-----|---------|---------|
| `pk:""` | Primary key | `ID pgparty.UUID[T] \`pk:""\`` |
| `key:"idx_name"` | Index (multi-column if shared) | `key:"traceidx"` |
| `unikey:"idx_name"` | Unique index | `unikey:"appidx"` |
| `ginkey:"idx_name"` | GIN index (JSONB) | `ginkey:"data_gin"` |
| `len:"N"` | VARCHAR length | `len:"255"` |
| `prec:"N"` | Numeric precision | `prec:"2"` |
| `defval:"X"` | Default SQL value | `defval:"NOW()"` |
| `sql:"type"` | Override column SQL type | `sql:"TEXT"` |
| `fulltext:""` | Full-text search index | `fulltext:""` |
| `db:"name"` | Override column name in DB | `db:"created_at"` |

Index tags support options: `key:"myidx concurrently unique"` or `key:"myidx gin jsonb_path_ops"`.

### 6. Model Registration

Models must be registered with a shard **before** migration or query execution:

```go
pgparty.Register(shard, pgparty.MD[MyModel]{})
```

This populates `Store.modelDescriptions` and `Store.queryReplacers`. Models are also cached globally in `mdRepo`.

### 7. Replace is Upsert

`pgparty.Replace[T]()` generates `INSERT ... ON CONFLICT(id) DO UPDATE`. It skips `ID` and `CreatedAt` fields in the UPDATE clause. Use `skipFields` parameter to exclude additional fields.

### 8. View Models

Implement `Viewable` for regular views, `MaterializedViewable` for materialized views:

```go
func (MyView) ViewQuery() string { return "SELECT :ID,:Name FROM &MyModel" }
func (MyView) MaterializedView() bool { return true }
```

Views are created/dropped on migration. Materialized views also get indexes.

### 9. RowScanner Interface

Types implementing `RowScanner` get special handling in `Select`/`Get`:

```go
type RowScanner interface {
    RowScan(rows sqlx.ColScanner) error
}
```

`*ModelObject`, `*SQLView[T]`, `*JsonView[T]` all implement this. Regular structs fall back to `sqlx.GetContext`/`SelectContext`.

### 10. In() Expands Slices for WHERE IN

The `In()` function in `inbind.go` expands slice arguments into proper `$1,$2,...` placeholders. Empty slices return an error.

## File Organization

```
pgparty/
├── pgparty.go         — PgStore, WithTx, transactions
├── shard.go           — Shard type, CtxShard
├── shards.go          — Shards registry, CtxShards
├── store.go           — Store (model descriptions, query replacers)
├── md.go              — ModelDesc, MD[T], Register()
├── mdflds.go          — FieldDescription, NewFDByStructField()
├── structmodeller.go  — StructModel[T], auto Modeller
├── modelobject.go     — ModelObject, RowScanner
├── modelobjects.go    — UniqueObjects (dedup + sync.Pool)
├── modelvaluer.go     — Field[T,F](), FieldByFD()
├── prepare.go         — Select[T], Get[T], Exec(), Query(), cursor walk
├── replace.go         — Replace[T] (upsert)
├── qreplace.go        — AnalyzeAndReplaceQuery()
├── rebind.go          — Rebind (? → $N)
├── inbind.go          — In() (slice expansion)
├── migrate.go         — Migrate(), Start/Stop/CheckStarted
├── migratesql.go      — MD2SQLModel, SQLCreate/Alter, Field2SQLColumn
├── migrateschema.go   — SaveModelConfig, EnsureModelSchema
├── migridx.go         — DBIndexDef, CurrentSchemaIndexes
├── migrpatch.go       — PatchTable, PatchView, all Patch* types
├── dbconfig.go        — DbConfigTable, DBColumnsInfo
├── sqltyps.go         — SQLType(), SQLDefaultValue(), type→SQL mapping
├── uuid.go            — UUIDv4, UUID58
├── xid.go             — XID[T], XIDPrefix, AppXID, TraceXID
├── null*.go           — NullInt64, NullString, NullBool, etc.
├── jsonb.go           — JsonB, NullJsonB
├── time.go, nulltime.go — Time, NullTime (UTC)
├── jsonview.go        — JsonView[T]
├── viewerr.go         — JsonViewErr[T], SQLViewErr[T]
├── sqlview.go         — SQLView[T]
├── errors.go          — ErrorNoTransaction, ErrorNotFound
├── logq.go            — WithLoggingQuery
├── simpleprotocol.go  — IsSimpleProtocol
├── initdb.go          — InitDB()
├── snake.go           — SnakeCase
├── collections.go     — Various collection helpers
├── md.go              — ModelsAndFields (mdjs)
├── storeq.go          — PgSelect builder
├── crud/              — NestJS CRUD parser/builder
│   ├── crud.go        — Package doc
│   ├── parser.go      — RequestQuery, ParseQuery
│   ├── builder.go     — RequestQueryBuilder
│   ├── query_types.go — QueryFilter, SCondition, operators
│   ├── errs.go        — CRUD errors
│   └── httpresp.go    — HTTP response helpers
├── list/              — Generic doubly-linked list (stdlib copy)
├── modelcols/         — SQLColumn, SQLIndex, SQLModel
└── utils/             — reflect helpers, deepcopy
```

## Adding New Features

### New SQL Type Wrapper

1. Create `null<type>.go` or `<type>.go`
2. Implement all interfaces from Rule #2
3. Add to `sqlTypesMap` in `sqltyps.go` if it's a primitive mapping
4. Add test in `*_test.go` that verifies DB round-trip

### New CRUD Operation

`Replace[T]` is the only built-in write operation. For deletes/updates:
- Use `Exec(ctx, "DELETE FROM &Model WHERE id=?", id)`
- The `&Model` replacement and `?` → `$1` rebind happen automatically

### New Index Type

Edit `Field2SQLColumn()` in `migratesql.go` — add a new tag handler similar to `GinIndexes` or `UniqIndexes`.

### Model Without Modeller Interface

Use `StructModel[T]` — it auto-generates `TypeName()`, `DatabaseName()`, `Fields()` from any struct. The `DatabaseName()` defaults to `sqlx.NameMapper()` (snake_case of struct name) unless the struct implements `DatabaseName()`.

## Code Style

- **Package-level functions** for generic operations: `Select[T]()`, `Get[T]()`, `Replace[T]()`, `Exec()`
- **Methods on `*PgStore`** for store-specific operations: `PrepSelect()`, `PrepExec()`, `MD2SQLModel()`
- **Generics** for type safety: `UUID[T Modeller]`, `XID[T XIDType]`, `JsonView[T Modeller]`
- **Context** always first argument, always carries shard
- **Error wrapping** with `%w` for sentinel errors (`ErrorNoTransaction`)
- **Logging** via `log.Print` (not structured logger)
- **No public mutable globals** except `mdRepo` (thread-safe with RWMutex)
- **Panics** for programming errors (nil model, unregistered type) — not for runtime errors
