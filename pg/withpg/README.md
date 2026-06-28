# golazy.dev/pg/withpg

`withpg` starts an embedded PostgreSQL server and runs a command with both
`DATABASE_URL` and `GOLAZY_PG_DATABASE_URL` set.

```sh
go run ./withpg/cmd/withpg --version 16 --version 17 -- go test ./...
```

Run the command from the `golazy.dev/pg` module root. Repeating `--version`
runs the command once for each PostgreSQL version.

Tests can use `withpg.Test` to create subtests:

```go
withpg.Test(t, withpg.Config{PgVersions: []string{"16", "17"}}, func(t *testing.T, db withpg.DB) {
	t.Parallel()
	// Use db.URL() or db.Client(t.Context()).
})
```
