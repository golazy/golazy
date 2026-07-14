// Package jsoncode provides structured JSON object edits as lazycode
// operations.
//
// For example, an installer can plan a package dependency update:
//
//	workspace, err := lazycode.Load(".", "package.json")
//	if err != nil {
//		return err
//	}
//
//	_, err = workspace.Plan(
//		jsoncode.Dependency(
//			"package.json",
//			"dependencies",
//			"@hotwired/turbo",
//			"8.0.13",
//		),
//	)
//	if err != nil {
//		return err
//	}
//	if result.Changed() {
//		fmt.Println("package.json would change")
//	}
//
// JSON indentation and the final newline are preserved. Object key order is
// normalized through encoding/json.
package jsoncode
