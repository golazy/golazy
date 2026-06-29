package lazyassets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"

	"golazy.dev/lazyview"
)

const (
	defaultMaxAssetSize = 5 << 20
	defaultHashLength   = 12
)

// OpenFunc opens an asset stream for AddReader or filesystem-backed records.
type OpenFunc func() (io.ReadCloser, error)

// OversizePolicy controls how disk or generated assets larger than MaxAssetSize
// participate in hashing, rewriting, serving, and export.
type OversizePolicy int

const (
	// OversizeIgnoreForPipeline serves oversized disk assets by logical path but
	// leaves them out of the fingerprinting pipeline. They do not get permanent
	// paths, ETags, integrity metadata, or CSS URL rewriting. This is the default
	// for files registered through AddFS.
	OversizeIgnoreForPipeline OversizePolicy = iota

	// OversizeError rejects oversized assets during registration.
	OversizeError

	// OversizeAllow allows oversized assets to be read, hashed, rewritten, and
	// exported like any other asset.
	OversizeAllow

	// OversizeSkipServing ignores oversized assets entirely, so Handler will fall
	// through to the next handler for their paths.
	OversizeSkipServing
)

// CachePolicy is the literal Cache-Control value used when serving or uploading
// logical and permanent asset paths.
type CachePolicy string

// Options configures a Registry.
type Options struct {
	// URLPrefix mounts registered assets below a path prefix. With /assets, an
	// asset added as /app.js is served and reported as /assets/app.js.
	URLPrefix string
	// MaxAssetSize limits assets that are read into the fingerprinting pipeline.
	// Negative values disable the limit.
	MaxAssetSize int64
	// OversizePolicy selects behavior for assets larger than MaxAssetSize.
	OversizePolicy OversizePolicy
	// HashLength is the number of hexadecimal hash characters inserted into
	// permanent paths.
	HashLength int
	// LogicalCache is sent for logical paths such as /styles.css.
	LogicalCache CachePolicy
	// PermanentCache is sent for permanent paths such as /styles-<hash>.css.
	PermanentCache CachePolicy
	// RewriteCSSURLs rewrites local CSS url(...) references to permanent paths
	// for registered target assets.
	RewriteCSSURLs bool
	// BaseURL makes helpers and rewritten CSS/importmap URLs absolute without
	// changing the request paths that Handler serves or Upload writes.
	BaseURL string
	// Development disables permanent paths, cache headers, ETags, integrity
	// values, and CSS rewriting so filesystem assets can be served fresh.
	Development bool
}

// Option configures New.
type Option func(*Options)

// WithURLPrefix mounts assets below prefix.
func WithURLPrefix(prefix string) Option {
	return func(options *Options) {
		options.URLPrefix = prefix
	}
}

// WithMaxAssetSize sets the largest asset read into the fingerprinting
// pipeline.
func WithMaxAssetSize(size int64) Option {
	return func(options *Options) {
		options.MaxAssetSize = size
	}
}

// WithOversizePolicy sets the behavior for assets larger than MaxAssetSize.
func WithOversizePolicy(policy OversizePolicy) Option {
	return func(options *Options) {
		options.OversizePolicy = policy
	}
}

// WithHashLength sets the number of hexadecimal hash characters inserted into
// permanent paths.
func WithHashLength(length int) Option {
	return func(options *Options) {
		options.HashLength = length
	}
}

// WithCachePolicies sets Cache-Control values for logical and permanent paths.
func WithCachePolicies(logical, permanent CachePolicy) Option {
	return func(options *Options) {
		options.LogicalCache = logical
		options.PermanentCache = permanent
	}
}

// WithDevelopmentMode serves logical asset paths directly without permanent
// hashed URLs, cache headers, ETags, integrity values, or CSS URL rewriting.
func WithDevelopmentMode(enabled bool) Option {
	return func(options *Options) {
		options.Development = enabled
	}
}

