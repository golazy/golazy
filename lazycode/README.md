# golazy.dev/lazycode

`golazy.dev/lazycode` plans deterministic source changes without writing
files. It is shared by GoLazy add-on installation, `lazy upgrade`, and future
generators.

The root package provides an immutable-baseline `Workspace`, operations,
diagnostics, SHA-256-bound `FileEdit` values, and a dry-run-friendly `Result`.
Format packages provide bounded editors:

- `gocode`: Go AST rewrites, imports, and formatted generated files.
- `tomlcode`: conservative comment-preserving TOML key and table changes.
- `jsoncode`: structured JSON and package dependency changes.
- `jscode`: exact JavaScript imports and explicitly owned managed blocks.

The module deliberately has no filesystem write or command-execution API. A
caller plans all changes in memory, verifies each baseline immediately before
writing, and owns transaction, rollback, and user-confirmation policy.

```go
workspace, err := lazycode.Load(root, "init/app.go")
if err != nil {
	return err
}

result, err := workspace.Plan(
	gocode.Rewrite("init/app.go", rewriteConfig),
)
if err != nil {
	return err
}

for _, edit := range result.Files {
	// Re-read edit.Path and require edit.VerifyBaseline before applying it.
}
```

The module stays independent of a specific installer. Callers retain ownership
of policy, user confirmation, transactional application, and rollback.
