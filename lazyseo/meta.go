package lazyseo

import "time"

const htmlContentType = "text/html; charset=utf-8"

type setter interface {
	Set(string, any)
}

// Meta contains the metadata emitted by the seo view helper.
type Meta struct {
	Title       string
	SiteName    string
	Description string
	Author      string
	Language    string
	URL         string
	Canonical   string
	Alternates  []Alternate
	Image       string
	OpenGraph   OpenGraph
	Type        string
	SchemaType  string
	Locale      string
	Twitter     TwitterCard
	TwitterType string
	JSONLD      []any
	UpdatedTime time.Time
}

// Alternate describes an alternate URL for the current page.
type Alternate struct {
	Language string
	URL      string
	Media    string
	Type     string
	Title    string
}

// Option configures Meta values.
type Option func(*Meta)

// Set stores request-local SEO metadata on a controller.
func Set(controller setter, options ...Option) *Meta {
	meta := New(options...)
	controller.Set("seo", meta)
	return meta
}

// New builds Meta with the supplied options.
func New(options ...Option) *Meta {
	meta := &Meta{}
	for _, option := range options {
		option(meta)
	}
	return meta
}

func merge(defaults, current *Meta) Meta {
	var meta Meta
	if defaults != nil {
		meta = *defaults
	}
	if current == nil {
		return meta
	}
	if current.Title != "" {
		meta.Title = current.Title
	}
	if current.SiteName != "" {
		meta.SiteName = current.SiteName
	}
	if current.Description != "" {
		meta.Description = current.Description
	}
	if current.Author != "" {
		meta.Author = current.Author
	}
	if current.Language != "" {
		meta.Language = current.Language
	}
	if current.URL != "" {
		meta.URL = current.URL
	}
	if current.Canonical != "" {
		meta.Canonical = current.Canonical
	}
	if len(current.Alternates) > 0 {
		meta.Alternates = append([]Alternate(nil), current.Alternates...)
	}
	if current.Image != "" {
		meta.Image = current.Image
	}
	if !isZeroOpenGraph(current.OpenGraph) {
		meta.OpenGraph = current.OpenGraph
	}
	if current.Type != "" {
		meta.Type = current.Type
	}
	if current.SchemaType != "" {
		meta.SchemaType = current.SchemaType
	}
	if current.Locale != "" {
		meta.Locale = current.Locale
	}
	if !isZeroTwitterCard(current.Twitter) {
		meta.Twitter = current.Twitter
	}
	if current.TwitterType != "" {
		meta.TwitterType = current.TwitterType
	}
	if len(current.JSONLD) > 0 {
		meta.JSONLD = append([]any(nil), current.JSONLD...)
	}
	if !current.UpdatedTime.IsZero() {
		meta.UpdatedTime = current.UpdatedTime
	}
	return meta
}