// WithCSSURLRewrite enables or disables rewriting local CSS url(...) references
// to permanent asset URLs.
func WithCSSURLRewrite(enabled bool) Option {
	return func(options *Options) {
		options.RewriteCSSURLs = enabled
	}
}

// WithBaseURL makes helpers return absolute asset URLs while keeping request
// routing and storage keys path-based.
func WithBaseURL(baseURL string) Option {
	return func(options *Options) {
		options.BaseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	}
}

// Source registers assets in a Registry.
//
// Generated asset packages can implement Source so lazyapp or custom startup
// code can add generated files to the same Registry as filesystem assets.
type Source interface {
	Assets(*Registry) error
}

// SourceFunc adapts a function into a Source.
type SourceFunc func(*Registry) error

// Assets calls fn(registry).
func (fn SourceFunc) Assets(registry *Registry) error {
	return fn(registry)
}

// Registry owns the registered asset index.
//
// It maps logical request paths and content-hashed permanent request paths to
// the same Asset records. The manifest is derived from the registered records;
// callers do not need to load or create one before serving.
type Registry struct {
	options   Options
	logical   map[string]*Asset
	permanent map[string]*Asset
	assets    []*Asset
}

// Asset describes one registered asset as it appears in a Manifest.
type Asset struct {
	// Path is the logical public path, for example /styles.css.
	Path string `json:"path"`
	// Permanent is the content-hashed public path, for example
	// /styles-<hash>.css. It is empty for development-mode or ignored assets.
	Permanent string `json:"permanent,omitempty"`
	// ContentType is the MIME type sent by Handler and written by Upload.
	ContentType string `json:"content_type,omitempty"`
	// Size is the served byte length, or -1 when development mode reads the
	// file each time and the size is not fixed in the manifest.
	Size int64 `json:"size"`
	// Hash is the full SHA-256 hex digest of the final served bytes.
	Hash string `json:"hash,omitempty"`
	// ETag is the strong ETag for the final served bytes.
	ETag string `json:"etag,omitempty"`
	// Integrity is the subresource-integrity value for the final served bytes.
	Integrity string `json:"integrity,omitempty"`
	// Source identifies the source file or generator, when supplied.
	Source string `json:"source,omitempty"`
	// Generated is true for assets registered from in-memory/generated bytes.
	Generated bool `json:"generated,omitempty"`
	// Ignored is true for oversized assets that are served by logical path but
	// skipped by hashing, permanent URL generation, and rewriting.
	Ignored bool `json:"ignored,omitempty"`

	content []byte
	raw     []byte
	open    OpenFunc
}

// Manifest is a snapshot of all registered assets and their computed metadata.
type Manifest struct {
	Assets []Asset `json:"assets"`
}

// SourceOption configures AddFS.
type SourceOption func(*sourceOptions)

type sourceOptions struct {
	source  string
	replace bool
}

// SourceName records name as the source for assets registered from AddFS.
func SourceName(name string) SourceOption {
	return func(options *sourceOptions) {
		options.source = name
	}
}

// Replace allows AddFS to replace previously registered assets with the same
// logical path.
func Replace() SourceOption {
	return func(options *sourceOptions) {
		options.replace = true
	}
}

// AssetOption configures Add and AddReader.
type AssetOption func(*assetOptions)

type assetOptions struct {
	contentType string
	source      string
	replace     bool
}

// ContentType sets the asset MIME type instead of inferring it from the path
// and content.
func ContentType(contentType string) AssetOption {
	return func(options *assetOptions) {
		options.contentType = contentType
	}
}

// AssetSource records name as the source for one registered asset.
func AssetSource(name string) AssetOption {
	return func(options *assetOptions) {
		options.source = name
	}
}

// ReplaceAsset allows Add or AddReader to replace a previously registered asset
// with the same logical path.
func ReplaceAsset() AssetOption {
	return func(options *assetOptions) {
		options.replace = true
	}
}

