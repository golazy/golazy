package lazycontroller

import (
	"time"

	"golazy.dev/lazyseo"
)

func (b *Base) Metadata(model any) *lazyseo.Meta {
	meta := b.ensureSEO()
	if model == nil {
		return meta
	}
	applyMetadataCore(meta, model)
	applyMetadataSocial(meta, model)
	applyMetadataJSONLD(meta, model)
	return meta
}

func applyMetadataCore(meta *lazyseo.Meta, model any) {
	if provider, ok := model.(interface{ Title() string }); ok {
		lazyseo.Title(provider.Title())(meta)
	}
	if provider, ok := model.(interface{ Description() string }); ok {
		lazyseo.Description(provider.Description())(meta)
	}
	if provider, ok := model.(interface{ Author() string }); ok {
		lazyseo.Author(provider.Author())(meta)
	}
	if provider, ok := model.(interface{ Language() string }); ok {
		lazyseo.Language(provider.Language())(meta)
	}
	if provider, ok := model.(interface{ URL() string }); ok {
		lazyseo.URL(provider.URL())(meta)
	}
	if provider, ok := model.(interface{ Canonical() string }); ok {
		lazyseo.Canonical(provider.Canonical())(meta)
	}
	if provider, ok := model.(interface{ Image() string }); ok {
		lazyseo.Image(provider.Image())(meta)
	}
	if provider, ok := model.(interface{ ImageAlt() string }); ok {
		lazyseo.ImageAlt(provider.ImageAlt())(meta)
	}
	if provider, ok := model.(interface{ Kind() lazyseo.PageKind }); ok {
		lazyseo.Kind(provider.Kind())(meta)
	}
	if provider, ok := model.(interface{ PublishedTime() time.Time }); ok {
		lazyseo.PublishedTime(provider.PublishedTime())(meta)
	}
	if provider, ok := model.(interface{ LastUpdated() time.Time }); ok {
		lazyseo.LastUpdated(provider.LastUpdated())(meta)
	}
	if provider, ok := model.(interface{ Alternates() []lazyseo.Alternate }); ok {
		for _, alternate := range provider.Alternates() {
			lazyseo.AlternateLink(alternate)(meta)
		}
	}
}

func applyMetadataSocial(meta *lazyseo.Meta, model any) {
	if provider, ok := model.(interface{ OpenGraph() lazyseo.OpenGraph }); ok {
		lazyseo.OpenGraphData(provider.OpenGraph())(meta)
	}
	if provider, ok := model.(interface{ TwitterCard() lazyseo.TwitterCard }); ok {
		lazyseo.TwitterCardData(provider.TwitterCard())(meta)
	}
}

func applyMetadataJSONLD(meta *lazyseo.Meta, model any) {
	if provider, ok := model.(interface{ JSONLD() any }); ok {
		appendJSONLD(meta, provider.JSONLD())
	}
	if len(meta.JSONLD) == 0 {
		appendJSONLD(meta, lazyseo.DefaultJSONLD(*meta))
	}
}

func appendJSONLD(meta *lazyseo.Meta, value any) {
	if value == nil {
		return
	}
	if values, ok := value.([]any); ok {
		for _, item := range values {
			if item != nil {
				meta.JSONLD = append(meta.JSONLD, item)
			}
		}
		return
	}
	meta.JSONLD = append(meta.JSONLD, value)
}
