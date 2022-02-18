# pgparty
Postgres database/sql access layer with Go generics 1.18+.

You can easily create go structures as models with tag descriptions for tables, fields and indexes in postgres schemas.

`pgparty` implements it in the database, with automatic migration between model descriptions in your service and database. 

We use a special table with model descriptions for migration.
