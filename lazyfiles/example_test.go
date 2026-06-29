package lazyfiles_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golazy.dev/lazyfiles"
	"golazy.dev/lazyfiles/jsonl"
	"golazy.dev/lazystorage"
)

func Example() {
	ctx := context.Background()

	dir, err := os.MkdirTemp("", "lazyfiles-example-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	repo, err := jsonl.New(filepath.Join(dir, "files.jsonl"))
	if err != nil {
		log.Fatal(err)
	}
	files := &lazyfiles.Files{
		Repository: repo,
		Storages: map[string]lazystorage.Storage{
			"local": lazystorage.NewFilesystem(filepath.Join(dir, "objects")),
		},
		DefaultStorage: "local",
	}

	file, _, err := files.Put(
		ctx,
		strings.NewReader("hello files"),
		lazyfiles.Filename{Name: "hello.txt"},
		lazystorage.ContentType{Value: "text/plain"},
	)
	if err != nil {
		log.Fatal(err)
	}

	opened, catalog, _, err := files.Open(ctx, file.ID)
	if err != nil {
		log.Fatal(err)
	}
	defer opened.Close()

	body, err := io.ReadAll(opened)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(catalog.Filename)
	fmt.Println(catalog.ContentType)
	fmt.Println(string(body))

	// Output:
	// hello.txt
	// text/plain
	// hello files
}
