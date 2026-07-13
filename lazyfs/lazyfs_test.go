package lazyfs_test

import (
	"errors"
	"io"
	"io/fs"
	"reflect"
	"sort"
	"sync"
	"testing"
	"testing/fstest"

	"golazy.dev/lazyfs"
)

func TestLayeredFilesAndDirectories(t *testing.T) {
	files := lazyfs.New()
	if err := files.Add(fstest.MapFS{
		"shared.txt":        {Data: []byte("framework")},
		"framework.txt":     {Data: []byte("framework")},
		"templates/base":    {Data: []byte("framework base")},
		"templates/shared":  {Data: []byte("framework shared")},
		"file-to-dir":       {Data: []byte("framework file")},
		"dir-to-file/child": {Data: []byte("framework child")},
	}, lazyfs.Name("framework"), lazyfs.Owner("golazy.dev")); err != nil {
		t.Fatalf("Add(framework) error = %v", err)
	}
	if err := files.Add(fstest.MapFS{
		"shared.txt":        {Data: []byte("application")},
		"application.txt":   {Data: []byte("application")},
		"templates/page":    {Data: []byte("application page")},
		"templates/shared":  {Data: []byte("application shared")},
		"file-to-dir/child": {Data: []byte("application child")},
		"dir-to-file":       {Data: []byte("application file")},
	}, lazyfs.Name("application")); err != nil {
		t.Fatalf("Add(application) error = %v", err)
	}
	if err := files.Seal(); err != nil {
		t.Fatalf("Seal() error = %v", err)
	}

	assertContent(t, files, "shared.txt", "application")
	assertContent(t, files, "framework.txt", "framework")
	assertContent(t, files, "application.txt", "application")
	assertContent(t, files, "templates/shared", "application shared")
	assertContent(t, files, "dir-to-file", "application file")

	if got, want := entryNames(t, files, "templates"), []string{"base", "page", "shared"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadDir(templates) = %v, want %v", got, want)
	}
	if got, want := entryNames(t, files, "file-to-dir"), []string{"child"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadDir(file-to-dir) = %v, want %v", got, want)
	}
	if _, err := fs.ReadDir(files, "dir-to-file"); err == nil {
		t.Fatal("ReadDir(dir-to-file) error = nil, want file error")
	}

	root, err := files.Open(".")
	if err != nil {
		t.Fatalf("Open(.) error = %v", err)
	}
	defer root.Close()
	directory, ok := root.(fs.ReadDirFile)
	if !ok {
		t.Fatalf("Open(.) type = %T, want fs.ReadDirFile", root)
	}
	var openedNames []string
	for {
		entries, err := directory.ReadDir(2)
		for _, entry := range entries {
			openedNames = append(openedNames, entry.Name())
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("ReadDir(2) error = %v", err)
		}
	}
	if !sort.StringsAreSorted(openedNames) {
		t.Fatalf("opened root names are not sorted: %v", openedNames)
	}

	matches, err := fs.Glob(files, "templates/*")
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if want := []string{"templates/base", "templates/page", "templates/shared"}; !reflect.DeepEqual(matches, want) {
		t.Fatalf("Glob() = %v, want %v", matches, want)
	}

	if err := fstest.TestFS(files,
		"shared.txt",
		"framework.txt",
		"application.txt",
		"templates/base",
		"templates/page",
		"templates/shared",
		"file-to-dir/child",
		"dir-to-file",
	); err != nil {
		t.Fatalf("fstest.TestFS() error = %v", err)
	}
}

