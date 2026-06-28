# golazy.dev/pg

`golazy.dev/pg` contains PostgreSQL implementations for GoLazy framework
packages. The module keeps PostgreSQL driver dependencies out of the core
`golazy.dev` module.

Current packages:

- `pg`: shared `pgxpool` connection helpers.
- `pgmigrate`: `lazymigrate.Backend` for PostgreSQL. Migrations use
  `-- +lazy Up` and `-- +lazy Down` sections.
- `pgjobs`: PostgreSQL `lazyjobs.Backend` plus embedded lazy job migrations.
- `withpg`: embedded PostgreSQL helper for local integration tests.

Reserved packages:

- `pgfiles`
- `pgstorage`
- `pgassets`
- `pgmigrations`

Integration tests read `GOLAZY_PG_DATABASE_URL`. The `golazy.dev/pg/withpg`
helper can run those tests against one or more PostgreSQL versions:

```sh
go run ./withpg/cmd/withpg --version 16 --version 17 -- go test ./...
```

Tests can also use `withpg.Test` to create one subtest per configured
PostgreSQL version.