// New creates an empty Registry.
//
// By default, logical paths are revalidation-friendly, permanent paths are
// immutable, CSS local URLs are rewritten, and permanent paths use a 12
// character hash prefix.
func New(options ...Option) *Registry {
	config := Options{
		URLPrefix:      "/",
		MaxAssetSize:   defaultMaxAssetSize,
		HashLength:     defaultHashLength,
		LogicalCache:   "public, max-age=0, must-revalidate",
		PermanentCache: "public, max-age=31536000, immutable",
		RewriteCSSURLs: true,
	}
	for _, option := range options {
		option(&config)
	}
	if config.Development {
		config.LogicalCache = ""
		config.PermanentCache = ""
		config.RewriteCSSURLs = false
	}
	config.URLPrefix = normalizePrefix(config.URLPrefix)
	if config.HashLength <= 0 {
		config.HashLength = defaultHashLength
	}
	return &Registry{
		options:   config,
		logical:   map[string]*Asset{},
		permanent: map[string]*Asset{},
	}
}

// AddFS registers every file in files under its slash-separated path.
//
// AddFS computes permanent paths and manifest metadata immediately unless
// development mode or the oversize policy keeps an asset out of the
// fingerprinting pipeline. Directories are skipped.
func (r *Registry) AddFS(files fs.FS, options ...SourceOption) error {
	if r == nil {
		return fmt.Errorf("lazyassets: registry is nil")
	}
	var opts sourceOptions
	for _, option := range options {
		option(&opts)
	}

	var names []string
	if err := fs.WalkDir(files, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		names = append(names, name)
		return nil
	}); err != nil {
		return fmt.Errorf("walk assets: %w", err)
	}
	sort.Strings(names)

	for _, name := range names {
		info, err := fs.Stat(files, name)
		if err != nil {
			return fmt.Errorf("stat %s: %w", name, err)
		}
		name := name
		open := func() (io.ReadCloser, error) {
			return files.Open(name)
		}
		if err := r.addFile("/"+name, info.Size(), open, assetOptions{
			contentType: contentTypeForName(name, nil),
			source:      firstNonEmpty(opts.source, name),
			replace:     opts.replace,
		}); err != nil {
			return err
		}
	}
	return nil
}

// Add registers an in-memory asset at path.
//
// Generated assets are fingerprinted like filesystem assets and appear in the
// manifest with Generated set. The content slice is copied before registration.
func (r *Registry) Add(path string, content []byte, options ...AssetOption) error {
	if r == nil {
		return fmt.Errorf("lazyassets: registry is nil")
	}
	var opts assetOptions
	for _, option := range options {
		option(&opts)
	}
	data := append([]byte(nil), content...)
	return r.addBytes(path, data, true, opts)
}

// AddReader registers an asset opened on demand by open.
//
// In production mode AddReader reads the content during registration so it can
// compute the hash, ETag, integrity value, permanent path, and CSS rewrites. In
// development mode it keeps open and reads the asset for each request.
func (r *Registry) AddReader(path string, open OpenFunc, options ...AssetOption) error {
	if r == nil {
		return fmt.Errorf("lazyassets: registry is nil")
	}
	if open == nil {
		return fmt.Errorf("lazyassets: open function is nil")
	}
	var opts assetOptions
	for _, option := range options {
		option(&opts)
	}
	if r.options.Development {
		return r.addOpen(path, open, true, opts)
	}

	file, err := open()
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	data, err := readAsset(file, r.options.MaxAssetSize, r.options.OversizePolicy == OversizeAllow)
	if err != nil {
		return err
	}
	return r.addBytes(path, data, true, opts)
}

// Path returns the URL to use for assetPath in HTML or generated output.
//
// It returns the permanent content-hashed path when the asset has one. In
// development mode and for ignored oversized assets, it returns the logical
// path. WithBaseURL makes the returned URL absolute.
func (r *Registry) Path(assetPath string) (string, error) {
	asset, ok := r.findLogical(assetPath)
	if !ok {
		return "", fmt.Errorf("lazyassets: asset %q not found", assetPath)
	}
	if asset.Permanent != "" {
		return r.assetURL(asset.Permanent), nil
	}
	return r.assetURL(asset.Path), nil
}

