package lazyassets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
)

const (
	defaultMaxAssetSize = 5 << 20
	defaultHashLength   = 12
)

type OpenFunc func() (io.ReadCloser, error)

type OversizePolicy int

const (
	OversizeIgnoreForPipeline OversizePolicy = iota
	OversizeError
	OversizeAllow
	OversizeSkipServing
)

type CachePolicy string

type Options struct {
	URLPrefix      string
	MaxAssetSize   int64
	OversizePolicy OversizePolicy
	HashLength     int
	LogicalCache   CachePolicy
	PermanentCache CachePolicy
}

type Option func(*Options)

func WithURLPrefix(prefix string) Option {
	return func(options *Options) {
		options.URLPrefix = prefix
	}
}

func WithMaxAssetSize(size int64) Option {
	return func(options *Options) {
		options.MaxAssetSize = size
	}
}

func WithOversizePolicy(policy OversizePolicy) Option {
	return func(options *Options) {
		options.OversizePolicy = policy
	}
}

func WithHashLength(length int) Option {
	return func(options *Options) {
		options.HashLength = length
	}
}

func WithCachePolicies(logical, permanent CachePolicy) Option {
	return func(options *Options) {
		options.LogicalCache = logical
		options.PermanentCache = permanent
	}
}

type Source interface {
	Assets(*Registry) error
}

type Registry struct {
	options   Options
	logical   map[string]*Asset
	permanent map[string]*Asset
	assets    []*Asset
}

type Asset struct {
	Path        string `json:"path"`
	Permanent   string `json:"permanent,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Size        int64  `json:"size"`
	Hash        string `json:"hash,omitempty"`
	ETag        string `json:"etag,omitempty"`
	Integrity   string `json:"integrity,omitempty"`
	Source      string `json:"source,omitempty"`
	Generated   bool   `json:"generated,omitempty"`
	Ignored     bool   `json:"ignored,omitempty"`

	content []byte
	open    OpenFunc
}

type Manifest struct {
	Assets []Asset `json:"assets"`
}

type SourceOption func(*sourceOptions)

type sourceOptions struct {
	source  string
	replace bool
}

func SourceName(name string) SourceOption {
	return func(options *sourceOptions) {
		options.source = name
	}
}

func Replace() SourceOption {
	return func(options *sourceOptions) {
		options.replace = true
	}
}

type AssetOption func(*assetOptions)

type assetOptions struct {
	contentType string
	source      string
	replace     bool
}

func ContentType(contentType string) AssetOption {
	return func(options *assetOptions) {
		options.contentType = contentType
	}
}

func AssetSource(name string) AssetOption {
	return func(options *assetOptions) {
		options.source = name
	}
}

func ReplaceAsset() AssetOption {
	return func(options *assetOptions) {
		options.replace = true
	}
}

func New(options ...Option) *Registry {
	config := Options{
		URLPrefix:      "/",
		MaxAssetSize:   defaultMaxAssetSize,
		HashLength:     defaultHashLength,
		LogicalCache:   "public, max-age=0, must-revalidate",
		PermanentCache: "public, max-age=31536000, immutable",
	}
	for _, option := range options {
		option(&config)
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

func (r *Registry) Path(assetPath string) (string, error) {
	asset, ok := r.findLogical(assetPath)
	if !ok {
		return "", fmt.Errorf("lazyassets: asset %q not found", assetPath)
	}
	if asset.Permanent != "" {
		return asset.Permanent, nil
	}
	return asset.Path, nil
}

func (r *Registry) MustPath(assetPath string) string {
	result, err := r.Path(assetPath)
	if err != nil {
		panic(err)
	}
	return result
}

func (r *Registry) Integrity(assetPath string) (string, error) {
	asset, ok := r.findLogical(assetPath)
	if !ok {
		return "", fmt.Errorf("lazyassets: asset %q not found", assetPath)
	}
	return asset.Integrity, nil
}

func (r *Registry) Manifest() Manifest {
	manifest := Manifest{Assets: make([]Asset, 0, len(r.assets))}
	for _, asset := range r.assets {
		manifest.Assets = append(manifest.Assets, asset.snapshot())
	}
	return manifest
}

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

func (r *Registry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.Handler(http.NotFoundHandler()).ServeHTTP(w, req)
}

func (r *Registry) Helpers() map[string]any {
	return map[string]any{
		"asset_path": func(path string) (string, error) {
			return r.Path(path)
		},
		"asset_integrity": func(path string) (string, error) {
			return r.Integrity(path)
		},
		"permalink": func(path string) (string, error) {
			return r.Path(path)
		},
	}
}

type UnpackMode int

const (
	UnpackBoth UnpackMode = iota
	UnpackLogical
	UnpackPermanent
)

type UnpackOption func(*unpackOptions)

type unpackOptions struct {
	mode UnpackMode
}

func WithUnpackMode(mode UnpackMode) UnpackOption {
	return func(options *unpackOptions) {
		options.mode = mode
	}
}

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
