// Package assetmigrate adapts lazyassets uploads to lazymigrate.
package assetmigrate

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"os"
	"path"
	"strings"
	"time"
	"unicode"

	"golazy.dev/lazyassets"
	"golazy.dev/lazycache"
	"golazy.dev/lazymigrate"
	"golazy.dev/lazystorage"
)

const (
	defaultMarkerPrefix    = ".migrations"
	defaultStaleAfter      = 30 * time.Second
	defaultHeartbeatEvery  = 10 * time.Second
	defaultWaitMin         = 15 * time.Second
	defaultWaitMax         = 30 * time.Second
	defaultPostDoneRefresh = 30 * time.Second
)

// ErrSchemaUnsupported is returned by schema load/dump methods because asset
// uploads do not have schema snapshots.
var ErrSchemaUnsupported = errors.New("assetmigrate: schema load and dump are not supported")

// Storage is the object storage capability required by asset migrations.
type Storage interface {
	lazystorage.Storage
	lazystorage.Writer
}

// Config describes one asset upload migration.
type Config struct {
	Registry        *lazyassets.Registry
	Storage         Storage
	ID              string
	MarkerPrefix    string
	Mode            lazyassets.UnpackMode
	UploadPrefix    string
	WriteManifest   bool
	StaleAfter      time.Duration
	HeartbeatEvery  time.Duration
	WaitMin         time.Duration
	WaitMax         time.Duration
	PostDoneRefresh time.Duration
}

// DB returns a lazymigrate DB containing one source migration and one backend.
func DB(ctx context.Context, config Config) (lazymigrate.DB, error) {
	runtime, err := normalize(ctx, config)
	if err != nil {
		return lazymigrate.DB{}, err
	}
	backend := &Backend{config: runtime}
	return lazymigrate.DB{
		Backend: backend,
		Sources: []lazymigrate.Source{source{metadata: runtime.metadata}},
	}, nil
}

// Backend stores asset migration state in object storage and uploads the
// registry when its source migration is pending.
type Backend struct {
	config runtimeConfig
}

var _ lazymigrate.Backend = (*Backend)(nil)

type runtimeConfig struct {
	registry        *lazyassets.Registry
	storage         Storage
	metadata        metadata
	markerPrefix    string
	mode            lazyassets.UnpackMode
	uploadPrefix    string
	writeManifest   bool
	staleAfter      time.Duration
	heartbeatEvery  time.Duration
	waitMin         time.Duration
	waitMax         time.Duration
	postDoneRefresh time.Duration
}

type metadata struct {
	ID             string `json:"id"`
	BuildVersion   string `json:"build_version"`
	ManifestHash   string `json:"manifest_hash"`
	UploadPlanHash string `json:"upload_plan_hash"`
}

