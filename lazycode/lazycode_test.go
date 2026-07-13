package lazycode_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golazy.dev/lazycode"
)

func TestPlanReturnsStableBaselineBoundEditsWithoutMutatingWorkspace(t *testing.T) {
	workspace, err := lazycode.FromFiles("", map[string][]byte{
		"update.txt": []byte("before"),
		"delete.txt": []byte("delete me"),
	})
	if err != nil {
		t.Fatal(err)
	}

	operation := lazycode.OperationFunc(func(workspace *lazycode.Workspace) error {
		if err := workspace.Replace("update.txt", []byte("after")); err != nil {
			return err
		}
		if err := workspace.Replace("create.txt", []byte("new")); err != nil {
			return err
		}
		if err := workspace.Remove("delete.txt"); err != nil {
			return err
		}
		return workspace.Diagnose(lazycode.Diagnostic{Path: "update.txt", Message: "updated"})
	})

	first, err := workspace.Plan(operation)
	if err != nil {
		t.Fatal(err)
	}
	second, err := workspace.Plan(operation)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Files) != 3 || len(second.Files) != 3 {
		t.Fatalf("edits = %d and %d, want 3", len(first.Files), len(second.Files))
	}
	if first.Files[0].Path != "create.txt" || first.Files[0].Kind != lazycode.EditCreate || first.Files[0].BaselineHash != lazycode.AbsentHash {
		t.Fatalf("create edit = %#v", first.Files[0])
	}
	if first.Files[1].Path != "delete.txt" || first.Files[1].Kind != lazycode.EditDelete || !first.Files[1].VerifyBaseline([]byte("delete me"), true) {
		t.Fatalf("delete edit = %#v", first.Files[1])
	}
	if first.Files[2].Path != "update.txt" || first.Files[2].Kind != lazycode.EditUpdate || !first.Files[2].VerifyBaseline([]byte("before"), true) {
		t.Fatalf("update edit = %#v", first.Files[2])
	}
	if first.Files[2].VerifyBaseline([]byte("drifted"), true) {
		t.Fatal("drifted baseline verified")
	}
	data, err := workspace.Read("update.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "before" {
		t.Fatalf("workspace mutated to %q", data)
	}
	if len(first.Diagnostics) != 1 || first.Diagnostics[0].Severity != lazycode.SeverityInfo {
		t.Fatalf("diagnostics = %#v", first.Diagnostics)
	}
}

func TestLoadAndPlanNeverWriteFiles(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config.txt")
	if err := os.WriteFile(path, []byte("disk"), 0o644); err != nil {
		t.Fatal(err)
	}
	workspace, err := lazycode.Load(root, "config.txt")
	if err != nil {
		t.Fatal(err)
	}
	result, err := workspace.Plan(lazycode.OperationFunc(func(workspace *lazycode.Workspace) error {
		return workspace.Replace("config.txt", []byte("planned"))
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 || string(result.Files[0].After) != "planned" {
		t.Fatalf("result = %#v", result)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "disk" {
		t.Fatalf("disk content = %q", data)
	}
}

func TestPlanRejectsInvalidPathsAndDoesNotReturnPartialResults(t *testing.T) {
	workspace := lazycode.New("")
	_, err := workspace.Plan(lazycode.OperationFunc(func(workspace *lazycode.Workspace) error {
		if err := workspace.Replace("ok.txt", []byte("partial")); err != nil {
			return err
		}
		return errors.New("stop")
	}))
	if err == nil {
		t.Fatal("Plan error = nil")
	}
	if workspace.Exists("ok.txt") {
		t.Fatal("failed plan mutated original workspace")
	}
	if err := workspace.Replace("../escape", nil); err == nil {
		t.Fatal("Replace invalid path error = nil")
	}
}