// MustPath is like Path, but panics on error.
func (r *Registry) MustPath(assetPath string) string {
	result, err := r.Path(assetPath)
	if err != nil {
		panic(err)
	}
	return result
}

// Integrity returns the subresource-integrity value for assetPath.
//
// Development-mode and ignored oversized assets do not have integrity metadata,
// so Integrity returns an empty string for those registered assets.
func (r *Registry) Integrity(assetPath string) (string, error) {
	asset, ok := r.findLogical(assetPath)
	if !ok {
		return "", fmt.Errorf("lazyassets: asset %q not found", assetPath)
	}
	return asset.Integrity, nil
}

func (r *Registry) content(assetPath string) ([]byte, error) {
	asset, ok := r.findLogical(assetPath)
	if !ok {
		return nil, fmt.Errorf("lazyassets: asset %q not found", assetPath)
	}
	if asset.content != nil {
		return append([]byte(nil), asset.content...), nil
	}
	if asset.raw != nil {
		return append([]byte(nil), asset.raw...), nil
	}
	if asset.open == nil {
		return nil, nil
	}

	file, err := asset.open()
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", assetPath, err)
	}
	defer file.Close()

	data, err := readAsset(file, r.options.MaxAssetSize, r.options.OversizePolicy == OversizeAllow)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Manifest returns a snapshot of the current registered assets.
//
// The manifest is computed from assets already added to the Registry; callers
// do not need to create or load it before calling Handler, Path, Upload, or
// Unpack.
func (r *Registry) Manifest() Manifest {
	manifest := Manifest{Assets: make([]Asset, 0, len(r.assets))}
	for _, asset := range r.assets {
		manifest.Assets = append(manifest.Assets, asset.snapshot())
	}
	return manifest
}

// Empty reports whether the Registry has no registered assets.
func (r *Registry) Empty() bool {
	return r == nil || len(r.assets) == 0
}

// Handler serves registered assets and falls through to next for misses.
//
// Requests are matched against logical paths such as /styles.css and permanent
// paths such as /styles-<hash>.css. Requests for / and paths ending in / look
// for index.html below that directory. Only GET and HEAD are served; other
// methods for known asset paths return 405 with Allow: GET, HEAD. Unknown paths
// are passed to next.
//
// The original logical files remain available through Handler unless an asset
// was skipped entirely by OversizeSkipServing. Permanent paths are generated
// when assets are added; no separate manifest creation step is required.
func (r *Registry) Handler(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		asset, permanent, ok := r.findRequest(req.URL.Path)
		if !ok {
			next.ServeHTTP(w, req)
			return
		}
		if req.Method != http.MethodGet && req.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		r.serveAsset(w, req, asset, permanent)
	})
}

// ServeHTTP serves the Registry as a standalone http.Handler.
func (r *Registry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.Handler(http.NotFoundHandler()).ServeHTTP(w, req)
}

// Helpers returns lazyview-compatible template helpers for registered assets.
//
// lazyapp calls Helpers during startup and adds the returned functions to
// lazyview. Templates then use helpers such as asset_path, stylesheet,
// importmap, asset_integrity, and the compatibility alias permalink.
func (r *Registry) Helpers() map[string]any {
	return map[string]any{
		"asset_path": func(path string) (string, error) {
			return r.Path(path)
		},
		"asset_integrity": func(path string) (string, error) {
			return r.Integrity(path)
		},
		"stylesheet": func(path string) (lazyview.Fragment, error) {
			permanent, err := r.Path(path)
			if err != nil {
				return lazyview.Fragment{}, err
			}
			return lazyview.Fragment{
				ContentType: "text/html; charset=utf-8",
				Body:        `<link rel="stylesheet" href="` + html.EscapeString(permanent) + `">`,
			}, nil
		},
		"importmap": func(path string) (lazyview.Fragment, error) {
			data, err := r.content(path)
			if err != nil {
				return lazyview.Fragment{}, err
			}
			data, err = r.rewriteImportmapURLs(data)
			if err != nil {
				return lazyview.Fragment{}, err
			}
			if !json.Valid(data) {
				return lazyview.Fragment{}, fmt.Errorf("lazyassets: importmap %q is not valid JSON", path)
			}
			var escaped bytes.Buffer
			json.HTMLEscape(&escaped, data)
			return lazyview.Fragment{
				ContentType: "text/html; charset=utf-8",
				Body:        `<script type="importmap">` + escaped.String() + `</script>`,
			}, nil
		},
		"permalink": func(path string) (string, error) {
			return r.Path(path)
		},
	}
}

