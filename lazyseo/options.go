package lazyseo

import (
	"strings"
	"time"
)

func Title(value string) Option {
	return func(meta *Meta) {
		meta.Title = strings.TrimSpace(value)
	}
}

func SiteName(value string) Option {
	return func(meta *Meta) {
		meta.SiteName = strings.TrimSpace(value)
	}
}

func Description(value string) Option {
	return func(meta *Meta) {
		meta.Description = strings.TrimSpace(value)
	}
}

func Author(value string) Option {
	return func(meta *Meta) {
		meta.Author = strings.TrimSpace(value)
	}
}

func Language(value string) Option {
	return func(meta *Meta) {
		meta.Language = strings.TrimSpace(value)
	}
}

func URL(value string) Option {
	return func(meta *Meta) {
		meta.URL = strings.TrimSpace(value)
	}
}

func Canonical(value string) Option {
	return func(meta *Meta) {
		meta.Canonical = strings.TrimSpace(value)
	}
}

func AlternateURL(language, url string) Option {
	return func(meta *Meta) {
		alternate := Alternate{
			Language: strings.TrimSpace(language),
			URL:      strings.TrimSpace(url),
		}
		if alternate.Language == "" || alternate.URL == "" {
			return
		}
		meta.Alternates = append(meta.Alternates, alternate)
	}
}

func AlternateLink(alternate Alternate) Option {
	return func(meta *Meta) {
		alternate.Language = strings.TrimSpace(alternate.Language)
		alternate.URL = strings.TrimSpace(alternate.URL)
		alternate.Media = strings.TrimSpace(alternate.Media)
		alternate.Type = strings.TrimSpace(alternate.Type)
		alternate.Title = strings.TrimSpace(alternate.Title)
		if alternate.URL == "" {
			return
		}
		meta.Alternates = append(meta.Alternates, alternate)
	}
}

func Image(value string) Option {
	return func(meta *Meta) {
		meta.Image = strings.TrimSpace(value)
	}
}

func ImageAlt(value string) Option {
	return func(meta *Meta) {
		meta.ImageAlt = strings.TrimSpace(value)
	}
}

func OpenGraphData(value OpenGraph) Option {
	return func(meta *Meta) {
		meta.OpenGraph = normalizeOpenGraph(value)
	}
}

func Kind(kind PageKind) Option {
	return func(meta *Meta) {
		meta.Type = strings.TrimSpace(kind.OpenGraph)
		meta.SchemaType = strings.TrimSpace(kind.Schema)
	}
}

func Type(value string) Option {
	return OpenGraphType(value)
}

func OpenGraphType(value string) Option {
	return func(meta *Meta) {
		meta.Type = strings.TrimSpace(value)
	}
}

func SchemaType(value string) Option {
	return func(meta *Meta) {
		meta.SchemaType = strings.TrimSpace(value)
	}
}

func Locale(value string) Option {
	return func(meta *Meta) {
		meta.Locale = strings.TrimSpace(value)
	}
}

func TwitterCardType(value string) Option {
	return func(meta *Meta) {
		meta.TwitterType = strings.TrimSpace(value)
	}
}

func TwitterCardData(value TwitterCard) Option {
	return func(meta *Meta) {
		meta.Twitter = normalizeTwitterCard(value)
	}
}

func JSONLD(value any) Option {
	return func(meta *Meta) {
		if value == nil {
			return
		}
		meta.JSONLD = append(meta.JSONLD, value)
	}
}

func UpdatedTime(value time.Time) Option {
	return func(meta *Meta) {
		meta.UpdatedTime = value
	}
}

func PublishedTime(value time.Time) Option {
	return func(meta *Meta) {
		meta.PublishedTime = value
	}
}

func LastUpdated(value time.Time) Option {
	return UpdatedTime(value)
}
