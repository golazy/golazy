// Package tomlcode provides conservative, comment-preserving TOML edits as
// lazycode operations.
//
// A typed operation can be combined with other edits in one lazycode plan:
//
//	workspace, err := lazycode.Load(".", "lazy.toml")
//	if err != nil {
//		return err
//	}
//
//	_, err = workspace.Plan(
//		tomlcode.SetString(
//			"lazy.toml",
//			"database",
//			"url_variable",
//			"DATABASE_URL",
//		),
//	)
//	if err != nil {
//		return err
//	}
//	if result.Changed() {
//		fmt.Println("lazy.toml would change")
//	}
//
// The editor preserves unrelated text and comments. It supports ordinary
// tables and single keys; ambiguous or unsupported TOML returns an error
// instead of causing a broad reformat.
package tomlcode
