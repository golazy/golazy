// Package pg contains PostgreSQL helpers shared by concrete GoLazy
// PostgreSQL backends and applications.
//
// Most GoLazy applications install the conventional pool and migration wiring
// with:
//
//	lazy add postgres
//
// Applications with custom lifecycle requirements can open and attach the
// application-owned pool directly:
//
//	pool, err := pg.OpenEnv(ctx)
//	if err != nil {
//		return err
//	}
//	defer pool.Close()
//	ctx = pg.WithPool(ctx, pool)
//
// Backend packages such as pgauth, pgmigrate, pgjobs, and other pg/*
// implementations can then share that pool with pg.FromContext.
package pg
