package lazyseo

import "strings"

// OpenGraph overrides metadata emitted for Open Graph tags.
type OpenGraph struct {
	Title       string
	Description string
	URL         string
	Image       string
	ImageAlt    string
	ImageWidth  int
	ImageHeight int
	Type        string
	SiteName    string
	Locale      string
}

// TwitterCard overrides metadata emitted for Twitter card tags.
type TwitterCard struct {
	Card        string
	Title       string
	Description string
	Image       string
	ImageAlt    string
	Site        string
	Creator     string
}

func normalizeOpenGraph(value OpenGraph) OpenGraph {
	value.Title = strings.TrimSpace(value.Title)
	value.Description = strings.TrimSpace(value.Description)
	value.URL = strings.TrimSpace(value.URL)
	value.Image = strings.TrimSpace(value.Image)
	value.ImageAlt = strings.TrimSpace(value.ImageAlt)
	value.Type = strings.TrimSpace(value.Type)
	value.SiteName = strings.TrimSpace(value.SiteName)
	value.Locale = strings.TrimSpace(value.Locale)
	return value
}

func normalizeTwitterCard(value TwitterCard) TwitterCard {
	value.Card = strings.TrimSpace(value.Card)
	value.Title = strings.TrimSpace(value.Title)
	value.Description = strings.TrimSpace(value.Description)
	value.Image = strings.TrimSpace(value.Image)
	value.ImageAlt = strings.TrimSpace(value.ImageAlt)
	value.Site = strings.TrimSpace(value.Site)
	value.Creator = strings.TrimSpace(value.Creator)
	return value
}

func isZeroOpenGraph(value OpenGraph) bool {
	return value == OpenGraph{}
}

func isZeroTwitterCard(value TwitterCard) bool {
	return value == TwitterCard{}
}