func TestHigherFileHidesLowerDescendants(t *testing.T) {
	files := lazyfs.New()
	if err := files.Add(fstest.MapFS{
		"foo/bar.txt":       {Data: []byte("lower bar")},
		"foo/nested/value":  {Data: []byte("lower nested")},
		"visible/lower.txt": {Data: []byte("lower visible")},
	}, lazyfs.Name("lower")); err != nil {
		t.Fatal(err)
	}
	if err := files.Add(fstest.MapFS{
		"foo":               {Data: []byte("higher file")},
		"visible/upper.txt": {Data: []byte("upper visible")},
	}, lazyfs.Name("higher")); err != nil {
		t.Fatal(err)
	}

	assertContent(t, files, "foo", "higher file")
	for _, name := range []string{"foo/bar.txt", "foo/nested", "foo/nested/value"} {
		assertHiddenPath(t, files, name)
		if _, err := files.Resolve(name); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("Resolve(%q) error = %v, want fs.ErrNotExist", name, err)
		}
	}
	if _, err := fs.ReadDir(files, "foo"); err == nil {
		t.Fatal("ReadDir(foo) error = nil, want higher file to win")
	}
	if _, err := fs.Sub(files, "foo"); !errors.Is(err, fs.ErrInvalid) {
		t.Fatalf("Sub(foo) error = %v, want fs.ErrInvalid", err)
	}
	if _, err := fs.Sub(files, "foo/nested"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Sub(foo/nested) error = %v, want fs.ErrNotExist", err)
	}

	for _, pattern := range []string{"foo/*", "foo/nested/*", "*/*.txt"} {
		matches, err := fs.Glob(files, pattern)
		if err != nil {
			t.Fatalf("Glob(%q) error = %v", pattern, err)
		}
		if pattern == "*/*.txt" {
			if got, want := matches, []string{"visible/lower.txt", "visible/upper.txt"}; !reflect.DeepEqual(got, want) {
				t.Fatalf("Glob(%q) = %v, want %v", pattern, got, want)
			}
			continue
		}
		if len(matches) != 0 {
			t.Fatalf("Glob(%q) = %v, want no hidden descendants", pattern, matches)
		}
	}
}

