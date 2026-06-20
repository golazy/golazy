package lazyapp

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
	"time"
)

func (m *metadataFiles) renderSitemap() ([]byte, time.Time, error) {
	urls, err := m.sitemapURLs()
	if err != nil {
		return nil, time.Time{}, err
	}

	var updated time.Time
	doc := sitemapURLSet{
		XMLNS:   "http://www.sitemaps.org/schemas/sitemap/0.9",
		XHTML:   "http://www.w3.org/1999/xhtml",
		Entries: make([]sitemapEntry, 0, len(urls)),
	}
	for _, entry := range urls {
		location := absoluteURL(m.sitemap.BaseURL, entry.Location)
		if location == "" {
			continue
		}
		updated = latestTime(updated, entry.LastUpdated)
		doc.Entries = append(doc.Entries, sitemapEntry{
			Location:   location,
			LastMod:    sitemapTime(entry.LastUpdated),
			ChangeFreq: strings.TrimSpace(entry.ChangeFreq),
			Priority:   sitemapPriority(entry.Priority),
			Links:      sitemapLinks(m.sitemap.BaseURL, entry.Alternates),
		})
	}

	body, err := encodeSitemap(doc)
	return body, updated, err
}

func (m *metadataFiles) sitemapURLs() ([]SitemapURL, error) {
	urls := append([]SitemapURL(nil), m.sitemap.URLs...)
	for _, source := range m.sitemap.Sources {
		if source == nil {
			return nil, fmt.Errorf("lazyapp: sitemap source is nil")
		}
		sourceURLs, err := source.SitemapURLs()
		if err != nil {
			return nil, fmt.Errorf("lazyapp: sitemap source: %w", err)
		}
		urls = append(urls, sourceURLs...)
	}
	sort.SliceStable(urls, func(i, j int) bool {
		return urls[i].Location < urls[j].Location
	})
	return urls, nil
}

func encodeSitemap(doc sitemapURLSet) ([]byte, error) {
	var out bytes.Buffer
	out.WriteString(xml.Header)
	encoder := xml.NewEncoder(&out)
	encoder.Indent("", "  ")
	if err := encoder.Encode(doc); err != nil {
		return nil, err
	}
	out.WriteByte('\n')
	return out.Bytes(), nil
}

type sitemapURLSet struct {
	XMLName xml.Name       `xml:"urlset"`
	XMLNS   string         `xml:"xmlns,attr"`
	XHTML   string         `xml:"xmlns:xhtml,attr,omitempty"`
	Entries []sitemapEntry `xml:"url"`
}

type sitemapEntry struct {
	Location   string        `xml:"loc"`
	LastMod    string        `xml:"lastmod,omitempty"`
	ChangeFreq string        `xml:"changefreq,omitempty"`
	Priority   string        `xml:"priority,omitempty"`
	Links      []sitemapLink `xml:"xhtml:link,omitempty"`
}

type sitemapLink struct {
	Rel      string `xml:"rel,attr"`
	Language string `xml:"hreflang,attr"`
	Location string `xml:"href,attr"`
}

func sitemapLinks(baseURL string, alternates []SitemapAlternate) []sitemapLink {
	links := make([]sitemapLink, 0, len(alternates))
	for _, alternate := range alternates {
		language := strings.TrimSpace(alternate.Language)
		location := absoluteURL(baseURL, alternate.Location)
		if language == "" || location == "" {
			continue
		}
		links = append(links, sitemapLink{Rel: "alternate", Language: language, Location: location})
	}
	return links
}

func sitemapTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format("2006-01-02")
}

func sitemapPriority(value float64) string {
	if value <= 0 {
		return ""
	}
	if value > 1 {
		value = 1
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", value), "0"), ".")
}
