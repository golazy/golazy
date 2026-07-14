// Package postgres registers the base PostgreSQL add-on for GoLazy apps.
//
// Most applications install the add-on and its generated wiring from the
// project root:
//
//	lazy add postgres
//
// Applications that construct their add-on selection manually can use:
//
//	app := lazyapp.New(lazyapp.Config{
//		Addons: lazyaddon.Select(postgres.AddonID),
//	})
//
// The add-on reads the connection URL from DATABASE_URL by default. To use a
// different environment variable, configure its name rather than placing the
// connection URL in add-on configuration:
//
//	selection := lazyaddon.Selection{Addons: []lazyaddon.Use{{
//		ID: postgres.AddonID,
//		Config: map[string]string{
//			"database_url_variable": "APP_DATABASE_URL",
//		},
//	}}}
//	app := lazyapp.New(lazyapp.Config{Addons: selection})
//
// During application initialization the add-on opens the pool, adds it to the
// dependency context, and configures the PostgreSQL migration backend. After
// lazyapp.New returns, another selected add-on can obtain the pool with:
//
//	pool, err := lazyaddon.Require(app.Addons, postgres.PoolCapability)
package postgres
