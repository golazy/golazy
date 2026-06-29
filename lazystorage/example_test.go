package lazystorage_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golazy.dev/lazystorage"
)

func Example() {
	ctx := context.Background()
	dir, err := os.MkdirTemp("", "lazystorage-example-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	storage := lazystorage.NewFilesystem(
		dir,
		lazystorage.WithBaseURL("https://cdn.example.test"),
	)

	info, _, err := storage.Put(ctx, "uploads/hello.txt", strings.NewReader("hello"),
		lazystorage.ContentType{Value: "text/plain"},
	)
	if err != nil {
		panic(err)
	}

	file, _, err := storage.Open(ctx, info.Key)
	if err != nil {
		panic(err)
	}
	body, err := io.ReadAll(file)
	closeErr := file.Close()
	if err != nil {
		panic(err)
	}
	if closeErr != nil {
		panic(closeErr)
	}

	iterator, _, err := storage.List(ctx, "uploads")
	if err != nil {
		panic(err)
	}
	defer iterator.Close()

	var keys []string
	for {
		next, err := iterator.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			panic(err)
		}
		keys = append(keys, next.Key)
	}

	resolved, _, err := storage.URL(ctx, info.Key)
	if err != nil {
		panic(err)
	}

	fmt.Println(info.ContentType, string(body))
	fmt.Println(strings.Join(keys, ","))
	fmt.Println(resolved.String)

	// Output:
	// text/plain hello
	// uploads/hello.txt
	// https://cdn.example.test/uploads/hello.txt
}