type marker struct {
	metadata
	Owner       string    `json:"owner,omitempty"`
	StartedAt   time.Time `json:"started_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CompletedAt time.Time `json:"completed_at"`
}

type source struct {
	metadata metadata
}

func (s source) LoadMigrations(context.Context) ([]lazymigrate.Migration, error) {
	content, err := json.Marshal(s.metadata)
	if err != nil {
		return nil, fmt.Errorf("assetmigrate: marshal migration metadata: %w", err)
	}
	return []lazymigrate.Migration{{
		ID:      s.metadata.ID,
		Prefix:  "lazyassets",
		Path:    "assetmigrate:" + s.metadata.ID,
		Content: content,
	}}, nil
}

func normalize(ctx context.Context, config Config) (runtimeConfig, error) {
	registry := config.Registry
	if registry == nil {
		var ok bool
		registry, ok = lazyassets.FromContext(ctx)
		if !ok {
			return runtimeConfig{}, fmt.Errorf("assetmigrate: asset registry is required")
		}
	}
	if config.Storage == nil {
		return runtimeConfig{}, fmt.Errorf("assetmigrate: storage is required")
	}
	markerPrefix := strings.Trim(strings.TrimSpace(config.MarkerPrefix), "/")
	if markerPrefix == "" {
		markerPrefix = defaultMarkerPrefix
	}
	if err := lazystorage.ValidateKey(markerPrefix); err != nil {
		return runtimeConfig{}, fmt.Errorf("assetmigrate: marker prefix: %w", err)
	}

	manifestHash, uploadPlanHash, err := planHashes(registry, config)
	if err != nil {
		return runtimeConfig{}, err
	}
	buildVersion := lazycache.BuildVersionFromContext(ctx)
	id := strings.TrimSpace(config.ID)
	if id == "" {
		id = "lazyassets-" + sanitizeIDPart(buildVersion) + "-" + shortHash(uploadPlanHash)
	}
	if err := lazystorage.ValidateKey(path.Join(markerPrefix, id)); err != nil {
		return runtimeConfig{}, fmt.Errorf("assetmigrate: migration id %q is not storage-safe: %w", id, err)
	}

	return runtimeConfig{
		registry:       registry,
		storage:        config.Storage,
		metadata:       metadata{ID: id, BuildVersion: buildVersion, ManifestHash: manifestHash, UploadPlanHash: uploadPlanHash},
		markerPrefix:   markerPrefix,
		mode:           config.Mode,
		uploadPrefix:   strings.Trim(strings.TrimSpace(config.UploadPrefix), "/"),
		writeManifest:  config.WriteManifest,
		staleAfter:     defaultDuration(config.StaleAfter, defaultStaleAfter),
		heartbeatEvery: defaultDuration(config.HeartbeatEvery, defaultHeartbeatEvery),
		waitMin:        defaultDuration(config.WaitMin, defaultWaitMin),
		waitMax:        defaultDuration(config.WaitMax, defaultWaitMax),
		postDoneRefresh: defaultDuration(
			config.PostDoneRefresh,
			defaultPostDoneRefresh,
		),
	}, nil
}

func planHashes(registry *lazyassets.Registry, config Config) (string, string, error) {
	manifestData, err := json.Marshal(registry.Manifest())
	if err != nil {
		return "", "", fmt.Errorf("assetmigrate: marshal asset manifest: %w", err)
	}
	uploadData, err := json.Marshal(struct {
		Mode          lazyassets.UnpackMode `json:"mode"`
		UploadPrefix  string                `json:"upload_prefix"`
		WriteManifest bool                  `json:"write_manifest"`
	}{
		Mode:          config.Mode,
		UploadPrefix:  strings.Trim(strings.TrimSpace(config.UploadPrefix), "/"),
		WriteManifest: config.WriteManifest,
	})
	if err != nil {
		return "", "", fmt.Errorf("assetmigrate: marshal upload config: %w", err)
	}
	planData := bytes.Join([][]byte{manifestData, uploadData}, []byte{'\n'})
	return digest(manifestData), digest(planData), nil
}

func defaultDuration(value, fallback time.Duration) time.Duration {
	if value > 0 {
		return value
	}
	return fallback
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func shortHash(value string) string {
	value = strings.TrimPrefix(value, "sha256:")
	if len(value) > 12 {
		return value[:12]
	}
	return value
}

func sanitizeIDPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "(devel)" {
		value = "devel"
	}
	var out strings.Builder
	lastDash := false
	for _, r := range value {
		if r <= unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
			out.WriteRune(unicode.ToLower(r))
			lastDash = false
			continue
		}
		if r == '.' || r == '_' || r == '-' {
			out.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			out.WriteByte('-')
			lastDash = true
		}
	}
	cleaned := strings.Trim(out.String(), "-_.")
	if cleaned == "" {
		return "devel"
	}
	return cleaned
}

// Setup validates the backend. Object storage metadata is represented by marker
// objects, so no backend table or bucket is created here.
func (b *Backend) Setup(context.Context) error {
	return b.validate()
}

// List reports the configured migration as applied when its done marker exists
// and matches the current asset manifest and upload config.
func (b *Backend) List(ctx context.Context) ([]lazymigrate.BackendMigration, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}
	done, _, err := b.readMarker(ctx, b.doneKey())
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !done.matches(b.config.metadata) {
		return nil, fmt.Errorf("assetmigrate: done marker %s does not match current migration metadata", b.doneKey())
	}
	return []lazymigrate.BackendMigration{{ID: b.config.metadata.ID}}, nil
}

// Run applies the asset upload migration.
func (b *Backend) Run(ctx context.Context, step lazymigrate.Step) error {
	if err := b.validate(); err != nil {
		return err
	}
	if step.Migration.ID != b.config.metadata.ID {
		return fmt.Errorf("assetmigrate: unexpected migration id %q, want %q", step.Migration.ID, b.config.metadata.ID)
	}
	switch step.Direction {
	case lazymigrate.DirectionUp:
	case lazymigrate.DirectionDown:
		return fmt.Errorf("assetmigrate: down migrations are not supported")
	default:
		return fmt.Errorf("assetmigrate: unsupported direction %q", step.Direction)
	}
	applied, err := b.applied(ctx)
	if err != nil || applied {
		return err
	}
	lease, owned, err := b.acquire(ctx)
	if err != nil {
		return err
	}
	if !owned {
		return nil
	}

	uploadCtx, cancel := context.WithCancel(ctx)
	heartbeatErr := b.startHeartbeat(uploadCtx, cancel, lease)
	uploadErr := b.config.registry.Upload(uploadCtx, b.config.storage, b.uploadOptions()...)
	if uploadErr == nil {
		select {
		case uploadErr = <-heartbeatErr:
		default:
		}
	}
	if uploadErr != nil {
		cancel()
		return uploadErr
	}
	if err := b.writeDone(ctx); err != nil {
		cancel()
		return err
	}
	if b.config.postDoneRefresh <= 0 {
		cancel()
		return nil
	}
	time.AfterFunc(b.config.postDoneRefresh, cancel)
	return nil
}

func (b *Backend) DumpSchema(context.Context) ([]byte, error) {
	return nil, ErrSchemaUnsupported
}

func (b *Backend) LoadSchema(context.Context, []byte) error {
	return ErrSchemaUnsupported
}

func (b *Backend) validate() error {
	if b == nil {
		return fmt.Errorf("assetmigrate: backend is nil")
	}
	if b.config.registry == nil {
		return fmt.Errorf("assetmigrate: asset registry is required")
	}
	if b.config.storage == nil {
		return fmt.Errorf("assetmigrate: storage is required")
	}
	if b.config.metadata.ID == "" {
		return fmt.Errorf("assetmigrate: migration id is required")
	}
	return nil
}

func (b *Backend) applied(ctx context.Context) (bool, error) {
	migrations, err := b.List(ctx)
	if err != nil {
		return false, err
	}
	return len(migrations) > 0, nil
}

type lease struct {
	key   string
	etag  string
	owner string
}

func (b *Backend) acquire(ctx context.Context) (lease, bool, error) {
	owner := newOwner()
	for {
		applied, err := b.applied(ctx)
		if err != nil {
			return lease{}, false, err
		}
		if applied {
			return lease{}, false, nil
		}
		etag, err := b.writeLease(ctx, owner, "", lazystorage.IfAbsent{})
		if err == nil {
			return lease{key: b.leaseKey(), etag: etag, owner: owner}, true, nil
		}
		if !errors.Is(err, lazystorage.ErrPreconditionFailed) {
			return lease{}, false, err
		}
		current, info, err := b.readMarker(ctx, b.leaseKey())
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return lease{}, false, err
		}
		if time.Since(current.timestamp(info)) < b.config.staleAfter {
			if err := sleep(ctx, waitDuration(b.config.waitMin, b.config.waitMax)); err != nil {
				return lease{}, false, err
			}
			continue
		}
		etag, err = b.writeLease(ctx, owner, info.Checksum, lazystorage.IfETag{Value: info.Checksum})
		if err == nil {
			return lease{key: b.leaseKey(), etag: etag, owner: owner}, true, nil
		}
		if errors.Is(err, lazystorage.ErrPreconditionFailed) {
			continue
		}
		return lease{}, false, err
	}
}

func (b *Backend) startHeartbeat(ctx context.Context, cancel context.CancelFunc, lease lease) <-chan error {
	errs := make(chan error, 1)
	if b.config.heartbeatEvery <= 0 {
		return errs
	}
	go func() {
		ticker := time.NewTicker(b.config.heartbeatEvery)
		defer ticker.Stop()
		etag := lease.etag
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				next, err := b.writeLease(ctx, lease.owner, etag, lazystorage.IfETag{Value: etag})
				if err != nil {
					select {
					case errs <- err:
					default:
					}
					cancel()
					return
				}
				etag = next
			}
		}
	}()
	return errs
}

func (b *Backend) writeLease(ctx context.Context, owner, etag string, conditions ...any) (string, error) {
	now := time.Now().UTC()
	lease := marker{
		metadata:  b.config.metadata,
		Owner:     owner,
		StartedAt: now,
		UpdatedAt: now,
	}
	if etag != "" {
		lease.StartedAt = time.Time{}
	}
	return b.writeMarker(ctx, b.leaseKey(), lease, conditions...)
}

func (b *Backend) writeDone(ctx context.Context) error {
	done := marker{
		metadata:    b.config.metadata,
		CompletedAt: time.Now().UTC(),
	}
	_, err := b.writeMarker(ctx, b.doneKey(), done)
	return err
}

func (b *Backend) writeMarker(ctx context.Context, key string, marker marker, conditions ...any) (string, error) {
	data, err := json.Marshal(marker)
	if err != nil {
		return "", fmt.Errorf("assetmigrate: marshal marker %s: %w", key, err)
	}
	options := []any{
		lazystorage.ContentType{Value: "application/json"},
		lazystorage.CacheControl{Value: "no-store"},
	}
	options = append(options, conditions...)
	info, remaining, err := b.config.storage.Put(ctx, key, bytes.NewReader(append(data, '\n')), options...)
	if err != nil {
		return "", fmt.Errorf("assetmigrate: write marker %s: %w", key, err)
	}
	if hasPrecondition(remaining) {
		return "", fmt.Errorf("assetmigrate: storage did not consume conditional write options for %s", key)
	}
	if info.Checksum != "" {
		return info.Checksum, nil
	}
	_, stat, err := b.readMarker(ctx, key)
	if err != nil {
		return "", err
	}
	if stat.Checksum == "" {
		return "", fmt.Errorf("assetmigrate: marker %s did not return an ETag or checksum", key)
	}
	return stat.Checksum, nil
}

func (b *Backend) readMarker(ctx context.Context, key string) (marker, lazystorage.Info, error) {
	file, _, err := b.config.storage.Open(ctx, key)
	if err != nil {
		return marker{}, lazystorage.Info{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return marker{}, lazystorage.Info{}, fmt.Errorf("assetmigrate: stat marker %s: %w", key, err)
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return marker{}, lazystorage.Info{}, fmt.Errorf("assetmigrate: read marker %s: %w", key, err)
	}
	var out marker
	if err := json.Unmarshal(data, &out); err != nil {
		return marker{}, lazystorage.Info{}, fmt.Errorf("assetmigrate: parse marker %s: %w", key, err)
	}
	return out, info, nil
}

func (b *Backend) uploadOptions() []lazyassets.UploadOption {
	options := []lazyassets.UploadOption{
		lazyassets.WithUploadMode(b.config.mode),
	}
	if b.config.uploadPrefix != "" {
		options = append(options, lazyassets.WithUploadPrefix(b.config.uploadPrefix))
	}
	if !b.config.writeManifest {
		options = append(options, lazyassets.WithoutUploadManifest())
	}
	return options
}

func (b *Backend) leaseKey() string {
	return path.Join(b.config.markerPrefix, b.config.metadata.ID)
}

func (b *Backend) doneKey() string {
	return path.Join(b.config.markerPrefix, b.config.metadata.ID+"-done")
}

func (m marker) matches(expected metadata) bool {
	return m.ID == expected.ID &&
		m.BuildVersion == expected.BuildVersion &&
		m.ManifestHash == expected.ManifestHash &&
		m.UploadPlanHash == expected.UploadPlanHash
}

func (m marker) timestamp(info lazystorage.Info) time.Time {
	if !m.UpdatedAt.IsZero() {
		return m.UpdatedAt
	}
	if !m.StartedAt.IsZero() {
		return m.StartedAt
	}
	return info.ModifiedAt
}

func hasPrecondition(options []any) bool {
	for _, option := range options {
		switch option.(type) {
		case lazystorage.IfAbsent, lazystorage.IfETag:
			return true
		}
	}
	return false
}

func waitDuration(min, max time.Duration) time.Duration {
	if max <= min {
		return min
	}
	return min + time.Duration(rand.Int63n(int64(max-min)))
}

func sleep(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func newOwner() string {
	hostname, _ := os.Hostname()
	hostname = sanitizeIDPart(hostname)
	return fmt.Sprintf("%s-%d-%d", hostname, os.Getpid(), time.Now().UnixNano())
}
