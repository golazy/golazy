package browser

import "testing"

func TestGetHTML(t *testing.T) {

	_, err := GetHTML("https://example.com")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetArticle(t *testing.T) {
	article, err := GetArticle("https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	if article.Title != "Example Domain" {
		t.Fatal(article.Title)
	}
}
