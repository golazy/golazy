package asset_manager

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/andybalholm/brotli"
)

type blob struct {
	content    []byte
	isCompress bool
	sha        string
	mime       string
}

type assetPaths struct {
	fullPath string
	permlink string
}

type AssetManager struct {
	Prefix      string
	toPath      map[string]assetPaths
	byPath      map[string]*blob
	byPermalink map[string]*blob
}

func shouldCompress(mimeType string) bool {
	return strings.HasPrefix(mimeType, "text/") ||
		strings.HasPrefix(mimeType, "application/javascript") ||
		strings.HasPrefix(mimeType, "application/json") ||
		strings.HasPrefix(mimeType, "application/ld+json") ||
		strings.HasPrefix(mimeType, "image/svg+xml") ||
		strings.HasPrefix(mimeType, "application/xhtml+xml") ||
		strings.HasPrefix(mimeType, "application/xml") ||
		strings.HasPrefix(mimeType, "application/xhtml+xml") ||
		strings.HasPrefix(mimeType, "application/xhtml+xml") ||
		strings.HasPrefix(mimeType, "application/xhtml+xml")
}

func newAsset(content []byte, name, mimeType string) *blob {
	h := sha256.New()
	h.Write([]byte(content))
	sha := h.Sum(nil)

	// Extract the mimeType from the file extension
	if mimeType == "" {
		mimeType = mime.TypeByExtension(path.Ext(name))
	}

	// If the mimeType is still empty, get it from the content
	if mimeType == "" {
		mimeType = http.DetectContentType(content)
	}

	b := &blob{
		content: content,
		sha:     hex.EncodeToString(sha),
		mime:    mimeType,
	}

	// Compress and/or minify the content
	if shouldCompress(mimeType) {
		b.isCompress = true
		buf := &bytes.Buffer{}
		comp := brotli.NewWriterLevel(buf, brotli.BestCompression)
		comp.Write(content)
		comp.Close()
		b.content = buf.Bytes()
	}

	return b
}

func (am *AssetManager) Add(name, content string) {
	am.addBlob(name, newAsset([]byte(content), name, ""))
}

func (am *AssetManager) AddReader(name string, r io.Reader) {
	data, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}
	am.addBlob(name, newAsset([]byte(data), name, ""))
}

func (am *AssetManager) AddFS(files fs.ReadDirFS, prefix ...string) {
	// recursively add all files in the fs
	entries, err := files.ReadDir(path.Join(".", path.Join(prefix...)))
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			am.AddFS(files, entry.Name())
			continue
		}
		pPrefix := make([]string, len(prefix))
		copy(pPrefix, prefix)
		fullPath := path.Join(append(pPrefix, entry.Name())...)
		f, err := files.Open(fullPath)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		am.AddReader(fullPath, f)
	}

}

func (am *AssetManager) pathForName(name string) string {
	return "/" + path.Join(am.Prefix, name)
}

func (am *AssetManager) newAssetPaths(name, hash string) assetPaths {
	fp := strings.TrimPrefix(name, "/")
	if am.Prefix != "" {
		fp = path.Join(am.Prefix, fp)
	}
	fp = "/" + fp

	return assetPaths{
		fullPath: fp,
		permlink: fp[:len(fp)-len(path.Ext(fp))] + "-" + hash + path.Ext(fp),
	}
}

func (am *AssetManager) addBlob(name string, b *blob) {
	if am.byPath == nil {
		am.byPath = make(map[string]*blob)
		am.byPermalink = make(map[string]*blob)
		am.toPath = make(map[string]assetPaths)
	}

	if _, ok := am.toPath[name]; ok {
		panic("asset " + name + " already exists")
	}

	paths := am.newAssetPaths(name, b.sha)

	am.toPath[name] = paths

	if am.byPath[paths.fullPath] != nil {
		panic("asset " + name + " already exists")
	}
	am.byPath[paths.fullPath] = b

	if am.byPermalink[paths.permlink] != nil {
		panic("asset " + paths.permlink + " already exists")
	}
	am.byPermalink[paths.permlink] = b
}

func (am *AssetManager) FullPath(name string) string {
	return am.pathForName(name)
}

func (am *AssetManager) Permalink(name string) string {
	p := am.pathForName(name)
	b, ok := am.byPath[p]
	if !ok {
		panic("asset " + p + " not found")
	}
	ext := path.Ext(p)
	return p[:len(p)-len(ext)] + "-" + b.sha + ext
}

func (am *AssetManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b, ok := am.byPath[r.URL.Path]
	if !ok {
		b, ok = am.byPermalink[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "max-age=31536000")

	}
	w.Header().Set("Vary", "Accept-Encoding")
	w.Header().Set("ETag", `"`+b.sha+`"`)
	if b.mime != "" {
		w.Header().Set("Content-Type", b.mime)
	}

	if strings.Contains(r.Header.Get("If-None-Match"), b.sha) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	if !b.isCompress {
		w.Write(b.content)
		return
	}

	if strings.Contains(r.Header.Get("Accept-Encoding"), "br") {
		w.Header().Set("Content-Encoding", "br")
		w.Write(b.content)
		return
	}

	// Decompress the content
	reader := brotli.NewReader(bytes.NewReader(b.content))
	io.Copy(w, reader)

}

func (am *AssetManager) Has(path string) bool {
	return am.byPath[path] != nil || am.byPermalink[path] != nil
}
