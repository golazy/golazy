package lazyseo

import "golazy.dev/lazyseo/jsonld"

// DefaultJSONLD builds a conventional schema.org value from metadata.
func DefaultJSONLD(meta Meta) any {
	if meta.Title == "" {
		return nil
	}
	schemaType := firstNonEmpty(meta.SchemaType, WebPage.Schema)
	url := firstNonEmpty(meta.Canonical, meta.URL)
	switch schemaType {
	case Article.Schema, BlogPosting.Schema, NewsArticle.Schema:
		article := jsonld.NewArticle(meta.Title)
		article.Type = schemaType
		article.Description = meta.Description
		article.URL = url
		article.Image = meta.Image
		article.DateModified = jsonld.Date(meta.UpdatedTime)
		return article
	case WebSite.Schema:
		site := jsonld.NewWebSite(meta.Title)
		site.URL = url
		return site
	case Organization.Schema:
		organization := jsonld.NewOrganization(meta.Title)
		organization.URL = url
		organization.Logo = meta.Image
		return organization
	default:
		page := jsonld.NewWebPage(meta.Title)
		page.Type = schemaType
		page.Description = meta.Description
		page.URL = url
		if meta.Image != "" {
			page.PrimaryImage = jsonld.NewImageObject(meta.Image)
		}
		return page
	}
}
