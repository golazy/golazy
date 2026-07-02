package assetmigrate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"sync"
	"testing"
	"time"

	"golazy.dev/lazyassets"
	"golazy.dev/lazycache"
	"golazy.dev/lazymigrate"
	"golazy.dev/lazystorage"
)

func TestBackendUploadsAssetsAndSkipsAlreadyDone(t *testing.T) {
	ctx, registry := testContext(t)
	storage := newMemoryStorage()
	migrator, id := testMigrator(t, ctx, Config{
		Registry:        registry,
		Storage:         storage,
		Mode:            lazyassets.UnpackBoth,
		WriteManifest:   true,
		PostDoneRefresh: time.Millisecond,
	})

	if _, err := migrator.Up(ctx, 0); err != nil {
		t.Fatal(err)
	}
	if !storage.exists(".migrations/" + id + "-done") {
		t.Fatalf("done marker was not written")
	}
	if !storage.exists("styles.css") {
		t.Fatalf("logical asset was not uploaded")
	}
	permanent, err := registry.Path("/styles.css")
	if err != nil {
		t.Fatal(err)
	}
	if !storage.exists(strings.TrimPrefix(permanent, "/")) {
		t.Fatalf("permanent asset %s was not uploaded", permanent)
	}
	firstPuts := storage.totalPuts()

	if _, err := migrator.Up(ctx, 0); err != nil {
		t.Fatal(err)
	}
	if got := storage.totalPuts(); got != firstPuts {
		t.Fatalf("puts after already-done migration = %d, want %d", got, firstPuts)
	}
}

func TestBackendWaitsForFreshLease(t *testing.T) {
	ctx, registry := testContext(t)
	storage := newMemoryStorage()
	migrator, id := testMigrator(t, ctx, Config{
		Registry:       registry,
		Storage:        storage,
		WaitMin:        5 * time.Millisecond,
		WaitMax:        5 * time.Millisecond,
		StaleAfter:     time.Hour,
		HeartbeatEvery: time.Hour,
	})
	storage.writeObject(".migrations/"+id, marker{
		metadata:  storage.metadataFor(t, migrator, ctx),
		Owner:     "other",
		StartedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	})
	waitCtx, cancel := context.WithTimeout(ctx, 15*time.Millisecond)
	defer cancel()

	_, err := migrator.Up(waitCtx, 0)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Up error = %v, want context deadline while waiting on fresh lease", err)
	}
	if storage.exists("styles.css") {
		t.Fatalf("asset uploaded while fresh lease was held")
	}
}

func TestBackendTakesStaleLeaseByETag(t *testing.T) {
	ctx, registry := testContext(t)
	storage := newMemoryStorage()
	migrator, id := testMigrator(t, ctx, Config{
		Registry:        registry,
		Storage:         storage,
		StaleAfter:      time.Millisecond,
		WaitMin:         time.Millisecond,
		WaitMax:         time.Millisecond,
		HeartbeatEvery:  time.Hour,
		PostDoneRefresh: time.Millisecond,
	})
	storage.writeObject(".migrations/"+id, marker{
		metadata:  storage.metadataFor(t, migrator, ctx),
		Owner:     "other",
		StartedAt: time.Now().Add(-time.Hour).UTC(),
		UpdatedAt: time.Now().Add(-time.Hour).UTC(),
	})

	if _, err := migrator.Up(ctx, 0); err != nil {
		t.Fatal(err)
	}
	if !storage.exists(".migrations/" + id + "-done") {
		t.Fatalf("done marker was not written after stale lease takeover")
	}
}

func TestBackendHeartbeatRefreshesLeaseDuringUpload(t *testing.T) {
	ctx, registry := testContext(t)
	storage := newMemoryStorage()
	storage.beforePut = func(ctx context.Context, key string) error {
		if key == "styles.css" {
			timer := time.NewTimer(20 * time.Millisecond)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
			}
		}
		return nil
	}
	migrator, id := testMigrator(t, ctx, Config{
		Registry:        registry,
		Storage:         storage,
		Mode:            lazyassets.UnpackBoth,
		HeartbeatEvery:  5 * time.Millisecond,
		PostDoneRefresh: time.Millisecond,
	})

	if _, err := migrator.Up(ctx, 0); err != nil {
		t.Fatal(err)
	}
	if puts := storage.puts(".migrations/" + id); puts < 2 {
		t.Fatalf("lease marker puts = %d, want heartbeat refresh", puts)
	}
}