func (r *Registry) rewriteImportmapURLs(data []byte) ([]byte, error) {
	if r.options.BaseURL == "" {
		return data, nil
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return data, nil
	}
	rewriteStringMap := func(value any) {
		values, ok := value.(map[string]any)
		if !ok {
			return
		}
		for name, raw := range values {
			ref, ok := raw.(string)
			if ok && strings.HasPrefix(ref, "/") {
				values[name] = r.assetURL(ref)
			}
		}
	}
	rewriteStringMap(doc["imports"])
	if scopes, ok := doc["scopes"].(map[string]any); ok {
		for _, scope := range scopes {
			rewriteStringMap(scope)
		}
	}
	rewritten, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("lazyassets: rewrite importmap URLs: %w", err)
	}
	return rewritten, nil
}

// UnpackMode selects which asset path forms Unpack and Upload write.
type UnpackMode int

const (
	// UnpackBoth writes logical files and permanent content-hashed files.
	UnpackBoth UnpackMode = iota
	// UnpackLogical writes only logical files such as styles.css.
	UnpackLogical
	// UnpackPermanent writes only permanent files such as styles-<hash>.css.
	UnpackPermanent
)

// UnpackOption configures Unpack.
type UnpackOption func(*unpackOptions)

type unpackOptions struct {
	mode UnpackMode
}

// WithUnpackMode selects which asset path forms Unpack writes.
func WithUnpackMode(mode UnpackMode) UnpackOption {
	return func(options *unpackOptions) {
		options.mode = mode
	}
}

