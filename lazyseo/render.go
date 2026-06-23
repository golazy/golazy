package lazyseo

import (
	"encoding/json"
	"html"
	"strconv"
	"strings"
	"time"
)

func render(meta Meta) string {
	var out strings.Builder
	if meta.Title != "" {
		element(&out, "title", displayTitle(meta))
	}
	if meta.Description != "" {
		name(&out, "description", meta.Description)
	}
	if meta.Canonical != "" {
		link(&out, "canonical", meta.Canonical)
	}
	for _, alternate := range meta.Alternates {
		alternateLink(&out, alternate)
	}
	if meta.Author != "" {
		name(&out, "author", meta.Author)
	}
	renderOpenGraph(&out, meta)
	renderTwitterCard(&out, meta)
	if !meta.PublishedTime.IsZero() {
		property(&out, "article:published_time", meta.PublishedTime.Format(time.RFC3339))
	}
	if !meta.UpdatedTime.IsZero() {
		property(&out, "article:modified_time", meta.UpdatedTime.Format(time.RFC3339))
	}
	for _, value := range meta.JSONLD {
		jsonldScript(&out, value)
	}
	return out.String()
}

func renderOpenGraph(out *strings.Builder, meta Meta) {
	if title := firstNonEmpty(meta.OpenGraph.Title, displayTitle(meta)); title != "" {
		property(out, "og:title", title)
	}
	if description := firstNonEmpty(meta.OpenGraph.Description, meta.Description); description != "" {
		property(out, "og:description", description)
	}
	if siteName := firstNonEmpty(meta.OpenGraph.SiteName, meta.SiteName); siteName != "" {
		property(out, "og:site_name", siteName)
	}
	if url := firstNonEmpty(meta.OpenGraph.URL, openGraphURL(meta)); url != "" {
		property(out, "og:url", url)
	}
	if image := firstNonEmpty(meta.OpenGraph.Image, meta.Image); image != "" {
		property(out, "og:image", image)
		if strings.HasPrefix(image, "https://") {
			property(out, "og:image:secure_url", image)
		}
		if meta.OpenGraph.ImageWidth > 0 {
			property(out, "og:image:width", strconv.Itoa(meta.OpenGraph.ImageWidth))
		}
		if meta.OpenGraph.ImageHeight > 0 {
			property(out, "og:image:height", strconv.Itoa(meta.OpenGraph.ImageHeight))
		}
		if alt := firstNonEmpty(meta.OpenGraph.ImageAlt, meta.ImageAlt); alt != "" {
			property(out, "og:image:alt", alt)
		}
	}
	if typ := firstNonEmpty(meta.OpenGraph.Type, meta.Type); typ != "" {
		property(out, "og:type", typ)
	}
	if locale := firstNonEmpty(meta.OpenGraph.Locale, meta.Locale); locale != "" {
		property(out, "og:locale", locale)
	}
}

func renderTwitterCard(out *strings.Builder, meta Meta) {
	if card := firstNonEmpty(meta.Twitter.Card, meta.TwitterType); card != "" {
		name(out, "twitter:card", card)
	}
	if title := firstNonEmpty(meta.Twitter.Title, displayTitle(meta)); title != "" {
		name(out, "twitter:title", title)
	}
	if description := firstNonEmpty(meta.Twitter.Description, meta.Description); description != "" {
		name(out, "twitter:description", description)
	}
	if image := firstNonEmpty(meta.Twitter.Image, meta.Image); image != "" {
		name(out, "twitter:image", image)
		if alt := firstNonEmpty(meta.Twitter.ImageAlt, meta.ImageAlt); alt != "" {
			name(out, "twitter:image:alt", alt)
		}
	}
	if site := meta.Twitter.Site; site != "" {
		name(out, "twitter:site", site)
	}
	if creator := meta.Twitter.Creator; creator != "" {
		name(out, "twitter:creator", creator)
	}
}

func displayTitle(meta Meta) string {
	if meta.Title == "" {
		return ""
	}
	title := meta.Title
	if meta.SiteName != "" && title != meta.SiteName && !strings.Contains(title, "|") && !strings.HasSuffix(title, " - "+meta.SiteName) {
		title += " - " + meta.SiteName
	}
	return title
}

func openGraphURL(meta Meta) string {
	if meta.URL != "" {
		return meta.URL
	}
	return meta.Canonical
}

func element(out *strings.Builder, tag, value string) {
	out.WriteString("<")
	out.WriteString(tag)
	out.WriteString(">")
	out.WriteString(html.EscapeString(value))
	out.WriteString("</")
	out.WriteString(tag)
	out.WriteString(">\n")
}

func name(out *strings.Builder, key, value string) {
	meta(out, "name", key, value)
}

func property(out *strings.Builder, key, value string) {
	meta(out, "property", key, value)
}

func link(out *strings.Builder, rel, href string) {
	out.WriteString(`<link rel="`)
	out.WriteString(html.EscapeString(rel))
	out.WriteString(`" href="`)
	out.WriteString(html.EscapeString(href))
	out.WriteString(`">`)
	out.WriteString("\n")
}

func alternateLink(out *strings.Builder, alternate Alternate) {
	out.WriteString(`<link rel="alternate"`)
	if alternate.Language != "" {
		out.WriteString(` hreflang="`)
		out.WriteString(html.EscapeString(alternate.Language))
		out.WriteString(`"`)
	}
	if alternate.Media != "" {
		out.WriteString(` media="`)
		out.WriteString(html.EscapeString(alternate.Media))
		out.WriteString(`"`)
	}
	if alternate.Type != "" {
		out.WriteString(` type="`)
		out.WriteString(html.EscapeString(alternate.Type))
		out.WriteString(`"`)
	}
	if alternate.Title != "" {
		out.WriteString(` title="`)
		out.WriteString(html.EscapeString(alternate.Title))
		out.WriteString(`"`)
	}
	out.WriteString(` href="`)
	out.WriteString(html.EscapeString(alternate.URL))
	out.WriteString(`">`)
	out.WriteString("\n")
}

func jsonldScript(out *strings.Builder, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	out.WriteString(`<script type="application/ld+json">`)
	out.Write(data)
	out.WriteString(`</script>`)
	out.WriteString("\n")
}

func meta(out *strings.Builder, attr, key, value string) {
	out.WriteString(`<meta `)
	out.WriteString(attr)
	out.WriteString(`="`)
	out.WriteString(html.EscapeString(key))
	out.WriteString(`" content="`)
	out.WriteString(html.EscapeString(value))
	out.WriteString(`">`)
	out.WriteString("\n")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
