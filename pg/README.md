# golazy.dev/pg

`golazy.dev/pg` contains PostgreSQL implementations for GoLazy framework
packages. The module keeps PostgreSQL driver dependencies out of the core
`golazy.dev` module.

Current packages:

- `pg`: shared `pgxpool` connection and app-context helpers.
- `pgauth`: PostgreSQL `lazyauth.Authenticator` plus embedded auth user
  migrations.
- `pgfiles`: PostgreSQL `lazyfiles.Repository` plus embedded lazy file
  catalog migrations.
- `pgmedia`: PostgreSQL `lazymedia.Repository` plus embedded lazy media
  variant migrations.
- `pgmigrate`: `lazymigrate.Backend` for PostgreSQL. Migrations use
  `-- +lazy Up` and `-- +lazy Down` sections, run under a PostgreSQL advisory
  lock, and treat stale same-checksum concurrent steps as no-ops.
- `pgjobs`: PostgreSQL `lazyjobs.Backend` plus embedded lazy job migrations.
- `pgstorage`: PostgreSQL `lazystorage` backend for object reads, writes,
  deletes, and listing, including storage targets for `lazyassets.Upload`.
- `withpg`: embedded PostgreSQL helper for local integration tests.

Integration tests read `GOLAZY_PG_DATABASE_URL`. The `golazy.dev/pg/withpg`
helper can run those tests against one or more PostgreSQL versions:

```sh
go run ./withpg/cmd/withpg --version 16 --version 17 -- go test ./...
```

Tests can also use `withpg.Test` to create one subtest per configured
PostgreSQL version.

Applications that open a pool during GoLazy dependency initialization can add
it to the app context with `pg.WithPool(ctx, pool)` and read it later with
`pg.FromContext(ctx)`. Package migrations such as `pgauth.Migrations()` should
be registered beside app-owned migrations before the corresponding backend is
used.