// Unpack writes registered assets and manifest.json into dir.
//
// By default it writes both logical and permanent asset files. The manifest is
// always written and is derived from the Registry's current asset records.
func (r *Registry) Unpack(dir string, options ...UnpackOption) error {
	var opts unpackOptions
	for _, option := range options {
		option(&opts)
	}
	for _, asset := range r.assets {
		if opts.mode != UnpackPermanent {
			if err := r.unpackAsset(dir, asset.Path, asset); err != nil {
				return err
			}
		}
		if opts.mode != UnpackLogical && asset.Permanent != "" {
			if err := r.unpackAsset(dir, asset.Permanent, asset); err != nil {
				return err
			}
		}
	}

	manifestPath := path.Join(dir, "manifest.json")
	data, err := json.MarshalIndent(r.Manifest(), "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := os.MkdirAll(path.Dir(manifestPath), 0o755); err != nil {
		return fmt.Errorf("create manifest directory: %w", err)
	}
	if err := os.WriteFile(manifestPath, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}

func (r *Registry) addFile(assetPath string, size int64, open OpenFunc, opts assetOptions) error {
	if r.options.MaxAssetSize >= 0 && size > r.options.MaxAssetSize && r.options.OversizePolicy != OversizeAllow {
		switch r.options.OversizePolicy {
		case OversizeError:
			return fmt.Errorf("lazyassets: asset %q is %d bytes, larger than max %d", assetPath, size, r.options.MaxAssetSize)
		case OversizeSkipServing:
			return nil
		default:
			return r.addIgnored(assetPath, size, open, opts)
		}
	}
	if r.options.Development {
		return r.addOpen(assetPath, open, false, opts)
	}

	file, err := open()
	if err != nil {
		return fmt.Errorf("open %s: %w", assetPath, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("read %s: %w", assetPath, err)
	}
	return r.addBytes(assetPath, data, false, opts)
}

func (r *Registry) addBytes(assetPath string, content []byte, generated bool, opts assetOptions) error {
	if r.options.MaxAssetSize >= 0 && int64(len(content)) > r.options.MaxAssetSize && r.options.OversizePolicy != OversizeAllow {
		switch r.options.OversizePolicy {
		case OversizeSkipServing:
			return nil
		default:
			return fmt.Errorf("lazyassets: generated asset %q is %d bytes, larger than max %d", assetPath, len(content), r.options.MaxAssetSize)
		}
	}

	logical, err := r.publicPath(assetPath)
	if err != nil {
		return err
	}
	if !opts.replace {
		if _, exists := r.logical[logical]; exists {
			return fmt.Errorf("lazyassets: asset %q already registered", logical)
		}
	}
	if r.options.Development {
		asset := &Asset{
			Path:        logical,
			ContentType: firstNonEmpty(opts.contentType, contentTypeForName(logical, content)),
			Size:        int64(len(content)),
			Source:      opts.source,
			Generated:   generated,
			content:     append([]byte(nil), content...),
			raw:         append([]byte(nil), content...),
		}
		r.register(asset, opts.replace)
		return nil
	}

	digest := newHash(content)
	contentType := firstNonEmpty(opts.contentType, contentTypeForName(logical, content))
	asset := &Asset{
		Path:        logical,
		Permanent:   withHash(logical, digest.Short(r.options.HashLength)),
		ContentType: contentType,
		Size:        int64(len(content)),
		Hash:        digest.Hex(),
		ETag:        digest.ETag(),
		Integrity:   digest.Integrity(),
		Source:      opts.source,
		Generated:   generated,
		content:     content,
		raw:         append([]byte(nil), content...),
	}
	r.register(asset, opts.replace)
	return nil
}

func (r *Registry) addOpen(assetPath string, open OpenFunc, generated bool, opts assetOptions) error {
	logical, err := r.publicPath(assetPath)
	if err != nil {
		return err
	}
	if !opts.replace {
		if _, exists := r.logical[logical]; exists {
			return fmt.Errorf("lazyassets: asset %q already registered", logical)
		}
	}
	asset := &Asset{
		Path:        logical,
		ContentType: firstNonEmpty(opts.contentType, contentTypeForName(logical, nil)),
		Size:        -1,
		Source:      opts.source,
		Generated:   generated,
		open:        open,
	}
	r.register(asset, opts.replace)
	return nil
}

func (r *Registry) addIgnored(assetPath string, size int64, open OpenFunc, opts assetOptions) error {
	logical, err := r.publicPath(assetPath)
	if err != nil {
		return err
	}
	if !opts.replace {
		if _, exists := r.logical[logical]; exists {
			return fmt.Errorf("lazyassets: asset %q already registered", logical)
		}
	}
	asset := &Asset{
		Path:        logical,
		ContentType: firstNonEmpty(opts.contentType, contentTypeForName(logical, nil)),
		Size:        size,
		Source:      opts.source,
		Ignored:     true,
		open:        open,
	}
	r.register(asset, opts.replace)
	return nil
}

func (r *Registry) register(asset *Asset, replace bool) {
	if existing, ok := r.logical[asset.Path]; ok && replace {
		r.remove(existing)
	}
	r.logical[asset.Path] = asset
	if asset.Permanent != "" {
		r.permanent[asset.Permanent] = asset
	}
	r.assets = append(r.assets, asset)
	r.rewriteCSSURLs()
}

func (r *Registry) remove(asset *Asset) {
	delete(r.logical, asset.Path)
	if asset.Permanent != "" {
		delete(r.permanent, asset.Permanent)
	}
	for index, candidate := range r.assets {
		if candidate == asset {
			r.assets = append(r.assets[:index], r.assets[index+1:]...)
			return
		}
	}
}

var cssURLPattern = regexp.MustCompile(`url\(\s*(['"]?)([^'")]+)['"]?\s*\)`)

func (r *Registry) rewriteCSSURLs() {
	if !r.options.RewriteCSSURLs {
		return
	}
	for _, asset := range r.assets {
		if asset.Ignored || !strings.HasPrefix(asset.ContentType, "text/css") || len(asset.raw) == 0 {
			continue
		}
		rewritten := []byte(r.rewriteCSSContent(asset, string(asset.raw)))
		if bytes.Equal(rewritten, asset.content) {
			continue
		}
		r.updateAssetContent(asset, rewritten)
	}
}

func (r *Registry) rewriteCSSContent(stylesheet *Asset, css string) string {
	return cssURLPattern.ReplaceAllStringFunc(css, func(match string) string {
		parts := cssURLPattern.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		quote := parts[1]
		ref := strings.TrimSpace(parts[2])
		if skipCSSURL(ref) {
			return match
		}
		refPath, suffix := splitURLSuffix(ref)
		if refPath == "" {
			return match
		}
		targetPath := refPath
		if !strings.HasPrefix(refPath, "/") {
			targetPath = path.Join(path.Dir(stylesheet.Path), refPath)
		}
		target, ok := r.findLogical(targetPath)
		if !ok || target == stylesheet || target.Permanent == "" {
			return match
		}
		return "url(" + quote + r.assetURL(target.Permanent) + suffix + quote + ")"
	})
}

func (r *Registry) assetURL(assetPath string) string {
	if r.options.BaseURL == "" {
		return assetPath
	}
	return r.options.BaseURL + "/" + strings.TrimPrefix(assetPath, "/")
}

func (r *Registry) updateAssetContent(asset *Asset, content []byte) {
	if asset.Permanent != "" {
		delete(r.permanent, asset.Permanent)
	}
	digest := newHash(content)
	asset.content = content
	asset.Size = int64(len(content))
	asset.Hash = digest.Hex()
	asset.ETag = digest.ETag()
	asset.Integrity = digest.Integrity()
	asset.Permanent = withHash(asset.Path, digest.Short(r.options.HashLength))
	r.permanent[asset.Permanent] = asset
}

func skipCSSURL(ref string) bool {
	ref = strings.TrimSpace(ref)
	if ref == "" || strings.HasPrefix(ref, "#") || strings.HasPrefix(ref, "//") {
		return true
	}
	colon := strings.Index(ref, ":")
	slash := strings.Index(ref, "/")
	return colon >= 0 && (slash == -1 || colon < slash)
}

func splitURLSuffix(ref string) (string, string) {
	index := strings.IndexAny(ref, "?#")
	if index == -1 {
		return ref, ""
	}
	return ref[:index], ref[index:]
}

func (r *Registry) serveAsset(w http.ResponseWriter, req *http.Request, asset *Asset, permanent bool) {
	if asset.ContentType != "" {
		w.Header().Set("Content-Type", asset.ContentType)
	}
	if asset.Size >= 0 {
		w.Header().Set("Content-Length", fmt.Sprint(asset.Size))
	}
	if asset.ETag != "" {
		w.Header().Set("ETag", asset.ETag)
	}
	if permanent {
		if r.options.PermanentCache != "" {
			w.Header().Set("Cache-Control", string(r.options.PermanentCache))
		}
	} else if r.options.LogicalCache != "" {
		w.Header().Set("Cache-Control", string(r.options.LogicalCache))
	}
	if asset.ETag != "" && etagMatches(req.Header.Get("If-None-Match"), asset.ETag) {
		w.Header().Del("Content-Length")
		w.Header().Del("Content-Type")
		w.WriteHeader(http.StatusNotModified)
		return
	}
	if req.Method == http.MethodHead {
		return
	}
	reader, err := asset.Open()
	if err != nil {
		http.Error(w, fmt.Errorf("open asset: %w", err).Error(), http.StatusInternalServerError)
		return
	}
	defer reader.Close()
	_, _ = io.Copy(w, reader)
}

func (r *Registry) unpackAsset(dir string, assetPath string, asset *Asset) error {
	target := path.Join(dir, strings.TrimPrefix(assetPath, "/"))
	if err := os.MkdirAll(path.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create asset directory: %w", err)
	}
	reader, err := asset.Open()
	if err != nil {
		return fmt.Errorf("open %s: %w", assetPath, err)
	}
	defer reader.Close()

	file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("create %s: %w", target, err)
	}
	defer file.Close()
	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("write %s: %w", target, err)
	}
	return nil
}

