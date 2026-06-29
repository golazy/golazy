package lazymedia_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"golazy.dev/lazymedia"
	"golazy.dev/lazystorage"
)

func ExampleMedia() {
	ctx := context.Background()
	files := newExampleFileStore()
	source := files.add("message.txt", "hello", "text/plain")

	media := &lazymedia.Media{
		Files:      files,
		Repository: newExampleRepository(),
		Processor: lazymedia.ProcessorFunc(func(_ context.Context, source lazymedia.Source, request lazymedia.Request, options ...any) (lazymedia.Result, []any, error) {
			data, err := io.ReadAll(source.Body)
			if err != nil {
				return lazymedia.Result{}, options, err
			}
			return lazymedia.Result{
				Body:        strings.NewReader(strings.ToUpper(string(data))),
				ContentType: source.File.ContentType,
				Filename:    request.VariantKey + "-" + source.File.Filename,
			}, options, nil
		}),
	}

	file, _, err := media.Variant(ctx, lazymedia.Request{
		SourceFileID: source.ID,
		VariantKey:   "preview",
	})
	if err != nil {
		panic(err)
	}
	url, _, err := media.URL(ctx, lazymedia.Request{
		SourceFileID: source.ID,
		VariantKey:   "preview",
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(file.Filename)
	fmt.Println(url)
	fmt.Println(files.body(file.ID))

	// Output:
	// preview-message.txt
	// /files/file-2
	// HELLO
}

type exampleFile struct {
	lazymedia.File
	body string
}

type exampleFileStore struct {
	files map[string]exampleFile
	next  int
}

func newExampleFileStore() *exampleFileStore {
	return &exampleFileStore{files: map[string]exampleFile{}}
}

func (store *exampleFileStore) add(filename, body, contentType string) lazymedia.File {
	store.next++
	file := lazymedia.File{
		ID:          fmt.Sprintf("file-%d", store.next),
		Filename:    filename,
		ContentType: contentType,
		Size:        int64(len(body)),
	}
	store.files[file.ID] = exampleFile{File: file, body: body}
	return file
}

func (store *exampleFileStore) Open(_ context.Context, id string, options ...any) (io.ReadCloser, lazymedia.File, []any, error) {
	file, ok := store.files[id]
	if !ok {
		return nil, lazymedia.File{}, options, fmt.Errorf("file %q not found", id)
	}
	return io.NopCloser(strings.NewReader(file.body)), file.File, options, nil
}

func (store *exampleFileStore) Put(_ context.Context, body io.Reader, options ...any) (lazymedia.File, []any, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return lazymedia.File{}, options, err
	}
	contentType, options, _ := lazystorage.Take[lazystorage.ContentType](options)
	filename, options, _ := lazystorage.Take[lazymedia.OutputFilename](options)
	store.next++
	file := lazymedia.File{
		ID:          fmt.Sprintf("file-%d", store.next),
		Filename:    filename.Name,
		ContentType: contentType.Value,
		Size:        int64(len(data)),
	}
	store.files[file.ID] = exampleFile{File: file, body: string(bytes.Clone(data))}
	return file, options, nil
}

func (store *exampleFileStore) URL(_ context.Context, id string, options ...any) (string, []any, error) {
	if _, ok := store.files[id]; !ok {
		return "", options, fmt.Errorf("file %q not found", id)
	}
	return "/files/" + id, options, nil
}

func (store *exampleFileStore) body(id string) string {
	return store.files[id].body
}

type exampleRepository struct {
	variants map[string]lazymedia.Variant
}

func newExampleRepository() *exampleRepository {
	return &exampleRepository{variants: map[string]lazymedia.Variant{}}
}

func (repo *exampleRepository) FindVariant(_ context.Context, sourceFileID, variantKey string, options ...any) (lazymedia.Variant, []any, error) {
	variant, ok := repo.variants[sourceFileID+"\x00"+variantKey]
	if !ok {
		return lazymedia.Variant{}, options, fmt.Errorf("variant %q/%q not found", sourceFileID, variantKey)
	}
	return variant, options, nil
}

func (repo *exampleRepository) SaveVariant(_ context.Context, variant lazymedia.Variant, options ...any) (lazymedia.Variant, []any, error) {
	if variant.Status == "" {
		variant.Status = lazymedia.StatusReady
	}
	repo.variants[variant.SourceFileID+"\x00"+variant.VariantKey] = variant
	return variant, options, nil
}

func (repo *exampleRepository) DeleteVariant(_ context.Context, sourceFileID, variantKey string, options ...any) ([]any, error) {
	delete(repo.variants, sourceFileID+"\x00"+variantKey)
	return options, nil
}
