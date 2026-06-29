package lazymigrate_test

import (
	"context"
	"fmt"
	"testing/fstest"

	"golazy.dev/lazymigrate"
	"golazy.dev/lazymigrate/fakemigrator"
)

func Example() {
	ctx := context.Background()
	files := fstest.MapFS{
		"migrations/postgres/202606280001_create_documents.sql": {
			Data: []byte("-- +lazy Up\nCREATE TABLE documents (id bigserial);\n-- +lazy Down\nDROP TABLE documents;\n"),
		},
	}

	backend := fakemigrator.New()
	migrator, err := lazymigrate.New(lazymigrate.Config{
		Backend: backend,
		Sources: []lazymigrate.Source{
			lazymigrate.ForDatabase(files, "postgres"),
		},
	})
	if err != nil {
		panic(err)
	}
	plan, err := migrator.Up(ctx, 0)
	if err != nil {
		panic(err)
	}

	fmt.Println(plan.Steps[0].Migration.ID)
	fmt.Println(plan.Steps[0].Direction)
	// Output:
	// 202606280001_create_documents
	// up
}
