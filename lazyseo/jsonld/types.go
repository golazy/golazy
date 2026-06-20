package jsonld

import "time"

const SchemaOrg = "https://schema.org"

func Date(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format("2006-01-02")
}

type Person struct {
	Context string `json:"@context,omitempty"`
	Type    string `json:"@type,omitempty"`
	Name    string `json:"name,omitempty"`
	URL     string `json:"url,omitempty"`
	Image   string `json:"image,omitempty"`
}

func NewPerson(name string) Person {
	return Person{Context: SchemaOrg, Type: "Person", Name: name}
}

type Organization struct {
	Context string `json:"@context,omitempty"`
	Type    string `json:"@type,omitempty"`
	Name    string `json:"name,omitempty"`
	URL     string `json:"url,omitempty"`
	Logo    string `json:"logo,omitempty"`
}

func NewOrganization(name string) Organization {
	return Organization{Context: SchemaOrg, Type: "Organization", Name: name}
}

type WebSite struct {
	Context   string `json:"@context,omitempty"`
	Type      string `json:"@type,omitempty"`
	Name      string `json:"name,omitempty"`
	URL       string `json:"url,omitempty"`
	Publisher any    `json:"publisher,omitempty"`
}

func NewWebSite(name string) WebSite {
	return WebSite{Context: SchemaOrg, Type: "WebSite", Name: name}
}

type WebPage struct {
	Context      string `json:"@context,omitempty"`
	Type         string `json:"@type,omitempty"`
	Name         string `json:"name,omitempty"`
	URL          string `json:"url,omitempty"`
	Description  string `json:"description,omitempty"`
	IsPartOf     any    `json:"isPartOf,omitempty"`
	PrimaryImage any    `json:"primaryImageOfPage,omitempty"`
}

func NewWebPage(name string) WebPage {
	return WebPage{Context: SchemaOrg, Type: "WebPage", Name: name}
}

type Article struct {
	Context          string `json:"@context,omitempty"`
	Type             string `json:"@type,omitempty"`
	Headline         string `json:"headline,omitempty"`
	Description      string `json:"description,omitempty"`
	URL              string `json:"url,omitempty"`
	Image            any    `json:"image,omitempty"`
	Author           any    `json:"author,omitempty"`
	Publisher        any    `json:"publisher,omitempty"`
	DatePublished    string `json:"datePublished,omitempty"`
	DateModified     string `json:"dateModified,omitempty"`
	MainEntityOfPage any    `json:"mainEntityOfPage,omitempty"`
}

func NewArticle(headline string) Article {
	return Article{Context: SchemaOrg, Type: "Article", Headline: headline}
}

type ImageObject struct {
	Context string `json:"@context,omitempty"`
	Type    string `json:"@type,omitempty"`
	URL     string `json:"url,omitempty"`
	Width   int    `json:"width,omitempty"`
	Height  int    `json:"height,omitempty"`
}

func NewImageObject(url string) ImageObject {
	return ImageObject{Context: SchemaOrg, Type: "ImageObject", URL: url}
}

type BreadcrumbList struct {
	Context string     `json:"@context,omitempty"`
	Type    string     `json:"@type,omitempty"`
	Items   []ListItem `json:"itemListElement,omitempty"`
}

func NewBreadcrumbList(items ...ListItem) BreadcrumbList {
	return BreadcrumbList{Context: SchemaOrg, Type: "BreadcrumbList", Items: items}
}

type ListItem struct {
	Type     string `json:"@type,omitempty"`
	Position int    `json:"position,omitempty"`
	Name     string `json:"name,omitempty"`
	Item     string `json:"item,omitempty"`
}

func NewListItem(position int, name, item string) ListItem {
	return ListItem{Type: "ListItem", Position: position, Name: name, Item: item}
}