// Open opens the asset content that Handler, Upload, and Unpack serve or write.
func (a *Asset) Open() (io.ReadCloser, error) {
	if a.open != nil {
		return a.open()
	}
	return io.NopCloser(bytes.NewReader(a.content)), nil
}

func (a *Asset) snapshot() Asset {
	return Asset{
		Path:        a.Path,
		Permanent:   a.Permanent,
		ContentType: a.ContentType,
		Size:        a.Size,
		Hash:        a.Hash,
		ETag:        a.ETag,
		Integrity:   a.Integrity,
		Source:      a.Source,
		Generated:   a.Generated,
		Ignored:     a.Ignored,
	}
}

func (r *Registry) findLogical(assetPath string) (*Asset, bool) {
	logical, err := r.publicPath(assetPath)
	if err != nil {
		return nil, false
	}
	asset, ok := r.logical[logical]
	return asset, ok
}

func (r *Registry) findRequest(requestPath string) (*Asset, bool, bool) {
	logical, ok := requestAssetPaths(requestPath)
	if !ok {
		return nil, false, false
	}
	if asset, ok := r.logical[logical]; ok {
		return asset, false, true
	}
	if asset, ok := r.permanent[logical]; ok {
		return asset, true, true
	}
	return nil, false, false
}

func (r *Registry) publicPath(assetPath string) (string, error) {
	normalized, err := normalizeAssetPath(assetPath)
	if err != nil {
		return "", err
	}
	if r.options.URLPrefix == "/" {
		return normalized, nil
	}
	if strings.HasPrefix(normalized, r.options.URLPrefix+"/") || normalized == r.options.URLPrefix {
		return normalized, nil
	}
	return path.Join(r.options.URLPrefix, normalized), nil
}

