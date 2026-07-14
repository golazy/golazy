// Package gocode provides syntax-aware Go edits as lazycode operations.
//
// For example, an installer can plan a blank import without changing the file
// on disk:
//
//	workspace, err := lazycode.Load(".", "init/app.go")
//	if err != nil {
//		return err
//	}
//
//	_, err = workspace.Plan(
//		gocode.Rewrite("init/app.go",
//			func(_ *token.FileSet, file *ast.File) (bool, error) {
//				return gocode.EnsureBlankImport(
//					file,
//					"example.com/addons/seo",
//				)
//			},
//		),
//	)
//	if err != nil {
//		return err
//	}
//	if result.Changed() {
//		fmt.Println("init/app.go would change")
//	}
//
// Rewrite parses the file with comments, applies the AST change, and formats
// changed output. EnsureFile plans a formatted generated Go sidecar.
package gocode
