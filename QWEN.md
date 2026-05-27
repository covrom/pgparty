# pgparty

**PostgreSQL database/sql access layer with Go generics 1.18+**

pgparty is a Go ORM-like library that provides automatic schema migration, sharding, and type-safe database access for PostgreSQL. It bridges Go struct definitions with database tables using struct tags and performs live migrations between model descriptions and the actual database schema.

## Architecture

### Core Concepts

- **Shards** — A `Shard` maps to a single PostgreSQL schema. Multiple shards can exist in a `*Shards` registry, each with its own database connection and schema. Shards are stored in `context.Context` and resolved at query time.
- **Models** — Go structs that implement `Storable` / `Modeller` interfaces. Struct fields are mapped to PostgreSQL columns via struct tags (`sql`, `store`, `key`, `pk`, `unikey`, `ginkey`, `fulltext`, `len`, `prec`, `defval`, `db`).
- **Automatic Migration** — On `Migrate()`, pgparty compares the Go model metadata with the database's `_config` table and generates `CREATE TABLE`, `ALTER TABLE`, and index DDL statements automatically.
- **Query Replacement** — SQL queries can use `&ModelName` syntax to have table names replaced with schema-qualified names at execution time. This enables writing queries without hardcoding schema prefixes.
- **Transactions** — Transaction scope is managed via `WithTx()` and `WithTxInShard()`. Nested calls reuse existing transactions. Panics inside transactions trigger automatic rollback.

### Key Types

| Type | File | Purpose |
|------|------|---------|
| `Shards`, `Shard` | `shards.go`, `shard.go` | Shard registry and context propagation |
| `PgStore` | `pgparty.go` | Per-shard store with transaction management |
| `Store` | `store.go` | Model descriptions and query replacers registry |
| `Model`, `Model58` | `model.go` | Base model with ID, timestamps, soft-delete |
| `UUIDv4`, `UUID58` | `uuid.go` | UUID types (v4 standard, v58 base58-encoded) |
| `XID[T]` | `xid.go` | Generic rs/xid wrapper with type-safe prefixes |
| `NullInt64`, `NullString`, etc. | `null*.go` | Nullable SQL types with JSON serialization |
| `JsonB`, `NullJsonB` | `jsonb.go` | PostgreSQL JSONB type wrappers |
| `Time`, `NullTime` | `time.go`, `nulltime.go` | UTC time types with PostgreSQL compliance |

### Subpackages

| Package | Purpose |
|---------|---------|
| `crud` | NestJS-style CRUD query parser and HTTP response helpers |
| `list` | Paginated list operations |
| `modelcols` | Model column introspection utilities |
| `utils` | Shared utility functions |
| `pg_query` | PostgreSQL parser bindings |

## Building and Running

```bash
# Run all tests (spins up PostgreSQL 13 via Docker)
go test ./...

# Run tests with verbose output
go test -v ./...

# Run a specific test file
go test -v -run TestBasic
```

Tests use `dockertest` to spin up a PostgreSQL 13 container automatically. Docker must be running.

## Coding Conventions

### Struct Tags for Model Mapping

```go
type MyModel struct {
    ID    pgparty.UUID[MyModel]    `json:"id" pk:""`
    Name  pgparty.NullString       `json:"name" sql:"name" len:"255"`
    Count pgparty.NullInt64        `json:"count" defval:"0"`
    Index pgparty.NullString       `json:"index" key:"my_idx" unikey:""`
}

func (MyModel) DatabaseName() string { return "my_models" }
func (MyModel) UUIDPrefix() string   { return "my_model_" }
```

Available struct tags:
- `sql` — column name override
- `store` — custom storage type
- `key` — index name
- `ginkey` — PostgreSQLGIN index
- `len` — column length (VARCHAR)
- `db` — database name override
- `prec` — numeric precision
- `defval` — default value
- `fulltext` — full-text search index
- `unikey` — unique index
- `pk` — primary key (empty string triggers it)

### Null Types Pattern

All nullable types follow the pattern: implement `sql.Scanner`, `driver.Valuer`, `json.Marshaler`, `json.Unmarshaler`, and the `PostgresType()`, `PostgresDefaultValue()`, `PostgresAllowNull()` interface methods for migration support.

### Transaction Pattern

```go
err := pgparty.WithTxInShard(ctx, shardID, func(ctx context.Context) error {
    return pgparty.Replace[MyModel](ctx, model)
})
```

Transactions are context-bound. Cross-shard transactions are not supported — calling `WithTxInShard` with a different shard ID while a transaction is active will return an error.

### Query Pattern with Table Replacement

```go
var results []MyModel
pgparty.Select[MyModel](ctx, "SELECT * FROM &MyModel", &results)
// Expands to: SELECT * FROM schema_name.my_models
```

## Testing

- Tests use `dockertest` with PostgreSQL 13
- `TestMain` in `main_test.go` sets up the Docker container
- Tests run against a real database (no mocking)
- Use `pgparty.WithLoggingQuery(ctx)` to enable query logging in tests