func TestDirectoryMergeStopsAtLowerFileBarrier(t *testing.T) {
	files := lazyfs.New()
	if err := files.Add(fstest.MapFS{
		"foo/lower.txt": {Data: []byte("lower")},
	}, lazyfs.Name("lower")); err != nil {
		t.Fatal(err)
	}
	if err := files.Add(fstest.MapFS{
		"foo": {Data: []byte("barrier")},
	}, lazyfs.Name("barrier")); err != nil {
		t.Fatal(err)
	}
	if err := files.Add(fstest.MapFS{
		"foo/higher.txt": {Data: []byte("higher")},
	}, lazyfs.Name("higher")); err != nil {
		t.Fatal(err)
	}

	if got, want := entryNames(t, files, "foo"), []string{"higher.txt"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadDir(foo) = %v, want %v", got, want)
	}
	assertContent(t, files, "foo/higher.txt", "higher")
	assertHiddenPath(t, files, "foo/lower.txt")
	if _, err := files.Resolve("foo/lower.txt"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Resolve(foo/lower.txt) error = %v, want fs.ErrNotExist", err)
	}

	opened, err := files.Open("foo")
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	directory, ok := opened.(fs.ReadDirFile)
	if !ok {
		t.Fatalf("Open(foo) type = %T, want fs.ReadDirFile", opened)
	}
	entries, err := directory.ReadDir(-1)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := dirEntryNames(entries), []string{"higher.txt"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Open(foo).ReadDir(-1) = %v, want %v", got, want)
	}

	matches, err := fs.Glob(files, "foo/*")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := matches, []string{"foo/higher.txt"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Glob(foo/*) = %v, want %v", got, want)
	}

	subFS, err := fs.Sub(files, "foo")
	if err != nil {
		t.Fatal(err)
	}
	sub, ok := subFS.(*lazyfs.FS)
	if !ok {
		t.Fatalf("Sub(foo) type = %T, want *lazyfs.FS", subFS)
	}
	if got, want := layerNames(sub.Layers()), []string{"higher"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("sub layers = %v, want %v", got, want)
	}
	if got, want := entryNames(t, sub, "."), []string{"higher.txt"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("sub entries = %v, want %v", got, want)
	}
}

func TestResolveAndSubPreserveProvenance(t *testing.T) {
	files := lazyfs.New()
	if err := files.Add(fstest.MapFS{
		"views/shared": {Data: []byte("framework")},
		"views/base":   {Data: []byte("base")},
	}, lazyfs.Name("framework"), lazyfs.Owner("core")); err != nil {
		t.Fatal(err)
	}
	if err := files.Add(fstest.MapFS{
		"views/shared": {Data: []byte("addon")},
	}, lazyfs.Name("seo"), lazyfs.Owner("golazy/seo")); err != nil {
		t.Fatal(err)
	}
	if err := files.Seal(); err != nil {
		t.Fatal(err)
	}

	resolution, err := files.Resolve("views/shared")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolution.Winner.Layer.Name != "seo" {
		t.Fatalf("winner = %q, want seo", resolution.Winner.Layer.Name)
	}
	if got, want := candidateNames(resolution.Candidates), []string{"seo", "framework"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("candidate names = %v, want %v", got, want)
	}

	subFS, err := fs.Sub(files, "views")
	if err != nil {
		t.Fatalf("Sub(views) error = %v", err)
	}
	sub, ok := subFS.(*lazyfs.FS)
	if !ok {
		t.Fatalf("Sub(views) type = %T, want *lazyfs.FS", subFS)
	}
	if !sub.Sealed() {
		t.Fatal("sub filesystem is not sealed")
	}
	if got, want := layerNames(sub.Layers()), []string{"framework", "seo"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("sub layers = %v, want %v", got, want)
	}
	assertContent(t, sub, "shared", "addon")
	assertContent(t, sub, "base", "base")
}

func TestConcurrentAddAndSeal(t *testing.T) {
	files := lazyfs.New()
	const count = 32
	var wait sync.WaitGroup
	for index := range count {
		wait.Add(1)
		go func() {
			defer wait.Done()
			name := "layer-" + string(rune('a'+index))
			path := name + ".txt"
			if err := files.Add(fstest.MapFS{path: {Data: []byte(path)}}, lazyfs.Name(name)); err != nil {
				t.Errorf("Add(%s) error = %v", name, err)
			}
		}()
	}
	wait.Wait()
	if err := files.Seal(); err != nil {
		t.Fatalf("Seal() error = %v", err)
	}
	if err := files.Seal(); err != nil {
		t.Fatalf("second Seal() error = %v", err)
	}
	if !files.Sealed() {
		t.Fatal("Sealed() = false")
	}
	if got := len(files.Layers()); got != count {
		t.Fatalf("len(Layers()) = %d, want %d", got, count)
	}
	if err := files.Add(fstest.MapFS{"late": {Data: []byte("late")}}); !errors.Is(err, lazyfs.ErrSealed) {
		t.Fatalf("Add() after Seal error = %v, want ErrSealed", err)
	}
}

func TestAddValidationAndFallbackErrors(t *testing.T) {
	files := lazyfs.New()
	if err := files.Add(nil); err == nil {
		t.Fatal("Add(nil) error = nil")
	}
	if err := files.Add(fstest.MapFS{"value": {Data: []byte("low")}}, lazyfs.Name("same")); err != nil {
		t.Fatal(err)
	}
	if err := files.Add(fstest.MapFS{"other": {Data: []byte("high")}}, lazyfs.Name("same")); err == nil {
		t.Fatal("Add() duplicate name error = nil")
	}
	if err := files.Add(permissionFS{}); err != nil {
		t.Fatal(err)
	}
	if _, err := files.Open("value"); !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("Open(value) error = %v, want permission error", err)
	}
	if _, err := files.Open("../invalid"); !errors.Is(err, fs.ErrInvalid) {
		t.Fatalf("Open(invalid) error = %v, want invalid path", err)
	}
}

func TestMountExposesNestedPrefixWithoutCopying(t *testing.T) {
	mounted, err := lazyfs.Mount("postgres/jobs", fstest.MapFS{
		"202607010101_jobs.sql": {Data: []byte("jobs")},
		"nested/readme.txt":     {Data: []byte("readme")},
	})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	if got, want := entryNames(t, mounted, "."), []string{"postgres"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("root entries = %v, want %v", got, want)
	}
	if got, want := entryNames(t, mounted, "postgres"), []string{"jobs"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("postgres entries = %v, want %v", got, want)
	}
	if got, want := entryNames(t, mounted, "postgres/jobs"), []string{"202607010101_jobs.sql", "nested"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("jobs entries = %v, want %v", got, want)
	}
	assertContent(t, mounted, "postgres/jobs/202607010101_jobs.sql", "jobs")
	info, err := fs.Stat(mounted, "postgres/jobs")
	if err != nil {
		t.Fatalf("Stat(postgres/jobs) error = %v", err)
	}
	if got, want := info.Name(), "jobs"; got != want {
		t.Fatalf("mounted root name = %q, want %q", got, want)
	}

	if err := fstest.TestFS(mounted,
		"postgres/jobs/202607010101_jobs.sql",
		"postgres/jobs/nested/readme.txt",
	); err != nil {
		t.Fatalf("fstest.TestFS() error = %v", err)
	}
}

func TestMountedFilesystemsMergeAsLayers(t *testing.T) {
	app, err := lazyfs.Mount("postgres/app", fstest.MapFS{
		"202607010101_app.sql": {Data: []byte("app")},
	})
	if err != nil {
		t.Fatal(err)
	}
	jobs, err := lazyfs.Mount("postgres/jobs", fstest.MapFS{
		"202607010102_jobs.sql": {Data: []byte("jobs")},
	})
	if err != nil {
		t.Fatal(err)
	}
	files := lazyfs.New()
	if err := files.Add(app, lazyfs.Name("app")); err != nil {
		t.Fatal(err)
	}
	if err := files.Add(jobs, lazyfs.Name("jobs")); err != nil {
		t.Fatal(err)
	}
	if got, want := entryNames(t, files, "postgres"), []string{"app", "jobs"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("postgres entries = %v, want %v", got, want)
	}
}

type permissionFS struct{}

func (permissionFS) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
}

func assertContent(t *testing.T, files fs.FS, name string, want string) {
	t.Helper()
	content, err := fs.ReadFile(files, name)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", name, err)
	}
	if got := string(content); got != want {
		t.Fatalf("ReadFile(%q) = %q, want %q", name, got, want)
	}
}

func assertHiddenPath(t *testing.T, files *lazyfs.FS, name string) {
	t.Helper()
	if _, err := files.Open(name); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Open(%q) error = %v, want fs.ErrNotExist", name, err)
	}
	if _, err := fs.ReadFile(files, name); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("ReadFile(%q) error = %v, want fs.ErrNotExist", name, err)
	}
	if _, err := fs.Stat(files, name); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Stat(%q) error = %v, want fs.ErrNotExist", name, err)
	}
	if _, err := fs.ReadDir(files, name); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("ReadDir(%q) error = %v, want fs.ErrNotExist", name, err)
	}
}

func entryNames(t *testing.T, files fs.FS, name string) []string {
	t.Helper()
	entries, err := fs.ReadDir(files, name)
	if err != nil {
		t.Fatalf("ReadDir(%q) error = %v", name, err)
	}
	return dirEntryNames(entries)
}

func dirEntryNames(entries []fs.DirEntry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

func candidateNames(candidates []lazyfs.Candidate) []string {
	names := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		names = append(names, candidate.Layer.Name)
	}
	return names
}

func layerNames(layers []lazyfs.Layer) []string {
	names := make([]string, 0, len(layers))
	for _, layer := range layers {
		names = append(names, layer.Name)
	}
	return names
}