func TestBackendErrorsWhenHeartbeatLosesLease(t *testing.T) {
	ctx, registry := testContext(t)
	storage := newMemoryStorage()
	leaseHijacked := make(chan struct{})
	storage.beforePut = func(ctx context.Context, key string) error {
		if key != "styles.css" {
			return nil
		}
		storage.hijack(".migrations/" + storage.idFor(t, registry, ctx))
		close(leaseHijacked)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
			return nil
		}
	}
	migrator, _ := testMigrator(t, ctx, Config{
		Registry:       registry,
		Storage:        storage,
		Mode:           lazyassets.UnpackBoth,
		HeartbeatEvery: 5 * time.Millisecond,
	})

	_, err := migrator.Up(ctx, 0)
	if err == nil {
		t.Fatalf("Up succeeded after lease was hijacked")
	}
	select {
	case <-leaseHijacked:
	default:
		t.Fatalf("test did not hijack the lease")
	}
}

func TestConcurrentRunnersUploadOnce(t *testing.T) {
	ctx, registry := testContext(t)
	storage := newMemoryStorage()
	uploadStarted := make(chan struct{})
	releaseUpload := make(chan struct{})
	var startOnce sync.Once
	storage.beforePut = func(ctx context.Context, key string) error {
		if key != "styles.css" {
			return nil
		}
		startOnce.Do(func() { close(uploadStarted) })
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-releaseUpload:
			return nil
		}
	}
	config := Config{
		Registry:        registry,
		Storage:         storage,
		Mode:            lazyassets.UnpackBoth,
		StaleAfter:      time.Hour,
		HeartbeatEvery:  5 * time.Millisecond,
		WaitMin:         2 * time.Millisecond,
		WaitMax:         2 * time.Millisecond,
		PostDoneRefresh: time.Millisecond,
	}
	first, id := testMigrator(t, ctx, config)
	second, _ := testMigrator(t, ctx, config)
	errs := make(chan error, 2)

	go func() {
		_, err := first.Up(ctx, 0)
		errs <- err
	}()
	select {
	case <-uploadStarted:
	case <-time.After(time.Second):
		t.Fatal("first runner did not start asset upload")
	}
	go func() {
		_, err := second.Up(ctx, 0)
		errs <- err
	}()
	close(releaseUpload)

	for range 2 {
		if err := <-errs; err != nil {
			t.Fatalf("runner error = %v", err)
		}
	}
	if puts := storage.puts("styles.css"); puts != 1 {
		t.Fatalf("logical asset puts = %d, want one upload", puts)
	}
	if !storage.exists(".migrations/" + id + "-done") {
		t.Fatalf("done marker was not written")
	}
}

func testContext(t *testing.T) (context.Context, *lazyassets.Registry) {
	t.Helper()
	registry := lazyassets.New()
	if err := registry.Add("/styles.css", []byte("body { color: black; }"), lazyassets.ContentType("text/css")); err != nil {
		t.Fatal(err)
	}
	ctx := lazycache.WithBuildVersion(context.Background(), "build.20260702")
	ctx = lazyassets.WithRegistry(ctx, registry)
	return ctx, registry
}

