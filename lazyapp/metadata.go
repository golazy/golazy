package lazyapp

import (
	"fmt"
	"net/http"
	"time"
)

type RobotsConfig struct {
	Disabled bool
	Rules    []RobotsRule
	Sitemaps []string
	Extra    []string
}

type RobotsRule struct {
	UserAgent  string
	Allow      []string
	Disallow   []string
	CrawlDelay string
}

type SitemapConfig struct {
	Disabled bool
	BaseURL  string
	URLs     []SitemapURL
	Sources  []SitemapSource
}

type SitemapURL struct {
	Location    string
	LastUpdated time.Time
	ChangeFreq  string
	Priority    float64
	Alternates  []SitemapAlternate
}

type SitemapAlternate struct {
	Language string
	Location string
}

type SitemapSource interface {
	SitemapURLs() ([]SitemapURL, error)
}

type SitemapSourceFunc func() ([]SitemapURL, error)

func (fn SitemapSourceFunc) SitemapURLs() ([]SitemapURL, error) {
	return fn()
}

type metadataFiles struct {
	robots  RobotsConfig
	sitemap SitemapConfig
	updated time.Time
	body    map[string][]byte
}

func newMetadataFiles(robots RobotsConfig, sitemap SitemapConfig) (*metadataFiles, error) {
	files := &metadataFiles{
		robots:  robots,
		sitemap: sitemap,
		body:    map[string][]byte{},
	}
	if !robots.Disabled {
		body, err := files.renderRobots()
		if err != nil {
			return nil, err
		}
		files.body["/robots.txt"] = body
	}
	if sitemap.enabled() {
		body, updated, err := files.renderSitemap()
		if err != nil {
			return nil, err
		}
		files.body["/sitemap.xml"] = body
		files.updated = latestTime(files.updated, updated)
	}
	return files, nil
}

func (m *metadataFiles) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := m.body[r.URL.Path]
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", metadataContentType(r.URL.Path))
		w.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")
		if m.checkFresh(w, r) {
			return
		}
		w.Header().Set("Content-Length", fmt.Sprint(len(body)))
		if r.Method == http.MethodHead {
			return
		}
		_, _ = w.Write(body)
	})
}

func (m *metadataFiles) checkFresh(w http.ResponseWriter, r *http.Request) bool {
	if m.updated.IsZero() {
		return false
	}
	w.Header().Set("Last-Modified", m.updated.UTC().Format(http.TimeFormat))
	since, err := time.Parse(http.TimeFormat, r.Header.Get("If-Modified-Since"))
	if err == nil && !m.updated.After(since) {
		w.WriteHeader(http.StatusNotModified)
		return true
	}
	return false
}

func metadataContentType(path string) string {
	if path == "/sitemap.xml" {
		return "application/xml; charset=utf-8"
	}
	return "text/plain; charset=utf-8"
}
