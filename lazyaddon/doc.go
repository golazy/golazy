// Package lazyaddon provides manifest-driven, per-application add-on
// selection, typed lifecycle hooks, and typed capabilities.
//
// Add-on packages normally register their manifest during package
// initialization. A lifecycle owner defines a versioned hook, and callbacks
// run only for application scopes that select their owning add-on:
//
//	type Routes struct {
//		Names []string
//	}
//
//	routes := lazyaddon.DefineHook[Routes]("example.com/app/routes", 1)
//	seo := lazyaddon.MustRegisterDefinition(lazyaddon.Definition{
//		ID:      "seo",
//		Version: "v1.0.0",
//	})
//
//	lazyaddon.MustOn(seo, routes, lazyaddon.CallbackOptions{ID: "routes"},
//		func(event *Routes) error {
//			event.Names = append(event.Names, "sitemap")
//			return nil
//		})
//
//	scope, err := lazyaddon.Resolve(lazyaddon.Select("seo"))
//	if err != nil {
//		return err
//	}
//	event := Routes{}
//	if err := lazyaddon.Run(scope, routes, &event); err != nil {
//		return err
//	}
//
// Resolved scopes are immutable and application-local. Capabilities exchange
// typed, versioned values between selected add-ons without process-global
// state. Use.Config is intended only for committed, non-secret configuration.
// Normal GoLazy applications let lazy add generate the imports and
// lazyapp.Config.Addons selection instead of assembling this wiring manually.
package lazyaddon
