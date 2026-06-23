package lazycontroller

import (
	"time"

	"golazy.dev/lazyseo"
)

func (b *Base) SEO(options ...lazyseo.Option) *lazyseo.Meta {
	meta := b.ensureSEO()
	for _, option := range options {
		option(meta)
	}
	return meta
}

func (b *Base) Title(value string) *lazyseo.Meta {
	return b.SEO(lazyseo.Title(value))
}

func (b *Base) Description(value string) *lazyseo.Meta {
	return b.SEO(lazyseo.Description(value))
}

func (b *Base) Author(value string) *lazyseo.Meta {
	return b.SEO(lazyseo.Author(value))
}

func (b *Base) Language(value string) *lazyseo.Meta {
	return b.SEO(lazyseo.Language(value))
}

func (b *Base) URL(value string) *lazyseo.Meta {
	return b.SEO(lazyseo.URL(value))
}

func (b *Base) Canonical(value string) *lazyseo.Meta {
	return b.SEO(lazyseo.Canonical(value))
}

func (b *Base) Alternate(language, url string) *lazyseo.Meta {
	return b.SEO(lazyseo.AlternateURL(language, url))
}

func (b *Base) OpenGraph(value lazyseo.OpenGraph) *lazyseo.Meta {
	return b.SEO(lazyseo.OpenGraphData(value))
}

func (b *Base) TwitterCard(value lazyseo.TwitterCard) *lazyseo.Meta {
	return b.SEO(lazyseo.TwitterCardData(value))
}

func (b *Base) JSONLD(value any) *lazyseo.Meta {
	return b.SEO(lazyseo.JSONLD(value))
}

func (b *Base) SEOImage(value string) *lazyseo.Meta {
	return b.SEO(lazyseo.Image(value))
}

func (b *Base) SEOImageAlt(value string) *lazyseo.Meta {
	return b.SEO(lazyseo.ImageAlt(value))
}

func (b *Base) Type(value string) *lazyseo.Meta {
	return b.SEO(lazyseo.Type(value))
}

func (b *Base) Kind(kind lazyseo.PageKind) *lazyseo.Meta {
	return b.SEO(lazyseo.Kind(kind))
}

func (b *Base) OpenGraphType(value string) *lazyseo.Meta {
	return b.SEO(lazyseo.OpenGraphType(value))
}

func (b *Base) SchemaType(value string) *lazyseo.Meta {
	return b.SEO(lazyseo.SchemaType(value))
}

func (b *Base) Locale(value string) *lazyseo.Meta {
	return b.SEO(lazyseo.Locale(value))
}

func (b *Base) TwitterCardType(value string) *lazyseo.Meta {
	return b.SEO(lazyseo.TwitterCardType(value))
}

func (b *Base) LastUpdated(value time.Time) *lazyseo.Meta {
	return b.SEO(lazyseo.LastUpdated(value))
}

func (b *Base) PublishedTime(value time.Time) *lazyseo.Meta {
	return b.SEO(lazyseo.PublishedTime(value))
}

func (b *Base) ensureSEO() *lazyseo.Meta {
	if b.data == nil {
		b.data = make(map[string]any)
	}
	if meta, ok := b.data["seo"].(*lazyseo.Meta); ok {
		return meta
	}
	if meta, ok := b.data["seo"].(lazyseo.Meta); ok {
		b.data["seo"] = &meta
		return &meta
	}
	meta := &lazyseo.Meta{}
	b.data["seo"] = meta
	return meta
}