func testMigrator(t *testing.T, ctx context.Context, config Config) (*lazymigrate.Migrator, string) {
	t.Helper()
	db, err := DB(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	migrations, err := db.Sources[0].LoadMigrations(ctx)
	if err != nil {
		t.Fatal(err)
	}
	migrator, err := db.Migrator("assets")
	if err != nil {
		t.Fatal(err)
	}
	return migrator, migrations[0].ID
}

type memoryObject struct {
	body        []byte
	etag        string
	contentType string
	modifiedAt  time.Time
}

type memoryStorage struct {
	mu        sync.Mutex
	objects   map[string]memoryObject
	putCounts map[string]int
	nextETag  int
	beforePut func(context.Context, string) error
}

func newMemoryStorage() *memoryStorage {
	return &memoryStorage{
		objects:   map[string]memoryObject{},
		putCounts: map[string]int{},
	}
}

func (s *memoryStorage) Put(ctx context.Context, key string, body io.Reader, options ...any) (lazystorage.Info, []any, error) {
	if err := ctx.Err(); err != nil {
		return lazystorage.Info{}, options, err
	}
	if s.beforePut != nil {
		if err := s.beforePut(ctx, key); err != nil {
			return lazystorage.Info{}, options, err
		}
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return lazystorage.Info{}, options, err
	}
	contentType, remaining, _ := lazystorage.Take[lazystorage.ContentType](options)
	_, remaining, _ = lazystorage.Take[lazystorage.CacheControl](remaining)
	_, remaining, ifAbsent := lazystorage.Take[lazystorage.IfAbsent](remaining)
	ifETag, remaining, hasIfETag := lazystorage.Take[lazystorage.IfETag](remaining)

	s.mu.Lock()
	defer s.mu.Unlock()
	current, exists := s.objects[key]
	if ifAbsent && exists {
		return lazystorage.Info{}, remaining, lazystorage.ErrPreconditionFailed
	}
	if hasIfETag && (!exists || current.etag != ifETag.Value) {
		return lazystorage.Info{}, remaining, lazystorage.ErrPreconditionFailed
	}
	s.nextETag++
	object := memoryObject{
		body:        append([]byte(nil), data...),
		etag:        fmt.Sprintf("etag-%d", s.nextETag),
		contentType: contentType.Value,
		modifiedAt:  time.Now().UTC(),
	}
	s.objects[key] = object
	s.putCounts[key]++
	return lazystorage.Info{
		Key:         key,
		ContentType: object.contentType,
		Size:        int64(len(object.body)),
		Checksum:    object.etag,
		ModifiedAt:  object.modifiedAt,
	}, remaining, nil
}

func (s *memoryStorage) Open(ctx context.Context, key string, options ...any) (lazystorage.File, []any, error) {
	if err := ctx.Err(); err != nil {
		return nil, options, err
	}
	s.mu.Lock()
	object, ok := s.objects[key]
	s.mu.Unlock()
	if !ok {
		return nil, options, fs.ErrNotExist
	}
	return &memoryFile{Reader: strings.NewReader(string(object.body)), object: object, key: key}, options, nil
}

func (s *memoryStorage) exists(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.objects[key]
	return ok
}

func (s *memoryStorage) puts(key string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.putCounts[key]
}

func (s *memoryStorage) totalPuts() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	total := 0
	for _, count := range s.putCounts {
		total += count
	}
	return total
}

func (s *memoryStorage) writeObject(key string, marker marker) {
	data, _ := json.Marshal(marker)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextETag++
	s.objects[key] = memoryObject{
		body:       append(data, '\n'),
		etag:       fmt.Sprintf("etag-%d", s.nextETag),
		modifiedAt: marker.timestamp(lazystorage.Info{ModifiedAt: time.Now().UTC()}),
	}
}

func (s *memoryStorage) hijack(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.objects[key]
	s.nextETag++
	current.etag = fmt.Sprintf("hijacked-%d", s.nextETag)
	current.modifiedAt = time.Now().UTC()
	s.objects[key] = current
}

func (s *memoryStorage) idFor(t *testing.T, registry *lazyassets.Registry, ctx context.Context) string {
	t.Helper()
	_, id := testMigrator(t, ctx, Config{Registry: registry, Storage: s, Mode: lazyassets.UnpackBoth})
	return id
}

func (s *memoryStorage) metadataFor(t *testing.T, migrator *lazymigrate.Migrator, ctx context.Context) metadata {
	t.Helper()
	statuses, err := migrator.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 {
		t.Fatalf("statuses = %d, want 1", len(statuses))
	}
	var out metadata
	if err := json.Unmarshal(statuses[0].Migration.Content, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

type memoryFile struct {
	*strings.Reader
	object memoryObject
	key    string
}

func (f *memoryFile) Close() error {
	return nil
}

func (f *memoryFile) Stat() (lazystorage.Info, error) {
	return lazystorage.Info{
		Key:         f.key,
		ContentType: f.object.contentType,
		Size:        int64(len(f.object.body)),
		Checksum:    f.object.etag,
		ModifiedAt:  f.object.modifiedAt,
	}, nil
}
