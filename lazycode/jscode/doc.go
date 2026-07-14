// Package jscode provides deliberately bounded JavaScript text edits as
// lazycode operations.
//
// Exact imports and explicitly owned blocks can be planned together:
//
//	workspace, err := lazycode.Load(".", "app/js/app.js")
//	if err != nil {
//		return err
//	}
//
//	_, err = workspace.Plan(
//		jscode.Import(
//			"app/js/app.js",
//			`import "@hotwired/turbo";`,
//		),
//		jscode.ManagedBlock(
//			"app/js/app.js",
//			"seo",
//			`initializeSEO();`,
//		),
//	)
//	if err != nil {
//		return err
//	}
//	if result.Changed() {
//		fmt.Println("app/js/app.js would change")
//	}
//
// Import manages exact single-line ES module imports. ManagedBlock owns only
// text between matching GoLazy markers. The package is intentionally not a
// general JavaScript parser or rewriter.
package jscode