func requestAssetPaths(requestPath string) (string, bool) {
	normalized, err := normalizeAssetPath(requestPath)
	if err != nil {
		return "", false
	}
	if normalized == "/" {
		return "/index.html", true
	}
	if strings.HasSuffix(requestPath, "/") {
		return path.Join(normalized, "index.html"), true
	}
	return normalized, true
}

func normalizeAssetPath(assetPath string) (string, error) {
	if strings.TrimSpace(assetPath) == "" {
		return "", fmt.Errorf("lazyassets: asset path is required")
	}
	assetPath = "/" + strings.TrimPrefix(assetPath, "/")
	for _, segment := range strings.Split(assetPath, "/") {
		if segment == ".." {
			return "", fmt.Errorf("lazyassets: asset path %q escapes root", assetPath)
		}
	}
	normalized := path.Clean(assetPath)
	if normalized == "." {
		normalized = "/"
	}
	return normalized, nil
}

func normalizePrefix(prefix string) string {
	prefix = "/" + strings.Trim(prefix, "/")
	if prefix == "/" {
		return "/"
	}
	return path.Clean(prefix)
}

func readAsset(reader io.Reader, maxSize int64, allowOversize bool) ([]byte, error) {
	if maxSize < 0 || allowOversize {
		return io.ReadAll(reader)
	}
	data, err := io.ReadAll(io.LimitReader(reader, maxSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxSize {
		return nil, fmt.Errorf("lazyassets: generated asset is %d bytes, larger than max %d", len(data), maxSize)
	}
	return data, nil
}

func contentTypeForName(name string, data []byte) string {
	if contentType := mime.TypeByExtension(path.Ext(name)); contentType != "" {
		return contentType
	}
	if len(data) != 0 {
		return http.DetectContentType(data)
	}
	return "application/octet-stream"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
