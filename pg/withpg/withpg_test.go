package withpg

import (
	"context"
	"strings"
	"testing"
)

func TestRunConfigsDefaultsToPostgres18(t *testing.T) {
	configs, err := runConfigs(Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(configs) != 1 {
		t.Fatalf("len(configs) = %d, want 1", len(configs))
	}
	if configs[0].PgVersion != "18" {
		t.Fatalf("PgVersion = %q, want 18", configs[0].PgVersion)
	}
}

func TestRunConfigsExpandsPgVersions(t *testing.T) {
	configs, err := runConfigs(Config{PgVersions: []string{"16", "17"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(configs) != 2 {
		t.Fatalf("len(configs) = %d, want 2", len(configs))
	}
	for index, want := range []string{"16", "17"} {
		if configs[index].PgVersion != want {
			t.Fatalf("configs[%d].PgVersion = %q, want %q", index, configs[index].PgVersion, want)
		}
		if len(configs[index].PgVersions) != 0 {
			t.Fatalf("configs[%d].PgVersions = %#v, want empty", index, configs[index].PgVersions)
		}
	}
}

func TestRunConfigsRejectsPgVersionAndPgVersions(t *testing.T) {
	_, err := runConfigs(Config{PgVersion: "16", PgVersions: []string{"17"}})
	if err == nil {
		t.Fatal("runConfigs succeeded, want error")
	}
	if !strings.Contains(err.Error(), "set either PgVersion or PgVersions") {
		t.Fatalf("error = %v", err)
	}
}

func TestStartRejectsPgVersions(t *testing.T) {
	db, err := Start(context.Background(), Config{PgVersions: []string{"16", "17"}})
	if err == nil {
		t.Fatal("Start succeeded, want error")
	}
	if db != nil {
		t.Fatalf("db = %#v, want nil", db)
	}
	if !strings.Contains(err.Error(), "Start does not accept PgVersions") {
		t.Fatalf("error = %v", err)
	}
}
