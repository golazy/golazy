// Package lazycode plans source-file changes without writing them.
//
// Format-specific packages create Operations that edit an in-memory Workspace.
// Plan returns baseline-bound edits for a caller to review and apply
// transactionally:
//
//	workspace, err := lazycode.Load(".", "js.toml")
//	if err != nil {
//		return err
//	}
//
//	result, err := workspace.Plan(
//		tomlcode.SetString(
//			"js.toml",
//			"libraries",
//			"turbo",
//			"@hotwired/turbo@8.0.13",
//		),
//	)
//	if err != nil {
//		return err
//	}
//
//	for _, edit := range result.Files {
//		fmt.Printf("%s %s\n", edit.Kind, edit.Path)
//	}
//
// Planning never writes to disk. Before applying an edit, the caller should
// re-read its path and call FileEdit.VerifyBaseline. The caller owns
// confirmation, atomic writes, rollback, and command execution.
package lazycode
